package profiles

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestOAuthClientStore(t *testing.T) (*OAuthClientStore, string) {
	t.Helper()
	dir := t.TempDir()
	s, err := NewOAuthClientStore(WithConfigDir(dir))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	return s, dir
}

func TestOAuthClientStore_MissingFileIsAMiss(t *testing.T) {
	s, _ := newTestOAuthClientStore(t)
	rec, ok, err := s.Get("https://api.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Errorf("expected miss, got record %+v", rec)
	}
}

func TestOAuthClientStore_PutGetDeleteRoundTrip(t *testing.T) {
	s, dir := newTestOAuthClientStore(t)

	rec := OAuthClientRecord{
		ClientID:                "cid-1",
		RegistrationAccessToken: "ratoken-1",
		RedirectURI:             "http://localhost:8080/callback",
	}
	if err := s.Put("https://api.example.com", rec); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// File should exist at 0600.
	info, err := os.Stat(filepath.Join(dir, OAuthClientsFileName))
	if err != nil {
		t.Fatalf("file not written: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0600 {
		t.Errorf("file mode = %o, want 0600", mode)
	}

	got, ok, err := s.Get("https://api.example.com")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !ok {
		t.Fatal("expected hit after Put")
	}
	if got != rec {
		t.Errorf("Get returned %+v, want %+v", got, rec)
	}

	if err := s.Delete("https://api.example.com"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if _, ok, _ := s.Get("https://api.example.com"); ok {
		t.Error("expected miss after Delete")
	}

	// Deleting a missing record is a no-op.
	if err := s.Delete("https://api.example.com"); err != nil {
		t.Errorf("Delete on missing record returned error: %v", err)
	}
}

func TestOAuthClientStore_EquivalentURLsCollapse(t *testing.T) {
	s, _ := newTestOAuthClientStore(t)

	rec := OAuthClientRecord{ClientID: "cid", RedirectURI: "http://localhost/cb"}
	if err := s.Put("HTTPS://API.example.com/", rec); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	for _, equivalent := range []string{
		"https://api.example.com",
		"https://api.example.com/",
		"https://api.example.com?x=1",
		"HTTPS://api.EXAMPLE.com/",
		"https://api.example.com#frag",
	} {
		got, ok, err := s.Get(equivalent)
		if err != nil {
			t.Errorf("Get(%q) returned error: %v", equivalent, err)
			continue
		}
		if !ok {
			t.Errorf("Get(%q) missed; expected canonicalization to hit", equivalent)
			continue
		}
		if got != rec {
			t.Errorf("Get(%q) returned %+v, want %+v", equivalent, got, rec)
		}
	}
}

func TestOAuthClientStore_CorruptFileSurfacesAndPutQuarantines(t *testing.T) {
	s, dir := newTestOAuthClientStore(t)

	path := filepath.Join(dir, OAuthClientsFileName)
	if err := os.WriteFile(path, []byte("{not valid json"), 0600); err != nil {
		t.Fatalf("seed write failed: %v", err)
	}

	// Get on a corrupt file surfaces the corruption rather than silently
	// missing — operators need to know the cache is unreadable.
	_, _, err := s.Get("https://api.example.com")
	if !errors.Is(err, ErrOAuthClientsFileCorrupt) {
		t.Fatalf("expected ErrOAuthClientsFileCorrupt on corrupt file, got %v", err)
	}

	// Put on a corrupt file quarantines the bad file (renames it with a
	// .corrupt-<timestamp> suffix) and then writes the new record cleanly.
	// Previous registrations are lost from the active file but preserved on
	// disk for manual recovery.
	want := OAuthClientRecord{ClientID: "cid", RedirectURI: "http://localhost/cb"}
	if err := s.Put("https://api.example.com", want); err != nil {
		t.Fatalf("Put on corrupt file failed: %v", err)
	}

	// Subsequent Get reads the new record cleanly.
	got, ok, err := s.Get("https://api.example.com")
	if err != nil {
		t.Fatalf("Get after Put failed: %v", err)
	}
	if !ok || got != want {
		t.Errorf("Get returned (%+v, %v), want (%+v, true)", got, ok, want)
	}

	// A quarantine file should now exist in the directory.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	var quarantineCount int
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), OAuthClientsFileName+".corrupt-") {
			quarantineCount++
		}
	}
	if quarantineCount != 1 {
		t.Errorf("expected exactly 1 quarantine file, got %d", quarantineCount)
	}
}

