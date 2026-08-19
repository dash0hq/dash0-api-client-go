package profiles

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	dash0 "github.com/dash0hq/dash0-api-client-go"
)

// seedOAuthProfile writes a single OAuth profile named "test" into a fresh
// store and returns the store, its config directory, and a Configuration that
// mirrors what a resolver would hand a caller.
func seedOAuthProfile(t *testing.T, apiUrl, accessToken string, expiresAt time.Time) (*Store, string, *Configuration) {
	t.Helper()
	store, dir := newTestStore(t)
	cfg := &Configuration{
		ApiUrl:      apiUrl,
		AuthToken:   accessToken,
		ProfileName: "test",
		OAuth: &OAuthState{
			ClientID:     "cid",
			RefreshToken: "rt",
			ExpiresAt:    expiresAt,
		},
	}
	persisted := *cfg
	createTestProfilesFile(t, dir, []Profile{{Name: "test", Configuration: persisted}})
	return store, dir, cfg
}

func TestConfigurationAuthTokenProvider(t *testing.T) {
	t.Run("serves a static token unchanged when the profile is not OAuth", func(t *testing.T) {
		cfg := &Configuration{AuthToken: "auth_static"}
		provider := cfg.AuthTokenProvider()

		got, err := provider.AuthToken(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "auth_static" {
			t.Errorf("AuthToken = %q, want %q", got, "auth_static")
		}
	})

	t.Run("does not refresh a token that is not near expiry", func(t *testing.T) {
		var calls atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			calls.Add(1)
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		_, dir, cfg := seedOAuthProfile(t, server.URL, "dash0_at_current", time.Now().Add(1*time.Hour))
		provider := cfg.AuthTokenProvider(WithConfigDir(dir))

		got, err := provider.AuthToken(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "dash0_at_current" {
			t.Errorf("AuthToken = %q, want %q", got, "dash0_at_current")
		}
		if n := calls.Load(); n != 0 {
			t.Errorf("token endpoint called %d times, want 0", n)
		}
	})

	t.Run("refreshes a token inside the refresh threshold", func(t *testing.T) {
		server := newTokenServer(t, tokenServerResponse{
			AccessToken: "dash0_at_refreshed",
			ExpiresIn:   int64((15 * time.Minute).Seconds()),
		}, nil)
		defer server.Close()

		_, dir, cfg := seedOAuthProfile(t, server.URL, "dash0_at_stale", time.Now().Add(1*time.Minute))
		provider := cfg.AuthTokenProvider(WithConfigDir(dir))

		got, err := provider.AuthToken(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "dash0_at_refreshed" {
			t.Errorf("AuthToken = %q, want %q", got, "dash0_at_refreshed")
		}
	})

	t.Run("leaves the caller's Configuration untouched", func(t *testing.T) {
		server := newTokenServer(t, tokenServerResponse{
			AccessToken: "dash0_at_refreshed",
			ExpiresIn:   int64((15 * time.Minute).Seconds()),
		}, nil)
		defer server.Close()

		_, dir, cfg := seedOAuthProfile(t, server.URL, "dash0_at_stale", time.Now().Add(1*time.Minute))
		provider := cfg.AuthTokenProvider(WithConfigDir(dir))

		if _, err := provider.AuthToken(context.Background()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.AuthToken != "dash0_at_stale" {
			t.Errorf("caller's AuthToken = %q, want it left at %q", cfg.AuthToken, "dash0_at_stale")
		}
	})

	t.Run("serves the refreshed token on subsequent calls without refreshing again", func(t *testing.T) {
		var calls atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/oauth/token" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			calls.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "dash0_at_refreshed",
				"token_type":   "Bearer",
				"expires_in":   int64((15 * time.Minute).Seconds()),
			})
		}))
		defer server.Close()

		_, dir, cfg := seedOAuthProfile(t, server.URL, "dash0_at_stale", time.Now().Add(1*time.Minute))
		provider := cfg.AuthTokenProvider(WithConfigDir(dir))

		for i := range 5 {
			got, err := provider.AuthToken(context.Background())
			if err != nil {
				t.Fatalf("call %d: unexpected error: %v", i, err)
			}
			if got != "dash0_at_refreshed" {
				t.Fatalf("call %d: AuthToken = %q, want %q", i, got, "dash0_at_refreshed")
			}
		}
		if n := calls.Load(); n != 1 {
			t.Errorf("token endpoint called %d times, want 1", n)
		}
	})

	t.Run("concurrent callers trigger exactly one refresh", func(t *testing.T) {
		var calls atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/oauth/token" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			calls.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "dash0_at_refreshed",
				"token_type":   "Bearer",
				"expires_in":   int64((15 * time.Minute).Seconds()),
			})
		}))
		defer server.Close()

		_, dir, cfg := seedOAuthProfile(t, server.URL, "dash0_at_stale", time.Now().Add(1*time.Minute))
		provider := cfg.AuthTokenProvider(WithConfigDir(dir))

		const goroutines = 10
		var start sync.WaitGroup
		start.Add(1)
		var done sync.WaitGroup
		tokens := make([]string, goroutines)
		errs := make([]error, goroutines)
		for i := range goroutines {
			done.Add(1)
			go func() {
				defer done.Done()
				start.Wait()
				tokens[i], errs[i] = provider.AuthToken(context.Background())
			}()
		}
		start.Done()
		done.Wait()

		for i := range goroutines {
			if errs[i] != nil {
				t.Fatalf("goroutine %d: unexpected error: %v", i, errs[i])
			}
			if tokens[i] != "dash0_at_refreshed" {
				t.Errorf("goroutine %d: AuthToken = %q, want %q", i, tokens[i], "dash0_at_refreshed")
			}
		}
		if n := calls.Load(); n != 1 {
			t.Errorf("token endpoint called %d times, want 1", n)
		}
	})

	t.Run("reports a configuration that does not name its profile", func(t *testing.T) {
		cfg := &Configuration{
			ApiUrl:    "https://api.example.com",
			AuthToken: "dash0_at_stale",
			OAuth: &OAuthState{
				ClientID:     "cid",
				RefreshToken: "rt",
				ExpiresAt:    time.Now().Add(1 * time.Minute),
			},
		}
		provider := cfg.AuthTokenProvider(WithConfigDir(t.TempDir()))

		_, err := provider.AuthToken(context.Background())
		if !errors.Is(err, ErrNoAssociatedProfile) {
			t.Fatalf("error = %v, want it to match ErrNoAssociatedProfile", err)
		}
		// This reaches an end user whenever a front end forgets to translate it,
		// so the wording must not carry Go identifiers or internal concepts.
		for _, leak := range []string{"ProfileName", "ConfigDir", "Store", "store", "Configuration"} {
			if strings.Contains(err.Error(), leak) {
				t.Errorf("error = %q, leaks the internal term %q", err, leak)
			}
		}

	})

	t.Run("surfaces ErrReauthenticationRequired when the refresh token is rejected", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": "invalid_grant"})
		}))
		defer server.Close()

		_, dir, cfg := seedOAuthProfile(t, server.URL, "dash0_at_stale", time.Now().Add(1*time.Minute))
		provider := cfg.AuthTokenProvider(WithConfigDir(dir))

		_, err := provider.AuthToken(context.Background())
		if !errors.Is(err, ErrReauthenticationRequired) {
			t.Fatalf("error = %v, want it to match ErrReauthenticationRequired", err)
		}

		// The state is terminal: a second call must not hand back an empty
		// token now that the stored OAuth state has been cleared.
		_, err = provider.AuthToken(context.Background())
		if !errors.Is(err, ErrReauthenticationRequired) {
			t.Errorf("second call error = %v, want it to match ErrReauthenticationRequired", err)
		}
	})
}

