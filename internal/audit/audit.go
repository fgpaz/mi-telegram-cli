package audit

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const EventVersion = "audit-jsonl-v1"

type Event struct {
	EventVersion   string    `json:"eventVersion"`
	EventID        string    `json:"eventId"`
	StartedAtUTC   time.Time `json:"startedAtUtc"`
	CompletedAtUTC time.Time `json:"completedAtUtc"`
	Operation      string    `json:"operation"`
	Profile        string    `json:"profile"`
	ProjectCwd     string    `json:"projectCwd,omitempty"`
	PID            int       `json:"pid"`
	DaemonPID      int       `json:"daemonPid,omitempty"`
	QueueMs        int64     `json:"queueMs"`
	DurationMs     int64     `json:"durationMs"`
	OK             bool      `json:"ok"`
	ExitCode       int       `json:"exitCode"`
	ErrorCode      string    `json:"errorCode,omitempty"`
	ErrorKind      string    `json:"errorKind,omitempty"`
	PeerQuery      string    `json:"peerQuery,omitempty"`
}

type Recorder struct {
	baseRoot string
	now      func() time.Time
}

type SummaryFilter struct {
	Since      time.Time
	Profile    string
	Operation  string
	ErrorsOnly bool
}

type Summary struct {
	Count       int                      `json:"count"`
	OK          int                      `json:"ok"`
	Errors      int                      `json:"errors"`
	ByOperation map[string]SummaryBucket `json:"byOperation"`
	ByProfile   map[string]SummaryBucket `json:"byProfile"`
	ByProject   map[string]SummaryBucket `json:"byProject"`
}

type SummaryBucket struct {
	Count         int   `json:"count"`
	Errors        int   `json:"errors"`
	QueueP50Ms    int64 `json:"queueP50Ms"`
	QueueP95Ms    int64 `json:"queueP95Ms"`
	DurationP50Ms int64 `json:"durationP50Ms"`
	DurationP95Ms int64 `json:"durationP95Ms"`
}

func NewRecorder(baseRoot string, now func() time.Time) *Recorder {
	if now == nil {
		now = time.Now().UTC
	}
	return &Recorder{baseRoot: baseRoot, now: now}
}

func NewEventID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(b[:])
}

func (r *Recorder) Append(event Event) error {
	if event.EventVersion == "" {
		event.EventVersion = EventVersion
	}
	if event.EventID == "" {
		event.EventID = NewEventID()
	}
	if event.CompletedAtUTC.IsZero() {
		event.CompletedAtUTC = r.now()
	}
	if event.DurationMs == 0 && !event.StartedAtUTC.IsZero() {
		event.DurationMs = event.CompletedAtUTC.Sub(event.StartedAtUTC).Milliseconds()
	}

	path := r.dailyPath(event.CompletedAtUTC)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	enc.SetEscapeHTML(false)
	return enc.Encode(event)
}

func (r *Recorder) Export(w io.Writer, filter SummaryFilter) error {
	events, err := r.ReadEvents(filter)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	for _, event := range events {
		if err := enc.Encode(event); err != nil {
			return err
		}
	}
	return nil
}

func (r *Recorder) Summarize(filter SummaryFilter) (Summary, error) {
	events, err := r.ReadEvents(filter)
	if err != nil {
		return Summary{}, err
	}
	s := Summary{
		ByOperation: map[string]SummaryBucket{},
		ByProfile:   map[string]SummaryBucket{},
		ByProject:   map[string]SummaryBucket{},
	}
	operations := map[string]*rawBucket{}
	profiles := map[string]*rawBucket{}
	projects := map[string]*rawBucket{}
	add := func(m map[string]*rawBucket, key string, event Event) {
		if strings.TrimSpace(key) == "" {
			key = "(empty)"
		}
		b := m[key]
		if b == nil {
			b = &rawBucket{}
			m[key] = b
		}
		b.count++
		if !event.OK {
			b.errors++
		}
		b.queue = append(b.queue, event.QueueMs)
		b.durations = append(b.durations, event.DurationMs)
	}
	for _, event := range events {
		s.Count++
		if event.OK {
			s.OK++
		} else {
			s.Errors++
		}
		add(operations, event.Operation, event)
		add(profiles, event.Profile, event)
		add(projects, event.ProjectCwd, event)
	}
	for k, b := range operations {
		s.ByOperation[k] = summarizeBucket(b)
	}
	for k, b := range profiles {
		s.ByProfile[k] = summarizeBucket(b)
	}
	for k, b := range projects {
		s.ByProject[k] = summarizeBucket(b)
	}
	return s, nil
}

func (r *Recorder) ReadEvents(filter SummaryFilter) ([]Event, error) {
	dir := filepath.Join(r.baseRoot, "audit")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []Event{}, nil
		}
		return nil, err
	}
	var events []Event
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "events-") || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		file, err := os.Open(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			var event Event
			if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
				_ = file.Close()
				return nil, err
			}
			if include(event, filter) {
				events = append(events, event)
			}
		}
		if err := scanner.Err(); err != nil {
			_ = file.Close()
			return nil, err
		}
		if err := file.Close(); err != nil {
			return nil, err
		}
	}
	sort.Slice(events, func(i, j int) bool {
		return events[i].StartedAtUTC.Before(events[j].StartedAtUTC)
	})
	return events, nil
}

func (r *Recorder) dailyPath(day time.Time) string {
	return filepath.Join(r.baseRoot, "audit", "events-"+day.UTC().Format("2006-01-02")+".jsonl")
}

func include(event Event, filter SummaryFilter) bool {
	if !filter.Since.IsZero() && event.StartedAtUTC.Before(filter.Since) {
		return false
	}
	if filter.Profile != "" && event.Profile != filter.Profile {
		return false
	}
	if filter.Operation != "" && event.Operation != filter.Operation {
		return false
	}
	if filter.ErrorsOnly && event.OK {
		return false
	}
	return true
}

func summarizeBucket(b *rawBucket) SummaryBucket {
	return SummaryBucket{
		Count:         b.count,
		Errors:        b.errors,
		QueueP50Ms:    percentile(b.queue, 50),
		QueueP95Ms:    percentile(b.queue, 95),
		DurationP50Ms: percentile(b.durations, 50),
		DurationP95Ms: percentile(b.durations, 95),
	}
}

type rawBucket struct {
	count     int
	errors    int
	queue     []int64
	durations []int64
}

func percentile(values []int64, p int) int64 {
	if len(values) == 0 {
		return 0
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	index := (len(values)*p + 99) / 100
	if index < 1 {
		index = 1
	}
	if index > len(values) {
		index = len(values)
	}
	return values[index-1]
}
