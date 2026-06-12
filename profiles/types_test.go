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
