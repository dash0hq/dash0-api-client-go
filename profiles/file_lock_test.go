package profiles

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestAcquireProfileLock_AcquireAndRelease(t *testing.T) {
	dir := t.TempDir()
	release, err := acquireProfileLock(context.Background(), dir)
	if err != nil {
		t.Fatalf("first acquire failed: %v", err)
	}
	release()

	// Re-acquiring after release should succeed.
	release2, err := acquireProfileLock(context.Background(), dir)
	if err != nil {
		t.Fatalf("second acquire failed: %v", err)
	}
	release2()

	// Sentinel file should exist at the expected path.
	if _, err := os.Stat(filepath.Join(dir, profileLockFileName)); err != nil {
		t.Errorf("expected lock sentinel file to exist: %v", err)
	}
}

func TestAcquireProfileLock_CreatesConfigDirWhenMissing(t *testing.T) {
	parent := t.TempDir()
	configDir := filepath.Join(parent, "missing", "subdir")
	release, err := acquireProfileLock(context.Background(), configDir)
	if err != nil {
		t.Fatalf("acquire failed when configDir does not yet exist: %v", err)
	}
	defer release()

	info, err := os.Stat(configDir)
	if err != nil {
		t.Fatalf("expected configDir to be created: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("expected configDir to be a directory")
	}
}

func TestAcquireProfileLock_CancelledContextDoesNotBlockForever(t *testing.T) {
	dir := t.TempDir()

	// Hold the lock with one acquirer; a second acquirer using an
	// already-cancelled context must return promptly rather than block.
	release, err := acquireProfileLock(context.Background(), dir)
	if err != nil {
		t.Fatalf("primary acquire failed: %v", err)
	}
	defer release()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	deadline := time.After(2 * time.Second)
	done := make(chan error, 1)
	go func() {
		r, err := acquireProfileLock(ctx, dir)
		if err == nil {
			r()
		}
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected an error when context is already cancelled, got nil")
		}
	case <-deadline:
		t.Fatal("acquireProfileLock blocked despite cancelled context")
	}
}

func TestAcquireProfileLock_SerializesContending(t *testing.T) {
	dir := t.TempDir()

	const goroutines = 8
	var inFlight, peak atomic.Int32
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			release, err := acquireProfileLock(context.Background(), dir)
			if err != nil {
				t.Errorf("acquire failed: %v", err)
				return
			}
			n := inFlight.Add(1)
			for {
				p := peak.Load()
				if n <= p || peak.CompareAndSwap(p, n) {
					break
				}
			}
			// Hold the lock briefly so contention is observable.
			time.Sleep(5 * time.Millisecond)
			inFlight.Add(-1)
			release()
		}()
	}
	wg.Wait()

	if got := peak.Load(); got > 1 {
		t.Errorf("peak concurrent holders = %d, want <= 1 (lock failed to serialize)", got)
	}
}
