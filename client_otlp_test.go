package dash0

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// newTestTraces creates a ptrace.Traces with a single span for testing.
func newTestTraces() ptrace.Traces {
	td := ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr("service.name", "test-service")
	ss := rs.ScopeSpans().AppendEmpty()
	span := ss.Spans().AppendEmpty()
	span.SetName("test-span")
	span.SetTraceID(pcommon.TraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}))
	span.SetSpanID(pcommon.SpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8}))
	return td
}

// newTestMetrics creates a pmetric.Metrics with a single gauge data point.
func newTestMetrics() pmetric.Metrics {
	md := pmetric.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()
	rm.Resource().Attributes().PutStr("service.name", "test-service")
	sm := rm.ScopeMetrics().AppendEmpty()
	m := sm.Metrics().AppendEmpty()
	m.SetName("test.metric")
	dp := m.SetEmptyGauge().DataPoints().AppendEmpty()
	dp.SetDoubleValue(42.0)
	return md
}

// newTestLogs creates a plog.Logs with a single log record.
func newTestLogs() plog.Logs {
	ld := plog.NewLogs()
	rl := ld.ResourceLogs().AppendEmpty()
	rl.Resource().Attributes().PutStr("service.name", "test-service")
	sl := rl.ScopeLogs().AppendEmpty()
	lr := sl.LogRecords().AppendEmpty()
	lr.Body().SetStr("test log message")
	lr.SetSeverityNumber(plog.SeverityNumberInfo)
	return ld
}

// otlpRequest captures an HTTP request received by the mock server.
type otlpRequest struct {
	method      string
	path        string
	contentType string
	authHeader  string
	body        []byte
}

// newMockOTLPServer creates an httptest.Server that records requests and responds with 200.
func newMockOTLPServer(t *testing.T) (*httptest.Server, *[]otlpRequest) {
	t.Helper()
	var mu sync.Mutex
	var requests []otlpRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		mu.Lock()
		requests = append(requests, otlpRequest{
			method:      r.Method,
			path:        r.URL.Path,
			contentType: r.Header.Get("Content-Type"),
			authHeader:  r.Header.Get("Authorization"),
			body:        body,
		})
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}"))
	}))

	return server, &requests
}

