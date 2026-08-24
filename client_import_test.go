package dash0

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// conflictBody is the shape the Dash0 API returns when a write loses the
// dataset's optimistic-concurrency check.
const conflictBody = `{"error":{"code":409,"message":"dataset version conflict, please retry"}}`

func TestImportDashboard_RetryOnConflict(t *testing.T) {
	// newImportServer replies with a 409 for the first conflicts requests, then
	// with the imported dashboard.
	newImportServer := func(t *testing.T, conflicts int32) (*httptest.Server, *atomic.Int32) {
		t.Helper()
		var requests atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assertEqual(t, "method", r.Method, http.MethodPost)
			assertEqual(t, "path", r.URL.Path, "/api/import/dashboard")

			var body DashboardDefinition
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("failed to decode request body: %v", err)
			}
			if body.Kind == "" {
				t.Error("request body lost its kind on replay")
			}

			if requests.Add(1) <= conflicts {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(conflictBody))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(body)
		}))
		t.Cleanup(server.Close)
		return server, &requests
	}

	dashboard := &DashboardDefinition{
		Kind: DashboardDefinitionKind("Dash0Dashboard"),
	}

	t.Run("retries until the conflict clears", func(t *testing.T) {
		server, requests := newImportServer(t, 2)

		c, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
			WithRetryOnConflict(),
			WithMaxRetries(3),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		result, err := c.ImportDashboard(context.Background(), dashboard, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected the imported dashboard")
		}
		if got := requests.Load(); got != 3 {
			t.Errorf("requests = %d, want 3", got)
		}
	})

	t.Run("surfaces the conflict when the option is off", func(t *testing.T) {
		server, requests := newImportServer(t, 2)

		c := newTestClient(t, server.URL)

		_, err := c.ImportDashboard(context.Background(), dashboard, nil)
		if err == nil {
			t.Fatal("expected the conflict to surface as an error")
		}
		if !IsConflict(err) {
			t.Errorf("expected a 409 error, got %v", err)
		}
		if got := requests.Load(); got != 1 {
			t.Errorf("requests = %d, want 1", got)
		}
	})

	t.Run("surfaces the conflict once the retry budget is spent", func(t *testing.T) {
		server, requests := newImportServer(t, 10)

		c, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
			WithRetryOnConflict(),
			WithMaxRetries(2),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, err = c.ImportDashboard(context.Background(), dashboard, nil)
		if !IsConflict(err) {
			t.Errorf("expected a 409 error, got %v", err)
		}
		if got := requests.Load(); got != 3 {
			t.Errorf("requests = %d, want 3", got)
		}
	})
}
