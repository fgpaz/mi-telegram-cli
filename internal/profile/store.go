package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"
)

var (
	ErrProfileAlreadyExists = errors.New("profile already exists")
	ErrProfileNotFound      = errors.New("profile not found")
	ErrProfileLocked        = errors.New("profile locked")
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

type Store struct {
	baseRoot string
	now      func() time.Time
}

type Lock struct {
	path string
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

	return &Lock{path: path}, nil
}

func (l *Lock) Release() error {
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
