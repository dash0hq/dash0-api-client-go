package profiles

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

// profileLockFileName is the sentinel file used for OS-level locking.
// Storing the lock as a dedicated file (rather than the profile file itself)
// avoids interaction with the temp-file + rename pattern used by
// [writeFileAtomic] — renaming over the profile file would invalidate any
// flock held on the original inode.
const profileLockFileName = ".profile-lock"

// profileLockPollInterval is how often [acquireProfileLock] retries when the
// lock is held by another process.
const profileLockPollInterval = 50 * time.Millisecond

// acquireProfileLock acquires an exclusive OS-level lock on a sentinel file in
// configDir, blocking until the lock is granted or ctx is cancelled.
// The returned release function must be called to release the lock.
//
// Locking is delegated to [flock] which provides a cross-platform
// implementation (flock on Unix, LockFileEx on Windows).
// The lock prevents two CLI processes sharing a config directory from
// concurrently refreshing OAuth tokens and clobbering each other's rotated
// refresh token.
func acquireProfileLock(ctx context.Context, configDir string) (func(), error) {
	if err := os.MkdirAll(configDir, configDirMode); err != nil {
		return nil, fmt.Errorf("failed to create config directory %s: %w", configDir, err)
	}
	lockPath := filepath.Join(configDir, profileLockFileName)
	lock := flock.New(lockPath)
	locked, err := lock.TryLockContext(ctx, profileLockPollInterval)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire profile lock %s: %w", lockPath, err)
	}
	if !locked {
		return nil, fmt.Errorf("failed to acquire profile lock %s: context cancelled", lockPath)
	}
	return func() {
		_ = lock.Unlock()
	}, nil
}
