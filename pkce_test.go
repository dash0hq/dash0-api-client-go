package dash0

import (
	"crypto/sha256"
	"encoding/base64"
	"regexp"
	"testing"
)

// base64urlNoPad matches the unpadded base64url alphabet (RFC 4648 §5).
var base64urlNoPad = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func TestGeneratePKCEPair_Format(t *testing.T) {
	pair, err := GeneratePKCEPair()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pair.Verifier) != 43 {
		t.Errorf("verifier length = %d, want 43", len(pair.Verifier))
	}
	if !base64urlNoPad.MatchString(pair.Verifier) {
		t.Errorf("verifier %q is not unpadded base64url", pair.Verifier)
	}
	if len(pair.Challenge) != 43 {
		t.Errorf("challenge length = %d, want 43", len(pair.Challenge))
	}
	if !base64urlNoPad.MatchString(pair.Challenge) {
		t.Errorf("challenge %q is not unpadded base64url", pair.Challenge)
	}
}

func TestGeneratePKCEPair_ChallengeDerivedFromVerifier(t *testing.T) {
	pair, err := GeneratePKCEPair()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sum := sha256.Sum256([]byte(pair.Verifier))
	want := base64.RawURLEncoding.EncodeToString(sum[:])
	if pair.Challenge != want {
		t.Errorf("challenge = %q, want %q (base64url(sha256(verifier)))",
			pair.Challenge, want)
	}
}

// TestGeneratePKCEPair_RFC7636AppendixB locks in the S256 derivation against
// the canonical example verifier/challenge pair from RFC 7636 Appendix B.
func TestGeneratePKCEPair_RFC7636AppendixB(t *testing.T) {
	const (
		verifier  = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
		challenge = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
	)
	sum := sha256.Sum256([]byte(verifier))
	got := base64.RawURLEncoding.EncodeToString(sum[:])
	if got != challenge {
		t.Errorf("RFC 7636 Appendix B challenge = %q, want %q", got, challenge)
	}
}

func TestGeneratePKCEPair_FreshOnEveryCall(t *testing.T) {
	a, err := GeneratePKCEPair()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, err := GeneratePKCEPair()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Verifier == b.Verifier {
		t.Error("consecutive calls returned identical verifiers")
	}
	if a.Challenge == b.Challenge {
		t.Error("consecutive calls returned identical challenges")
	}
}

func TestGenerateOAuthState_Format(t *testing.T) {
	state, err := GenerateOAuthState()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(state) != 43 {
		t.Errorf("state length = %d, want 43", len(state))
	}
	if !base64urlNoPad.MatchString(state) {
		t.Errorf("state %q is not unpadded base64url", state)
	}
}

func TestGenerateOAuthState_FreshOnEveryCall(t *testing.T) {
	a, err := GenerateOAuthState()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, err := GenerateOAuthState()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a == b {
		t.Error("consecutive calls returned identical state values")
	}
}
