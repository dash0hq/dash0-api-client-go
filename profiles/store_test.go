package profiles

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestValidateProfileName(t *testing.T) {
	t.Run("rejects", func(t *testing.T) {
		cases := []struct {
			name   string
			input  string
			reason string
		}{
			{"empty", "", "empty"},
			{"newline", "dev\nprod", "control character"},
			{"NUL", "dev\x00prod", "control character"},
			{"leading dot", ".profile-lock", "must not start with '.'"},
			{"forward slash", "dev/prod", "path separators"},
			{"backslash", "dev\\prod", "path separators"},
			{"too long", strings.Repeat("a", MaxProfileNameLength+1), "exceeds maximum length"},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				err := validateProfileName(c.input)
				if err == nil {
					t.Fatalf("expected error for %q", c.input)
				}
				if !strings.Contains(err.Error(), c.reason) {
					t.Errorf("error %q does not mention %q", err.Error(), c.reason)
				}
			})
		}
	})

	t.Run("accepts", func(t *testing.T) {
		for _, name := range []string{
			"dev",
			"prod",
			"my-profile",
			"team_a.qa",
			"Profile1",
			strings.Repeat("a", MaxProfileNameLength),
		} {
			t.Run(name, func(t *testing.T) {
				if err := validateProfileName(name); err != nil {
					t.Errorf("unexpected error for %q: %v", name, err)
				}
			})
		}
	})
}

func TestAddProfile_RejectsInvalidName(t *testing.T) {
	store, _ := newTestStore(t)
	err := store.AddProfile(Profile{
		Name: "bad\nname",
		Configuration: Configuration{
			ApiUrl:    "https://api.example.com",
			AuthToken: "auth_x",
		},
	})
	if err == nil {
		t.Fatal("expected error for invalid profile name")
	}
	if !strings.Contains(err.Error(), "control character") {
		t.Errorf("error should mention control character, got: %v", err)
	}
}

func TestEnsureConfigDirMode_DowngradesBroadPermissions(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatalf("seed chmod: %v", err)
	}

	if err := ensureConfigDirMode(dir); err != nil {
		t.Fatalf("ensureConfigDirMode: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != configDirMode {
		t.Errorf("perm = %o, want %o", perm, configDirMode)
	}
}

func TestEnsureConfigDirMode_NoopWhenAlreadyTight(t *testing.T) {
	dir := t.TempDir()
	// Seed at exactly configDirMode; ensureConfigDirMode must not touch it.
	if err := os.Chmod(dir, configDirMode); err != nil {
		t.Fatalf("seed chmod: %v", err)
	}
	infoBefore, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat before: %v", err)
	}

	if err := ensureConfigDirMode(dir); err != nil {
		t.Fatalf("ensureConfigDirMode: %v", err)
	}

	infoAfter, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat after: %v", err)
	}
	if infoBefore.Mode().Perm() != infoAfter.Mode().Perm() {
		t.Errorf("perm changed: before=%o after=%o", infoBefore.Mode().Perm(), infoAfter.Mode().Perm())
	}
}

func TestEnsureConfigDirMode_TolerantOfTighterExisting(t *testing.T) {
	// A directory already at 0600 has no execute bit so we can't `cd` into
	// it, but ensureConfigDirMode must not consider it broader-than-needed
	// and try to "fix" it. The bits 0600 are a strict subset of 0700.
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o600); err != nil {
		t.Fatalf("seed chmod: %v", err)
	}
	defer func() { _ = os.Chmod(dir, configDirMode) }() // restore for t.TempDir cleanup

	if err := ensureConfigDirMode(dir); err != nil {
		t.Fatalf("ensureConfigDirMode: %v", err)
	}
}

