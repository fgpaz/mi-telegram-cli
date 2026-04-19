package profile_test

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"mi-telegram-cli/internal/profile"
)

func fixedNow() time.Time {
	return time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
}

func TestStoreLifecycleAndAuthProjection(t *testing.T) {
	root := t.TempDir()
	store := profile.NewStore(root, fixedNow)

	created, err := store.Create("qa-dev", "QA Dev", "")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if created.ID != "qa-dev" {
		t.Fatalf("Create() id = %q, want %q", created.ID, "qa-dev")
	}

	view, err := store.Get("qa-dev")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if view.AuthorizationStatus != profile.AuthorizationUnauthorized {
		t.Fatalf("Get() auth status = %q, want %q", view.AuthorizationStatus, profile.AuthorizationUnauthorized)
	}

	if err := store.SaveAuthState(profile.AuthState{
		ProfileID:            "qa-dev",
		AuthorizationStatus: profile.AuthorizationAuthorized,
		AuthorizedAtUTC:      ptrTime(fixedNow()),
		LastCheckedAtUTC:     ptrTime(fixedNow()),
	}); err != nil {
		t.Fatalf("SaveAuthState() error = %v", err)
	}

	if err := store.WriteSession("qa-dev", []byte("session-bytes")); err != nil {
		t.Fatalf("WriteSession() error = %v", err)
	}

	views, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(views) != 1 {
		t.Fatalf("List() len = %d, want 1", len(views))
	}

	if views[0].AuthorizationStatus != profile.AuthorizationAuthorized {
		t.Fatalf("List() auth status = %q, want %q", views[0].AuthorizationStatus, profile.AuthorizationAuthorized)
	}

	sessionPath := filepath.Join(created.StorageRoot, "session.bin")
	if got, err := store.ReadSession("qa-dev"); err != nil || string(got) != "session-bytes" {
		t.Fatalf("ReadSession() = %q, %v, want %q, nil from %s", string(got), err, "session-bytes", sessionPath)
	}

	if err := store.Delete("qa-dev"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if _, err := store.Get("qa-dev"); !errors.Is(err, profile.ErrProfileNotFound) {
		t.Fatalf("Get() error after delete = %v, want ErrProfileNotFound", err)
	}
}

func TestStoreLockingRejectsConcurrentAcquire(t *testing.T) {
	root := t.TempDir()
	store := profile.NewStore(root, fixedNow)

	if _, err := store.Create("qa-dev", "QA Dev", ""); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	lock, err := store.AcquireLock("qa-dev")
	if err != nil {
		t.Fatalf("AcquireLock() first error = %v", err)
	}
	defer func() {
		_ = lock.Release()
	}()

	if _, err := store.AcquireLock("qa-dev"); !errors.Is(err, profile.ErrProfileLocked) {
		t.Fatalf("AcquireLock() second error = %v, want ErrProfileLocked", err)
	}
}

func ptrTime(v time.Time) *time.Time {
	return &v
}
