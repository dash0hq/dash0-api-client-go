package profiles

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	dash0 "github.com/dash0hq/dash0-api-client-go"
)

func TestRefreshOAuthToken(t *testing.T) {
	t.Run("no OAuth state", func(t *testing.T) {
		store, _ := newTestStore(t)
		cfg := &Configuration{AuthToken: "auth_something"}
		err := refreshOAuthToken(store, "test", cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.AuthToken != "auth_something" {
			t.Errorf("expected AuthToken unchanged, got %s", cfg.AuthToken)
		}
	})

	t.Run("token not near expiry", func(t *testing.T) {
		store, dir := newTestStore(t)
		cfg := &Configuration{
			ApiUrl:    "https://api.example.com",
			AuthToken: "auth_current-token",
			OAuth: &OAuthState{
				ClientID:     "cid",
				RefreshToken: "rt",
				ExpiresAt:    time.Now().Add(1 * time.Hour),
			},
		}
		createTestProfilesFile(t, dir, []Profile{
			{Name: "test", Configuration: *cfg},
		})

		err := refreshOAuthToken(store, "test", cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.AuthToken != "auth_current-token" {
			t.Errorf("expected AuthToken unchanged, got %s", cfg.AuthToken)
		}
	})

	t.Run("token near expiry refresh succeeds", func(t *testing.T) {
		var lastReq map[string]string
		server := newTokenServer(t, tokenServerResponse{
			AccessToken: "dash0_at_refreshed-token",
			ExpiresIn:   3600,
		}, &lastReq)
		defer server.Close()

		store, dir := newTestStore(t)
		profile := Profile{
			Name: "test",
			Configuration: Configuration{
				ApiUrl:    server.URL,
				AuthToken: "dash0_at_old-token",
				OAuth: &OAuthState{
					ClientID:     "my-client-id",
					RefreshToken: "my-refresh-token",
					ExpiresAt:    time.Now().Add(2 * time.Minute),
				},
			},
		}
		createTestProfilesFile(t, dir, []Profile{profile})

		cfg := &profile.Configuration
		err := refreshOAuthToken(store, "test", cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.AuthToken != "dash0_at_refreshed-token" {
			t.Errorf("expected dash0_at_refreshed-token, got %s", cfg.AuthToken)
		}
		if time.Until(cfg.OAuth.ExpiresAt) < 59*time.Minute {
			t.Error("expected ExpiresAt to be roughly 1 hour from now")
		}

		// Verify request body sent to token endpoint.
		if lastReq["grant_type"] != "refresh_token" {
			t.Errorf("expected grant_type refresh_token, got %s", lastReq["grant_type"])
		}
		if lastReq["refresh_token"] != "my-refresh-token" {
			t.Errorf("expected refresh_token my-refresh-token, got %s", lastReq["refresh_token"])
		}
		if lastReq["client_id"] != "my-client-id" {
			t.Errorf("expected client_id my-client-id, got %s", lastReq["client_id"])
		}

		// Verify persisted to disk.
		profiles, err := store.GetProfiles()
		if err != nil {
			t.Fatalf("failed to read profiles: %v", err)
		}
		persisted := profiles[0].Configuration
		if persisted.AuthToken != "dash0_at_refreshed-token" {
			t.Errorf("expected persisted AuthToken dash0_at_refreshed-token, got %s",
				persisted.AuthToken)
		}
		if persisted.OAuth.RefreshToken != "my-refresh-token" {
			t.Errorf("expected persisted RefreshToken unchanged, got %s",
				persisted.OAuth.RefreshToken)
		}
		if time.Until(persisted.OAuth.ExpiresAt) < 59*time.Minute {
			t.Error("expected persisted ExpiresAt to be roughly 1 hour from now")
		}
	})

	t.Run("refresh token rotated", func(t *testing.T) {
		server := newTokenServer(t, tokenServerResponse{
			AccessToken:  "dash0_at_new-token",
			ExpiresIn:    7200,
			RefreshToken: dash0.Ptr("new-rt"),
		}, nil)
		defer server.Close()

		store, dir := newTestStore(t)
		profile := Profile{
			Name: "test",
			Configuration: Configuration{
				ApiUrl:    server.URL,
				AuthToken: "dash0_at_old-token",
				OAuth: &OAuthState{
					ClientID:     "cid",
					RefreshToken: "old-rt",
					ExpiresAt:    time.Now().Add(1 * time.Minute),
				},
			},
		}
		createTestProfilesFile(t, dir, []Profile{profile})

		cfg := &profile.Configuration
		err := refreshOAuthToken(store, "test", cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.OAuth.RefreshToken != "new-rt" {
			t.Errorf("expected rotated refresh token new-rt, got %s", cfg.OAuth.RefreshToken)
		}

		// Verify persisted.
		profiles, err := store.GetProfiles()
		if err != nil {
			t.Fatalf("failed to read profiles: %v", err)
		}
		if profiles[0].Configuration.OAuth.RefreshToken != "new-rt" {
			t.Errorf("expected persisted RefreshToken new-rt, got %s",
				profiles[0].Configuration.OAuth.RefreshToken)
		}
	})

	t.Run("refresh fails", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error":             "invalid_grant",
				"error_description": "refresh token expired",
			})
		}))
		defer server.Close()

		store, dir := newTestStore(t)
		profile := Profile{
			Name: "test",
			Configuration: Configuration{
				ApiUrl:    server.URL,
				AuthToken: "dash0_at_old-token",
				OAuth: &OAuthState{
					ClientID:     "cid",
					RefreshToken: "bad-rt",
					ExpiresAt:    time.Now().Add(-1 * time.Minute),
				},
			},
		}
		createTestProfilesFile(t, dir, []Profile{profile})

		cfg := &profile.Configuration
		err := refreshOAuthToken(store, "test", cfg)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if cfg.AuthToken != "dash0_at_old-token" {
			t.Errorf("expected AuthToken unchanged after failure, got %s", cfg.AuthToken)
		}
	})

	t.Run("missing API URL", func(t *testing.T) {
		store, dir := newTestStore(t)
		profile := Profile{
			Name: "test",
			Configuration: Configuration{
				AuthToken: "dash0_at_old-token",
				OAuth: &OAuthState{
					ClientID:     "cid",
					RefreshToken: "rt",
					ExpiresAt:    time.Now().Add(1 * time.Minute),
				},
			},
		}
		createTestProfilesFile(t, dir, []Profile{profile})

		cfg := &profile.Configuration
		err := refreshOAuthToken(store, "test", cfg)
		if err == nil {
			t.Fatal("expected error for missing API URL, got nil")
		}
		if cfg.AuthToken != "dash0_at_old-token" {
			t.Errorf("expected AuthToken unchanged after failure, got %s", cfg.AuthToken)
		}
	})

	t.Run("concurrent refresh only fires one request", func(t *testing.T) {
		var requestCount atomic.Int32
		var release sync.WaitGroup
		release.Add(1)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount.Add(1)
			// Hold the first request so the other goroutines pile up at
			// the mutex. Subsequent requests (if any) are not held.
			release.Wait()
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token":  "dash0_at_rotated",
				"token_type":    "Bearer",
				"expires_in":    3600,
				"refresh_token": "rt-rotated",
			})
		}))
		defer server.Close()

		store, dir := newTestStore(t)
		createTestProfilesFile(t, dir, []Profile{
			{Name: "test", Configuration: Configuration{
				ApiUrl:    server.URL,
				AuthToken: "dash0_at_old",
				OAuth: &OAuthState{
					ClientID:     "cid",
					RefreshToken: "rt-original",
					ExpiresAt:    time.Now().Add(1 * time.Minute),
				},
			}},
		})

		const goroutines = 8
		cfgs := make([]*Configuration, goroutines)
		errs := make([]error, goroutines)
		var start, done sync.WaitGroup
		start.Add(1)
		done.Add(goroutines)
		for i := range goroutines {
			go func() {
				defer done.Done()
				// Each goroutine starts with its own snapshot of the
				// configuration (as if it had just read the profile).
				cfgs[i] = &Configuration{
					ApiUrl:    server.URL,
					AuthToken: "dash0_at_old",
					OAuth: &OAuthState{
						ClientID:     "cid",
						RefreshToken: "rt-original",
						ExpiresAt:    time.Now().Add(1 * time.Minute),
					},
				}
				start.Wait()
				errs[i] = refreshOAuthToken(store, "test", cfgs[i])
			}()
		}
		start.Done()
		// Give every goroutine a chance to enter refreshOAuthToken and
		// block on the mutex before the in-flight request returns.
		time.Sleep(50 * time.Millisecond)
		release.Done()
		done.Wait()

		if got := requestCount.Load(); got != 1 {
			t.Errorf("expected exactly 1 token request, got %d", got)
		}
		for i, err := range errs {
			if err != nil {
				t.Errorf("goroutine %d: unexpected error: %v", i, err)
			}
			if cfgs[i].AuthToken != "dash0_at_rotated" {
				t.Errorf("goroutine %d: expected dash0_at_rotated, got %s",
					i, cfgs[i].AuthToken)
			}
			if cfgs[i].OAuth == nil || cfgs[i].OAuth.RefreshToken != "rt-rotated" {
				t.Errorf("goroutine %d: expected rt-rotated, got %+v",
					i, cfgs[i].OAuth)
			}
		}

		// Persisted state should reflect the rotation exactly once.
		profiles, err := store.GetProfiles()
		if err != nil {
			t.Fatalf("failed to read profiles: %v", err)
		}
		persisted := profiles[0].Configuration
		if persisted.AuthToken != "dash0_at_rotated" {
			t.Errorf("persisted AuthToken = %s, want dash0_at_rotated", persisted.AuthToken)
		}
		if persisted.OAuth == nil || persisted.OAuth.RefreshToken != "rt-rotated" {
			t.Errorf("persisted RefreshToken = %+v, want rt-rotated", persisted.OAuth)
		}
	})

	t.Run("token already expired refresh succeeds", func(t *testing.T) {
		server := newTokenServer(t, tokenServerResponse{
			AccessToken: "dash0_at_fresh-token",
			ExpiresIn:   3600,
		}, nil)
		defer server.Close()

		store, dir := newTestStore(t)
		profile := Profile{
			Name: "test",
			Configuration: Configuration{
				ApiUrl:    server.URL,
				AuthToken: "dash0_at_expired-token",
				OAuth: &OAuthState{
					ClientID:     "cid",
					RefreshToken: "rt",
					ExpiresAt:    time.Now().Add(-10 * time.Minute),
				},
			},
		}
		createTestProfilesFile(t, dir, []Profile{profile})

		cfg := &profile.Configuration
		err := refreshOAuthToken(store, "test", cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.AuthToken != "dash0_at_fresh-token" {
			t.Errorf("expected dash0_at_fresh-token, got %s", cfg.AuthToken)
		}
	})
}

