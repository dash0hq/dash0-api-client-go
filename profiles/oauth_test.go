package profiles

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
		err := refreshOAuthToken(context.Background(), store, "test", cfg)
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

		err := refreshOAuthToken(context.Background(), store, "test", cfg)
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
		err := refreshOAuthToken(context.Background(), store, "test", cfg)
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
		err := refreshOAuthToken(context.Background(), store, "test", cfg)
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

	t.Run("invalid_grant clears OAuth state and returns ErrReauthenticationRequired", func(t *testing.T) {
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
		err := refreshOAuthToken(context.Background(), store, "test", cfg)
		if !errors.Is(err, ErrReauthenticationRequired) {
			t.Fatalf("expected ErrReauthenticationRequired, got: %v", err)
		}
		if cfg.OAuth != nil {
			t.Errorf("expected cfg.OAuth cleared after invalid_grant, got %+v", cfg.OAuth)
		}
		if cfg.AuthToken != "" {
			t.Errorf("expected cfg.AuthToken cleared after invalid_grant, got %q", cfg.AuthToken)
		}
		// Verify the on-disk state was also cleared so a subsequent call
		// does not retry the dead credential.
		ps, err := store.GetProfiles()
		if err != nil {
			t.Fatalf("GetProfiles: %v", err)
		}
		if ps[0].Configuration.OAuth != nil {
			t.Errorf("expected persisted OAuth cleared after invalid_grant, got %+v", ps[0].Configuration.OAuth)
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
		err := refreshOAuthToken(context.Background(), store, "test", cfg)
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
				errs[i] = refreshOAuthToken(context.Background(), store, "test", cfgs[i])
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

	t.Run("cross-process flock serializes concurrent refreshes via separate Store instances", func(t *testing.T) {
		// Two distinct *Store instances over the same config directory
		// mimic two CLI invocations: the in-process refreshMu does not
		// connect them, so only the file lock prevents both from hitting
		// the token endpoint simultaneously and rotating the same family.
		var inFlight, peak atomic.Int32
		var release sync.WaitGroup
		release.Add(1)
		var requestCount atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			n := inFlight.Add(1)
			defer inFlight.Add(-1)
			for {
				p := peak.Load()
				if n <= p || peak.CompareAndSwap(p, n) {
					break
				}
			}
			requestCount.Add(1)
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

		_, dir := newTestStore(t)
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

		const processes = 4
		errs := make([]error, processes)
		var done sync.WaitGroup
		done.Add(processes)
		for i := range processes {
			go func() {
				defer done.Done()
				storeI, _ := NewStore(WithConfigDir(dir))
				cfg := &Configuration{
					ApiUrl:    server.URL,
					AuthToken: "dash0_at_old",
					OAuth: &OAuthState{
						ClientID:     "cid",
						RefreshToken: "rt-original",
						ExpiresAt:    time.Now().Add(1 * time.Minute),
					},
				}
				errs[i] = refreshOAuthToken(context.Background(), storeI, "test", cfg)
			}()
		}
		// Give every goroutine a chance to enter the locked section before
		// the first request returns.
		time.Sleep(100 * time.Millisecond)
		release.Done()
		done.Wait()

		for i, err := range errs {
			if err != nil {
				t.Errorf("process %d: unexpected error: %v", i, err)
			}
		}
		// At most one process should ever be in-flight at the token
		// endpoint at once — that is the property the flock guarantees.
		if got := peak.Load(); got > 1 {
			t.Errorf("peak concurrent token-endpoint requests = %d, want 1 (cross-process flock failed)", got)
		}
		// Subsequent processes pick up the persisted rotated token and
		// short-circuit, so the request count is usually 1; allow up to 2
		// in case scheduler timing lets two processes both pass the expiry
		// check before either acquires the lock.
		if got := requestCount.Load(); got > 2 {
			t.Errorf("expected at most 2 token-endpoint requests, got %d", got)
		}
	})

	t.Run("GetProfiles failure inside the lock surfaces a wrapped error", func(t *testing.T) {
		// Corrupt profiles.json so GetProfiles fails after the refresh
		// mutex is acquired. The function should return a wrapped error
		// instead of panicking or silently zeroing cfg.
		store, dir := newTestStore(t)
		profilesPath := filepath.Join(dir, ProfilesFileName)
		if err := os.WriteFile(profilesPath, []byte("not json"), 0o600); err != nil {
			t.Fatalf("seed corrupt profile: %v", err)
		}

		cfg := &Configuration{
			ApiUrl:    "https://api.eu-west-1.aws.dash0.com",
			AuthToken: "dash0_at_old",
			OAuth: &OAuthState{
				ClientID:     "cid",
				RefreshToken: "rt",
				ExpiresAt:    time.Now().Add(-1 * time.Minute),
			},
		}
		err := refreshOAuthToken(context.Background(), store, "test", cfg)
		if err == nil {
			t.Fatal("expected error from corrupted profile, got nil")
		}
		if !strings.Contains(err.Error(), "re-read profiles") {
			t.Errorf("error should mention re-read failure, got: %v", err)
		}
	})

	t.Run("profile removed concurrently returns dedicated error", func(t *testing.T) {
		// The profile vanished between snapshot and post-lock re-read.
		// Surface a dedicated error rather than silently zeroing cfg.
		store, _ := newTestStore(t)
		cfg := &Configuration{
			ApiUrl:    "https://api.eu-west-1.aws.dash0.com",
			AuthToken: "dash0_at_old",
			OAuth: &OAuthState{
				ClientID:     "cid",
				RefreshToken: "rt",
				ExpiresAt:    time.Now().Add(-1 * time.Minute),
			},
		}
		err := refreshOAuthToken(context.Background(), store, "missing-profile", cfg)
		if err == nil {
			t.Fatal("expected error for missing profile, got nil")
		}
		if !strings.Contains(err.Error(), "no longer exists") {
			t.Errorf("error should mention missing profile, got: %v", err)
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
		err := refreshOAuthToken(context.Background(), store, "test", cfg)
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
		err := revokeOAuthTokens(context.Background(), cfg)
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
		err := revokeOAuthTokens(context.Background(), cfg)
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
		err := revokeOAuthTokens(context.Background(), cfg)
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
		err := revokeOAuthTokens(context.Background(), cfg)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "OAuth token revocation failed") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

func TestValidateRefreshResponse(t *testing.T) {
	cases := []struct {
		name    string
		resp    *dash0.OAuthTokenResponse
		wantErr string
	}{
		{
			name:    "nil response",
			resp:    nil,
			wantErr: "nil response",
		},
		{
			name:    "empty access_token",
			resp:    &dash0.OAuthTokenResponse{TokenType: "Bearer", ExpiresIn: 3600},
			wantErr: "empty access_token",
		},
		{
			name:    "non-Bearer token_type",
			resp:    &dash0.OAuthTokenResponse{AccessToken: "at", TokenType: "MAC", ExpiresIn: 3600},
			wantErr: "unsupported token_type",
		},
		{
			name:    "zero expires_in",
			resp:    &dash0.OAuthTokenResponse{AccessToken: "at", TokenType: "Bearer", ExpiresIn: 0},
			wantErr: "non-positive expires_in",
		},
		{
			name:    "negative expires_in",
			resp:    &dash0.OAuthTokenResponse{AccessToken: "at", TokenType: "Bearer", ExpiresIn: -1},
			wantErr: "non-positive expires_in",
		},
		{
			name:    "expires_in beyond 24h ceiling",
			resp:    &dash0.OAuthTokenResponse{AccessToken: "at", TokenType: "Bearer", ExpiresIn: int64((25 * time.Hour) / time.Second)},
			wantErr: "implausibly large expires_in",
		},
		{
			name: "happy path with empty token_type defaults to OK",
			resp: &dash0.OAuthTokenResponse{AccessToken: "at", ExpiresIn: 3600},
		},
		{
			name: "happy path with Bearer (case-insensitive)",
			resp: &dash0.OAuthTokenResponse{AccessToken: "at", TokenType: "bearer", ExpiresIn: 3600},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateRefreshResponse(c.resp)
			if c.wantErr == "" {
				if err != nil {
					t.Errorf("expected nil error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", c.wantErr)
			}
			if !strings.Contains(err.Error(), c.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), c.wantErr)
			}
		})
	}
}

func TestIsTransientRefreshError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"5xx APIError", &dash0.APIError{StatusCode: 503}, true},
		{"4xx APIError", &dash0.APIError{StatusCode: 401}, false},
		{"OAuthTokenError (invalid_grant)", &dash0.OAuthTokenError{StatusCode: 400, Code: "invalid_grant"}, false},
		{"OAuthTokenError (invalid_client)", &dash0.OAuthTokenError{StatusCode: 400, Code: "invalid_client"}, false},
		{"unclassified network-like error", errors.New("connection refused"), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isTransientRefreshError(c.err); got != c.want {
				t.Errorf("isTransientRefreshError = %v, want %v", got, c.want)
			}
		})
	}
}

