package dash0

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func newTestSignalToMetrics() SignalToMetricsDefinition {
	return SignalToMetricsDefinition{
		Kind: Dash0SignalToMetrics,
		Metadata: SignalToMetricsMetadata{
			Name: "checkout-span-count",
			Labels: &SignalToMetricsLabels{
				Dash0Comid:      Ptr("s2m-123"),
				Dash0Comdataset: Ptr("default"),
			},
		},
		Spec: SignalToMetricsSpec{
			Enabled: true,
			Display: SignalToMetricsDisplay{Name: "Checkout span count"},
			Match: SignalToMetricsMatch{
				Signal:  SignalToMetricsSignalTypeSpans,
				Filters: FilterCriteria{},
			},
			Output: SignalToMetricsOutput{
				Name:     "checkout_spans_total",
				Interval: "60s",
			},
		},
	}
}

func TestSignalToMetrics_Integration(t *testing.T) {
	t.Run("ListSignalToMetrics returns rules", func(t *testing.T) {
		rule := newTestSignalToMetrics()
		second := newTestSignalToMetrics()
		second.Metadata.Name = "error-log-count"

		var gotURL *url.URL
		var gotMethod string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotURL = r.URL
			gotMethod = r.Method
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(GetSignalToMetricsResponse{
				SignalToMetrics: []SignalToMetricsDefinition{rule, second},
			})
		}))
		defer server.Close()

		got, err := newTestClient(t, server.URL).ListSignalToMetrics(context.Background(), nil)
		if err != nil {
			t.Fatalf("ListSignalToMetrics failed: %v", err)
		}

		assertEqual(t, "method", gotMethod, http.MethodGet)
		assertEqual(t, "path", gotURL.Path, "/api/signal-to-metrics")
		if _, ok := gotURL.Query()["dataset"]; ok {
			t.Error("dataset query parameter should be absent when dataset is nil")
		}
		if len(got) != 2 {
			t.Fatalf("expected 2 rules, got %d", len(got))
		}
		assertEqual(t, "rules[0].Metadata.Name", got[0].Metadata.Name, "checkout-span-count")
		assertEqual(t, "rules[1].Metadata.Name", got[1].Metadata.Name, "error-log-count")
	})

	t.Run("ListSignalToMetrics sends the dataset query parameter", func(t *testing.T) {
		var gotURL *url.URL
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotURL = r.URL
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(GetSignalToMetricsResponse{})
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).ListSignalToMetrics(context.Background(), Ptr("production"))
		if err != nil {
			t.Fatalf("ListSignalToMetrics failed: %v", err)
		}
		assertEqual(t, "dataset", gotURL.Query().Get("dataset"), "production")
	})

	t.Run("ListSignalToMetrics sends the originPrefix query parameter", func(t *testing.T) {
		var gotURL *url.URL
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotURL = r.URL
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(GetSignalToMetricsResponse{})
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).ListSignalToMetrics(context.Background(), Ptr("production"), WithOriginPrefix("dash0-operator_abc_"))
		if err != nil {
			t.Fatalf("ListSignalToMetrics failed: %v", err)
		}
		assertEqual(t, "originPrefix", gotURL.Query().Get("originPrefix"), "dash0-operator_abc_")
		assertEqual(t, "dataset", gotURL.Query().Get("dataset"), "production")
	})

	t.Run("ListSignalToMetrics omits originPrefix when unset", func(t *testing.T) {
		var gotURL *url.URL
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotURL = r.URL
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(GetSignalToMetricsResponse{})
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).ListSignalToMetrics(context.Background(), nil, WithOriginPrefix(""))
		if err != nil {
			t.Fatalf("ListSignalToMetrics failed: %v", err)
		}
		if _, ok := gotURL.Query()["originPrefix"]; ok {
			t.Error("originPrefix query parameter should be absent for an empty prefix")
		}
		if _, ok := gotURL.Query()["offset"]; ok {
			t.Error("offset query parameter should be absent on the first page")
		}
	})

	t.Run("ListSignalToMetrics follows pages until hasMore is false", func(t *testing.T) {
		first := newTestSignalToMetrics()
		second := newTestSignalToMetrics()
		second.Metadata.Name = "error-log-count"
		var gotOffsets []string
		var gotPrefixes []string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotOffsets = append(gotOffsets, r.URL.Query().Get("offset"))
			gotPrefixes = append(gotPrefixes, r.URL.Query().Get("originPrefix"))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			resp := GetSignalToMetricsResponse{SignalToMetrics: []SignalToMetricsDefinition{first}, HasMore: Ptr(true)}
			if r.URL.Query().Get("offset") == "1" {
				resp = GetSignalToMetricsResponse{SignalToMetrics: []SignalToMetricsDefinition{second}, HasMore: Ptr(false)}
			}
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		got, err := newTestClient(t, server.URL).ListSignalToMetrics(context.Background(), nil, WithOriginPrefix("tf_"))
		if err != nil {
			t.Fatalf("ListSignalToMetrics failed: %v", err)
		}
		if len(gotOffsets) != 2 {
			t.Fatalf("expected 2 requests, got %d", len(gotOffsets))
		}
		assertEqual(t, "offsets[0]", gotOffsets[0], "")
		assertEqual(t, "offsets[1]", gotOffsets[1], "1")
		assertEqual(t, "prefixes[1]", gotPrefixes[1], "tf_")
		if len(got) != 2 {
			t.Fatalf("expected 2 rules, got %d", len(got))
		}
		assertEqual(t, "rules[0].Metadata.Name", got[0].Metadata.Name, "checkout-span-count")
		assertEqual(t, "rules[1].Metadata.Name", got[1].Metadata.Name, "error-log-count")
	})

	t.Run("ListSignalToMetrics fails on an empty page that claims hasMore", func(t *testing.T) {
		requests := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(GetSignalToMetricsResponse{SignalToMetrics: []SignalToMetricsDefinition{}, HasMore: Ptr(true)})
		}))
		defer server.Close()

		got, err := newTestClient(t, server.URL).ListSignalToMetrics(context.Background(), nil)
		if err == nil {
			t.Fatal("expected an error for an empty page that claims hasMore")
		}
		if requests != 1 {
			t.Errorf("expected 1 request, got %d", requests)
		}
		if got != nil {
			t.Errorf("expected nil result, got %d rules", len(got))
		}
	})

	t.Run("ListSignalToMetrics parses a wire-minimum rule", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"signalToMetrics":[{"kind":"Dash0SignalToMetrics","metadata":{"name":"minimal"},"spec":{"display":{"name":""},"enabled":false,"match":{"signal":"logs","filters":[]},"output":{"name":"m","interval":"1m"}}}]}`))
		}))
		defer server.Close()

		got, err := newTestClient(t, server.URL).ListSignalToMetrics(context.Background(), nil)
		if err != nil {
			t.Fatalf("ListSignalToMetrics failed: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 rule, got %d", len(got))
		}
		assertEqual(t, "GetSignalToMetricsName", GetSignalToMetricsName(got[0]), "minimal")
		assertEqual(t, "GetSignalToMetricsID", GetSignalToMetricsID(got[0]), "")
		assertEqual(t, "GetSignalToMetricsDataset", GetSignalToMetricsDataset(got[0]), "")
		StripSignalToMetricsServerFields(got[0])
		ClearSignalToMetricsID(got[0])
	})

	t.Run("ListSignalToMetrics handles a null rule array", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"signalToMetrics": null}`))
		}))
		defer server.Close()

		got, err := newTestClient(t, server.URL).ListSignalToMetrics(context.Background(), nil)
		if err != nil {
			t.Fatalf("ListSignalToMetrics failed: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("expected an empty slice, got %d entries", len(got))
		}
	})

	t.Run("ListSignalToMetrics reports an unparsed 200 body", func(t *testing.T) {
		// No JSON content type, so the generated parser leaves JSON200 nil and the
		// client's own guard is what reports the problem.
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).ListSignalToMetrics(context.Background(), nil)
		if err == nil {
			t.Fatal("expected an error for a 200 with no parsable body")
		}
		assertEqual(t, "error", err.Error(), "dash0: unexpected nil response")
	})

	t.Run("ListSignalToMetrics wraps a malformed JSON body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{not json`))
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).ListSignalToMetrics(context.Background(), nil)
		if err == nil {
			t.Fatal("expected an error for a malformed JSON body")
		}
		if got := err.Error(); !strings.Contains(got, "dash0: list signal-to-metrics failed") {
			t.Errorf("error = %q, want it to wrap %q", got, "dash0: list signal-to-metrics failed")
		}
	})

	t.Run("GetSignalToMetrics returns the rule", func(t *testing.T) {
		rule := newTestSignalToMetrics()
		var gotURL *url.URL
		var gotMethod string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotURL = r.URL
			gotMethod = r.Method
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(rule)
		}))
		defer server.Close()

		got, err := newTestClient(t, server.URL).GetSignalToMetrics(context.Background(), "s2m-123", nil)
		if err != nil {
			t.Fatalf("GetSignalToMetrics failed: %v", err)
		}

		assertEqual(t, "method", gotMethod, http.MethodGet)
		assertEqual(t, "path", gotURL.Path, "/api/signal-to-metrics/s2m-123")
		assertEqual(t, "Metadata.Name", got.Metadata.Name, "checkout-span-count")
	})

	t.Run("GetSignalToMetrics handles 404", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeAPIError(w, http.StatusNotFound, "Signal-to-metrics rule not found")
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).GetSignalToMetrics(context.Background(), "nope", nil)
		if err == nil {
			t.Fatal("expected an error for a 404 response")
		}
		if !IsNotFound(err) {
			t.Errorf("IsNotFound(err) = false, want true (err = %v)", err)
		}
	})

	t.Run("GetSignalToMetrics sends the dataset query parameter", func(t *testing.T) {
		rule := newTestSignalToMetrics()
		var gotURL *url.URL
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotURL = r.URL
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(rule)
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).GetSignalToMetrics(context.Background(), "s2m-123", Ptr("production"))
		if err != nil {
			t.Fatalf("GetSignalToMetrics failed: %v", err)
		}
		assertEqual(t, "dataset", gotURL.Query().Get("dataset"), "production")
	})

	t.Run("CreateSignalToMetrics posts the definition", func(t *testing.T) {
		rule := newTestSignalToMetrics()
		var gotURL *url.URL
		var gotMethod string
		var gotBody SignalToMetricsDefinition
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotURL = r.URL
			gotMethod = r.Method
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(rule)
		}))
		defer server.Close()

		got, err := newTestClient(t, server.URL).CreateSignalToMetrics(context.Background(), &rule, nil)
		if err != nil {
			t.Fatalf("CreateSignalToMetrics failed: %v", err)
		}

		assertEqual(t, "method", gotMethod, http.MethodPost)
		assertEqual(t, "path", gotURL.Path, "/api/signal-to-metrics")
		assertEqual(t, "request body Metadata.Name", gotBody.Metadata.Name, "checkout-span-count")
		assertEqual(t, "response Metadata.Name", got.Metadata.Name, "checkout-span-count")
	})

	t.Run("CreateSignalToMetrics handles 201", func(t *testing.T) {
		rule := newTestSignalToMetrics()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(rule)
		}))
		defer server.Close()

		got, err := newTestClient(t, server.URL).CreateSignalToMetrics(context.Background(), &rule, nil)
		if err != nil {
			t.Fatalf("CreateSignalToMetrics failed: %v", err)
		}
		assertEqual(t, "Metadata.Name", got.Metadata.Name, "checkout-span-count")
		assertEqual(t, "ID", GetSignalToMetricsID(got), "s2m-123")
	})

	t.Run("CreateSignalToMetrics handles 400", func(t *testing.T) {
		rule := newTestSignalToMetrics()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeAPIError(w, http.StatusBadRequest, "spec.sample.interval is required")
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).CreateSignalToMetrics(context.Background(), &rule, nil)
		if err == nil {
			t.Fatal("expected an error for a 400 response")
		}
		if !IsBadRequest(err) {
			t.Errorf("IsBadRequest(err) = false, want true (err = %v)", err)
		}
		apiErr, ok := err.(*APIError)
		if !ok {
			t.Fatalf("expected *APIError, got %T", err)
		}
		assertEqual(t, "APIError.Message", apiErr.Message, "spec.sample.interval is required")
	})

	t.Run("CreateSignalToMetrics handles 403", func(t *testing.T) {
		rule := newTestSignalToMetrics()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeAPIError(w, http.StatusForbidden, "insufficient permissions")
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).CreateSignalToMetrics(context.Background(), &rule, nil)
		if err == nil {
			t.Fatal("expected an error for a 403 response")
		}
		if !IsForbidden(err) {
			t.Errorf("IsForbidden(err) = false, want true (err = %v)", err)
		}
	})

	t.Run("CreateSignalToMetrics sends the dataset query parameter", func(t *testing.T) {
		rule := newTestSignalToMetrics()
		var gotURL *url.URL
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotURL = r.URL
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(rule)
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).CreateSignalToMetrics(context.Background(), &rule, Ptr("production"))
		if err != nil {
			t.Fatalf("CreateSignalToMetrics failed: %v", err)
		}
		assertEqual(t, "dataset", gotURL.Query().Get("dataset"), "production")
	})

	t.Run("CreateSignalToMetrics wraps a 201 body it cannot parse", func(t *testing.T) {
		rule := newTestSignalToMetrics()
		// On a 201 the generated parser decodes into ErrorResponse, which ignores
		// unknown fields, so JSON200 stays nil and the manual fallback runs. A
		// numeric "kind" parses as an ErrorResponse but not as a definition, which
		// is what reaches the fallback's error branch.
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"kind": 123}`))
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).CreateSignalToMetrics(context.Background(), &rule, nil)
		if err == nil {
			t.Fatal("expected an error for a 201 body that is not a definition")
		}
		if !strings.Contains(err.Error(), "dash0: failed to parse signal-to-metrics response") {
			t.Errorf("error = %q, want it to wrap the parse failure", err.Error())
		}
	})

	t.Run("CreateSignalToMetrics wraps an empty 201 body", func(t *testing.T) {
		rule := newTestSignalToMetrics()
		// No JSON content type and no body, so no parser case matches and the
		// fallback has nothing to decode.
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).CreateSignalToMetrics(context.Background(), &rule, nil)
		if err == nil {
			t.Fatal("expected an error for a 201 with an empty body")
		}
		if !strings.Contains(err.Error(), "dash0: failed to parse signal-to-metrics response") {
			t.Errorf("error = %q, want it to wrap the parse failure", err.Error())
		}
	})

	t.Run("CreateSignalToMetrics rejects a nil rule", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("the request should not reach the server")
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).CreateSignalToMetrics(context.Background(), nil, nil)
		if err == nil {
			t.Fatal("expected an error for a nil rule")
		}
		assertEqual(t, "error", err.Error(), "dash0: create signal-to-metrics requires a non-nil rule")
	})

	t.Run("UpdateSignalToMetrics rejects a nil rule", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("the request should not reach the server")
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).UpdateSignalToMetrics(context.Background(), "s2m-123", nil, nil)
		if err == nil {
			t.Fatal("expected an error for a nil rule")
		}
		assertEqual(t, "error", err.Error(), "dash0: update signal-to-metrics requires a non-nil rule")
	})

	t.Run("UpdateSignalToMetrics puts the definition", func(t *testing.T) {
		rule := newTestSignalToMetrics()
		var gotURL *url.URL
		var gotMethod string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotURL = r.URL
			gotMethod = r.Method
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(rule)
		}))
		defer server.Close()

		got, err := newTestClient(t, server.URL).UpdateSignalToMetrics(context.Background(), "s2m-123", &rule, nil)
		if err != nil {
			t.Fatalf("UpdateSignalToMetrics failed: %v", err)
		}

		assertEqual(t, "method", gotMethod, http.MethodPut)
		assertEqual(t, "path", gotURL.Path, "/api/signal-to-metrics/s2m-123")
		assertEqual(t, "Metadata.Name", got.Metadata.Name, "checkout-span-count")
	})

	t.Run("UpdateSignalToMetrics handles 404", func(t *testing.T) {
		rule := newTestSignalToMetrics()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeAPIError(w, http.StatusNotFound, "Signal-to-metrics rule not found")
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).UpdateSignalToMetrics(context.Background(), "nope", &rule, nil)
		if err == nil {
			t.Fatal("expected an error for a 404 response")
		}
		if !IsNotFound(err) {
			t.Errorf("IsNotFound(err) = false, want true (err = %v)", err)
		}
	})

	t.Run("UpdateSignalToMetrics sends the dataset query parameter", func(t *testing.T) {
		rule := newTestSignalToMetrics()
		var gotURL *url.URL
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotURL = r.URL
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(rule)
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).UpdateSignalToMetrics(context.Background(), "s2m-123", &rule, Ptr("production"))
		if err != nil {
			t.Fatalf("UpdateSignalToMetrics failed: %v", err)
		}
		assertEqual(t, "dataset", gotURL.Query().Get("dataset"), "production")
	})

	t.Run("DeleteSignalToMetrics handles 200", func(t *testing.T) {
		var gotURL *url.URL
		var gotMethod string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotURL = r.URL
			gotMethod = r.Method
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		if err := newTestClient(t, server.URL).DeleteSignalToMetrics(context.Background(), "s2m-123", nil); err != nil {
			t.Fatalf("DeleteSignalToMetrics failed: %v", err)
		}
		assertEqual(t, "method", gotMethod, http.MethodDelete)
		assertEqual(t, "path", gotURL.Path, "/api/signal-to-metrics/s2m-123")
	})

	t.Run("DeleteSignalToMetrics handles 204", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		if err := newTestClient(t, server.URL).DeleteSignalToMetrics(context.Background(), "s2m-123", nil); err != nil {
			t.Fatalf("DeleteSignalToMetrics failed: %v", err)
		}
	})

	t.Run("DeleteSignalToMetrics handles 403", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeAPIError(w, http.StatusForbidden, "insufficient permissions")
		}))
		defer server.Close()

		err := newTestClient(t, server.URL).DeleteSignalToMetrics(context.Background(), "s2m-123", nil)
		if err == nil {
			t.Fatal("expected an error for a 403 response")
		}
		if !IsForbidden(err) {
			t.Errorf("IsForbidden(err) = false, want true (err = %v)", err)
		}
	})

	t.Run("DeleteSignalToMetrics sends the dataset query parameter", func(t *testing.T) {
		var gotURL *url.URL
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotURL = r.URL
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		if err := newTestClient(t, server.URL).DeleteSignalToMetrics(context.Background(), "s2m-123", Ptr("production")); err != nil {
			t.Fatalf("DeleteSignalToMetrics failed: %v", err)
		}
		assertEqual(t, "dataset", gotURL.Query().Get("dataset"), "production")
	})

	t.Run("ListSignalToMetricsIter passes the originPrefix option", func(t *testing.T) {
		var gotQuery url.Values
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotQuery = r.URL.Query()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(GetSignalToMetricsResponse{
				SignalToMetrics: []SignalToMetricsDefinition{newTestSignalToMetrics()},
			})
		}))
		defer server.Close()

		iter := newTestClient(t, server.URL).ListSignalToMetricsIter(context.Background(), Ptr("production"), WithOriginPrefix("dash0-operator_abc_"))
		count := 0
		for iter.Next() {
			count++
		}
		if err := iter.Err(); err != nil {
			t.Fatalf("iterator error: %v", err)
		}
		if count != 1 {
			t.Fatalf("expected 1 rule, got %d", count)
		}
		assertEqual(t, "originPrefix", gotQuery.Get("originPrefix"), "dash0-operator_abc_")
		assertEqual(t, "dataset", gotQuery.Get("dataset"), "production")
	})

	t.Run("ListSignalToMetricsIter iterates every rule", func(t *testing.T) {
		rule := newTestSignalToMetrics()
		second := newTestSignalToMetrics()
		second.Metadata.Name = "error-log-count"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(GetSignalToMetricsResponse{
				SignalToMetrics: []SignalToMetricsDefinition{rule, second},
			})
		}))
		defer server.Close()

		iter := newTestClient(t, server.URL).ListSignalToMetricsIter(context.Background(), nil)
		var names []string
		for iter.Next() {
			names = append(names, iter.Current().Metadata.Name)
		}
		if err := iter.Err(); err != nil {
			t.Fatalf("iterator error: %v", err)
		}
		if len(names) != 2 {
			t.Fatalf("expected 2 rules, got %d", len(names))
		}
		assertEqual(t, "names[0]", names[0], "checkout-span-count")
		assertEqual(t, "names[1]", names[1], "error-log-count")
	})

	t.Run("ListSignalToMetricsIter surfaces the list error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeAPIError(w, http.StatusInternalServerError, "boom")
		}))
		defer server.Close()

		iter := newTestClient(t, server.URL).ListSignalToMetricsIter(context.Background(), nil)
		if iter.Next() {
			t.Error("Next() = true, want false on a failed list")
		}
		if iter.Err() == nil {
			t.Fatal("expected the iterator to surface the list error")
		}
		if !IsServerError(iter.Err()) {
			t.Errorf("IsServerError(err) = false, want true (err = %v)", iter.Err())
		}
	})
}

func TestSignalToMetrics_APINotConfigured(t *testing.T) {
	c, err := NewClient(
		WithOtlpEndpoint(OtlpEncodingJson, "https://ingress.eu-west-1.aws.dash0.com"),
		WithAuthToken("auth_test123"),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	rule := newTestSignalToMetrics()
	ctx := context.Background()

	if _, err := c.ListSignalToMetrics(ctx, nil); !errors.Is(err, ErrAPINotConfigured) {
		t.Errorf("ListSignalToMetrics error = %v, want ErrAPINotConfigured", err)
	}
	if _, err := c.GetSignalToMetrics(ctx, "s2m-123", nil); !errors.Is(err, ErrAPINotConfigured) {
		t.Errorf("GetSignalToMetrics error = %v, want ErrAPINotConfigured", err)
	}
	if _, err := c.CreateSignalToMetrics(ctx, &rule, nil); !errors.Is(err, ErrAPINotConfigured) {
		t.Errorf("CreateSignalToMetrics error = %v, want ErrAPINotConfigured", err)
	}
	if _, err := c.UpdateSignalToMetrics(ctx, "s2m-123", &rule, nil); !errors.Is(err, ErrAPINotConfigured) {
		t.Errorf("UpdateSignalToMetrics error = %v, want ErrAPINotConfigured", err)
	}
	if err := c.DeleteSignalToMetrics(ctx, "s2m-123", nil); !errors.Is(err, ErrAPINotConfigured) {
		t.Errorf("DeleteSignalToMetrics error = %v, want ErrAPINotConfigured", err)
	}
	iter := c.ListSignalToMetricsIter(ctx, nil)
	if !errors.Is(iter.Err(), ErrAPINotConfigured) {
		t.Errorf("ListSignalToMetricsIter error = %v, want ErrAPINotConfigured", iter.Err())
	}
}

func TestStripSignalToMetricsServerFields(t *testing.T) {
	createdAt := time.Now()
	updatedAt := time.Now()
	deletedAt := time.Now()
	source := CrdSource("terraform")

	rule := &SignalToMetricsDefinition{
		Metadata: SignalToMetricsMetadata{
			Name: "keep-this-name",
			Annotations: &SignalToMetricsGroupAnnotations{
				Dash0ComfolderPath: Ptr("/infra"),
				Dash0ComcreatedAt:  &createdAt,
				Dash0ComupdatedAt:  &updatedAt,
				Dash0ComdeletedAt:  &deletedAt,
			},
			Labels: &SignalToMetricsLabels{
				Dash0Comid:      Ptr("keep-this-id"),
				Dash0Comversion: Ptr("2"),
				Dash0Comsource:  &source,
				Dash0Comdataset: Ptr("ds"),
				Dash0Comorigin:  Ptr("my-origin"),
			},
		},
		Spec: SignalToMetricsSpec{Enabled: true},
	}

	StripSignalToMetricsServerFields(rule)

	if rule.Metadata.Annotations.Dash0ComcreatedAt != nil {
		t.Error("Dash0ComcreatedAt should be nil")
	}
	if rule.Metadata.Annotations.Dash0ComupdatedAt != nil {
		t.Error("Dash0ComupdatedAt should be nil")
	}
	if rule.Metadata.Annotations.Dash0ComdeletedAt != nil {
		t.Error("Dash0ComdeletedAt should be nil")
	}
	if rule.Metadata.Labels.Dash0Comversion != nil {
		t.Error("Dash0Comversion should be nil")
	}
	if rule.Metadata.Labels.Dash0Comsource != nil {
		t.Error("Dash0Comsource should be nil")
	}
	if rule.Metadata.Labels.Dash0Comdataset != nil {
		t.Error("Dash0Comdataset should be nil")
	}
	if rule.Metadata.Labels.Dash0Comorigin != nil {
		t.Error("Dash0Comorigin should be nil")
	}

	// The ID survives, so a Get -> Strip -> Update round-trip keeps addressing the
	// same asset. Callers who want it gone call ClearSignalToMetricsID.
	if got := GetSignalToMetricsID(rule); got != "keep-this-id" {
		t.Errorf("Dash0Comid = %q, want %q (the ID must be preserved)", got, "keep-this-id")
	}
	assertEqual(t, "Metadata.Name", rule.Metadata.Name, "keep-this-name")
	assertEqual(t, "Annotations.Dash0ComfolderPath", StringValue(rule.Metadata.Annotations.Dash0ComfolderPath), "/infra")
	if !rule.Spec.Enabled {
		t.Error("Spec should be preserved")
	}
}

func TestStripSignalToMetricsServerFields_ZeroValue(t *testing.T) {
	rule := &SignalToMetricsDefinition{}
	StripSignalToMetricsServerFields(rule) // should not panic
	if rule.Metadata.Labels != nil {
		t.Error("Labels should remain nil")
	}
	if rule.Metadata.Annotations != nil {
		t.Error("Annotations should remain nil")
	}
}

func TestStripSignalToMetricsServerFields_Nil(t *testing.T) {
	StripSignalToMetricsServerFields(nil) // should not panic
}

func TestGetSignalToMetricsName(t *testing.T) {
	tests := []struct {
		name string
		rule *SignalToMetricsDefinition
		want string
	}{
		{
			"from display name",
			&SignalToMetricsDefinition{
				Spec: SignalToMetricsSpec{
					Display: SignalToMetricsDisplay{Name: "Display Name"},
				},
			},
			"Display Name",
		},
		{
			"falls back to metadata name on a zero-value display",
			&SignalToMetricsDefinition{
				Metadata: SignalToMetricsMetadata{Name: "meta-name"},
			},
			"meta-name",
		},
		{
			"falls back to metadata name when display name is empty",
			&SignalToMetricsDefinition{
				Metadata: SignalToMetricsMetadata{Name: "meta-name"},
				Spec: SignalToMetricsSpec{
					Display: SignalToMetricsDisplay{Name: ""},
				},
			},
			"meta-name",
		},
		{"nil rule", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetSignalToMetricsName(tt.rule); got != tt.want {
				t.Errorf("GetSignalToMetricsName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetSignalToMetricsID(t *testing.T) {
	tests := []struct {
		name string
		rule *SignalToMetricsDefinition
		want string
	}{
		{
			"with ID",
			&SignalToMetricsDefinition{
				Metadata: SignalToMetricsMetadata{
					Labels: &SignalToMetricsLabels{Dash0Comid: Ptr("s2m-123")},
				},
			},
			"s2m-123",
		},
		{"nil rule", nil, ""},
		{"nil labels", &SignalToMetricsDefinition{}, ""},
		{
			"nil ID",
			&SignalToMetricsDefinition{
				Metadata: SignalToMetricsMetadata{Labels: &SignalToMetricsLabels{}},
			},
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetSignalToMetricsID(tt.rule); got != tt.want {
				t.Errorf("GetSignalToMetricsID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetSignalToMetricsDataset(t *testing.T) {
	tests := []struct {
		name string
		rule *SignalToMetricsDefinition
		want string
	}{
		{
			"with dataset",
			&SignalToMetricsDefinition{
				Metadata: SignalToMetricsMetadata{
					Labels: &SignalToMetricsLabels{Dash0Comdataset: Ptr("production")},
				},
			},
			"production",
		},
		{"nil rule", nil, ""},
		{"nil labels", &SignalToMetricsDefinition{}, ""},
		{
			"nil dataset",
			&SignalToMetricsDefinition{
				Metadata: SignalToMetricsMetadata{Labels: &SignalToMetricsLabels{}},
			},
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetSignalToMetricsDataset(tt.rule); got != tt.want {
				t.Errorf("GetSignalToMetricsDataset() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetSignalToMetricsDataset(t *testing.T) {
	rule := &SignalToMetricsDefinition{}
	SetSignalToMetricsDataset(rule, "production")
	if rule.Metadata.Labels == nil {
		t.Fatal("expected the labels struct to be initialized")
	}
	assertPtrEqual(t, "Dash0Comdataset", rule.Metadata.Labels.Dash0Comdataset, "production")
}

func TestSetSignalToMetricsDataset_Nil(t *testing.T) {
	SetSignalToMetricsDataset(nil, "production") // should not panic
}

func TestSetSignalToMetricsID(t *testing.T) {
	rule := &SignalToMetricsDefinition{}
	SetSignalToMetricsID(rule, "tsa-new")
	if rule.Metadata.Labels == nil {
		t.Fatal("expected the labels struct to be initialized")
	}
	assertPtrEqual(t, "Dash0Comid", rule.Metadata.Labels.Dash0Comid, "tsa-new")
}

func TestSetSignalToMetricsID_Overwrites(t *testing.T) {
	rule := &SignalToMetricsDefinition{
		Metadata: SignalToMetricsMetadata{
			Labels: &SignalToMetricsLabels{Dash0Comid: Ptr("existing-id")},
		},
	}
	SetSignalToMetricsID(rule, "tsa-new")
	assertPtrEqual(t, "Dash0Comid", rule.Metadata.Labels.Dash0Comid, "tsa-new")
}

func TestSetSignalToMetricsID_Nil(t *testing.T) {
	SetSignalToMetricsID(nil, "tsa-new") // should not panic
}

func TestSetSignalToMetricsIDIfAbsent(t *testing.T) {
	rule := &SignalToMetricsDefinition{}
	SetSignalToMetricsIDIfAbsent(rule, "tsa-new")
	if rule.Metadata.Labels == nil {
		t.Fatal("expected the labels struct to be initialized")
	}
	assertPtrEqual(t, "Dash0Comid", rule.Metadata.Labels.Dash0Comid, "tsa-new")
}

func TestSetSignalToMetricsIDIfAbsent_NoOpWhenAlreadySet(t *testing.T) {
	rule := &SignalToMetricsDefinition{
		Metadata: SignalToMetricsMetadata{
			Labels: &SignalToMetricsLabels{Dash0Comid: Ptr("existing-id")},
		},
	}
	SetSignalToMetricsIDIfAbsent(rule, "tsa-new")
	assertPtrEqual(t, "Dash0Comid", rule.Metadata.Labels.Dash0Comid, "existing-id")
}

func TestSetSignalToMetricsIDIfAbsent_Nil(t *testing.T) {
	SetSignalToMetricsIDIfAbsent(nil, "tsa-new") // should not panic
}

func TestClearSignalToMetricsID(t *testing.T) {
	rule := &SignalToMetricsDefinition{
		Metadata: SignalToMetricsMetadata{
			Labels: &SignalToMetricsLabels{Dash0Comid: Ptr("s2m-123")},
		},
	}
	ClearSignalToMetricsID(rule)
	if rule.Metadata.Labels.Dash0Comid != nil {
		t.Error("Dash0Comid should be nil")
	}
}

func TestClearSignalToMetricsID_NilLabels(t *testing.T) {
	ClearSignalToMetricsID(&SignalToMetricsDefinition{}) // should not panic
}

func TestClearSignalToMetricsID_Nil(t *testing.T) {
	ClearSignalToMetricsID(nil) // should not panic
}

/*
TestSignalToMetricsDefinition_JSONRoundTrip unmarshals a fully populated rule into the typed struct, marshals it again, and compares both documents.
Filter and matcher values are generated as json.RawMessage unions, so this is what proves a caller holding untyped JSON (for example the operator's CRD map) loses no field by going through SignalToMetricsDefinition.
*/
func TestSignalToMetricsDefinition_JSONRoundTrip(t *testing.T) {
	input := `{
		"kind": "Dash0SignalToMetrics",
		"metadata": {
			"name": "checkout-span-count",
			"labels": {
				"dash0.com/id": "s2m-123",
				"dash0.com/dataset": "default",
				"dash0.com/origin": "dash0-operator_uid_default_ns_checkout",
				"dash0.com/source": "operator",
				"dash0.com/version": "3"
			},
			"annotations": {
				"dash0.com/folder-path": "/shop/checkout",
				"dash0.com/sharing": "team:team_01abc",
				"dash0.com/created-at": "2026-10-01T10:00:00Z",
				"dash0.com/updated-at": "2026-10-01T11:00:00Z"
			}
		},
		"spec": {
			"enabled": true,
			"display": {"name": "Checkout span count"},
			"match": {
				"signal": "spans",
				"filters": [
					{"key": "service.name", "operator": "is", "value": "checkout"},
					{"key": "http.response.status_code", "operator": "is_one_of", "values": ["500", "503"]},
					{"key": "k8s.namespace.name", "operator": "is_set"}
				]
			},
			"output": {
				"name": "checkout_spans_total",
				"description": "Spans emitted by the checkout service",
				"interval": "1m",
				"keepResourceAttributes": [
					{"operator": "is", "value": "service.name"},
					{"operator": "is_one_of", "values": ["k8s.namespace.name", "k8s.pod.name"]}
				],
				"keepSignalAttributes": [
					{"operator": "starts_with", "value": "http."}
				]
			},
			"targetDatasetMode": "both",
			"alternativeDatasetId": "metrics-only"
		}
	}`

	var rule SignalToMetricsDefinition
	if err := json.Unmarshal([]byte(input), &rule); err != nil {
		t.Fatalf("unmarshal into SignalToMetricsDefinition: %v", err)
	}
	output, err := json.Marshal(&rule)
	if err != nil {
		t.Fatalf("marshal SignalToMetricsDefinition: %v", err)
	}

	var want, got map[string]any
	if err := json.Unmarshal([]byte(input), &want); err != nil {
		t.Fatalf("unmarshal input: %v", err)
	}
	if err := json.Unmarshal(output, &got); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	wantJSON, _ := json.MarshalIndent(want, "", "  ")
	gotJSON, _ := json.MarshalIndent(got, "", "  ")
	if string(wantJSON) != string(gotJSON) {
		t.Errorf("round trip changed the document\nwant:\n%s\ngot:\n%s", wantJSON, gotJSON)
	}
}