func TestConfigurationClientOptionsUsesProviderForOAuth(t *testing.T) {
	t.Run("OAuth configuration yields a client that builds without a static token", func(t *testing.T) {
		// No t.Setenv(EnvConfigDir, ...) here on purpose: the configuration
		// carries the directory it was resolved from, so the provider finds its
		// profile without help from the environment.
		store, dir, _ := seedOAuthProfile(t, "https://api.example.com", "dash0_at_current", time.Now().Add(1*time.Hour))
		t.Setenv(EnvConfigDir, "")
		_ = dir

		cfg, err := store.GetConfigurationForProfile(context.Background(), "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		client, err := dash0.NewClient(cfg.ClientOptions()...)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer func() { _ = client.Close(context.Background()) }()
	})

	t.Run("static configuration still maps to WithAuthToken", func(t *testing.T) {
		cfg := &Configuration{ApiUrl: "https://api.example.com", AuthToken: "auth_static"}

		client, err := dash0.NewClient(cfg.ClientOptions()...)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer func() { _ = client.Close(context.Background()) }()
	})
}

// TestRefreshSurvivesCallerCancellation covers the hazard exchangeRefreshToken's
// own doc cites: a refresh cancelled between the server rotating the refresh
// token and the response being read burns the stored credential.
//
// The provider is consulted on the API request's deadline, so without detaching
// the refresh an ordinary request timeout would be enough to force an
// interactive re-login.
func TestRefreshSurvivesCallerCancellation(t *testing.T) {
	released := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/token" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		// Hold the response until the caller's context has been cancelled.
		<-released
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "dash0_at_rotated",
			"refresh_token": "rt-rotated",
			"token_type":    "Bearer",
			"expires_in":    int64((15 * time.Minute).Seconds()),
		})
	}))
	defer server.Close()

	_, dir, cfg := seedOAuthProfile(t, server.URL, "dash0_at_stale", time.Now().Add(1*time.Minute))
	provider := cfg.AuthTokenProvider(WithConfigDir(dir))

	ctx, cancel := context.WithCancel(context.Background())
	type result struct {
		token string
		err   error
	}
	results := make(chan result, 1)
	go func() {
		token, err := provider.AuthToken(ctx)
		results <- result{token, err}
	}()

	// Cancel while the authorization server is mid-rotation, then let it reply.
	cancel()
	close(released)

	got := <-results
	if got.err != nil {
		t.Fatalf("refresh should not inherit the caller's cancellation: %v", got.err)
	}
	if got.token != "dash0_at_rotated" {
		t.Errorf("token = %q, want the rotation to have completed and been persisted", got.token)
	}

	// The rotated credential must be on disk, or the next refresh gets
	// invalid_grant.
	store, err := NewStore(WithConfigDir(dir))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	persisted, err := store.GetConfigurationForProfile(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if persisted.AuthToken != "dash0_at_rotated" {
		t.Errorf("persisted AuthToken = %q, want the rotated token", persisted.AuthToken)
	}
	if persisted.OAuth == nil || persisted.OAuth.RefreshToken != "rt-rotated" {
		t.Errorf("persisted refresh token was not updated; the stored credential is burned")
	}
}

