package dash0

import (
	"context"
	"strings"
	"testing"
)

func TestStaticAuthTokenProvider(t *testing.T) {
	t.Run("returns the token it was built with", func(t *testing.T) {
		provider := StaticAuthTokenProvider("auth_static")

		got, err := provider.AuthToken(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertEqual(t, "AuthToken", got, "auth_static")
	})

	t.Run("returns the same token on every call", func(t *testing.T) {
		provider := StaticAuthTokenProvider("dash0_at_static")

		for i := range 3 {
			got, err := provider.AuthToken(context.Background())
			if err != nil {
				t.Fatalf("call %d: unexpected error: %v", i, err)
			}
			assertEqual(t, "AuthToken", got, "dash0_at_static")
		}
	})

	t.Run("does not implement RefreshingAuthTokenProvider", func(t *testing.T) {
		// A fixed token cannot be refreshed, so the client must surface a 401
		// rather than attempt a replay.
		if _, ok := StaticAuthTokenProvider("auth_static").(RefreshingAuthTokenProvider); ok {
			t.Error("StaticAuthTokenProvider must not implement RefreshingAuthTokenProvider")
		}
	})

	t.Run("tolerates an empty token", func(t *testing.T) {
		// Rejecting the shape is validateAuthToken's job, at the point of use.
		// The provider itself is a plain carrier.
		got, err := StaticAuthTokenProvider("").AuthToken(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertEqual(t, "AuthToken", got, "")
	})
}

func TestValidateAuthToken(t *testing.T) {
	tests := []struct {
		name      string
		authToken string
		wantErr   string
	}{
		{name: "static token prefix", authToken: "auth_abc123"},
		{name: "OAuth access token prefix", authToken: "dash0_at_abc123"},
		{name: "prefix only", authToken: "auth_"},
		{name: "empty", authToken: "", wantErr: "auth token is empty"},
		{name: "no recognized prefix", authToken: "abc123", wantErr: "must start with"},
		{name: "prefix not at the start", authToken: "xauth_abc123", wantErr: "must start with"},
		{name: "wrong case", authToken: "AUTH_abc123", wantErr: "must start with"},
		{name: "bearer prefix included", authToken: "Bearer auth_abc", wantErr: "must start with"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateAuthToken(tc.authToken)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected an error containing %q", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err, tc.wantErr)
			}
		})
	}
}
