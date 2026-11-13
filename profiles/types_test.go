package profiles

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestOAuthState_ZeroValueElidesAllFields(t *testing.T) {
	data, err := json.Marshal(OAuthState{})
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if got := string(data); got != "{}" {
		t.Errorf("zero-valued OAuthState marshaled to %s, want {}", got)
	}

	var round OAuthState
	if err := json.Unmarshal(data, &round); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if round != (OAuthState{}) {
		t.Errorf("round-trip produced %+v, want zero value", round)
	}
	if !round.ExpiresAt.IsZero() {
		t.Errorf("ExpiresAt = %v, want zero time", round.ExpiresAt)
	}
}

func TestOAuthState_PopulatedRoundTrip(t *testing.T) {
	original := OAuthState{
		ClientID:     "cid",
		RefreshToken: "rt",
		ExpiresAt:    time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC),
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var round OAuthState
	if err := json.Unmarshal(data, &round); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if round != original {
		t.Errorf("round-trip produced %+v, want %+v", round, original)
	}
}

func TestConfigurationHasCredentials(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  *Configuration
		want bool
	}{
		{name: "nil", cfg: nil, want: false},
		{name: "nothing set", cfg: &Configuration{}, want: false},
		{name: "static token", cfg: &Configuration{AuthToken: "auth_x"}, want: true},
		{name: "OAuth profile never logged in", cfg: &Configuration{OAuth: &OAuthState{}}, want: false},
		{
			name: "OAuth state with an access token",
			cfg:  &Configuration{OAuth: &OAuthState{AccessToken: "dash0_at_x"}},
			want: true,
		},
		{
			name: "empty token is not a credential",
			cfg:  &Configuration{ApiUrl: "https://api.example.com"},
			want: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.cfg.HasCredentials(); got != tc.want {
				t.Errorf("HasCredentials() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestConfigurationDatasetPtrNil(t *testing.T) {
	var cfg *Configuration
	if got := cfg.DatasetPtr(); got != nil {
		t.Errorf("DatasetPtr() = %v, want nil", got)
	}
}

func TestResolvedConfigurationToken(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  Configuration
		want string
	}{
		{name: "static token", cfg: Configuration{AuthToken: "auth_x"}, want: "auth_x"},
		{
			name: "OAuth access token",
			cfg:  Configuration{OAuth: &OAuthState{AccessToken: "dash0_at_x"}},
			want: "dash0_at_x",
		},
		{name: "OAuth profile never logged in", cfg: Configuration{OAuth: &OAuthState{}}, want: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := newResolved(tc.cfg, "test", "/config")
			if got := cfg.Token(); got != tc.want {
				t.Errorf("Token() = %q, want %q", got, tc.want)
			}
			if got := cfg.HasCredentials(); got != (tc.want != "") {
				t.Errorf("HasCredentials() = %v, want %v", got, tc.want != "")
			}
		})
	}
}

func TestResolvedConfigurationOAuthReturnsACopy(t *testing.T) {
	cfg := newResolved(Configuration{OAuth: &OAuthState{AccessToken: "dash0_at_x"}}, "test", "/config")

	cfg.OAuth().AccessToken = "changed"

	if got := cfg.Token(); got != "dash0_at_x" {
		t.Errorf("Token() = %q after changing the returned OAuth state, want %q", got, "dash0_at_x")
	}
}

func TestResolvedConfigurationNil(t *testing.T) {
	var cfg *ResolvedConfiguration
	if cfg.ApiUrl() != "" || cfg.OtlpUrl() != "" || cfg.Dataset() != "" || cfg.Token() != "" ||
		cfg.ProfileName() != "" || cfg.ConfigDir() != "" {
		t.Error("expected empty strings from a nil ResolvedConfiguration")
	}
	if cfg.DatasetPtr() != nil {
		t.Error("DatasetPtr() on nil, want nil")
	}
	if cfg.OAuth() != nil {
		t.Error("OAuth() on nil, want nil")
	}
	if cfg.HasCredentials() {
		t.Error("HasCredentials() on nil, want false")
	}
	if opts := cfg.ClientOptions(); len(opts) != 0 {
		t.Errorf("ClientOptions() on nil returned %d options, want 0", len(opts))
	}
	if token, _ := cfg.AuthTokenProvider().AuthToken(context.Background()); token != "" {
		t.Errorf("AuthTokenProvider() on nil served %q, want an empty token", token)
	}
}

func TestResolvedConfigurationSource_NilAndUnknownField(t *testing.T) {
	var nilConfig *ResolvedConfiguration
	if got := nilConfig.Source(FieldApiUrl); got != SourceUnset {
		t.Errorf("nil Source(FieldApiUrl) = %q, want %q", got, SourceUnset)
	}

	cfg := &ResolvedConfiguration{}
	cfg.setApiUrl("https://api.example.com", SourceArgument)
	for _, field := range []ConfigurationField{-1, fieldCount, 99} {
		if got := cfg.Source(field); got != SourceUnset {
			t.Errorf("Source(%d) = %q, want %q", field, got, SourceUnset)
		}
	}
}
