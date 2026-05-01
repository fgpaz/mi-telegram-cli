package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var (
	ErrProfileAlreadyExists = errors.New("profile already exists")
	ErrProfileNotFound      = errors.New("profile not found")
	ErrProfileLocked        = errors.New("profile locked")
	ErrQueueTimeout         = errors.New("queue timeout")
	ErrDaemonLeaseDenied    = errors.New("daemon lease denied")
	ErrDaemonLeaseExpired   = errors.New("daemon lease expired")
)

type AuthorizationStatus string

const (
	AuthorizationUnauthorized AuthorizationStatus = "Unauthorized"
	AuthorizationPendingCode  AuthorizationStatus = "PendingCode"
	AuthorizationAuthorized   AuthorizationStatus = "Authorized"
	AuthorizationLoggedOut    AuthorizationStatus = "LoggedOut"
)

type ProfileStatus string

const (
	StatusCreated    ProfileStatus = "Created"
	StatusConfigured ProfileStatus = "Configured"
)

type Profile struct {
	ID           string        `json:"profileId"`
	DisplayName  string        `json:"displayName"`
	StorageRoot  string        `json:"storageRoot"`
	CreatedAtUTC time.Time     `json:"createdAtUtc"`
	Status       ProfileStatus `json:"status"`
}

type AuthState struct {
	ProfileID           string              `json:"profileId"`
	AuthorizationStatus AuthorizationStatus `json:"authorizationStatus"`
	AuthorizedAtUTC     *time.Time          `json:"authorizedAtUtc,omitempty"`
	LastCheckedAtUTC    *time.Time          `json:"lastCheckedAtUtc,omitempty"`
	LogoutAtUTC         *time.Time          `json:"logoutAtUtc,omitempty"`
}

type ProfileView struct {
	Profile
	AuthorizationStatus AuthorizationStatus `json:"authorizationStatus"`
}

type lockMetadata struct {
	ProfileID     string    `json:"profileId"`
	PID           int       `json:"pid"`
	AcquiredAtUTC time.Time `json:"acquiredAtUtc"`
}

type leaseMetadata struct {
	ProfileID     string    `json:"profileId"`
	PID           int       `json:"pid"`
	Operation     string    `json:"operation"`
	LeaseID       string    `json:"leaseId"`
	AcquiredAtUTC time.Time `json:"acquiredAtUtc"`
	ExpiresAtUTC  time.Time `json:"expiresAtUtc"`
}

type Store struct {
	baseRoot string
	now      func() time.Time
}

type Lock struct {
	path       string
	queuePath  string
	acquiredAt time.Time
}

type Lease struct {
	path  string
	id    string
	store *Store
}

func NewStore(baseRoot string, now func() time.Time) *Store {
	if now == nil {
		now = time.Now().UTC
	}

	return &Store{
		baseRoot: baseRoot,
		now:      now,
	}
}

func (s *Store) Create(id, displayName, storageRootOverride string) (Profile, error) {
	if id == "" {
		return Profile{}, fmt.Errorf("empty profile id")
	}

	root := s.profileRoot(id, storageRootOverride)
	profilePath := filepath.Join(root, "profile.json")
	if _, err := os.Stat(profilePath); err == nil {
		return Profile{}, ErrProfileAlreadyExists
	}

	if err := os.MkdirAll(root, 0o700); err != nil {
		return Profile{}, err
	}

	profile := Profile{
		ID:           id,
		DisplayName:  displayName,
		StorageRoot:  root,
		CreatedAtUTC: s.now(),
		Status:       StatusCreated,
	}

	if err := writeJSON(profilePath, profile); err != nil {
		_ = os.RemoveAll(root)
		return Profile{}, err
	}
	if err := writeJSON(filepath.Join(root, "auth-state.json"), AuthState{
		ProfileID:           id,
		AuthorizationStatus: AuthorizationUnauthorized,
	}); err != nil {
		_ = os.RemoveAll(root)
		return Profile{}, err
	}

	return profile, nil
}

func (s *Store) Get(id string) (ProfileView, error) {
	profile, err := s.readProfile(id)
	if err != nil {
		return ProfileView{}, err
	}

	auth, err := s.LoadAuthState(id)
	if err != nil {
		return ProfileView{}, err
	}

	return ProfileView{
		Profile:             profile,
		AuthorizationStatus: auth.AuthorizationStatus,
	}, nil
}

