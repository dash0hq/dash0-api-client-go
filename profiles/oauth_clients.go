package profiles

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// OAuthClientsFileName is the file storing cached dynamic client registrations.
const OAuthClientsFileName = "oauth-clients.json"

// ErrOAuthClientsFileCorrupt is wrapped into the error returned by
// [OAuthClientStore.Get] when the on-disk file exists but is unparseable.
// [OAuthClientStore.Put] and [OAuthClientStore.Delete] handle corruption
// internally by quarantining the bad file (see [OAuthClientStore.load]),
// rather than failing.
var ErrOAuthClientsFileCorrupt = errors.New("oauth-clients.json is corrupt")

// OAuthClientRecord is a cached dynamic client registration (RFC 7591).
type OAuthClientRecord struct {
	ClientID                string `json:"clientId"`
	RegistrationAccessToken string `json:"registrationAccessToken,omitempty"`
	RedirectURI             string `json:"redirectUri"`
}

// OAuthClientStore caches dynamic client registrations keyed by canonical
// API URL.
// The file is created mode 0600 because RegistrationAccessToken (RFC 7592)
// is a long-lived management credential.
//
// In-process mutation (Get/Put/Delete) is serialized by mu; cross-process
// mutation is serialized via an OS-level advisory lock on
// [oauthClientsLockFileName] so two CLI invocations sharing the same config
// directory cannot lose-update each other's RegistrationAccessToken values.
type OAuthClientStore struct {
	configDir string
	mu        sync.Mutex
}

// oauthClientsFile is the on-disk layout: clients keyed by canonical API URL.
type oauthClientsFile struct {
	Clients map[string]OAuthClientRecord `json:"clients"`
}

// NewOAuthClientStore mirrors [NewStore]: explicit [WithConfigDir] >
// DASH0_CONFIG_DIR > ~/.dash0/.
// Same best-effort hygiene as [NewStore]: best-effort dir-mode tightening and
// stale-tempfile cleanup when the directory already exists.
func NewOAuthClientStore(opts ...StoreOption) (*OAuthClientStore, error) {
	configDir, err := resolveConfigDir(opts)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(configDir); err == nil {
		_ = ensureConfigDirMode(configDir)
		cleanupStaleTempFiles(configDir)
	}
	return &OAuthClientStore{configDir: configDir}, nil
}

// CanonicalAPIURL normalises an API URL into the key form used by
// [OAuthClientStore].
// It lowercases the scheme and host, strips userinfo (so credentials never
// reach the on-disk key), drops the port when it matches the scheme default
// (so "https://api.example.com" and "https://api.example.com:443" share one
// cache entry), normalises the path with [path.Clean], trims a trailing slash
// from the path, and drops query and fragment components.
// Returns an error when the input is not a parseable absolute URL.
func CanonicalAPIURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid API URL %q: %w", raw, err)
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("invalid API URL %q: missing scheme or host", raw)
	}
	u.Scheme = strings.ToLower(u.Scheme)
	// Strip userinfo. Credentials in an API URL would otherwise be persisted
	// verbatim into the on-disk cache key, leaking them into oauth-clients.json.
	u.User = nil
	// u.Hostname() strips IPv6 brackets, so re-bracket when the host contains
	// a colon (a literal IPv6 address; DNS hostnames cannot contain colons).
	// Without this, a URL like https://[::1]:8443/v1 would canonicalize to
	// https://::1:8443/v1, which the next url.Parse round-trip cannot recover.
	host := strings.ToLower(u.Hostname())
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	if port := u.Port(); port != "" && !isDefaultPort(u.Scheme, port) {
		u.Host = host + ":" + port
	} else {
		u.Host = host
	}
	if u.Path != "" {
		// path.Clean collapses ".", "..", and duplicate "/" segments. We also
		// strip a trailing "/" so "/v1" and "/v1/" collapse.
		u.Path = strings.TrimSuffix(path.Clean(u.Path), "/")
		if u.Path == "." {
			u.Path = ""
		}
	}
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

// isDefaultPort reports whether port is the default for scheme.
// Only the schemes the library accepts (http, https) are recognised.
func isDefaultPort(scheme, port string) bool {
	switch scheme {
	case "https":
		return port == "443"
	case "http":
		return port == "80"
	}
	return false
}