func TestExchangeRefreshTokenWithRetry(t *testing.T) {
	t.Run("retries on 5xx and eventually succeeds", func(t *testing.T) {
		var hits atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			n := hits.Add(1)
			if n < 2 {
				w.WriteHeader(http.StatusBadGateway)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "dash0_at_ok",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
		}))
		defer server.Close()

		oauthClient, err := dash0.NewOAuthClient(dash0.WithApiUrl(server.URL))
		if err != nil {
			t.Fatalf("create client: %v", err)
		}
		defer func() { _ = oauthClient.Close(context.Background()) }()

		resp, err := exchangeRefreshTokenWithRetry(context.Background(), oauthClient, "cid", "rt")
		if err != nil {
			t.Fatalf("expected success after retry, got %v", err)
		}
		if resp.AccessToken != "dash0_at_ok" {
			t.Errorf("AccessToken = %q, want dash0_at_ok", resp.AccessToken)
		}
		if got := hits.Load(); got != 2 {
			t.Errorf("expected 2 requests (1 transient + 1 success), got %d", got)
		}
	})

	t.Run("invalid_grant short-circuits without retry", func(t *testing.T) {
		var hits atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hits.Add(1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant"})
		}))
		defer server.Close()

		oauthClient, err := dash0.NewOAuthClient(dash0.WithApiUrl(server.URL))
		if err != nil {
			t.Fatalf("create client: %v", err)
		}
		defer func() { _ = oauthClient.Close(context.Background()) }()

		_, err = exchangeRefreshTokenWithRetry(context.Background(), oauthClient, "cid", "rt")
		if !dash0.IsOAuthInvalidGrant(err) {
			t.Fatalf("expected invalid_grant, got %v", err)
		}
		if got := hits.Load(); got != 1 {
			t.Errorf("expected 1 request (no retry on invalid_grant), got %d", got)
		}
	})

	t.Run("cancelled context returns ctx.Err without burning extra attempts", func(t *testing.T) {
		var hits atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hits.Add(1)
			w.WriteHeader(http.StatusBadGateway)
		}))
		defer server.Close()

		oauthClient, err := dash0.NewOAuthClient(dash0.WithApiUrl(server.URL))
		if err != nil {
			t.Fatalf("create client: %v", err)
		}
		defer func() { _ = oauthClient.Close(context.Background()) }()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err = exchangeRefreshTokenWithRetry(ctx, oauthClient, "cid", "rt")
		if err == nil {
			t.Fatal("expected error from cancelled context, got nil")
		}
		// One attempt may fire before the cancellation is observed; the
		// retry loop's inter-attempt select must then bail rather than
		// burn additional round-trips.
		if got := hits.Load(); got > 1 {
			t.Errorf("expected at most 1 request with cancelled context, got %d", got)
		}
	})
}