func TestOAuthClientStore_CrossProcessPutSerializes(t *testing.T) {
	// Two distinct OAuthClientStore instances against the same config dir
	// mimic two CLI invocations. Each does N Puts against a distinct API URL;
	// after both finish, all 2N records must be present — the cross-process
	// flock guarantees no read-modify-write loses an entry.
	dir := t.TempDir()
	const perProcess = 10
	storeA, _ := NewOAuthClientStore(WithConfigDir(dir))
	storeB, _ := NewOAuthClientStore(WithConfigDir(dir))

	done := make(chan error, 2)
	put := func(store *OAuthClientStore, prefix string) {
		for i := range perProcess {
			rec := OAuthClientRecord{
				ClientID:    fmt.Sprintf("%s-cid-%d", prefix, i),
				RedirectURI: "http://localhost/cb",
			}
			apiURL := fmt.Sprintf("https://api-%s-%d.example.com", prefix, i)
			if err := store.Put(apiURL, rec); err != nil {
				done <- err
				return
			}
		}
		done <- nil
	}
	go put(storeA, "a")
	go put(storeB, "b")
	for range 2 {
		if err := <-done; err != nil {
			t.Fatalf("Put failed: %v", err)
		}
	}

	// Read every record back via a fresh store.
	verify, _ := NewOAuthClientStore(WithConfigDir(dir))
	for _, prefix := range []string{"a", "b"} {
		for i := range perProcess {
			apiURL := fmt.Sprintf("https://api-%s-%d.example.com", prefix, i)
			rec, ok, err := verify.Get(apiURL)
			if err != nil {
				t.Errorf("Get(%s): %v", apiURL, err)
				continue
			}
			if !ok {
				t.Errorf("Get(%s): record missing — cross-process serialization lost an update", apiURL)
				continue
			}
			wantCID := fmt.Sprintf("%s-cid-%d", prefix, i)
			if rec.ClientID != wantCID {
				t.Errorf("Get(%s): ClientID = %q, want %q", apiURL, rec.ClientID, wantCID)
			}
		}
	}
}

func TestOAuthClientStore_MultipleURLsCoexist(t *testing.T) {
	s, _ := newTestOAuthClientStore(t)

	recEU := OAuthClientRecord{ClientID: "cid-eu", RedirectURI: "http://localhost/cb"}
	recUS := OAuthClientRecord{ClientID: "cid-us", RedirectURI: "http://localhost/cb"}
	if err := s.Put("https://api.eu-west-1.aws.dash0.com", recEU); err != nil {
		t.Fatalf("Put EU failed: %v", err)
	}
	if err := s.Put("https://api.us-west-2.aws.dash0.com", recUS); err != nil {
		t.Fatalf("Put US failed: %v", err)
	}

	gotEU, okEU, _ := s.Get("https://api.eu-west-1.aws.dash0.com")
	if !okEU || gotEU != recEU {
		t.Errorf("EU Get = (%+v, %v), want (%+v, true)", gotEU, okEU, recEU)
	}
	gotUS, okUS, _ := s.Get("https://api.us-west-2.aws.dash0.com")
	if !okUS || gotUS != recUS {
		t.Errorf("US Get = (%+v, %v), want (%+v, true)", gotUS, okUS, recUS)
	}

	if err := s.Delete("https://api.eu-west-1.aws.dash0.com"); err != nil {
		t.Fatalf("Delete EU failed: %v", err)
	}
	if _, ok, _ := s.Get("https://api.eu-west-1.aws.dash0.com"); ok {
		t.Error("EU should be gone after Delete")
	}
	gotUS, okUS, _ = s.Get("https://api.us-west-2.aws.dash0.com")
	if !okUS || gotUS != recUS {
		t.Errorf("US still expected after EU Delete: got (%+v, %v)", gotUS, okUS)
	}
}

func TestCanonicalAPIURL(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"https://api.example.com", "https://api.example.com"},
		{"https://api.example.com/", "https://api.example.com"},
		{"HTTPS://API.example.com/", "https://api.example.com"},
		{"https://api.example.com/v1/", "https://api.example.com/v1"},
		{"https://api.example.com?x=1", "https://api.example.com"},
		{"https://api.example.com#frag", "https://api.example.com"},
		// Userinfo must not survive into the cache key.
		{"https://user:pass@api.example.com/", "https://api.example.com"},
		{"https://user@api.example.com/", "https://api.example.com"},
		// Default port for the scheme is dropped so two equivalent URLs
		// collapse to the same key.
		{"https://api.example.com:443/", "https://api.example.com"},
		{"http://api.example.com:80/", "http://api.example.com"},
		// Non-default ports survive.
		{"https://api.example.com:8443/", "https://api.example.com:8443"},
		// path.Clean collapses "." and "..".
		{"https://api.example.com/v1/./", "https://api.example.com/v1"},
		{"https://api.example.com/a/b/../v1/", "https://api.example.com/a/v1"},
	}
	for _, c := range cases {
		got, err := CanonicalAPIURL(c.raw)
		if err != nil {
			t.Errorf("CanonicalAPIURL(%q) returned error: %v", c.raw, err)
			continue
		}
		if got != c.want {
			t.Errorf("CanonicalAPIURL(%q) = %q, want %q", c.raw, got, c.want)
		}
	}

	// Inputs missing scheme or host should error.
	for _, raw := range []string{"", "not-a-url", "/path/only"} {
		if _, err := CanonicalAPIURL(raw); err == nil {
			t.Errorf("CanonicalAPIURL(%q) expected error, got nil", raw)
		}
	}
}

func TestNewOAuthClientStore_ConfigDirResolution(t *testing.T) {
	t.Run("explicit WithConfigDir wins", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv(EnvConfigDir, "/tmp/should-not-be-used")
		s, err := NewOAuthClientStore(WithConfigDir(dir))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.configDir != dir {
			t.Errorf("configDir = %q, want %q", s.configDir, dir)
		}
	})

	t.Run("DASH0_CONFIG_DIR honoured", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv(EnvConfigDir, dir)
		s, err := NewOAuthClientStore()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.configDir != dir {
			t.Errorf("configDir = %q, want %q", s.configDir, dir)
		}
	})
}
