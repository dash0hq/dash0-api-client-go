package profiles

import (
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
		cfg  Configuration
		want bool
	}{
		{name: "nothing set", cfg: Configuration{}, want: false},
		{name: "static token", cfg: Configuration{AuthToken: "auth_x"}, want: true},
		{name: "OAuth state only", cfg: Configuration{OAuth: &OAuthState{}}, want: true},
		{
			name: "OAuth state with an access token",
			cfg:  Configuration{AuthToken: "dash0_at_x", OAuth: &OAuthState{}},
			want: true,
		},
		{
			name: "empty token is not a credential",
			cfg:  Configuration{ApiUrl: "https://api.example.com"},
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
