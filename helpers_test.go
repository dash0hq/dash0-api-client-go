package dash0

import (
	"net/http"
	"testing"
)

func assertEqual(t *testing.T, field, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %q, want %q", field, got, want)
	}
}

func assertPtrEqual[T comparable](t *testing.T, field string, got *T, want T) {
	t.Helper()
	if got == nil {
		t.Errorf("%s is nil, want %v", field, want)
		return
	}
	if *got != want {
		t.Errorf("%s = %v, want %v", field, *got, want)
	}
}

func newTestClient(t *testing.T, serverURL string) Client {
	t.Helper()
	c, err := NewClient(
		WithApiUrl(serverURL),
		WithAuthToken("auth_test123"),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	return c
}

// beneathAuthTransport returns the transport stack below the [authTransport]
// that [NewClient] installs at the top of every client's stack.
// Tests that assert on the retry or rate-limiting layers go through this helper
// so they describe the layer they mean rather than the outermost one.
func beneathAuthTransport(t *testing.T, roundTripper http.RoundTripper) http.RoundTripper {
	t.Helper()
	auth, ok := roundTripper.(*authTransport)
	if !ok {
		t.Fatalf("expected the outermost transport to be *authTransport, got %T", roundTripper)
	}
	return auth.base
}
