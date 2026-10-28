package dash0

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
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

func TestImportSignalToMetrics_Integration(t *testing.T) {
	t.Run("puts the rules into the dataset", func(t *testing.T) {
		first := newTestSignalToMetrics()
		first.Metadata.Labels.Dash0Comorigin = Ptr("tf_checkout")
		second := newTestSignalToMetrics()
		second.Metadata.Name = "error-log-count"
		second.Metadata.Labels = &SignalToMetricsLabels{Dash0Comorigin: Ptr("tf_errors")}

		var gotURL *url.URL
		var gotMethod string
		var gotBody SignalToMetricsBulkUpsertRequest
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotURL = r.URL
			gotMethod = r.Method
			if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
				t.Errorf("failed to decode request body: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(SignalToMetricsBulkUpsertResponse{Created: 1, Updated: 1})
		}))
		defer server.Close()

		got, err := newTestClient(t, server.URL).ImportSignalToMetrics(context.Background(), []*SignalToMetricsDefinition{&first, &second}, "production")
		if err != nil {
			t.Fatalf("ImportSignalToMetrics failed: %v", err)
		}

		assertEqual(t, "method", gotMethod, http.MethodPut)
		assertEqual(t, "path", gotURL.Path, "/api/import/signal-to-metrics")
		assertEqual(t, "dataset", gotURL.Query().Get("dataset"), "production")
		if len(gotBody.Items) != 2 {
			t.Fatalf("expected 2 items in the request body, got %d", len(gotBody.Items))
		}
		assertPtrEqual(t, "items[0] origin", gotBody.Items[0].Metadata.Labels.Dash0Comorigin, "tf_checkout")
		assertPtrEqual(t, "items[1] origin", gotBody.Items[1].Metadata.Labels.Dash0Comorigin, "tf_errors")
		if got.Created != 1 || got.Updated != 1 {
			t.Errorf("response = %+v, want {Created:1 Updated:1}", *got)
		}
	})

	t.Run("reports a batch that fails validation", func(t *testing.T) {
		rule := newTestSignalToMetrics()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeAPIError(w, http.StatusBadRequest, "items[0]: metadata.labels.dash0.com/origin is required")
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).ImportSignalToMetrics(context.Background(), []*SignalToMetricsDefinition{&rule}, "production")
		if !IsBadRequest(err) {
			t.Errorf("IsBadRequest(err) = false, want true (err = %v)", err)
		}
		var apiErr *APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("expected *APIError, got %T", err)
		}
		assertEqual(t, "APIError.Message", apiErr.Message, "items[0]: metadata.labels.dash0.com/origin is required")
	})

	t.Run("handles 403", func(t *testing.T) {
		rule := newTestSignalToMetrics()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeAPIError(w, http.StatusForbidden, "insufficient permissions")
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).ImportSignalToMetrics(context.Background(), []*SignalToMetricsDefinition{&rule}, "production")
		if !IsForbidden(err) {
			t.Errorf("IsForbidden(err) = false, want true (err = %v)", err)
		}
	})

	t.Run("rejects an empty dataset", func(t *testing.T) {
		rule := newTestSignalToMetrics()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("no request should reach the server")
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).ImportSignalToMetrics(context.Background(), []*SignalToMetricsDefinition{&rule}, "")
		if err == nil {
			t.Fatal("expected an error for an empty dataset")
		}
	})

	t.Run("rejects a nil rule", func(t *testing.T) {
		rule := newTestSignalToMetrics()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("no request should reach the server")
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).ImportSignalToMetrics(context.Background(), []*SignalToMetricsDefinition{&rule, nil}, "production")
		if err == nil {
			t.Fatal("expected an error for a nil rule")
		}
		if !strings.Contains(err.Error(), "rule 1 is nil") {
			t.Errorf("error = %q, want it to name the nil rule's index", err)
		}
	})

	t.Run("requires the API URL", func(t *testing.T) {
		c, err := NewClient(
			WithOtlpEndpoint(OtlpEncodingJson, "https://ingress.eu-west-1.aws.dash0.com"),
			WithAuthToken("auth_test123"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		rule := newTestSignalToMetrics()
		if _, err := c.ImportSignalToMetrics(context.Background(), []*SignalToMetricsDefinition{&rule}, "production"); !errors.Is(err, ErrAPINotConfigured) {
			t.Errorf("ImportSignalToMetrics error = %v, want ErrAPINotConfigured", err)
		}
	})
}
