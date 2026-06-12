package profiles

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

// profileLockFileName is the sentinel file used for OS-level locking of the
// profiles file.
// Storing the lock as a dedicated file (rather than the profile file itself)
// avoids interaction with the temp-file + rename pattern used by
// [writeFileAtomic] — renaming over the profile file would invalidate any
// flock held on the original inode.
const profileLockFileName = ".profile-lock"

// oauthClientsLockFileName is the sentinel file used for OS-level locking of
// the oauth-clients.json file.
// A separate sentinel from [profileLockFileName] keeps profile operations and
// dynamic-client-registration operations independent: a process refreshing a
// token need not wait for a sibling process registering a new client.
const oauthClientsLockFileName = ".oauth-clients-lock"

// profileLockPollInterval is how often [acquireProfileLock] retries when the
// lock is held by another process.
// 50ms balances responsiveness against syscall pressure on a hot lock; under a
// long-lived context this becomes a busy poll, so callers should bound the
// context.
const profileLockPollInterval = 50 * time.Millisecond

// acquireProfileLock acquires the profile-file lock; see [acquireLock].
func acquireProfileLock(ctx context.Context, configDir string) (func(), error) {
	return acquireLock(ctx, configDir, profileLockFileName)
}

// acquireOAuthClientsLock acquires the oauth-clients-file lock; see [acquireLock].
func acquireOAuthClientsLock(ctx context.Context, configDir string) (func(), error) {
	return acquireLock(ctx, configDir, oauthClientsLockFileName)
}

// acquireLock acquires an exclusive OS-level lock on the named sentinel file
// in configDir, blocking until the lock is granted or ctx is cancelled.
// The returned release function must be called to release the lock.
//
// Locking is delegated to [flock] which provides a cross-platform
// implementation (flock on Unix, LockFileEx on Windows).
// The lock prevents two CLI processes sharing a config directory from
// concurrently mutating the file the sentinel guards.
func acquireLock(ctx context.Context, configDir, sentinelName string) (func(), error) {
	if err := os.MkdirAll(configDir, configDirMode); err != nil {
		return nil, fmt.Errorf("failed to create config directory %s: %w", configDir, err)
	}
	lockPath := filepath.Join(configDir, sentinelName)
	lock := flock.New(lockPath)
	if _, err := lock.TryLockContext(ctx, profileLockPollInterval); err != nil {
		return nil, fmt.Errorf("failed to acquire lock %s: %w", lockPath, err)
	}
	return func() {
		_ = lock.Unlock()
	}, nil
}
