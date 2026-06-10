package dash0

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// PKCEPair is a freshly-generated PKCE code verifier and the matching
// S256 code challenge, suitable for an OAuth 2.0 authorization-code-with-PKCE
// flow (RFC 7636).
type PKCEPair struct {
	Verifier  string // pass back to ExchangeToken as CodeVerifier
	Challenge string // pass to AuthorizeURL as CodeChallenge with method S256
}

// GeneratePKCEPair returns a fresh verifier and its S256 challenge per RFC 7636.
// The verifier is base64url(32 random bytes) -- 43 characters, no padding.
func GeneratePKCEPair() (PKCEPair, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return PKCEPair{}, fmt.Errorf("dash0: PKCE verifier generation failed: %w", err)
	}
	verifier := base64.RawURLEncoding.EncodeToString(buf)
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	return PKCEPair{Verifier: verifier, Challenge: challenge}, nil
}

// GenerateOAuthState returns a fresh opaque value suitable for the OAuth
// `state` parameter (RFC 6749 §10.12). 43 base64url characters, no padding.
func GenerateOAuthState() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("dash0: OAuth state generation failed: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