func TestRevokeOAuthTokens(t *testing.T) {
	t.Run("no OAuth state", func(t *testing.T) {
		cfg := &Configuration{AuthToken: "auth_something"}
		err := revokeOAuthTokens(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("no API URL", func(t *testing.T) {
		cfg := &Configuration{
			AuthToken: "dash0_at_token",
			OAuth: &OAuthState{
				ClientID:     "cid",
				RefreshToken: "rt",
				ExpiresAt:    time.Now().Add(1 * time.Hour),
			},
		}
		err := revokeOAuthTokens(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("revokes refresh token", func(t *testing.T) {
		var lastReq map[string]string
		server := newRevokeServer(t, &lastReq)
		defer server.Close()

		cfg := &Configuration{
			ApiUrl:    server.URL,
			AuthToken: "dash0_at_token",
			OAuth: &OAuthState{
				ClientID:     "cid",
				RefreshToken: "my-refresh-token",
				ExpiresAt:    time.Now().Add(1 * time.Hour),
			},
		}
		err := revokeOAuthTokens(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if lastReq["token"] != "my-refresh-token" {
			t.Errorf("expected token my-refresh-token, got %s", lastReq["token"])
		}
		if lastReq["token_type_hint"] != "refresh_token" {
			t.Errorf("expected token_type_hint refresh_token, got %s", lastReq["token_type_hint"])
		}
	})

	t.Run("server error returns error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "internal error"})
		}))
		defer server.Close()

		cfg := &Configuration{
			ApiUrl:    server.URL,
			AuthToken: "dash0_at_token",
			OAuth: &OAuthState{
				ClientID:     "cid",
				RefreshToken: "rt",
				ExpiresAt:    time.Now().Add(1 * time.Hour),
			},
		}
		err := revokeOAuthTokens(cfg)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "OAuth token revocation failed") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}