func TestCleanupStaleTempFiles_RemovesLeftovers(t *testing.T) {
	dir := t.TempDir()
	// Stale temp files matching a managed prefix and older than the cutoff.
	stalePaths := []string{
		filepath.Join(dir, ProfilesFileName+".tmp-abc123"),
		filepath.Join(dir, OAuthClientsFileName+".tmp-def456"),
		filepath.Join(dir, ActiveProfileFileName+".tmp-ghi789"),
	}
	for _, p := range stalePaths {
		if err := os.WriteFile(p, []byte("partial"), 0600); err != nil {
			t.Fatalf("seed stale: %v", err)
		}
		oldTime := time.Now().Add(-(staleTempFileAge + time.Minute))
		if err := os.Chtimes(p, oldTime, oldTime); err != nil {
			t.Fatalf("backdate %s: %v", p, err)
		}
	}

	// A recent temp file from a concurrent in-flight writeFileAtomic must
	// be preserved.
	recentPath := filepath.Join(dir, ProfilesFileName+".tmp-recent")
	if err := os.WriteFile(recentPath, []byte("in-flight"), 0600); err != nil {
		t.Fatalf("seed recent: %v", err)
	}

	// A file whose name contains ".tmp-" but does not match a managed prefix
	// must be left alone — only managed temp files are eligible for sweep.
	unrelatedPath := filepath.Join(dir, "user-stuff.tmp-keep")
	if err := os.WriteFile(unrelatedPath, []byte("user data"), 0600); err != nil {
		t.Fatalf("seed unrelated: %v", err)
	}

	keepPath := filepath.Join(dir, ProfilesFileName)
	if err := os.WriteFile(keepPath, []byte(`{"profiles":[]}`), 0600); err != nil {
		t.Fatalf("seed keep: %v", err)
	}

	cleanupStaleTempFiles(dir)

	for _, p := range stalePaths {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("expected stale temp %s to be removed, stat err = %v", p, err)
		}
	}
	if _, err := os.Stat(recentPath); err != nil {
		t.Errorf("recent temp file must be preserved (concurrent write race): %v", err)
	}
	if _, err := os.Stat(unrelatedPath); err != nil {
		t.Errorf("unrelated file must be preserved: %v", err)
	}
	if _, err := os.Stat(keepPath); err != nil {
		t.Errorf("non-temp file should be preserved: %v", err)
	}
}

func TestNewStore(t *testing.T) {
	t.Run("with WithConfigDir", func(t *testing.T) {
		dir := t.TempDir()
		svc, err := NewStore(WithConfigDir(dir))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if svc.configDir != dir {
			t.Errorf("expected configDir %s, got %s", dir, svc.configDir)
		}
	})

	t.Run("with DASH0_CONFIG_DIR env var", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv(EnvConfigDir, dir)
		svc, err := NewStore()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if svc.configDir != dir {
			t.Errorf("expected configDir %s, got %s", dir, svc.configDir)
		}
	})

	t.Run("WithConfigDir takes precedence over env var", func(t *testing.T) {
		envDir := t.TempDir()
		optDir := t.TempDir()
		t.Setenv(EnvConfigDir, envDir)
		svc, err := NewStore(WithConfigDir(optDir))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if svc.configDir != optDir {
			t.Errorf("expected configDir %s, got %s", optDir, svc.configDir)
		}
	})

	t.Run("defaults to home dir", func(t *testing.T) {
		// Unset the env var so the default path is used.
		t.Setenv(EnvConfigDir, "")
		svc, err := NewStore()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		homeDir, _ := os.UserHomeDir()
		expected := filepath.Join(homeDir, ConfigDirName)
		if svc.configDir != expected {
			t.Errorf("expected configDir %s, got %s", expected, svc.configDir)
		}
	})
}

func TestGetProfiles(t *testing.T) {
	svc, dir := newTestStore(t)

	testProfiles := []Profile{
		{Name: "test1", Configuration: Configuration{ApiUrl: "https://test1.example.com", AuthToken: "token1"}},
		{Name: "test2", Configuration: Configuration{ApiUrl: "https://test2.example.com", AuthToken: "token2"}},
	}
	createTestProfilesFile(t, dir, testProfiles)

	profiles, err := svc.GetProfiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(profiles))
	}
	for i, p := range profiles {
		if p.Name != testProfiles[i].Name {
			t.Errorf("profile %d: expected name %s, got %s", i, testProfiles[i].Name, p.Name)
		}
		if p.Configuration.ApiUrl != testProfiles[i].Configuration.ApiUrl {
			t.Errorf("profile %d: expected API URL %s, got %s", i, testProfiles[i].Configuration.ApiUrl, p.Configuration.ApiUrl)
		}
	}
}

func TestGetProfiles_empty(t *testing.T) {
	svc, _ := newTestStore(t)
	profiles, err := svc.GetProfiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(profiles) != 0 {
		t.Errorf("expected 0 profiles, got %d", len(profiles))
	}
}

