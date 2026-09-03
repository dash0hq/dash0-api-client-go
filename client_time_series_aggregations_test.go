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

func newTestTimeSeriesAggregation() TimeSeriesAggregationDefinition {
	return TimeSeriesAggregationDefinition{
		Kind: Dash0TimeSeriesAggregation,
		Metadata: TimeSeriesAggregationMetadata{
			Name: "http-request-duration-rollup",
			Labels: &TimeSeriesAggregationLabels{
				Dash0Comid:      Ptr("tsa-123"),
				Dash0Comdataset: Ptr("default"),
			},
		},
		Spec: TimeSeriesAggregationSpec{
			Enabled: true,
			Display: &TimeSeriesAggregationDisplay{Name: "HTTP request duration rollup"},
			Match: TimeSeriesAggregationMetricNameMatch{
				MetricNameMatcher: Matcher{Operator: "is_set"},
			},
			Sample: TimeSeriesAggregationSample{
				Interval: "60s",
			},
		},
	}
}

func TestTimeSeriesAggregations_Integration(t *testing.T) {
	t.Run("ListTimeSeriesAggregations returns aggregations", func(t *testing.T) {
		aggregation := newTestTimeSeriesAggregation()
		second := newTestTimeSeriesAggregation()
		second.Metadata.Name = "cpu-rollup"

		var gotURL *url.URL
		var gotMethod string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotURL = r.URL
			gotMethod = r.Method
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(TimeSeriesAggregationListResponse{
				TimeSeriesAggregations: []TimeSeriesAggregationDefinition{aggregation, second},
			})
		}))
		defer server.Close()

		got, err := newTestClient(t, server.URL).ListTimeSeriesAggregations(context.Background(), nil)
		if err != nil {
			t.Fatalf("ListTimeSeriesAggregations failed: %v", err)
		}

		assertEqual(t, "method", gotMethod, http.MethodGet)
		assertEqual(t, "path", gotURL.Path, "/api/time-series-aggregations")
		if _, ok := gotURL.Query()["dataset"]; ok {
			t.Error("dataset query parameter should be absent when dataset is nil")
		}
		if len(got) != 2 {
			t.Fatalf("expected 2 aggregations, got %d", len(got))
		}
		assertEqual(t, "aggregations[0].Metadata.Name", got[0].Metadata.Name, "http-request-duration-rollup")
		assertEqual(t, "aggregations[1].Metadata.Name", got[1].Metadata.Name, "cpu-rollup")
	})

	t.Run("ListTimeSeriesAggregations sends the dataset query parameter", func(t *testing.T) {
		var gotURL *url.URL
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotURL = r.URL
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(TimeSeriesAggregationListResponse{})
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).ListTimeSeriesAggregations(context.Background(), Ptr("production"))
		if err != nil {
			t.Fatalf("ListTimeSeriesAggregations failed: %v", err)
		}
		assertEqual(t, "dataset", gotURL.Query().Get("dataset"), "production")
	})

	t.Run("ListTimeSeriesAggregations handles a null aggregation array", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"timeSeriesAggregations": null}`))
		}))
		defer server.Close()

		got, err := newTestClient(t, server.URL).ListTimeSeriesAggregations(context.Background(), nil)
		if err != nil {
			t.Fatalf("ListTimeSeriesAggregations failed: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("expected an empty slice, got %d entries", len(got))
		}
	})

	t.Run("ListTimeSeriesAggregations reports an unparsed 200 body", func(t *testing.T) {
		// No JSON content type, so the generated parser leaves JSON200 nil and the
		// client's own guard is what reports the problem.
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).ListTimeSeriesAggregations(context.Background(), nil)
		if err == nil {
			t.Fatal("expected an error for a 200 with no parsable body")
		}
		assertEqual(t, "error", err.Error(), "dash0: unexpected nil response")
	})

	t.Run("ListTimeSeriesAggregations wraps a malformed JSON body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{not json`))
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).ListTimeSeriesAggregations(context.Background(), nil)
		if err == nil {
			t.Fatal("expected an error for a malformed JSON body")
		}
		if got := err.Error(); !strings.Contains(got, "dash0: list time series aggregations failed") {
			t.Errorf("error = %q, want it to wrap %q", got, "dash0: list time series aggregations failed")
		}
	})

	t.Run("GetTimeSeriesAggregation returns the aggregation", func(t *testing.T) {
		aggregation := newTestTimeSeriesAggregation()
		var gotURL *url.URL
		var gotMethod string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotURL = r.URL
			gotMethod = r.Method
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(aggregation)
		}))
		defer server.Close()

		got, err := newTestClient(t, server.URL).GetTimeSeriesAggregation(context.Background(), "tsa-123", nil)
		if err != nil {
			t.Fatalf("GetTimeSeriesAggregation failed: %v", err)
		}

		assertEqual(t, "method", gotMethod, http.MethodGet)
		assertEqual(t, "path", gotURL.Path, "/api/time-series-aggregations/tsa-123")
		assertEqual(t, "Metadata.Name", got.Metadata.Name, "http-request-duration-rollup")
	})

	t.Run("GetTimeSeriesAggregation handles 404", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeAPIError(w, http.StatusNotFound, "Time series aggregation not found")
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).GetTimeSeriesAggregation(context.Background(), "nope", nil)
		if err == nil {
			t.Fatal("expected an error for a 404 response")
		}
		if !IsNotFound(err) {
			t.Errorf("IsNotFound(err) = false, want true (err = %v)", err)
		}
	})

	t.Run("GetTimeSeriesAggregation reports an unparsed 200 body", func(t *testing.T) {
		// No JSON content type, so the generated parser leaves JSON200 nil. Without
		// the client's own guard this returns (nil, nil) and the caller cannot tell
		// an empty result from a broken response.
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		got, err := newTestClient(t, server.URL).GetTimeSeriesAggregation(context.Background(), "tsa-123", nil)
		if err == nil {
			t.Fatal("expected an error for a 200 with no parsable body")
		}
		if got != nil {
			t.Errorf("expected a nil aggregation alongside the error, got %+v", got)
		}
		assertEqual(t, "error", err.Error(), "dash0: unexpected nil response")
	})

	t.Run("GetTimeSeriesAggregation sends the dataset query parameter", func(t *testing.T) {
		aggregation := newTestTimeSeriesAggregation()
		var gotURL *url.URL
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotURL = r.URL
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(aggregation)
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).GetTimeSeriesAggregation(context.Background(), "tsa-123", Ptr("production"))
		if err != nil {
			t.Fatalf("GetTimeSeriesAggregation failed: %v", err)
		}
		assertEqual(t, "dataset", gotURL.Query().Get("dataset"), "production")
	})

	t.Run("CreateTimeSeriesAggregation posts the definition", func(t *testing.T) {
		aggregation := newTestTimeSeriesAggregation()
		var gotURL *url.URL
		var gotMethod string
		var gotBody TimeSeriesAggregationDefinition
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotURL = r.URL
			gotMethod = r.Method
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(aggregation)
		}))
		defer server.Close()

		got, err := newTestClient(t, server.URL).CreateTimeSeriesAggregation(context.Background(), &aggregation, nil)
		if err != nil {
			t.Fatalf("CreateTimeSeriesAggregation failed: %v", err)
		}

		assertEqual(t, "method", gotMethod, http.MethodPost)
		assertEqual(t, "path", gotURL.Path, "/api/time-series-aggregations")
		assertEqual(t, "request body Metadata.Name", gotBody.Metadata.Name, "http-request-duration-rollup")
		assertEqual(t, "response Metadata.Name", got.Metadata.Name, "http-request-duration-rollup")
	})

	t.Run("CreateTimeSeriesAggregation handles 201", func(t *testing.T) {
		aggregation := newTestTimeSeriesAggregation()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(aggregation)
		}))
		defer server.Close()

		got, err := newTestClient(t, server.URL).CreateTimeSeriesAggregation(context.Background(), &aggregation, nil)
		if err != nil {
			t.Fatalf("CreateTimeSeriesAggregation failed: %v", err)
		}
		assertEqual(t, "Metadata.Name", got.Metadata.Name, "http-request-duration-rollup")
		assertEqual(t, "ID", GetTimeSeriesAggregationID(got), "tsa-123")
	})

	t.Run("CreateTimeSeriesAggregation handles 400", func(t *testing.T) {
		aggregation := newTestTimeSeriesAggregation()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeAPIError(w, http.StatusBadRequest, "spec.sample.interval is required")
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).CreateTimeSeriesAggregation(context.Background(), &aggregation, nil)
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

	t.Run("CreateTimeSeriesAggregation handles 403", func(t *testing.T) {
		aggregation := newTestTimeSeriesAggregation()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeAPIError(w, http.StatusForbidden, "insufficient permissions")
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).CreateTimeSeriesAggregation(context.Background(), &aggregation, nil)
		if err == nil {
			t.Fatal("expected an error for a 403 response")
		}
		if !IsForbidden(err) {
			t.Errorf("IsForbidden(err) = false, want true (err = %v)", err)
		}
	})

	t.Run("CreateTimeSeriesAggregation sends the dataset query parameter", func(t *testing.T) {
		aggregation := newTestTimeSeriesAggregation()
		var gotURL *url.URL
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotURL = r.URL
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(aggregation)
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).CreateTimeSeriesAggregation(context.Background(), &aggregation, Ptr("production"))
		if err != nil {
			t.Fatalf("CreateTimeSeriesAggregation failed: %v", err)
		}
		assertEqual(t, "dataset", gotURL.Query().Get("dataset"), "production")
	})

	t.Run("CreateTimeSeriesAggregation wraps a 201 body it cannot parse", func(t *testing.T) {
		aggregation := newTestTimeSeriesAggregation()
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

		_, err := newTestClient(t, server.URL).CreateTimeSeriesAggregation(context.Background(), &aggregation, nil)
		if err == nil {
			t.Fatal("expected an error for a 201 body that is not a definition")
		}
		if !strings.Contains(err.Error(), "dash0: failed to parse time series aggregation response") {
			t.Errorf("error = %q, want it to wrap the parse failure", err.Error())
		}
	})

	t.Run("CreateTimeSeriesAggregation wraps an empty 201 body", func(t *testing.T) {
		aggregation := newTestTimeSeriesAggregation()
		// No JSON content type and no body, so no parser case matches and the
		// fallback has nothing to decode.
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).CreateTimeSeriesAggregation(context.Background(), &aggregation, nil)
		if err == nil {
			t.Fatal("expected an error for a 201 with an empty body")
		}
		if !strings.Contains(err.Error(), "dash0: failed to parse time series aggregation response") {
			t.Errorf("error = %q, want it to wrap the parse failure", err.Error())
		}
	})

	t.Run("CreateTimeSeriesAggregation rejects a nil aggregation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("the request should not reach the server")
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).CreateTimeSeriesAggregation(context.Background(), nil, nil)
		if err == nil {
			t.Fatal("expected an error for a nil aggregation")
		}
		assertEqual(t, "error", err.Error(), "dash0: create time series aggregation requires a non-nil aggregation")
	})

	t.Run("UpdateTimeSeriesAggregation rejects a nil aggregation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("the request should not reach the server")
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).UpdateTimeSeriesAggregation(context.Background(), "tsa-123", nil, nil)
		if err == nil {
			t.Fatal("expected an error for a nil aggregation")
		}
		assertEqual(t, "error", err.Error(), "dash0: update time series aggregation requires a non-nil aggregation")
	})

	t.Run("UpdateTimeSeriesAggregation puts the definition", func(t *testing.T) {
		aggregation := newTestTimeSeriesAggregation()
		var gotURL *url.URL
		var gotMethod string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotURL = r.URL
			gotMethod = r.Method
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(aggregation)
		}))
		defer server.Close()

		got, err := newTestClient(t, server.URL).UpdateTimeSeriesAggregation(context.Background(), "tsa-123", &aggregation, nil)
		if err != nil {
			t.Fatalf("UpdateTimeSeriesAggregation failed: %v", err)
		}

		assertEqual(t, "method", gotMethod, http.MethodPut)
		assertEqual(t, "path", gotURL.Path, "/api/time-series-aggregations/tsa-123")
		assertEqual(t, "Metadata.Name", got.Metadata.Name, "http-request-duration-rollup")
	})

	t.Run("UpdateTimeSeriesAggregation handles 404", func(t *testing.T) {
		aggregation := newTestTimeSeriesAggregation()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeAPIError(w, http.StatusNotFound, "Time series aggregation not found")
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).UpdateTimeSeriesAggregation(context.Background(), "nope", &aggregation, nil)
		if err == nil {
			t.Fatal("expected an error for a 404 response")
		}
		if !IsNotFound(err) {
			t.Errorf("IsNotFound(err) = false, want true (err = %v)", err)
		}
	})

	t.Run("UpdateTimeSeriesAggregation reports an unparsed 200 body", func(t *testing.T) {
		aggregation := newTestTimeSeriesAggregation()
		// Same guard as Get: without it the caller gets (nil, nil) and cannot tell
		// a successful update from a response it could not read.
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		got, err := newTestClient(t, server.URL).UpdateTimeSeriesAggregation(context.Background(), "tsa-123", &aggregation, nil)
		if err == nil {
			t.Fatal("expected an error for a 200 with no parsable body")
		}
		if got != nil {
			t.Errorf("expected a nil aggregation alongside the error, got %+v", got)
		}
		assertEqual(t, "error", err.Error(), "dash0: unexpected nil response")
	})

	t.Run("UpdateTimeSeriesAggregation sends the dataset query parameter", func(t *testing.T) {
		aggregation := newTestTimeSeriesAggregation()
		var gotURL *url.URL
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotURL = r.URL
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(aggregation)
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).UpdateTimeSeriesAggregation(context.Background(), "tsa-123", &aggregation, Ptr("production"))
		if err != nil {
			t.Fatalf("UpdateTimeSeriesAggregation failed: %v", err)
		}
		assertEqual(t, "dataset", gotURL.Query().Get("dataset"), "production")
	})

	t.Run("DeleteTimeSeriesAggregation handles 200", func(t *testing.T) {
		var gotURL *url.URL
		var gotMethod string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotURL = r.URL
			gotMethod = r.Method
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		if err := newTestClient(t, server.URL).DeleteTimeSeriesAggregation(context.Background(), "tsa-123", nil); err != nil {
			t.Fatalf("DeleteTimeSeriesAggregation failed: %v", err)
		}
		assertEqual(t, "method", gotMethod, http.MethodDelete)
		assertEqual(t, "path", gotURL.Path, "/api/time-series-aggregations/tsa-123")
	})

	t.Run("DeleteTimeSeriesAggregation handles 204", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		if err := newTestClient(t, server.URL).DeleteTimeSeriesAggregation(context.Background(), "tsa-123", nil); err != nil {
			t.Fatalf("DeleteTimeSeriesAggregation failed: %v", err)
		}
	})

	t.Run("DeleteTimeSeriesAggregation handles 403", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeAPIError(w, http.StatusForbidden, "insufficient permissions")
		}))
		defer server.Close()

		err := newTestClient(t, server.URL).DeleteTimeSeriesAggregation(context.Background(), "tsa-123", nil)
		if err == nil {
			t.Fatal("expected an error for a 403 response")
		}
		if !IsForbidden(err) {
			t.Errorf("IsForbidden(err) = false, want true (err = %v)", err)
		}
	})

	t.Run("DeleteTimeSeriesAggregation sends the dataset query parameter", func(t *testing.T) {
		var gotURL *url.URL
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotURL = r.URL
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		if err := newTestClient(t, server.URL).DeleteTimeSeriesAggregation(context.Background(), "tsa-123", Ptr("production")); err != nil {
			t.Fatalf("DeleteTimeSeriesAggregation failed: %v", err)
		}
		assertEqual(t, "dataset", gotURL.Query().Get("dataset"), "production")
	})

	t.Run("ListTimeSeriesAggregationsIter iterates every aggregation", func(t *testing.T) {
		aggregation := newTestTimeSeriesAggregation()
		second := newTestTimeSeriesAggregation()
		second.Metadata.Name = "cpu-rollup"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(TimeSeriesAggregationListResponse{
				TimeSeriesAggregations: []TimeSeriesAggregationDefinition{aggregation, second},
			})
		}))
		defer server.Close()

		iter := newTestClient(t, server.URL).ListTimeSeriesAggregationsIter(context.Background(), nil)
		var names []string
		for iter.Next() {
			names = append(names, iter.Current().Metadata.Name)
		}
		if err := iter.Err(); err != nil {
			t.Fatalf("iterator error: %v", err)
		}
		if len(names) != 2 {
			t.Fatalf("expected 2 aggregations, got %d", len(names))
		}
		assertEqual(t, "names[0]", names[0], "http-request-duration-rollup")
		assertEqual(t, "names[1]", names[1], "cpu-rollup")
	})

	t.Run("ListTimeSeriesAggregationsIter surfaces the list error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeAPIError(w, http.StatusInternalServerError, "boom")
		}))
		defer server.Close()

		iter := newTestClient(t, server.URL).ListTimeSeriesAggregationsIter(context.Background(), nil)
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