func (s *Store) List() ([]ProfileView, error) {
	profilesDir := filepath.Join(s.baseRoot, "profiles")
	if _, err := os.Stat(profilesDir); errors.Is(err, fs.ErrNotExist) {
		return []ProfileView{}, nil
	}

	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		return nil, err
	}

	items := make([]ProfileView, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		view, err := s.Get(entry.Name())
		if err != nil {
			return nil, err
		}
		items = append(items, view)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})

	return items, nil
}

func (s *Store) SaveAuthState(state AuthState) error {
	if _, err := s.readProfile(state.ProfileID); err != nil {
		return err
	}

	if state.AuthorizationStatus == "" {
		state.AuthorizationStatus = AuthorizationUnauthorized
	}

	return writeJSON(filepath.Join(s.profileRoot(state.ProfileID, ""), "auth-state.json"), state)
}

func (s *Store) LoadAuthState(id string) (AuthState, error) {
	if _, err := s.readProfile(id); err != nil {
		return AuthState{}, err
	}

	path := filepath.Join(s.profileRoot(id, ""), "auth-state.json")
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return AuthState{
			ProfileID:           id,
			AuthorizationStatus: AuthorizationUnauthorized,
		}, nil
	}

	var state AuthState
	if err := readJSON(path, &state); err != nil {
		return AuthState{}, err
	}

	if state.AuthorizationStatus == "" {
		state.AuthorizationStatus = AuthorizationUnauthorized
	}

	return state, nil
}

func (s *Store) WriteSession(id string, data []byte) error {
	if _, err := s.readProfile(id); err != nil {
		return err
	}
	return os.WriteFile(s.sessionPath(id), data, 0o600)
}

func (s *Store) ReadSession(id string) ([]byte, error) {
	if _, err := s.readProfile(id); err != nil {
		return nil, err
	}
	return os.ReadFile(s.sessionPath(id))
}

func (s *Store) SessionExists(id string) (bool, error) {
	if _, err := s.readProfile(id); err != nil {
		return false, err
	}

	_, err := os.Stat(s.sessionPath(id))
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

func (s *Store) RemoveSession(id string) (bool, error) {
	if _, err := s.readProfile(id); err != nil {
		return false, err
	}

	err := os.Remove(s.sessionPath(id))
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

func (s *Store) Delete(id string) error {
	if _, err := s.readProfile(id); err != nil {
		return err
	}
	return os.RemoveAll(s.profileRoot(id, ""))
}

func (s *Store) AcquireLock(id string) (*Lock, error) {
	if _, err := s.readProfile(id); err != nil {
		return nil, err
	}

	path := filepath.Join(s.profileRoot(id, ""), "lock.json")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return nil, ErrProfileLocked
		}
		return nil, err
	}
	defer file.Close()

	payload, err := json.Marshal(lockMetadata{
		ProfileID:     id,
		PID:           os.Getpid(),
		AcquiredAtUTC: s.now(),
	})
	if err != nil {
		_ = os.Remove(path)
		return nil, err
	}

	if _, err := file.Write(payload); err != nil {
		_ = os.Remove(path)
		return nil, err
	}

	return &Lock{path: path, acquiredAt: s.now()}, nil
}

