package dash0

import (
	"net/http"
	"strings"
	"testing"
)

func TestNewClient(t *testing.T) {
	t.Run("requires API URL", func(t *testing.T) {
		_, err := NewClient(
			WithAuthToken("auth_test123"),
		)

		if err == nil {
			t.Fatal("expected error for missing API URL")
		}
		if !strings.Contains(err.Error(), "API URL is required") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("requires auth token", func(t *testing.T) {
		_, err := NewClient(
			WithApiUrl("https://api.example.com"),
		)

		if err == nil {
			t.Fatal("expected error for missing auth token")
		}
		if !strings.Contains(err.Error(), "auth token is required") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("auth token must start with auth_", func(t *testing.T) {
		_, err := NewClient(
			WithApiUrl("https://api.example.com"),
			WithAuthToken("invalid_token"),
		)

		if err == nil {
			t.Fatal("expected error for invalid auth token prefix")
		}
		if !strings.Contains(err.Error(), "must start with 'auth_'") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("accepts valid auth token", func(t *testing.T) {
		client, err := NewClient(
			WithApiUrl("https://api.example.com"),
			WithAuthToken("auth_validtoken123"),
		)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client == nil {
			t.Fatal("expected client to be created")
		}
	})

	t.Run("clamps max concurrent requests to minimum", func(t *testing.T) {
		c, err := NewClient(
			WithApiUrl("https://api.example.com"),
			WithAuthToken("auth_test"),
			WithMaxConcurrentRequests(0),
		)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		impl := c.(*client)
		if impl.config.maxConcurrent != 1 {
			t.Errorf("maxConcurrent = %d, want 1", impl.config.maxConcurrent)
		}
	})

	t.Run("clamps max concurrent requests to maximum", func(t *testing.T) {
		c, err := NewClient(
			WithApiUrl("https://api.example.com"),
			WithAuthToken("auth_test"),
			WithMaxConcurrentRequests(100),
		)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		impl := c.(*client)
		if impl.config.maxConcurrent != MaxConcurrentRequests {
			t.Errorf("maxConcurrent = %d, want %d", impl.config.maxConcurrent, MaxConcurrentRequests)
		}
	})

	t.Run("applies rate limiting with custom HTTP client", func(t *testing.T) {
		customClient := &http.Client{
			Transport: http.DefaultTransport,
		}

		c, err := NewClient(
			WithApiUrl("https://api.example.com"),
			WithAuthToken("auth_test"),
			WithHTTPClient(customClient),
		)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify the inner client's HTTP client has transport wrapping applied
		// The transport stack is: retryTransport -> rateLimitedTransport -> base
		impl := c.(*client)
		innerClient := impl.inner.ClientInterface.(*generatedClient)
		httpClient := innerClient.Client.(*http.Client)
		retry, isRetry := httpClient.Transport.(*retryTransport)
		if !isRetry {
			t.Fatal("expected retry transport to be applied")
		}
		_, isRateLimited := retry.base.(*rateLimitedTransport)
		if !isRateLimited {
			t.Error("expected rate limiting to be applied with custom HTTP client")
		}
	})

	t.Run("preserves custom HTTP client settings", func(t *testing.T) {
		customRedirect := func(req *http.Request, via []*http.Request) error {
			return nil
		}

		customClient := &http.Client{
			CheckRedirect: customRedirect,
		}

		c, err := NewClient(
			WithApiUrl("https://api.example.com"),
			WithAuthToken("auth_test"),
			WithHTTPClient(customClient),
		)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify CheckRedirect was preserved
		impl := c.(*client)
		innerClient := impl.inner.ClientInterface.(*generatedClient)
		httpClient := innerClient.Client.(*http.Client)
		if httpClient.CheckRedirect == nil {
			t.Error("expected CheckRedirect to be preserved")
		}
	})
}