func TestGetProfiles_invalidJSON(t *testing.T) {
	svc, dir := newTestStore(t)

	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ProfilesFileName), []byte(`{invalid`), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	_, err := svc.GetProfiles()
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse profiles file") {
		t.Errorf("expected parse error, got: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "Hint:") {
		t.Errorf("expected hint in error, got: %s", err.Error())
	}
}

func TestGetActiveProfile(t *testing.T) {
	svc, dir := newTestStore(t)

	testProfiles := []Profile{
		{Name: "test1", Configuration: Configuration{ApiUrl: "https://test1.example.com", AuthToken: "token1"}},
		{Name: "test2", Configuration: Configuration{ApiUrl: "https://test2.example.com", AuthToken: "token2"}},
	}
	createTestProfilesFile(t, dir, testProfiles)
	setActiveProfile(t, dir, "test2")

	profile, err := svc.GetActiveProfile()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile.Name != "test2" {
		t.Errorf("expected active profile test2, got %s", profile.Name)
	}
	if profile.Configuration.ApiUrl != "https://test2.example.com" {
		t.Errorf("expected API URL https://test2.example.com, got %s", profile.Configuration.ApiUrl)
	}
}

func TestGetActiveProfile_noActiveProfile(t *testing.T) {
	svc, _ := newTestStore(t)
	_, err := svc.GetActiveProfile()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != ErrNoActiveProfile {
		t.Errorf("expected ErrNoActiveProfile, got: %v", err)
	}
}

