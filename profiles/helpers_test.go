package profiles

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// newTestStore returns a fresh Store rooted at t.TempDir().
// It takes a [testing.TB] so benchmarks can use it too.
func newTestStore(t testing.TB) (*Store, string) {
	t.Helper()
	dir := t.TempDir()
	svc, err := NewStore(WithConfigDir(dir))
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	return svc, dir
}

// createTestProfilesFile seeds the profiles file in configDir.
func createTestProfilesFile(t *testing.T, configDir string, profiles []Profile) {
	t.Helper()
	profilesFile := ProfilesFile{Profiles: profiles}
	data, err := json.MarshalIndent(profilesFile, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal profiles: %v", err)
	}
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, ProfilesFileName), data, 0644); err != nil {
		t.Fatalf("Failed to write profiles file: %v", err)
	}
}

// setActiveProfile writes the active-profile pointer file directly, bypassing
// validation in setActiveProfileLocked. Useful for tests that need to seed a
// specific (possibly stale) pointer.
func setActiveProfile(t *testing.T, configDir, profileName string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(configDir, ActiveProfileFileName), []byte(profileName), 0644); err != nil {
		t.Fatalf("Failed to write active profile: %v", err)
	}
}

// parsePostForm parses an HTTP request body as a urlencoded form and returns
// the first value for each key as a flat map.
func parsePostForm(r *http.Request) (map[string]string, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}
	m := make(map[string]string, len(r.PostForm))
	for k, v := range r.PostForm {
		if len(v) > 0 {
			m[k] = v[0]
		}
	}
	return m, nil
}

// tokenServerResponse configures the mock token endpoint response.
type tokenServerResponse struct {
	AccessToken  string
	ExpiresIn    int64
	RefreshToken *string
}

// newTokenServer creates an httptest.Server that responds to POST /oauth/token
// with the given token response.
// It also records the parsed form values from the most recent request into
// *lastRequest if non-nil.
func newTokenServer(t *testing.T, resp tokenServerResponse, lastRequest *map[string]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/oauth/token" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if lastRequest != nil {
			if m, err := parsePostForm(r); err == nil {
				*lastRequest = m
			}
		}
		w.Header().Set("Content-Type", "application/json")
		body := map[string]any{
			"access_token": resp.AccessToken,
			"token_type":   "Bearer",
			"expires_in":   resp.ExpiresIn,
		}
		if resp.RefreshToken != nil {
			body["refresh_token"] = *resp.RefreshToken
		}
		_ = json.NewEncoder(w).Encode(body)
	}))
}

// newRevokeServer creates an httptest.Server that responds to POST /oauth/revoke
// with 200 OK.
// It records the parsed form values from the most recent request into
// *lastRequest if non-nil.
func newRevokeServer(t *testing.T, lastRequest *map[string]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/oauth/revoke" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if lastRequest != nil {
			if m, err := parsePostForm(r); err == nil {
				*lastRequest = m
			}
		}
		w.WriteHeader(http.StatusOK)
	}))
}