func TestTimeSeriesAggregations_APINotConfigured(t *testing.T) {
	c, err := NewClient(
		WithOtlpEndpoint(OtlpEncodingJson, "https://ingress.eu-west-1.aws.dash0.com"),
		WithAuthToken("auth_test123"),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	aggregation := newTestTimeSeriesAggregation()
	ctx := context.Background()

	if _, err := c.ListTimeSeriesAggregations(ctx, nil); !errors.Is(err, ErrAPINotConfigured) {
		t.Errorf("ListTimeSeriesAggregations error = %v, want ErrAPINotConfigured", err)
	}
	if _, err := c.GetTimeSeriesAggregation(ctx, "tsa-123", nil); !errors.Is(err, ErrAPINotConfigured) {
		t.Errorf("GetTimeSeriesAggregation error = %v, want ErrAPINotConfigured", err)
	}
	if _, err := c.CreateTimeSeriesAggregation(ctx, &aggregation, nil); !errors.Is(err, ErrAPINotConfigured) {
		t.Errorf("CreateTimeSeriesAggregation error = %v, want ErrAPINotConfigured", err)
	}
	if _, err := c.UpdateTimeSeriesAggregation(ctx, "tsa-123", &aggregation, nil); !errors.Is(err, ErrAPINotConfigured) {
		t.Errorf("UpdateTimeSeriesAggregation error = %v, want ErrAPINotConfigured", err)
	}
	if err := c.DeleteTimeSeriesAggregation(ctx, "tsa-123", nil); !errors.Is(err, ErrAPINotConfigured) {
		t.Errorf("DeleteTimeSeriesAggregation error = %v, want ErrAPINotConfigured", err)
	}
	iter := c.ListTimeSeriesAggregationsIter(ctx, nil)
	if !errors.Is(iter.Err(), ErrAPINotConfigured) {
		t.Errorf("ListTimeSeriesAggregationsIter error = %v, want ErrAPINotConfigured", iter.Err())
	}
}

func TestStripTimeSeriesAggregationServerFields(t *testing.T) {
	createdAt := time.Now()
	updatedAt := time.Now()
	deletedAt := time.Now()
	source := CrdSource("terraform")

	aggregation := &TimeSeriesAggregationDefinition{
		Metadata: TimeSeriesAggregationMetadata{
			Name: "keep-this-name",
			Annotations: &TimeSeriesAggregationAnnotations{
				Dash0ComcreatedAt: &createdAt,
				Dash0ComupdatedAt: &updatedAt,
				Dash0ComdeletedAt: &deletedAt,
			},
			Labels: &TimeSeriesAggregationLabels{
				Dash0Comid:      Ptr("keep-this-id"),
				Dash0Comversion: Ptr("2"),
				Dash0Comsource:  &source,
				Dash0Comdataset: Ptr("ds"),
				Dash0Comorigin:  Ptr("my-origin"),
				Custom:          &map[string]string{"team": "platform"},
			},
		},
		Spec: TimeSeriesAggregationSpec{Enabled: true},
	}

	StripTimeSeriesAggregationServerFields(aggregation)

	if aggregation.Metadata.Annotations.Dash0ComcreatedAt != nil {
		t.Error("Dash0ComcreatedAt should be nil")
	}
	if aggregation.Metadata.Annotations.Dash0ComupdatedAt != nil {
		t.Error("Dash0ComupdatedAt should be nil")
	}
	if aggregation.Metadata.Annotations.Dash0ComdeletedAt != nil {
		t.Error("Dash0ComdeletedAt should be nil")
	}
	if aggregation.Metadata.Labels.Dash0Comversion != nil {
		t.Error("Dash0Comversion should be nil")
	}
	if aggregation.Metadata.Labels.Dash0Comsource != nil {
		t.Error("Dash0Comsource should be nil")
	}
	if aggregation.Metadata.Labels.Dash0Comdataset != nil {
		t.Error("Dash0Comdataset should be nil")
	}
	if aggregation.Metadata.Labels.Dash0Comorigin != nil {
		t.Error("Dash0Comorigin should be nil")
	}

	// The ID survives, so a Get -> Strip -> Update round-trip keeps addressing the
	// same asset. Callers who want it gone call ClearTimeSeriesAggregationID.
	if got := GetTimeSeriesAggregationID(aggregation); got != "keep-this-id" {
		t.Errorf("Dash0Comid = %q, want %q (the ID must be preserved)", got, "keep-this-id")
	}
	assertEqual(t, "Metadata.Name", aggregation.Metadata.Name, "keep-this-name")
	if aggregation.Metadata.Labels.Custom == nil {
		t.Error("Labels.Custom should be preserved")
	}
	if !aggregation.Spec.Enabled {
		t.Error("Spec should be preserved")
	}
}

func TestStripTimeSeriesAggregationServerFields_ZeroValue(t *testing.T) {
	aggregation := &TimeSeriesAggregationDefinition{}
	StripTimeSeriesAggregationServerFields(aggregation) // should not panic
	if aggregation.Metadata.Labels != nil {
		t.Error("Labels should remain nil")
	}
	if aggregation.Metadata.Annotations != nil {
		t.Error("Annotations should remain nil")
	}
}

func TestStripTimeSeriesAggregationServerFields_Nil(t *testing.T) {
	StripTimeSeriesAggregationServerFields(nil) // should not panic
}

func TestGetTimeSeriesAggregationName(t *testing.T) {
	tests := []struct {
		name        string
		aggregation *TimeSeriesAggregationDefinition
		want        string
	}{
		{
			"from display name",
			&TimeSeriesAggregationDefinition{
				Spec: TimeSeriesAggregationSpec{
					Display: &TimeSeriesAggregationDisplay{Name: "Display Name"},
				},
			},
			"Display Name",
		},
		{
			"falls back to metadata name when display is nil",
			&TimeSeriesAggregationDefinition{
				Metadata: TimeSeriesAggregationMetadata{Name: "meta-name"},
			},
			"meta-name",
		},
		{
			"falls back to metadata name when display name is empty",
			&TimeSeriesAggregationDefinition{
				Metadata: TimeSeriesAggregationMetadata{Name: "meta-name"},
				Spec: TimeSeriesAggregationSpec{
					Display: &TimeSeriesAggregationDisplay{Name: ""},
				},
			},
			"meta-name",
		},
		{"nil aggregation", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetTimeSeriesAggregationName(tt.aggregation); got != tt.want {
				t.Errorf("GetTimeSeriesAggregationName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetTimeSeriesAggregationID(t *testing.T) {
	tests := []struct {
		name        string
		aggregation *TimeSeriesAggregationDefinition
		want        string
	}{
		{
			"with ID",
			&TimeSeriesAggregationDefinition{
				Metadata: TimeSeriesAggregationMetadata{
					Labels: &TimeSeriesAggregationLabels{Dash0Comid: Ptr("tsa-123")},
				},
			},
			"tsa-123",
		},
		{"nil aggregation", nil, ""},
		{"nil labels", &TimeSeriesAggregationDefinition{}, ""},
		{
			"nil ID",
			&TimeSeriesAggregationDefinition{
				Metadata: TimeSeriesAggregationMetadata{Labels: &TimeSeriesAggregationLabels{}},
			},
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetTimeSeriesAggregationID(tt.aggregation); got != tt.want {
				t.Errorf("GetTimeSeriesAggregationID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetTimeSeriesAggregationDataset(t *testing.T) {
	tests := []struct {
		name        string
		aggregation *TimeSeriesAggregationDefinition
		want        string
	}{
		{
			"with dataset",
			&TimeSeriesAggregationDefinition{
				Metadata: TimeSeriesAggregationMetadata{
					Labels: &TimeSeriesAggregationLabels{Dash0Comdataset: Ptr("production")},
				},
			},
			"production",
		},
		{"nil aggregation", nil, ""},
		{"nil labels", &TimeSeriesAggregationDefinition{}, ""},
		{
			"nil dataset",
			&TimeSeriesAggregationDefinition{
				Metadata: TimeSeriesAggregationMetadata{Labels: &TimeSeriesAggregationLabels{}},
			},
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetTimeSeriesAggregationDataset(tt.aggregation); got != tt.want {
				t.Errorf("GetTimeSeriesAggregationDataset() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetTimeSeriesAggregationDataset(t *testing.T) {
	aggregation := &TimeSeriesAggregationDefinition{}
	SetTimeSeriesAggregationDataset(aggregation, "production")
	if aggregation.Metadata.Labels == nil {
		t.Fatal("expected the labels struct to be initialized")
	}
	assertPtrEqual(t, "Dash0Comdataset", aggregation.Metadata.Labels.Dash0Comdataset, "production")
}

func TestSetTimeSeriesAggregationDataset_Nil(t *testing.T) {
	SetTimeSeriesAggregationDataset(nil, "production") // should not panic
}

func TestSetTimeSeriesAggregationID(t *testing.T) {
	aggregation := &TimeSeriesAggregationDefinition{}
	SetTimeSeriesAggregationID(aggregation, "tsa-new")
	if aggregation.Metadata.Labels == nil {
		t.Fatal("expected the labels struct to be initialized")
	}
	assertPtrEqual(t, "Dash0Comid", aggregation.Metadata.Labels.Dash0Comid, "tsa-new")
}

func TestSetTimeSeriesAggregationID_Overwrites(t *testing.T) {
	aggregation := &TimeSeriesAggregationDefinition{
		Metadata: TimeSeriesAggregationMetadata{
			Labels: &TimeSeriesAggregationLabels{Dash0Comid: Ptr("existing-id")},
		},
	}
	SetTimeSeriesAggregationID(aggregation, "tsa-new")
	assertPtrEqual(t, "Dash0Comid", aggregation.Metadata.Labels.Dash0Comid, "tsa-new")
}

func TestSetTimeSeriesAggregationID_Nil(t *testing.T) {
	SetTimeSeriesAggregationID(nil, "tsa-new") // should not panic
}

func TestSetTimeSeriesAggregationIDIfAbsent(t *testing.T) {
	aggregation := &TimeSeriesAggregationDefinition{}
	SetTimeSeriesAggregationIDIfAbsent(aggregation, "tsa-new")
	if aggregation.Metadata.Labels == nil {
		t.Fatal("expected the labels struct to be initialized")
	}
	assertPtrEqual(t, "Dash0Comid", aggregation.Metadata.Labels.Dash0Comid, "tsa-new")
}

func TestSetTimeSeriesAggregationIDIfAbsent_NoOpWhenAlreadySet(t *testing.T) {
	aggregation := &TimeSeriesAggregationDefinition{
		Metadata: TimeSeriesAggregationMetadata{
			Labels: &TimeSeriesAggregationLabels{Dash0Comid: Ptr("existing-id")},
		},
	}
	SetTimeSeriesAggregationIDIfAbsent(aggregation, "tsa-new")
	assertPtrEqual(t, "Dash0Comid", aggregation.Metadata.Labels.Dash0Comid, "existing-id")
}

func TestSetTimeSeriesAggregationIDIfAbsent_Nil(t *testing.T) {
	SetTimeSeriesAggregationIDIfAbsent(nil, "tsa-new") // should not panic
}

func TestClearTimeSeriesAggregationID(t *testing.T) {
	aggregation := &TimeSeriesAggregationDefinition{
		Metadata: TimeSeriesAggregationMetadata{
			Labels: &TimeSeriesAggregationLabels{Dash0Comid: Ptr("tsa-123")},
		},
	}
	ClearTimeSeriesAggregationID(aggregation)
	if aggregation.Metadata.Labels.Dash0Comid != nil {
		t.Error("Dash0Comid should be nil")
	}
}

func TestClearTimeSeriesAggregationID_NilLabels(t *testing.T) {
	ClearTimeSeriesAggregationID(&TimeSeriesAggregationDefinition{}) // should not panic
}

func TestClearTimeSeriesAggregationID_Nil(t *testing.T) {
	ClearTimeSeriesAggregationID(nil) // should not panic
}