// TestAuthTokenProviderUsesTheStoreDirectory pins that a configuration resolved
// from a non-default directory refreshes against that same profiles.json.
//
// Resolving the directory afresh inside the provider would take .profile-lock on
// the wrong directory and, if a same-named profile existed in the default one,
// rotate an unrelated profile's refresh token.
func TestAuthTokenProviderUsesTheStoreDirectory(t *testing.T) {
	server := newTokenServer(t, tokenServerResponse{
		AccessToken: "dash0_at_refreshed",
		ExpiresIn:   int64((15 * time.Minute).Seconds()),
	}, nil)
	defer server.Close()

	store, dir := newTestStore(t)
	createTestProfilesFile(t, dir, []Profile{{Name: "test", Configuration: Configuration{
		ApiUrl:    server.URL,
		AuthToken: "dash0_at_stale",
		OAuth: &OAuthState{
			ClientID:     "cid",
			RefreshToken: "rt",
			ExpiresAt:    time.Now().Add(1 * time.Minute),
		},
	}}})

	// Deliberately unset, so the provider has to learn the directory from the
	// configuration rather than the environment.
	t.Setenv(EnvConfigDir, "")

	cfg, err := store.GetConfigurationForProfile(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := cfg.AuthTokenProvider().AuthToken(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "dash0_at_refreshed" {
		t.Errorf("AuthToken = %q, want %q", got, "dash0_at_refreshed")
	}
}
