package profiles

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// OAuthClientsFileName is the file storing cached dynamic client registrations.
const OAuthClientsFileName = "oauth-clients.json"

// OAuthClientRecord is a cached dynamic client registration (RFC 7591).
type OAuthClientRecord struct {
	ClientID                string `json:"clientId"`
	RegistrationAccessToken string `json:"registrationAccessToken,omitempty"`
	RedirectURI             string `json:"redirectUri"`
}

// OAuthClientStore caches dynamic client registrations keyed by canonical
// API URL. The file is created mode 0600 because RegistrationAccessToken
// (RFC 7592) is a long-lived management credential.
type OAuthClientStore struct {
	configDir string
}

// oauthClientsFile is the on-disk layout: clients keyed by canonical API URL.
type oauthClientsFile struct {
	Clients map[string]OAuthClientRecord `json:"clients"`
}

// NewOAuthClientStore mirrors [NewStore]: explicit [WithConfigDir] >
// DASH0_CONFIG_DIR > ~/.dash0/.
func NewOAuthClientStore(opts ...StoreOption) (*OAuthClientStore, error) {
	cfg := &storeConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.configDir != "" {
		return &OAuthClientStore{configDir: cfg.configDir}, nil
	}
	if d := os.Getenv(EnvConfigDir); d != "" {
		return &OAuthClientStore{configDir: d}, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to determine home directory: %w", err)
	}
	return &OAuthClientStore{configDir: filepath.Join(home, ConfigDirName)}, nil
}

// CanonicalAPIURL normalises an API URL into the key form used by
// [OAuthClientStore]. It lowercases the scheme and host, trims trailing
// slashes from the path, and drops query and fragment components.
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
	u.Host = strings.ToLower(u.Host)
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

// Get returns the cached record for apiURL, or (zero, false, nil) on miss.
// A missing or unreadable file is a miss, not an error.
func (s *OAuthClientStore) Get(apiURL string) (OAuthClientRecord, bool, error) {
	key, err := CanonicalAPIURL(apiURL)
	if err != nil {
		return OAuthClientRecord{}, false, err
	}
	f := s.load()
	rec, ok := f.Clients[key]
	return rec, ok, nil
}

// Put upserts a record for apiURL.
func (s *OAuthClientStore) Put(apiURL string, rec OAuthClientRecord) error {
	key, err := CanonicalAPIURL(apiURL)
	if err != nil {
		return err
	}
	f := s.load()
	f.Clients[key] = rec
	return s.save(f)
}

// Delete removes the record for apiURL. A miss is a no-op.
func (s *OAuthClientStore) Delete(apiURL string) error {
	key, err := CanonicalAPIURL(apiURL)
	if err != nil {
		return err
	}
	f := s.load()
	if _, ok := f.Clients[key]; !ok {
		return nil
	}
	delete(f.Clients, key)
	return s.save(f)
}

// load reads the on-disk file. A missing, unreadable, or corrupt file
// yields an empty, ready-to-mutate result -- the next save overwrites cleanly.
func (s *OAuthClientStore) load() oauthClientsFile {
	empty := oauthClientsFile{Clients: map[string]OAuthClientRecord{}}
	data, err := os.ReadFile(filepath.Join(s.configDir, OAuthClientsFileName))
	if err != nil {
		return empty
	}
	var f oauthClientsFile
	if err := json.Unmarshal(data, &f); err != nil {
		return empty
	}
	if f.Clients == nil {
		f.Clients = map[string]OAuthClientRecord{}
	}
	return f
}

// save writes the file atomically-enough for a single-user CLI cache:
// directory created 0700 if missing, file written 0600.
func (s *OAuthClientStore) save(f oauthClientsFile) error {
	if err := os.MkdirAll(s.configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", s.configDir, err)
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal oauth clients: %w", err)
	}
	path := filepath.Join(s.configDir, OAuthClientsFileName)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write oauth clients file %s: %w", path, err)
	}
	return nil
}