// Get returns the cached record for apiURL, or (zero, false, nil) on miss.
// A missing file is a miss, not an error.
// A corrupt file returns [ErrOAuthClientsFileCorrupt] wrapped with the
// quarantine path so the operator can decide whether to investigate.
func (s *OAuthClientStore) Get(apiURL string) (OAuthClientRecord, bool, error) {
	key, err := CanonicalAPIURL(apiURL)
	if err != nil {
		return OAuthClientRecord{}, false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	f, loadErr := s.load(false)
	if loadErr != nil {
		return OAuthClientRecord{}, false, loadErr
	}
	rec, ok := f.Clients[key]
	return rec, ok, nil
}

// Put upserts a record for apiURL.
// The read-modify-write sequence is guarded by both an in-process mutex and a
// cross-process advisory lock on [oauthClientsLockFileName] so two CLI
// invocations sharing the same config directory cannot lose-update each
// other's RegistrationAccessToken values.
// If the on-disk file is corrupt it is quarantined (renamed with a timestamp
// suffix) before the new record is written; the previous registrations are
// lost from the active file but preserved on disk for manual recovery.
func (s *OAuthClientStore) Put(apiURL string, rec OAuthClientRecord) error {
	key, err := CanonicalAPIURL(apiURL)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.withFileLock(context.Background(), func() error {
		f, err := s.load(true)
		if err != nil {
			return err
		}
		f.Clients[key] = rec
		return s.save(f)
	})
}

// Delete removes the record for apiURL.
// A miss is a no-op.
// Cross-process serialization mirrors [OAuthClientStore.Put].
func (s *OAuthClientStore) Delete(apiURL string) error {
	key, err := CanonicalAPIURL(apiURL)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.withFileLock(context.Background(), func() error {
		f, err := s.load(true)
		if err != nil {
			return err
		}
		if _, ok := f.Clients[key]; !ok {
			return nil
		}
		delete(f.Clients, key)
		return s.save(f)
	})
}

// withFileLock acquires the cross-process OAuth-clients advisory lock for the
// duration of fn.
func (s *OAuthClientStore) withFileLock(ctx context.Context, fn func() error) error {
	release, err := acquireOAuthClientsLock(ctx, s.configDir)
	if err != nil {
		return err
	}
	defer release()
	return fn()
}

// load reads the on-disk file.
// A missing file always yields an empty, ready-to-mutate result and a nil
// error.
// A corrupt file:
//   - When quarantineCorrupt is true (write paths): the corrupt file is
//     renamed to oauth-clients.json.corrupt-<timestamp> and an empty result
//     is returned so the caller can proceed with the new write.
//     The previous registrations are lost from the active file but preserved
//     on disk for manual recovery.
//   - When quarantineCorrupt is false (read paths): an empty result is
//     returned alongside [ErrOAuthClientsFileCorrupt] so the caller can
//     surface the problem rather than silently treat it as a miss.
func (s *OAuthClientStore) load(quarantineCorrupt bool) (oauthClientsFile, error) {
	empty := oauthClientsFile{Clients: map[string]OAuthClientRecord{}}
	path := filepath.Join(s.configDir, OAuthClientsFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return empty, nil
		}
		return empty, fmt.Errorf("failed to read oauth clients file: %w", err)
	}
	var f oauthClientsFile
	if err := json.Unmarshal(data, &f); err != nil {
		if quarantineCorrupt {
			quarantinePath := fmt.Sprintf("%s.corrupt-%d", path, time.Now().UnixNano())
			_ = os.Rename(path, quarantinePath)
			return empty, nil
		}
		return empty, fmt.Errorf("%w: %v", ErrOAuthClientsFileCorrupt, err)
	}
	if f.Clients == nil {
		f.Clients = map[string]OAuthClientRecord{}
	}
	return f, nil
}

// save writes the file via a temp file + os.Rename so a crash mid-write
// leaves either the previous contents intact or the new contents fully
// written.
// The directory is created 0700 and the file 0600 to protect the long-lived
// RegistrationAccessToken (RFC 7592).
func (s *OAuthClientStore) save(f oauthClientsFile) error {
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal oauth clients: %w", err)
	}
	p := filepath.Join(s.configDir, OAuthClientsFileName)
	return writeFileAtomic(s.configDir, p, data)
}