// newOTLPClient creates a client with OTLP configured against the given server URL.
func newOTLPClient(t *testing.T, serverURL string) Client {
	t.Helper()
	// We also need a mock API server since WithApiUrl is required
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(apiServer.Close)

	c, err := NewClient(
		WithApiUrl(apiServer.URL),
		WithAuthToken("auth_test123"),
		WithOtlpEndpoint(OtlpEncodingJSON, serverURL),
		WithRetryWaitMin(1*time.Millisecond),
		WithRetryWaitMax(10*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	t.Cleanup(func() { _ = c.Close(context.Background()) })
	return c
}

func TestClient_OTLP(t *testing.T) {
	t.Run("SendTraces succeeds", func(t *testing.T) {
		server, requests := newMockOTLPServer(t)
		defer server.Close()

		c := newOTLPClient(t, server.URL)
		err := c.SendTraces(context.Background(), newTestTraces())
		if err != nil {
			t.Fatalf("SendTraces failed: %v", err)
		}

		found := false
		for _, req := range *requests {
			if req.path == "/v1/traces" {
				found = true
				if req.method != http.MethodPost {
					t.Errorf("expected POST, got %s", req.method)
				}
				if req.contentType != "application/json" {
					t.Errorf("expected Content-Type application/json, got %s", req.contentType)
				}
				if req.authHeader != "Bearer auth_test123" {
					t.Errorf("expected Authorization 'Bearer auth_test123', got %s", req.authHeader)
				}
				// Verify body is valid OTLP JSON by unmarshaling
				unmarshaler := &ptrace.JSONUnmarshaler{}
				td, err := unmarshaler.UnmarshalTraces(req.body)
				if err != nil {
					t.Fatalf("failed to unmarshal traces: %v", err)
				}
				if td.SpanCount() != 1 {
					t.Errorf("expected 1 span, got %d", td.SpanCount())
				}
				spanName := td.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0).Name()
				if spanName != "test-span" {
					t.Errorf("expected span name 'test-span', got %s", spanName)
				}
			}
		}
		if !found {
			t.Error("no request received at /v1/traces")
		}
	})

	t.Run("SendMetrics succeeds", func(t *testing.T) {
		server, requests := newMockOTLPServer(t)
		defer server.Close()

		c := newOTLPClient(t, server.URL)
		err := c.SendMetrics(context.Background(), newTestMetrics())
		if err != nil {
			t.Fatalf("SendMetrics failed: %v", err)
		}

		found := false
		for _, req := range *requests {
			if req.path == "/v1/metrics" {
				found = true
				if req.contentType != "application/json" {
					t.Errorf("expected Content-Type application/json, got %s", req.contentType)
				}
				unmarshaler := &pmetric.JSONUnmarshaler{}
				md, err := unmarshaler.UnmarshalMetrics(req.body)
				if err != nil {
					t.Fatalf("failed to unmarshal metrics: %v", err)
				}
				if md.MetricCount() != 1 {
					t.Errorf("expected 1 metric, got %d", md.MetricCount())
				}
			}
		}
		if !found {
			t.Error("no request received at /v1/metrics")
		}
	})

	t.Run("SendLogs succeeds", func(t *testing.T) {
		server, requests := newMockOTLPServer(t)
		defer server.Close()

		c := newOTLPClient(t, server.URL)
		err := c.SendLogs(context.Background(), newTestLogs())
		if err != nil {
			t.Fatalf("SendLogs failed: %v", err)
		}

		found := false
		for _, req := range *requests {
			if req.path == "/v1/logs" {
				found = true
				if req.contentType != "application/json" {
					t.Errorf("expected Content-Type application/json, got %s", req.contentType)
				}
				unmarshaler := &plog.JSONUnmarshaler{}
				ld, err := unmarshaler.UnmarshalLogs(req.body)
				if err != nil {
					t.Fatalf("failed to unmarshal logs: %v", err)
				}
				if ld.LogRecordCount() != 1 {
					t.Errorf("expected 1 log record, got %d", ld.LogRecordCount())
				}
			}
		}
		if !found {
			t.Error("no request received at /v1/logs")
		}
	})

	t.Run("SendTraces returns ErrOTLPNotConfigured", func(t *testing.T) {
		apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer apiServer.Close()

		c, err := NewClient(
			WithApiUrl(apiServer.URL),
			WithAuthToken("auth_test123"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		err = c.SendTraces(context.Background(), newTestTraces())
		if !errors.Is(err, ErrOTLPNotConfigured) {
			t.Errorf("expected ErrOTLPNotConfigured, got %v", err)
		}
	})

	t.Run("SendMetrics returns ErrOTLPNotConfigured", func(t *testing.T) {
		apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer apiServer.Close()

		c, err := NewClient(
			WithApiUrl(apiServer.URL),
			WithAuthToken("auth_test123"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		err = c.SendMetrics(context.Background(), newTestMetrics())
		if !errors.Is(err, ErrOTLPNotConfigured) {
			t.Errorf("expected ErrOTLPNotConfigured, got %v", err)
		}
	})

	t.Run("SendLogs returns ErrOTLPNotConfigured", func(t *testing.T) {
		apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer apiServer.Close()

		c, err := NewClient(
			WithApiUrl(apiServer.URL),
			WithAuthToken("auth_test123"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		err = c.SendLogs(context.Background(), newTestLogs())
		if !errors.Is(err, ErrOTLPNotConfigured) {
			t.Errorf("expected ErrOTLPNotConfigured, got %v", err)
		}
	})

	t.Run("rejects unsupported encoding", func(t *testing.T) {
		apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer apiServer.Close()

		_, err := NewClient(
			WithApiUrl(apiServer.URL),
			WithAuthToken("auth_test123"),
			WithOtlpEndpoint(OtlpEncoding("grpc"), "https://otlp.example.com"),
		)
		if err == nil {
			t.Fatal("expected error for unsupported encoding")
		}
		if !strings.Contains(err.Error(), "unsupported OTLP encoding") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("rejects endpoint with wrong scheme for otlp/json encoding", func(t *testing.T) {
		apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer apiServer.Close()

		_, err := NewClient(
			WithApiUrl(apiServer.URL),
			WithAuthToken("auth_test123"),
			WithOtlpEndpoint(OtlpEncodingJSON, "grpc://otlp.example.com"),
		)
		if err == nil {
			t.Fatal("expected error for endpoint with wrong scheme")
		}
		if !strings.Contains(err.Error(), "must start with http:// or https://") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("rejects endpoint with signal path suffix", func(t *testing.T) {
		apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer apiServer.Close()

		_, err := NewClient(
			WithApiUrl(apiServer.URL),
			WithAuthToken("auth_test123"),
			WithOtlpEndpoint(OtlpEncodingJSON, "https://otlp.example.com/v1/traces"),
		)
		if err == nil {
			t.Fatal("expected error for endpoint with signal path suffix")
		}
		if !strings.Contains(err.Error(), "must not end with") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("Close is no-op without OTLP", func(t *testing.T) {
		apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer apiServer.Close()

		c, err := NewClient(
			WithApiUrl(apiServer.URL),
			WithAuthToken("auth_test123"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		err = c.Close(context.Background())
		if err != nil {
			t.Fatalf("Close failed: %v", err)
		}
	})

	t.Run("SendTraces returns APIError on HTTP 401", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("x-trace-id", "abc123")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"message":"invalid token"}`))
		}))
		defer server.Close()

		c := newOTLPClient(t, server.URL)
		err := c.SendTraces(context.Background(), newTestTraces())
		if err == nil {
			t.Fatal("expected error")
		}

		apiErr, ok := err.(*APIError)
		if !ok {
			t.Fatalf("expected *APIError, got %T: %v", err, err)
		}
		if apiErr.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", apiErr.StatusCode)
		}
		if apiErr.Message != "invalid token" {
			t.Errorf("expected message 'invalid token', got %q", apiErr.Message)
		}
		if apiErr.TraceID != "abc123" {
			t.Errorf("expected trace ID 'abc123', got %q", apiErr.TraceID)
		}
		if !IsUnauthorized(err) {
			t.Error("expected IsUnauthorized to return true")
		}
	})

	t.Run("SendMetrics returns APIError on HTTP 500", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message":"internal error"}`))
		}))
		defer server.Close()

		c := newOTLPClient(t, server.URL)
		err := c.SendMetrics(context.Background(), newTestMetrics())
		if err == nil {
			t.Fatal("expected error")
		}

		if !IsServerError(err) {
			t.Errorf("expected IsServerError to return true, got error: %v", err)
		}
	})

	t.Run("SendLogs returns APIError on HTTP 429", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"message":"rate limited"}`))
		}))
		defer server.Close()

		c := newOTLPClient(t, server.URL)
		err := c.SendLogs(context.Background(), newTestLogs())
		if err == nil {
			t.Fatal("expected error")
		}

		if !IsRateLimited(err) {
			t.Errorf("expected IsRateLimited to return true, got error: %v", err)
		}
	})

	t.Run("Close succeeds after sending data", func(t *testing.T) {
		server, _ := newMockOTLPServer(t)
		defer server.Close()

		c := newOTLPClient(t, server.URL)
		_ = c.SendTraces(context.Background(), newTestTraces())
		_ = c.SendMetrics(context.Background(), newTestMetrics())
		_ = c.SendLogs(context.Background(), newTestLogs())

		err := c.Close(context.Background())
		if err != nil {
			t.Fatalf("Close failed: %v", err)
		}
	})
}