func (s *Store) AcquireQueuedLock(id string, timeout time.Duration) (*Lock, time.Duration, error) {
	if timeout <= 0 {
		lock, err := s.AcquireLock(id)
		return lock, 0, err
	}
	if _, err := s.readProfile(id); err != nil {
		return nil, 0, err
	}

	start := s.now()
	queuePath, err := s.createQueueTicket(id, start)
	if err != nil {
		return nil, 0, err
	}

	deadline := time.Now().Add(timeout)
	for {
		first, err := s.isFirstQueueTicket(id, queuePath)
		if err != nil {
			_ = os.Remove(queuePath)
			return nil, 0, err
		}
		if first {
			lock, err := s.AcquireLock(id)
			if err == nil {
				lock.queuePath = queuePath
				return lock, s.now().Sub(start), nil
			}
			if !errors.Is(err, ErrProfileLocked) {
				_ = os.Remove(queuePath)
				return nil, 0, err
			}
		}
		if time.Now().After(deadline) {
			_ = os.Remove(queuePath)
			return nil, s.now().Sub(start), ErrQueueTimeout
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func (s *Store) AcquireLease(id, operation string, ttl time.Duration) (*Lease, error) {
	if _, err := s.readProfile(id); err != nil {
		return nil, err
	}
	if ttl <= 0 {
		return nil, fmt.Errorf("lease ttl must be greater than zero")
	}

	path := filepath.Join(s.profileRoot(id, ""), "lease.json")
	now := s.now()
	if err := s.removeExpiredLease(path, now); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return nil, ErrDaemonLeaseDenied
		}
		return nil, err
	}
	defer file.Close()

	leaseID := fmt.Sprintf("%d-%d", now.UnixNano(), os.Getpid())
	payload, err := json.Marshal(leaseMetadata{
		ProfileID:     id,
		PID:           os.Getpid(),
		Operation:     strings.TrimSpace(operation),
		LeaseID:       leaseID,
		AcquiredAtUTC: now,
		ExpiresAtUTC:  now.Add(ttl),
	})
	if err != nil {
		_ = os.Remove(path)
		return nil, err
	}
	if _, err := file.Write(payload); err != nil {
		_ = os.Remove(path)
		return nil, err
	}

	return &Lease{path: path, id: leaseID, store: s}, nil
}

func (s *Store) createQueueTicket(id string, now time.Time) (string, error) {
	dir := filepath.Join(s.profileRoot(id, ""), "queue")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	for attempt := 0; attempt < 1000; attempt++ {
		name := fmt.Sprintf("%020d-%06d-%03d.ticket", now.UnixNano(), os.Getpid(), attempt)
		path := filepath.Join(dir, name)
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			if errors.Is(err, fs.ErrExist) {
				continue
			}
			return "", err
		}
		_, writeErr := fmt.Fprintf(file, `{"profileId":%q,"pid":%d,"createdAtUtc":%q}`, id, os.Getpid(), now.Format(time.RFC3339Nano))
		closeErr := file.Close()
		if writeErr != nil {
			_ = os.Remove(path)
			return "", writeErr
		}
		if closeErr != nil {
			_ = os.Remove(path)
			return "", closeErr
		}
		return path, nil
	}
	return "", fmt.Errorf("failed to allocate queue ticket")
}

func (s *Store) isFirstQueueTicket(id, ticketPath string) (bool, error) {
	dir := filepath.Join(s.profileRoot(id, ""), "queue")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	ticketName := filepath.Base(ticketPath)
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".ticket") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return len(names) > 0 && names[0] == ticketName, nil
}

func (s *Store) removeExpiredLease(path string, now time.Time) error {
	var current leaseMetadata
	if err := readJSON(path, &current); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	if now.After(current.ExpiresAtUTC) {
		return os.Remove(path)
	}
	return nil
}

func (l *Lock) Release() error {
	if l == nil || l.path == "" {
		return nil
	}

	err := os.Remove(l.path)
	if errors.Is(err, fs.ErrNotExist) {
		err = nil
	}
	if l.queuePath != "" {
		if queueErr := os.Remove(l.queuePath); err == nil && queueErr != nil && !errors.Is(queueErr, fs.ErrNotExist) {
			err = queueErr
		}
	}
	return err
}

func (l *Lease) Release() error {
	if l == nil || l.path == "" {
		return nil
	}
	err := os.Remove(l.path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

func (s *Store) SessionPath(id string) string {
	return s.sessionPath(id)
}

func (s *Store) profileRoot(id, override string) string {
	if override != "" {
		return override
	}
	return filepath.Join(s.baseRoot, "profiles", id)
}

func (s *Store) sessionPath(id string) string {
	return filepath.Join(s.profileRoot(id, ""), "session.bin")
}

func (s *Store) readProfile(id string) (Profile, error) {
	path := filepath.Join(s.profileRoot(id, ""), "profile.json")
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return Profile{}, ErrProfileNotFound
	}

	var profile Profile
	if err := readJSON(path, &profile); err != nil {
		return Profile{}, err
	}

	return profile, nil
}

func readJSON(path string, target any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}

func writeJSON(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o600)
}