func TestAddProfile(t *testing.T) {
	t.Run("first profile becomes active", func(t *testing.T) {
		svc, _ := newTestStore(t)

		err := svc.AddProfile(Profile{
			Name:          "new-profile",
			Configuration: Configuration{ApiUrl: "https://new.example.com", AuthToken: "new-token"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		profiles, err := svc.GetProfiles()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(profiles) != 1 {
			t.Fatalf("expected 1 profile, got %d", len(profiles))
		}
		if profiles[0].Name != "new-profile" {
			t.Errorf("expected name new-profile, got %s", profiles[0].Name)
		}

		active, err := svc.GetActiveProfile()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if active.Name != "new-profile" {
			t.Errorf("expected active profile new-profile, got %s", active.Name)
		}
	})

	t.Run("replaces existing profile with same name", func(t *testing.T) {
		svc, dir := newTestStore(t)
		createTestProfilesFile(t, dir, []Profile{
			{Name: "dev", Configuration: Configuration{ApiUrl: "https://old.example.com", AuthToken: "old-token"}},
		})
		setActiveProfile(t, dir, "dev")

		err := svc.AddProfile(Profile{
			Name:          "dev",
			Configuration: Configuration{ApiUrl: "https://new.example.com", AuthToken: "new-token"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		profiles, err := svc.GetProfiles()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(profiles) != 1 {
			t.Fatalf("expected 1 profile, got %d", len(profiles))
		}
		if profiles[0].Configuration.ApiUrl != "https://new.example.com" {
			t.Errorf("expected API URL https://new.example.com, got %s", profiles[0].Configuration.ApiUrl)
		}
	})
}

func TestUpdateProfile(t *testing.T) {
	t.Run("update single field", func(t *testing.T) {
		svc, dir := newTestStore(t)
		createTestProfilesFile(t, dir, []Profile{
			{Name: "dev", Configuration: Configuration{ApiUrl: "https://old.example.com", AuthToken: "token1"}},
		})

		err := svc.UpdateProfile("dev", func(cfg *Configuration) {
			cfg.ApiUrl = "https://new.example.com"
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		profiles, _ := svc.GetProfiles()
		if profiles[0].Configuration.ApiUrl != "https://new.example.com" {
			t.Errorf("expected API URL https://new.example.com, got %s", profiles[0].Configuration.ApiUrl)
		}
		if profiles[0].Configuration.AuthToken != "token1" {
			t.Errorf("expected auth token to remain token1, got %s", profiles[0].Configuration.AuthToken)
		}
	})

	t.Run("update multiple fields", func(t *testing.T) {
		svc, dir := newTestStore(t)
		createTestProfilesFile(t, dir, []Profile{
			{Name: "dev", Configuration: Configuration{ApiUrl: "https://old.example.com", AuthToken: "old-token"}},
		})

		err := svc.UpdateProfile("dev", func(cfg *Configuration) {
			cfg.ApiUrl = "https://new.example.com"
			cfg.AuthToken = "new-token"
			cfg.OtlpUrl = "https://otlp.example.com"
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		profiles, _ := svc.GetProfiles()
		if profiles[0].Configuration.ApiUrl != "https://new.example.com" {
			t.Errorf("expected API URL https://new.example.com, got %s", profiles[0].Configuration.ApiUrl)
		}
		if profiles[0].Configuration.AuthToken != "new-token" {
			t.Errorf("expected auth token new-token, got %s", profiles[0].Configuration.AuthToken)
		}
		if profiles[0].Configuration.OtlpUrl != "https://otlp.example.com" {
			t.Errorf("expected OTLP URL https://otlp.example.com, got %s", profiles[0].Configuration.OtlpUrl)
		}
	})

	t.Run("clear field by setting to empty string", func(t *testing.T) {
		svc, dir := newTestStore(t)
		createTestProfilesFile(t, dir, []Profile{
			{Name: "dev", Configuration: Configuration{
				ApiUrl: "https://api.example.com", AuthToken: "token1", OtlpUrl: "https://otlp.example.com",
			}},
		})

		err := svc.UpdateProfile("dev", func(cfg *Configuration) {
			cfg.OtlpUrl = ""
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		profiles, _ := svc.GetProfiles()
		if profiles[0].Configuration.OtlpUrl != "" {
			t.Errorf("expected OTLP URL to be empty, got %s", profiles[0].Configuration.OtlpUrl)
		}
		if profiles[0].Configuration.ApiUrl != "https://api.example.com" {
			t.Errorf("expected API URL to remain unchanged, got %s", profiles[0].Configuration.ApiUrl)
		}
	})

	t.Run("does not affect other profiles", func(t *testing.T) {
		svc, dir := newTestStore(t)
		createTestProfilesFile(t, dir, []Profile{
			{Name: "dev", Configuration: Configuration{ApiUrl: "https://dev.example.com", AuthToken: "dev-token"}},
			{Name: "prod", Configuration: Configuration{ApiUrl: "https://prod.example.com", AuthToken: "prod-token"}},
		})

		err := svc.UpdateProfile("dev", func(cfg *Configuration) {
			cfg.ApiUrl = "https://new-dev.example.com"
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		profiles, _ := svc.GetProfiles()
		if profiles[1].Configuration.ApiUrl != "https://prod.example.com" {
			t.Errorf("expected prod API URL to remain unchanged, got %s", profiles[1].Configuration.ApiUrl)
		}
	})

	t.Run("profile not found", func(t *testing.T) {
		svc, dir := newTestStore(t)
		createTestProfilesFile(t, dir, []Profile{
			{Name: "dev", Configuration: Configuration{ApiUrl: "https://dev.example.com", AuthToken: "token1"}},
		})

		err := svc.UpdateProfile("nonexistent", func(cfg *Configuration) {
			cfg.ApiUrl = "https://new.example.com"
		})
		if err == nil {
			t.Fatal("expected error for nonexistent profile, got nil")
		}
		if !strings.Contains(err.Error(), "nonexistent") {
			t.Errorf("expected error to contain profile name, got: %s", err.Error())
		}
	})
}

func TestRemoveProfile(t *testing.T) {
	t.Run("removes profile and updates active", func(t *testing.T) {
		svc, dir := newTestStore(t)
		createTestProfilesFile(t, dir, []Profile{
			{Name: "test1", Configuration: Configuration{ApiUrl: "https://test1.example.com", AuthToken: "token1"}},
			{Name: "test2", Configuration: Configuration{ApiUrl: "https://test2.example.com", AuthToken: "token2"}},
		})
		setActiveProfile(t, dir, "test2")

		err := svc.RemoveProfile("test2")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		profiles, err := svc.GetProfiles()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(profiles) != 1 {
			t.Fatalf("expected 1 profile, got %d", len(profiles))
		}
		if profiles[0].Name != "test1" {
			t.Errorf("expected remaining profile test1, got %s", profiles[0].Name)
		}

		active, err := svc.GetActiveProfile()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if active.Name != "test1" {
			t.Errorf("expected active profile test1, got %s", active.Name)
		}
	})

	t.Run("revokes OAuth refresh token", func(t *testing.T) {
		var lastReq map[string]string
		server := newRevokeServer(t, &lastReq)
		defer server.Close()

		svc, dir := newTestStore(t)
		createTestProfilesFile(t, dir, []Profile{
			{Name: "oauth-profile", Configuration: Configuration{
				ApiUrl:    server.URL,
				AuthToken: "dash0_at_token",
				OAuth: &OAuthState{
					ClientID:     "cid",
					RefreshToken: "rt-to-revoke",
					ExpiresAt:    time.Now().Add(1 * time.Hour),
				},
			}},
		})
		setActiveProfile(t, dir, "oauth-profile")

		err := svc.RemoveProfile("oauth-profile")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if lastReq["token"] != "rt-to-revoke" {
			t.Errorf("expected revoke token rt-to-revoke, got %s", lastReq["token"])
		}
		if lastReq["token_type_hint"] != "refresh_token" {
			t.Errorf("expected token_type_hint refresh_token, got %s", lastReq["token_type_hint"])
		}

		profiles, err := svc.GetProfiles()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(profiles) != 0 {
			t.Errorf("expected 0 profiles, got %d", len(profiles))
		}
	})

	t.Run("removes profile but surfaces revocation failure via ErrRevocationFailed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		svc, dir := newTestStore(t)
		createTestProfilesFile(t, dir, []Profile{
			{Name: "oauth-profile", Configuration: Configuration{
				ApiUrl:    server.URL,
				AuthToken: "dash0_at_token",
				OAuth: &OAuthState{
					ClientID:     "cid",
					RefreshToken: "rt",
					ExpiresAt:    time.Now().Add(1 * time.Hour),
				},
			}},
		})
		setActiveProfile(t, dir, "oauth-profile")

		err := svc.RemoveProfile("oauth-profile")
		if !errors.Is(err, ErrRevocationFailed) {
			t.Fatalf("expected ErrRevocationFailed, got: %v", err)
		}

		// The local removal still succeeded even though revocation failed.
		profiles, err := svc.GetProfiles()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(profiles) != 0 {
			t.Errorf("expected 0 profiles after failed revocation, got %d", len(profiles))
		}
	})

	t.Run("not found", func(t *testing.T) {
		svc, _ := newTestStore(t)
		err := svc.RemoveProfile("nonexistent")
		if err != ErrProfileNotFound {
			t.Errorf("expected ErrProfileNotFound, got: %v", err)
		}
	})
}

func TestSetActiveProfile(t *testing.T) {
	t.Run("sets active profile", func(t *testing.T) {
		svc, dir := newTestStore(t)
		createTestProfilesFile(t, dir, []Profile{
			{Name: "test1", Configuration: Configuration{ApiUrl: "https://test1.example.com", AuthToken: "token1"}},
			{Name: "test2", Configuration: Configuration{ApiUrl: "https://test2.example.com", AuthToken: "token2"}},
		})

		err := svc.SetActiveProfile("test2")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		active, err := svc.GetActiveProfile()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if active.Name != "test2" {
			t.Errorf("expected active profile test2, got %s", active.Name)
		}
	})

	t.Run("not found", func(t *testing.T) {
		svc, _ := newTestStore(t)
		err := svc.SetActiveProfile("nonexistent")
		if err != ErrProfileNotFound {
			t.Errorf("expected ErrProfileNotFound, got: %v", err)
		}
	})
}

func TestGetActiveConfiguration(t *testing.T) {
	t.Run("from env vars only", func(t *testing.T) {
		svc, _ := newTestStore(t)
		t.Setenv(EnvApiUrl, "https://env.example.com")
		t.Setenv(EnvAuthToken, "env-token")

		cfg, err := svc.GetActiveConfiguration()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.ApiUrl != "https://env.example.com" {
			t.Errorf("expected API URL https://env.example.com, got %s", cfg.ApiUrl)
		}
		if cfg.AuthToken != "env-token" {
			t.Errorf("expected auth token env-token, got %s", cfg.AuthToken)
		}
	})

	t.Run("from active profile", func(t *testing.T) {
		svc, dir := newTestStore(t)
		createTestProfilesFile(t, dir, []Profile{
			{Name: "test1", Configuration: Configuration{ApiUrl: "https://test1.example.com", AuthToken: "token1"}},
		})
		setActiveProfile(t, dir, "test1")

		cfg, err := svc.GetActiveConfiguration()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.ApiUrl != "https://test1.example.com" {
			t.Errorf("expected API URL https://test1.example.com, got %s", cfg.ApiUrl)
		}
		if cfg.AuthToken != "token1" {
			t.Errorf("expected auth token token1, got %s", cfg.AuthToken)
		}
	})

	t.Run("env vars override profile", func(t *testing.T) {
		svc, dir := newTestStore(t)
		createTestProfilesFile(t, dir, []Profile{
			{Name: "test1", Configuration: Configuration{
				ApiUrl: "https://profile.example.com", AuthToken: "profile-token", Dataset: "profile-dataset",
			}},
		})
		setActiveProfile(t, dir, "test1")
		t.Setenv(EnvDataset, "env-dataset")

		cfg, err := svc.GetActiveConfiguration()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Dataset != "env-dataset" {
			t.Errorf("expected dataset env-dataset, got %s", cfg.Dataset)
		}
		// Non-overridden fields come from the profile.
		if cfg.ApiUrl != "https://profile.example.com" {
			t.Errorf("expected API URL from profile, got %s", cfg.ApiUrl)
		}
	})

	t.Run("OTLP URL from env var", func(t *testing.T) {
		svc, _ := newTestStore(t)
		t.Setenv(EnvAuthToken, "env-token")
		t.Setenv(EnvOtlpUrl, "https://otlp.example.com")

		cfg, err := svc.GetActiveConfiguration()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.OtlpUrl != "https://otlp.example.com" {
			t.Errorf("expected OTLP URL https://otlp.example.com, got %s", cfg.OtlpUrl)
		}
	})

	t.Run("OTLP URL env var overrides profile", func(t *testing.T) {
		svc, dir := newTestStore(t)
		createTestProfilesFile(t, dir, []Profile{
			{Name: "test1", Configuration: Configuration{
				ApiUrl: "https://api.example.com", AuthToken: "token1", OtlpUrl: "https://otlp-profile.example.com",
			}},
		})
		setActiveProfile(t, dir, "test1")
		t.Setenv(EnvOtlpUrl, "https://otlp-env.example.com")

		cfg, err := svc.GetActiveConfiguration()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.OtlpUrl != "https://otlp-env.example.com" {
			t.Errorf("expected OTLP URL https://otlp-env.example.com, got %s", cfg.OtlpUrl)
		}
	})

	t.Run("dataset from profile", func(t *testing.T) {
		svc, dir := newTestStore(t)
		createTestProfilesFile(t, dir, []Profile{
			{Name: "test1", Configuration: Configuration{
				ApiUrl: "https://api.example.com", AuthToken: "token1", Dataset: "profile-dataset",
			}},
		})
		setActiveProfile(t, dir, "test1")

		cfg, err := svc.GetActiveConfiguration()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Dataset != "profile-dataset" {
			t.Errorf("expected dataset profile-dataset, got %s", cfg.Dataset)
		}
	})

	t.Run("OAuth token not near expiry uses existing auth token", func(t *testing.T) {
		svc, dir := newTestStore(t)
		createTestProfilesFile(t, dir, []Profile{
			{Name: "test1", Configuration: Configuration{
				ApiUrl:    "https://api.example.com",
				AuthToken: "dash0_at_current-token",
				OAuth: &OAuthState{
					ClientID:     "cid",
					RefreshToken: "rt",
					ExpiresAt:    time.Now().Add(1 * time.Hour),
				},
			}},
		})
		setActiveProfile(t, dir, "test1")

		cfg, err := svc.GetActiveConfiguration()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.AuthToken != "dash0_at_current-token" {
			t.Errorf("expected dash0_at_current-token, got %s", cfg.AuthToken)
		}
		if cfg.OAuth == nil {
			t.Error("expected OAuth state to be preserved")
		}
	})

	t.Run("OAuth token near expiry triggers refresh", func(t *testing.T) {
		server := newTokenServer(t, tokenServerResponse{
			AccessToken: "dash0_at_refreshed-token",
			ExpiresIn:   3600,
		}, nil)
		defer server.Close()

		svc, dir := newTestStore(t)
		createTestProfilesFile(t, dir, []Profile{
			{Name: "test1", Configuration: Configuration{
				ApiUrl:    server.URL,
				AuthToken: "dash0_at_old-token",
				OAuth: &OAuthState{
					ClientID:     "cid",
					RefreshToken: "rt",
					ExpiresAt:    time.Now().Add(2 * time.Minute),
				},
			}},
		})
		setActiveProfile(t, dir, "test1")

		cfg, err := svc.GetActiveConfiguration()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.AuthToken != "dash0_at_refreshed-token" {
			t.Errorf("expected dash0_at_refreshed-token, got %s", cfg.AuthToken)
		}
	})

	t.Run("DASH0_AUTH_TOKEN env var clears OAuth state", func(t *testing.T) {
		svc, dir := newTestStore(t)
		createTestProfilesFile(t, dir, []Profile{
			{Name: "test1", Configuration: Configuration{
				ApiUrl:    "https://api.example.com",
				AuthToken: "dash0_at_oauth-token",
				OAuth: &OAuthState{
					ClientID:     "cid",
					RefreshToken: "rt",
					ExpiresAt:    time.Now().Add(1 * time.Hour),
				},
			}},
		})
		setActiveProfile(t, dir, "test1")
		t.Setenv(EnvAuthToken, "auth_env-token")

		cfg, err := svc.GetActiveConfiguration()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.AuthToken != "auth_env-token" {
			t.Errorf("expected auth_env-token, got %s", cfg.AuthToken)
		}
		if cfg.OAuth != nil {
			t.Error("expected OAuth to be nil when env var overrides")
		}
	})
}

func TestResolveConfiguration(t *testing.T) {
	t.Run("from env vars", func(t *testing.T) {
		t.Setenv(EnvApiUrl, "https://env.example.com")
		t.Setenv(EnvAuthToken, "env-token")

		cfg, err := ResolveConfiguration("", "", WithConfigDir(t.TempDir()))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.ApiUrl != "https://env.example.com" {
			t.Errorf("expected API URL https://env.example.com, got %s", cfg.ApiUrl)
		}
		if cfg.AuthToken != "env-token" {
			t.Errorf("expected auth token env-token, got %s", cfg.AuthToken)
		}
	})

	t.Run("parameter overrides profile", func(t *testing.T) {
		dir := t.TempDir()
		createTestProfilesFile(t, dir, []Profile{
			{Name: "test", Configuration: Configuration{ApiUrl: "https://original.example.com", AuthToken: "original-token"}},
		})
		setActiveProfile(t, dir, "test")

		cfg, err := ResolveConfiguration("https://override.example.com", "", WithConfigDir(dir))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.ApiUrl != "https://override.example.com" {
			t.Errorf("expected API URL https://override.example.com, got %s", cfg.ApiUrl)
		}
		if cfg.AuthToken != "original-token" {
			t.Errorf("expected auth token original-token, got %s", cfg.AuthToken)
		}
	})

	t.Run("no configuration available", func(t *testing.T) {
		_, err := ResolveConfiguration("", "", WithConfigDir(t.TempDir()))
		if err == nil {
			t.Fatal("expected error for missing configuration, got nil")
		}
	})

	t.Run("OTLP URL only via env vars", func(t *testing.T) {
		t.Setenv(EnvOtlpUrl, "https://otlp.example.com")
		t.Setenv(EnvAuthToken, "env-token")

		cfg, err := ResolveConfiguration("", "", WithConfigDir(t.TempDir()))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.OtlpUrl != "https://otlp.example.com" {
			t.Errorf("expected OTLP URL https://otlp.example.com, got %s", cfg.OtlpUrl)
		}
		if cfg.ApiUrl != "" {
			t.Errorf("expected empty API URL, got %s", cfg.ApiUrl)
		}
	})

	t.Run("dataset from env var", func(t *testing.T) {
		t.Setenv(EnvApiUrl, "https://api.example.com")
		t.Setenv(EnvAuthToken, "env-token")
		t.Setenv(EnvDataset, "env-dataset")

		cfg, err := ResolveConfiguration("", "", WithConfigDir(t.TempDir()))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Dataset != "env-dataset" {
			t.Errorf("expected dataset env-dataset, got %s", cfg.Dataset)
		}
	})

	t.Run("dataset from flag override", func(t *testing.T) {
		t.Setenv(EnvApiUrl, "https://api.example.com")
		t.Setenv(EnvAuthToken, "env-token")
		t.Setenv(EnvDataset, "env-dataset")

		cfg, err := ResolveConfigurationWithOtlp("", "", "", "flag-dataset", WithConfigDir(t.TempDir()))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Dataset != "flag-dataset" {
			t.Errorf("expected dataset flag-dataset, got %s", cfg.Dataset)
		}
	})

	t.Run("OTLP URL flag override", func(t *testing.T) {
		t.Setenv(EnvAuthToken, "env-token")

		cfg, err := ResolveConfigurationWithOtlp("", "", "https://otlp-flag.example.com", "", WithConfigDir(t.TempDir()))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.OtlpUrl != "https://otlp-flag.example.com" {
			t.Errorf("expected OTLP URL https://otlp-flag.example.com, got %s", cfg.OtlpUrl)
		}
	})

	t.Run("OAuth state propagated from profile", func(t *testing.T) {
		dir := t.TempDir()
		createTestProfilesFile(t, dir, []Profile{
			{Name: "test", Configuration: Configuration{
				ApiUrl:    "https://api.example.com",
				AuthToken: "dash0_at_oauth-token",
				OAuth: &OAuthState{
					ClientID:     "cid",
					RefreshToken: "rt",
					ExpiresAt:    time.Now().Add(1 * time.Hour),
				},
			}},
		})
		setActiveProfile(t, dir, "test")

		cfg, err := ResolveConfiguration("", "", WithConfigDir(dir))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.OAuth == nil {
			t.Fatal("expected OAuth state to be propagated")
		}
		if cfg.OAuth.ClientID != "cid" {
			t.Errorf("expected ClientID cid, got %s", cfg.OAuth.ClientID)
		}
	})

	t.Run("explicit auth token clears OAuth state", func(t *testing.T) {
		dir := t.TempDir()
		createTestProfilesFile(t, dir, []Profile{
			{Name: "test", Configuration: Configuration{
				ApiUrl:    "https://api.example.com",
				AuthToken: "dash0_at_oauth-token",
				OAuth: &OAuthState{
					ClientID:     "cid",
					RefreshToken: "rt",
					ExpiresAt:    time.Now().Add(1 * time.Hour),
				},
			}},
		})
		setActiveProfile(t, dir, "test")

		cfg, err := ResolveConfiguration("", "auth_explicit-token", WithConfigDir(dir))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.AuthToken != "auth_explicit-token" {
			t.Errorf("expected auth_explicit-token, got %s", cfg.AuthToken)
		}
		if cfg.OAuth != nil {
			t.Error("expected OAuth to be nil when explicit auth token provided")
		}
	})

	t.Run("explicit auth token suppresses OAuth refresh", func(t *testing.T) {
		var requestCount atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "dash0_at_should_not_be_seen",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
		}))
		defer server.Close()

		dir := t.TempDir()
		createTestProfilesFile(t, dir, []Profile{
			{Name: "test", Configuration: Configuration{
				ApiUrl:    server.URL,
				AuthToken: "dash0_at_old",
				OAuth: &OAuthState{
					ClientID:     "cid",
					RefreshToken: "rt",
					// Near expiry: without the suppression this would fire a refresh.
					ExpiresAt: time.Now().Add(1 * time.Minute),
				},
			}},
		})
		setActiveProfile(t, dir, "test")

		cfg, err := ResolveConfiguration("", "auth_explicit-token", WithConfigDir(dir))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.AuthToken != "auth_explicit-token" {
			t.Errorf("expected auth_explicit-token, got %s", cfg.AuthToken)
		}
		if got := requestCount.Load(); got != 0 {
			t.Errorf("expected 0 token-endpoint requests, got %d", got)
		}
	})

	t.Run("no explicit auth token still refreshes OAuth", func(t *testing.T) {
		var requestCount atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "dash0_at_refreshed",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
		}))
		defer server.Close()

		dir := t.TempDir()
		createTestProfilesFile(t, dir, []Profile{
			{Name: "test", Configuration: Configuration{
				ApiUrl:    server.URL,
				AuthToken: "dash0_at_old",
				OAuth: &OAuthState{
					ClientID:     "cid",
					RefreshToken: "rt",
					ExpiresAt:    time.Now().Add(1 * time.Minute),
				},
			}},
		})
		setActiveProfile(t, dir, "test")

		cfg, err := ResolveConfiguration("", "", WithConfigDir(dir))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.AuthToken != "dash0_at_refreshed" {
			t.Errorf("expected dash0_at_refreshed, got %s", cfg.AuthToken)
		}
		if got := requestCount.Load(); got != 1 {
			t.Errorf("expected 1 token-endpoint request, got %d", got)
		}
	})
}
