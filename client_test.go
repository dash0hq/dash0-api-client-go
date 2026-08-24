package dash0

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	t.Run("requires at least API URL or OTLP endpoint", func(t *testing.T) {
		_, err := NewClient(
			WithAuthToken("auth_test123"),
		)

		if err == nil {
			t.Fatal("expected error for missing API URL and OTLP endpoint")
		}
		if !strings.Contains(err.Error(), "at least one of API URL or OTLP endpoint is required") {
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

	t.Run("auth token must start with auth_ or dash0_at_", func(t *testing.T) {
		_, err := NewClient(
			WithApiUrl("https://api.example.com"),
			WithAuthToken("invalid_token"),
		)

		if err == nil {
			t.Fatal("expected error for invalid auth token prefix")
		}
		if !strings.Contains(err.Error(), "must start with") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("accepts dash0_at_ token", func(t *testing.T) {
		client, err := NewClient(
			WithApiUrl("https://api.example.com"),
			WithAuthToken("dash0_at_oauthtoken123"),
		)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client == nil {
			t.Fatal("expected client to be created")
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
		retry, isRetry := beneathAuthTransport(t, httpClient.Transport).(*retryTransport)
		if !isRetry {
			t.Fatal("expected retry transport to be applied")
		}
		_, isRateLimited := retry.base.(*rateLimitedTransport)
		if !isRateLimited {
			t.Error("expected rate limiting to be applied with custom HTTP client")
		}
	})

	t.Run("WithRetryOnConflict reaches the retry transport", func(t *testing.T) {
		for _, enabled := range []bool{false, true} {
			opts := []ClientOption{
				WithApiUrl("https://api.example.com"),
				WithAuthToken("auth_test"),
			}
			if enabled {
				opts = append(opts, WithRetryOnConflict())
			}
			c, err := NewClient(opts...)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			impl := c.(*client)
			innerClient := impl.inner.ClientInterface.(*generatedClient)
			httpClient := innerClient.Client.(*http.Client)
			retry, isRetry := beneathAuthTransport(t, httpClient.Transport).(*retryTransport)
			if !isRetry {
				t.Fatal("expected retry transport to be applied")
			}
			if retry.retryOnConflict != enabled {
				t.Errorf("retryOnConflict = %v, want %v", retry.retryOnConflict, enabled)
			}
		}
	})

	t.Run("REST API methods return ErrAPINotConfigured without WithApiUrl", func(t *testing.T) {
		c, err := NewClient(
			WithAuthToken("auth_test123"),
			WithOtlpEndpoint(OtlpEncodingJson, "https://otlp.example.com"),
		)
		if err != nil {
			t.Fatalf("failed to create OTLP-only client: %v", err)
		}

		_, err = c.ListDashboards(context.Background(), nil)
		if !errors.Is(err, ErrAPINotConfigured) {
			t.Errorf("ListDashboards: expected ErrAPINotConfigured, got %v", err)
		}

		_, err = c.GetDashboard(context.Background(), "test", nil)
		if !errors.Is(err, ErrAPINotConfigured) {
			t.Errorf("GetDashboard: expected ErrAPINotConfigured, got %v", err)
		}

		if c.Inner() != nil {
			t.Error("Inner() should return nil for OTLP-only client")
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

	t.Run("accepts WithTransport", func(t *testing.T) {
		tr := NewTransport(
			WithTransportMaxRetries(3),
			WithTransportTimeout(10*time.Second),
		)

		c, err := NewClient(
			WithApiUrl("https://api.example.com"),
			WithAuthToken("auth_test"),
			WithTransport(tr),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		impl := c.(*client)
		innerClient := impl.inner.ClientInterface.(*generatedClient)
		httpClient := innerClient.Client.(*http.Client)

		// Verify the transport's RoundTripper is used
		if beneathAuthTransport(t, httpClient.Transport) != tr.RoundTripper() {
			t.Error("expected client to use the Transport's RoundTripper")
		}
		// Verify the transport's timeout is used
		if httpClient.Timeout != 10*time.Second {
			t.Errorf("timeout = %v, want 10s", httpClient.Timeout)
		}
	})

	t.Run("WithTransport with WithTimeout overrides transport timeout", func(t *testing.T) {
		tr := NewTransport(WithTransportTimeout(10 * time.Second))

		c, err := NewClient(
			WithApiUrl("https://api.example.com"),
			WithAuthToken("auth_test"),
			WithTransport(tr),
			WithTimeout(20*time.Second),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		impl := c.(*client)
		innerClient := impl.inner.ClientInterface.(*generatedClient)
		httpClient := innerClient.Client.(*http.Client)
		if httpClient.Timeout != 20*time.Second {
			t.Errorf("timeout = %v, want 20s (overridden by WithTimeout)", httpClient.Timeout)
		}
	})

	t.Run("WithTransport conflicts with WithMaxRetries", func(t *testing.T) {
		tr := NewTransport()
		_, err := NewClient(
			WithApiUrl("https://api.example.com"),
			WithAuthToken("auth_test"),
			WithTransport(tr),
			WithMaxRetries(2),
		)
		if err == nil {
			t.Fatal("expected error for conflicting options")
		}
		if !strings.Contains(err.Error(), "WithMaxRetries") {
			t.Errorf("error should mention WithMaxRetries: %v", err)
		}
	})

	t.Run("WithTransport conflicts with WithRetryWaitMin", func(t *testing.T) {
		tr := NewTransport()
		_, err := NewClient(
			WithApiUrl("https://api.example.com"),
			WithAuthToken("auth_test"),
			WithTransport(tr),
			WithRetryWaitMin(time.Second),
		)
		if err == nil {
			t.Fatal("expected error for conflicting options")
		}
		if !strings.Contains(err.Error(), "WithRetryWaitMin") {
			t.Errorf("error should mention WithRetryWaitMin: %v", err)
		}
	})

	t.Run("WithTransport conflicts with WithRetryWaitMax", func(t *testing.T) {
		tr := NewTransport()
		_, err := NewClient(
			WithApiUrl("https://api.example.com"),
			WithAuthToken("auth_test"),
			WithTransport(tr),
			WithRetryWaitMax(time.Second),
		)
		if err == nil {
			t.Fatal("expected error for conflicting options")
		}
		if !strings.Contains(err.Error(), "WithRetryWaitMax") {
			t.Errorf("error should mention WithRetryWaitMax: %v", err)
		}
	})

	t.Run("WithTransport conflicts with WithRetryOnConflict", func(t *testing.T) {
		tr := NewTransport()
		_, err := NewClient(
			WithApiUrl("https://api.example.com"),
			WithAuthToken("auth_test"),
			WithTransport(tr),
			WithRetryOnConflict(),
		)
		if err == nil {
			t.Fatal("expected error for conflicting options")
		}
		if !strings.Contains(err.Error(), "WithRetryOnConflict") {
			t.Errorf("error should mention WithRetryOnConflict: %v", err)
		}
	})

	t.Run("WithTransport conflicts with WithMaxConcurrentRequests", func(t *testing.T) {
		tr := NewTransport()
		_, err := NewClient(
			WithApiUrl("https://api.example.com"),
			WithAuthToken("auth_test"),
			WithTransport(tr),
			WithMaxConcurrentRequests(5),
		)
		if err == nil {
			t.Fatal("expected error for conflicting options")
		}
		if !strings.Contains(err.Error(), "WithMaxConcurrentRequests") {
			t.Errorf("error should mention WithMaxConcurrentRequests: %v", err)
		}
	})

	t.Run("WithTransport conflicts with WithHTTPClient", func(t *testing.T) {
		tr := NewTransport()
		_, err := NewClient(
			WithApiUrl("https://api.example.com"),
			WithAuthToken("auth_test"),
			WithTransport(tr),
			WithHTTPClient(&http.Client{}),
		)
		if err == nil {
			t.Fatal("expected error for conflicting options")
		}
		if !strings.Contains(err.Error(), "WithHTTPClient") {
			t.Errorf("error should mention WithHTTPClient: %v", err)
		}
	})

	t.Run("WithTransport reports all conflicting options", func(t *testing.T) {
		tr := NewTransport()
		_, err := NewClient(
			WithApiUrl("https://api.example.com"),
			WithAuthToken("auth_test"),
			WithTransport(tr),
			WithMaxRetries(2),
			WithMaxConcurrentRequests(5),
		)
		if err == nil {
			t.Fatal("expected error for conflicting options")
		}
		if !strings.Contains(err.Error(), "WithMaxConcurrentRequests") {
			t.Errorf("error should mention WithMaxConcurrentRequests: %v", err)
		}
		if !strings.Contains(err.Error(), "WithMaxRetries") {
			t.Errorf("error should mention WithMaxRetries: %v", err)
		}
	})

	t.Run("WithTransport works with OTLP-only client", func(t *testing.T) {
		tr := NewTransport()
		c, err := NewClient(
			WithAuthToken("auth_test"),
			WithOtlpEndpoint(OtlpEncodingJson, "https://otlp.example.com"),
			WithTransport(tr),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c == nil {
			t.Fatal("expected client to be created")
		}
	})

	t.Run("WithTransport shares rate-limit budget", func(t *testing.T) {
		tr := NewTransport(WithTransportMaxConcurrentRequests(1))
		c, err := NewClient(
			WithApiUrl("https://api.example.com"),
			WithAuthToken("auth_test"),
			WithTransport(tr),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Both the raw HTTPClient and the typed client use the same transport.
		rawClient := tr.HTTPClient()
		impl := c.(*client)
		innerClient := impl.inner.ClientInterface.(*generatedClient)
		typedClient := innerClient.Client.(*http.Client)

		if rawClient.Transport != beneathAuthTransport(t, typedClient.Transport) {
			t.Error("expected raw and typed clients to share the same RoundTripper")
		}
	})
}
