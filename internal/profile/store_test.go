package profile_test

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"mi-telegram-cli/internal/profile"
)

func fixedNow() time.Time {
	return time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
}

func TestStoreQueuedLockWaitsForActiveLock(t *testing.T) {
	root := t.TempDir()
	store := profile.NewStore(root, fixedNow)
	if _, err := store.Create("qa-dev", "QA Dev", ""); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	first, _, err := store.AcquireQueuedLock("qa-dev", time.Second)
	if err != nil {
		t.Fatalf("AcquireQueuedLock() first error = %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	acquired := make(chan error, 1)
	go func() {
		defer wg.Done()
		second, _, err := store.AcquireQueuedLock("qa-dev", time.Second)
		if err == nil {
			_ = second.Release()
		}
		acquired <- err
	}()

	time.Sleep(50 * time.Millisecond)
	if err := first.Release(); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	wg.Wait()

	if err := <-acquired; err != nil {
		t.Fatalf("second AcquireQueuedLock() error = %v", err)
	}
}

func TestStoreQueuedLockTimeout(t *testing.T) {
	root := t.TempDir()
	store := profile.NewStore(root, fixedNow)
	if _, err := store.Create("qa-dev", "QA Dev", ""); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	first, err := store.AcquireLock("qa-dev")
	if err != nil {
		t.Fatalf("AcquireLock() error = %v", err)
	}
	defer func() { _ = first.Release() }()

	if _, _, err := store.AcquireQueuedLock("qa-dev", 30*time.Millisecond); !errors.Is(err, profile.ErrQueueTimeout) {
		t.Fatalf("AcquireQueuedLock() error = %v, want ErrQueueTimeout", err)
	}
}

func TestStoreLeaseAcquireReleaseAndExpiry(t *testing.T) {
	now := fixedNow()
	store := profile.NewStore(t.TempDir(), func() time.Time { return now })
	if _, err := store.Create("qa-dev", "QA Dev", ""); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	lease, err := store.AcquireLease("qa-dev", "auth login", time.Minute)
	if err != nil {
		t.Fatalf("AcquireLease() error = %v", err)
	}
	if _, err := store.AcquireLease("qa-dev", "auth login", time.Minute); !errors.Is(err, profile.ErrDaemonLeaseDenied) {
		t.Fatalf("AcquireLease() denied error = %v, want ErrDaemonLeaseDenied", err)
	}
	if err := lease.Release(); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	if lease, err := store.AcquireLease("qa-dev", "auth login", time.Minute); err != nil {
		t.Fatalf("AcquireLease() after release error = %v", err)
	} else {
		_ = lease.Release()
	}

	if _, err := store.AcquireLease("qa-dev", "auth login", time.Minute); err != nil {
		t.Fatalf("AcquireLease() before expiry setup error = %v", err)
	}
	now = now.Add(2 * time.Minute)
	if lease, err := store.AcquireLease("qa-dev", "auth login", time.Minute); err != nil {
		t.Fatalf("AcquireLease() after expiry error = %v", err)
	} else {
		_ = lease.Release()
	}
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
		ProfileID:           "qa-dev",
		AuthorizationStatus: profile.AuthorizationAuthorized,
		AuthorizedAtUTC:     ptrTime(fixedNow()),
		LastCheckedAtUTC:    ptrTime(fixedNow()),
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
