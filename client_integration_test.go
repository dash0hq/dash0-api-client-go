package dash0

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_Integration(t *testing.T) {
	t.Run("ListDashboards returns empty list", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request
			if r.URL.Path != "/api/dashboards" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.Method != http.MethodGet {
				t.Errorf("unexpected method: %s", r.Method)
			}

			// Verify auth header
			auth := r.Header.Get("Authorization")
			if auth != "Bearer auth_test123" {
				t.Errorf("unexpected Authorization header: %s", auth)
			}

			// Verify User-Agent
			ua := r.Header.Get("User-Agent")
			if ua != DefaultUserAgent {
				t.Errorf("unexpected User-Agent: %s", ua)
			}

			// Return empty list
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]DashboardApiListItem{})
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		dashboards, err := client.ListDashboards(context.Background(), nil)
		if err != nil {
			t.Fatalf("ListDashboards failed: %v", err)
		}

		if len(dashboards) != 0 {
			t.Errorf("expected empty list, got %d dashboards", len(dashboards))
		}
	})

	t.Run("handles 401 Unauthorized", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("x-trace-id", "trace-abc123")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"message": "Invalid auth token",
			})
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_invalid"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, err = client.ListDashboards(context.Background(), nil)
		if err == nil {
			t.Fatal("expected error for 401 response")
		}

		if !IsUnauthorized(err) {
			t.Errorf("expected IsUnauthorized to return true, got false")
		}

		apiErr, ok := err.(*APIError)
		if !ok {
			t.Fatalf("expected *APIError, got %T", err)
		}
		if apiErr.TraceID != "trace-abc123" {
			t.Errorf("expected trace ID 'trace-abc123', got '%s'", apiErr.TraceID)
		}
		if apiErr.Message != "Invalid auth token" {
			t.Errorf("expected message 'Invalid auth token', got '%s'", apiErr.Message)
		}
	})

	t.Run("handles 404 Not Found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"message": "Dashboard not found",
			})
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, err = client.GetDashboard(context.Background(), "nonexistent", nil)
		if err == nil {
			t.Fatal("expected error for 404 response")
		}

		if !IsNotFound(err) {
			t.Errorf("expected IsNotFound to return true")
		}
	})

	t.Run("handles 500 Internal Server Error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("x-trace-id", "trace-error-500")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Internal Server Error"))
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, err = client.ListDashboards(context.Background(), nil)
		if err == nil {
			t.Fatal("expected error for 500 response")
		}

		if !IsServerError(err) {
			t.Errorf("expected IsServerError to return true")
		}

		apiErr, ok := err.(*APIError)
		if !ok {
			t.Fatalf("expected *APIError, got %T", err)
		}
		if apiErr.TraceID != "trace-error-500" {
			t.Errorf("expected trace ID 'trace-error-500', got '%s'", apiErr.TraceID)
		}
		if apiErr.Body != "Internal Server Error" {
			t.Errorf("expected body 'Internal Server Error', got '%s'", apiErr.Body)
		}
	})

	t.Run("handles 429 Rate Limited", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"message": "Rate limit exceeded",
			})
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, err = client.ListDashboards(context.Background(), nil)
		if err == nil {
			t.Fatal("expected error for 429 response")
		}

		if !IsRateLimited(err) {
			t.Errorf("expected IsRateLimited to return true")
		}
	})

	t.Run("sends dataset query parameter", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			dataset := r.URL.Query().Get("dataset")
			if dataset != "my-dataset" {
				t.Errorf("expected dataset 'my-dataset', got '%s'", dataset)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]DashboardApiListItem{})
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		dataset := "my-dataset"
		_, err = client.ListDashboards(context.Background(), &dataset)
		if err != nil {
			t.Fatalf("ListDashboards failed: %v", err)
		}
	})

	t.Run("custom User-Agent is sent", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ua := r.Header.Get("User-Agent")
			if ua != "custom-agent/1.0" {
				t.Errorf("expected User-Agent 'custom-agent/1.0', got '%s'", ua)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]DashboardApiListItem{})
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
			WithUserAgent("custom-agent/1.0"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, err = client.ListDashboards(context.Background(), nil)
		if err != nil {
			t.Fatalf("ListDashboards failed: %v", err)
		}
	})

	t.Run("retries on 500 and succeeds", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts < 3 {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("server error"))
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]DashboardApiListItem{})
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
			WithMaxRetries(3),
			WithRetryWaitMin(1*time.Millisecond),
			WithRetryWaitMax(10*time.Millisecond),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, err = client.ListDashboards(context.Background(), nil)
		if err != nil {
			t.Fatalf("expected request to succeed after retries, got: %v", err)
		}

		if attempts != 3 {
			t.Errorf("expected 3 attempts, got %d", attempts)
		}
	})

	t.Run("retries on 429 and succeeds", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts < 2 {
				w.Header().Set("Retry-After", "0")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte("rate limited"))
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]DashboardApiListItem{})
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
			WithMaxRetries(3),
			WithRetryWaitMin(1*time.Millisecond),
			WithRetryWaitMax(10*time.Millisecond),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, err = client.ListDashboards(context.Background(), nil)
		if err != nil {
			t.Fatalf("expected request to succeed after retry, got: %v", err)
		}

		if attempts != 2 {
			t.Errorf("expected 2 attempts, got %d", attempts)
		}
	})

	t.Run("fails after max retries exceeded", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("server error"))
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
			WithMaxRetries(2),
			WithRetryWaitMin(1*time.Millisecond),
			WithRetryWaitMax(10*time.Millisecond),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, err = client.ListDashboards(context.Background(), nil)
		if err == nil {
			t.Fatal("expected error after max retries")
		}

		if !IsServerError(err) {
			t.Errorf("expected server error, got: %v", err)
		}

		// 1 initial + 2 retries = 3 attempts
		if attempts != 3 {
			t.Errorf("expected 3 attempts (1 initial + 2 retries), got %d", attempts)
		}
	})

	t.Run("does not retry non-idempotent POST requests", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("server error"))
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
			WithMaxRetries(3),
			WithRetryWaitMin(1*time.Millisecond),
			WithRetryWaitMax(10*time.Millisecond),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		// CreateDashboard is a POST request, should not be retried
		_, err = client.CreateDashboard(context.Background(), &DashboardDefinition{}, nil)
		if err == nil {
			t.Fatal("expected error")
		}

		if attempts != 1 {
			t.Errorf("expected 1 attempt (no retries for POST), got %d", attempts)
		}
	})

	t.Run("retries POST requests marked as idempotent", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts < 2 {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("server error"))
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(GetSpansResponse{})
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
			WithMaxRetries(3),
			WithRetryWaitMin(1*time.Millisecond),
			WithRetryWaitMax(10*time.Millisecond),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		// Use withIdempotent to allow retrying this POST request
		ctx := withIdempotent(context.Background())
		_, err = client.GetSpans(ctx, &GetSpansRequest{})
		if err != nil {
			t.Fatalf("expected request to succeed after retry, got: %v", err)
		}

		if attempts != 2 {
			t.Errorf("expected 2 attempts, got %d", attempts)
		}
	})

	t.Run("does not retry on 4xx errors", func(t *testing.T) {
		attempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("bad request"))
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
			WithMaxRetries(3),
			WithRetryWaitMin(1*time.Millisecond),
			WithRetryWaitMax(10*time.Millisecond),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, err = client.ListDashboards(context.Background(), nil)
		if err == nil {
			t.Fatal("expected error")
		}

		if attempts != 1 {
			t.Errorf("expected 1 attempt (no retries for 4xx), got %d", attempts)
		}
	})
}
