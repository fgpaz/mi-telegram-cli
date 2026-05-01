package audit_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"mi-telegram-cli/internal/audit"
)

func TestRecorderAppendsRedactedEventAndSummarizes(t *testing.T) {
	now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	recorder := audit.NewRecorder(t.TempDir(), func() time.Time { return now })

	event := audit.Event{
		StartedAtUTC:   now.Add(-1500 * time.Millisecond),
		CompletedAtUTC: now,
		Operation:      "messages send",
		Profile:        "qa-dev",
		ProjectCwd:     `C:\repos\mios\multi-tedi`,
		PID:            123,
		DaemonPID:      456,
		QueueMs:        25,
		DurationMs:     1500,
		OK:             false,
		ExitCode:       1,
		ErrorCode:      "QueueTimeout",
		ErrorKind:      "coordination",
		PeerQuery:      "@multi_tedi_dev_bot",
	}
	if err := recorder.Append(event); err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	var buf bytes.Buffer
	if err := recorder.Export(&buf, audit.SummaryFilter{ErrorsOnly: true}); err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	raw := buf.String()
	for _, forbidden := range []string{"hello bot", "caption", "12345", "api_hash", "session.bin", "--file"} {
		if strings.Contains(raw, forbidden) {
			t.Fatalf("export leaked forbidden value %q: %s", forbidden, raw)
		}
	}

	var exported audit.Event
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &exported); err != nil {
		t.Fatalf("json.Unmarshal(export) error = %v", err)
	}
	if exported.EventVersion != audit.EventVersion {
		t.Fatalf("EventVersion = %q, want %q", exported.EventVersion, audit.EventVersion)
	}
	if exported.ErrorCode != "QueueTimeout" || exported.PeerQuery != "@multi_tedi_dev_bot" {
		t.Fatalf("exported event = %+v", exported)
	}

	summary, err := recorder.Summarize(audit.SummaryFilter{Profile: "qa-dev"})
	if err != nil {
		t.Fatalf("Summarize() error = %v", err)
	}
	if summary.Count != 1 || summary.Errors != 1 {
		t.Fatalf("summary counts = %+v, want one error", summary)
	}
	if got := summary.ByOperation["messages send"].QueueP95Ms; got != 25 {
		t.Fatalf("QueueP95Ms = %d, want 25", got)
	}
}
