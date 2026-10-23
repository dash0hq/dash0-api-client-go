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
)

func newTestSamplingRule(t *testing.T) SamplingDefinition {
	t.Helper()
	var conditions SamplingCondition
	if err := conditions.FromSamplingConditionError(SamplingConditionError{
		Kind: SamplingConditionErrorKindError,
	}); err != nil {
		t.Fatalf("failed to build sampling condition: %v", err)
	}
	return SamplingDefinition{
		Kind: Dash0Sampling,
		Metadata: SamplingMetadata{
			Name: "keep-errors",
			Labels: &SamplingLabels{
				Dash0Comid:      Ptr("sr-123"),
				Dash0Comdataset: Ptr("default"),
			},
		},
		Spec: SamplingSpec{
			Enabled:    true,
			Display:    &SamplingDisplay{Name: "Keep error traces"},
			Conditions: conditions,
		},
	}
}

func TestSamplingRules_Integration(t *testing.T) {
	t.Run("ListSamplingRules returns rules", func(t *testing.T) {
		rule := newTestSamplingRule(t)
		second := newTestSamplingRule(t)
		second.Metadata.Name = "keep-slow-checkouts"

		var gotURL *url.URL
		var gotMethod string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotURL = r.URL
			gotMethod = r.Method
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(GetSamplingRulesResponse{
				SamplingRules: []SamplingDefinition{rule, second},
			})
		}))
		defer server.Close()

		got, err := newTestClient(t, server.URL).ListSamplingRules(context.Background(), nil)
		if err != nil {
			t.Fatalf("ListSamplingRules failed: %v", err)
		}

		assertEqual(t, "method", gotMethod, http.MethodGet)
		assertEqual(t, "path", gotURL.Path, "/api/sampling-rules")
		if _, ok := gotURL.Query()["dataset"]; ok {
			t.Error("dataset query parameter should be absent when dataset is nil")
		}
		if len(got) != 2 {
			t.Fatalf("expected 2 rules, got %d", len(got))
		}
		assertEqual(t, "rules[0].Metadata.Name", got[0].Metadata.Name, "keep-errors")
		assertEqual(t, "rules[1].Metadata.Name", got[1].Metadata.Name, "keep-slow-checkouts")
	})

	t.Run("ListSamplingRules sends the dataset query parameter", func(t *testing.T) {
		var gotURL *url.URL
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotURL = r.URL
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(GetSamplingRulesResponse{})
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).ListSamplingRules(context.Background(), Ptr("production"))
		if err != nil {
			t.Fatalf("ListSamplingRules failed: %v", err)
		}
		assertEqual(t, "dataset", gotURL.Query().Get("dataset"), "production")
	})

	t.Run("ListSamplingRules handles a null rule array", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"samplingRules": null}`))
		}))
		defer server.Close()

		got, err := newTestClient(t, server.URL).ListSamplingRules(context.Background(), nil)
		if err != nil {
			t.Fatalf("ListSamplingRules failed: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("expected an empty slice, got %d entries", len(got))
		}
	})

	t.Run("ListSamplingRules reports an unparsed 200 body", func(t *testing.T) {
		// No JSON content type, so the generated parser leaves JSON200 nil and the
		// client's own guard is what reports the problem.
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).ListSamplingRules(context.Background(), nil)
		if err == nil {
			t.Fatal("expected an error for a 200 with no parsable body")
		}
		assertEqual(t, "error", err.Error(), "dash0: unexpected nil response")
	})

	t.Run("ListSamplingRules wraps a malformed JSON body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{not json`))
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).ListSamplingRules(context.Background(), nil)
		if err == nil {
			t.Fatal("expected an error for a malformed JSON body")
		}
		if got := err.Error(); !strings.Contains(got, "dash0: list sampling rules failed") {
			t.Errorf("error = %q, want it to wrap %q", got, "dash0: list sampling rules failed")
		}
	})

	t.Run("ListSamplingRules handles 403", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeAPIError(w, http.StatusForbidden, "insufficient permissions")
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).ListSamplingRules(context.Background(), nil)
		if err == nil {
			t.Fatal("expected an error for a 403 response")
		}
		if !IsForbidden(err) {
			t.Errorf("IsForbidden(err) = false, want true (err = %v)", err)
		}
	})

	t.Run("GetSamplingRule returns the rule", func(t *testing.T) {
		rule := newTestSamplingRule(t)
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

		got, err := newTestClient(t, server.URL).GetSamplingRule(context.Background(), "sr-123", nil)
		if err != nil {
			t.Fatalf("GetSamplingRule failed: %v", err)
		}

		assertEqual(t, "method", gotMethod, http.MethodGet)
		assertEqual(t, "path", gotURL.Path, "/api/sampling-rules/sr-123")
		assertEqual(t, "Metadata.Name", got.Metadata.Name, "keep-errors")
		assertEqual(t, "ID", GetSamplingRuleID(got), "sr-123")
		if !got.Spec.Enabled {
			t.Error("Spec.Enabled = false, want true")
		}
	})

	t.Run("GetSamplingRule handles 404", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeAPIError(w, http.StatusNotFound, "Sampling rule not found")
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).GetSamplingRule(context.Background(), "nope", nil)
		if err == nil {
			t.Fatal("expected an error for a 404 response")
		}
		if !IsNotFound(err) {
			t.Errorf("IsNotFound(err) = false, want true (err = %v)", err)
		}
	})

	t.Run("GetSamplingRule sends the dataset query parameter", func(t *testing.T) {
		rule := newTestSamplingRule(t)
		var gotURL *url.URL
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotURL = r.URL
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(rule)
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).GetSamplingRule(context.Background(), "sr-123", Ptr("production"))
		if err != nil {
			t.Fatalf("GetSamplingRule failed: %v", err)
		}
		assertEqual(t, "dataset", gotURL.Query().Get("dataset"), "production")
	})

	t.Run("CreateSamplingRule posts the definition", func(t *testing.T) {
		rule := newTestSamplingRule(t)
		var gotURL *url.URL
		var gotMethod string
		var gotBody SamplingDefinition
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotURL = r.URL
			gotMethod = r.Method
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(rule)
		}))
		defer server.Close()

		got, err := newTestClient(t, server.URL).CreateSamplingRule(context.Background(), &rule, nil)
		if err != nil {
			t.Fatalf("CreateSamplingRule failed: %v", err)
		}

		assertEqual(t, "method", gotMethod, http.MethodPost)
		assertEqual(t, "path", gotURL.Path, "/api/sampling-rules")
		assertEqual(t, "request body Metadata.Name", gotBody.Metadata.Name, "keep-errors")
		condition, err := gotBody.Spec.Conditions.AsSamplingConditionError()
		if err != nil {
			t.Fatalf("request body conditions are not an error condition: %v", err)
		}
		assertEqual(t, "request body condition kind", string(condition.Kind), string(SamplingConditionErrorKindError))
		assertEqual(t, "response Metadata.Name", got.Metadata.Name, "keep-errors")
	})

	t.Run("CreateSamplingRule handles 201", func(t *testing.T) {
		rule := newTestSamplingRule(t)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(rule)
		}))
		defer server.Close()

		got, err := newTestClient(t, server.URL).CreateSamplingRule(context.Background(), &rule, nil)
		if err != nil {
			t.Fatalf("CreateSamplingRule failed: %v", err)
		}
		assertEqual(t, "Metadata.Name", got.Metadata.Name, "keep-errors")
		assertEqual(t, "ID", GetSamplingRuleID(got), "sr-123")
	})

	t.Run("CreateSamplingRule handles 400", func(t *testing.T) {
		rule := newTestSamplingRule(t)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeAPIError(w, http.StatusBadRequest, "spec.conditions is required")
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).CreateSamplingRule(context.Background(), &rule, nil)
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
		assertEqual(t, "APIError.Message", apiErr.Message, "spec.conditions is required")
	})

	t.Run("CreateSamplingRule handles 403", func(t *testing.T) {
		rule := newTestSamplingRule(t)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeAPIError(w, http.StatusForbidden, "insufficient permissions")
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).CreateSamplingRule(context.Background(), &rule, nil)
		if err == nil {
			t.Fatal("expected an error for a 403 response")
		}
		if !IsForbidden(err) {
			t.Errorf("IsForbidden(err) = false, want true (err = %v)", err)
		}
	})

	t.Run("CreateSamplingRule sends the dataset query parameter", func(t *testing.T) {
		rule := newTestSamplingRule(t)
		var gotURL *url.URL
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotURL = r.URL
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(rule)
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).CreateSamplingRule(context.Background(), &rule, Ptr("production"))
		if err != nil {
			t.Fatalf("CreateSamplingRule failed: %v", err)
		}
		assertEqual(t, "dataset", gotURL.Query().Get("dataset"), "production")
	})

	t.Run("CreateSamplingRule wraps a 201 body it cannot parse", func(t *testing.T) {
		rule := newTestSamplingRule(t)
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

		_, err := newTestClient(t, server.URL).CreateSamplingRule(context.Background(), &rule, nil)
		if err == nil {
			t.Fatal("expected an error for a 201 body that is not a definition")
		}
		if !strings.Contains(err.Error(), "dash0: failed to parse sampling rule response") {
			t.Errorf("error = %q, want it to wrap the parse failure", err.Error())
		}
	})

	t.Run("CreateSamplingRule rejects a nil rule", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("the request should not reach the server")
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).CreateSamplingRule(context.Background(), nil, nil)
		if err == nil {
			t.Fatal("expected an error for a nil rule")
		}
		assertEqual(t, "error", err.Error(), "dash0: create sampling rule requires a non-nil rule")
	})

	t.Run("UpdateSamplingRule rejects a nil rule", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("the request should not reach the server")
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).UpdateSamplingRule(context.Background(), "sr-123", nil, nil)
		if err == nil {
			t.Fatal("expected an error for a nil rule")
		}
		assertEqual(t, "error", err.Error(), "dash0: update sampling rule requires a non-nil rule")
	})

	t.Run("UpdateSamplingRule puts the definition", func(t *testing.T) {
		rule := newTestSamplingRule(t)
		var gotURL *url.URL
		var gotMethod string
		var gotBody SamplingDefinition
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotURL = r.URL
			gotMethod = r.Method
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(rule)
		}))
		defer server.Close()

		got, err := newTestClient(t, server.URL).UpdateSamplingRule(context.Background(), "sr-123", &rule, nil)
		if err != nil {
			t.Fatalf("UpdateSamplingRule failed: %v", err)
		}

		assertEqual(t, "method", gotMethod, http.MethodPut)
		assertEqual(t, "path", gotURL.Path, "/api/sampling-rules/sr-123")
		assertEqual(t, "request body Metadata.Name", gotBody.Metadata.Name, "keep-errors")
		assertEqual(t, "Metadata.Name", got.Metadata.Name, "keep-errors")
	})

	t.Run("UpdateSamplingRule handles 400", func(t *testing.T) {
		rule := newTestSamplingRule(t)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeAPIError(w, http.StatusBadRequest, "spec.rateLimit.rate must be positive")
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).UpdateSamplingRule(context.Background(), "sr-123", &rule, nil)
		if err == nil {
			t.Fatal("expected an error for a 400 response")
		}
		if !IsBadRequest(err) {
			t.Errorf("IsBadRequest(err) = false, want true (err = %v)", err)
		}
	})

	t.Run("UpdateSamplingRule handles 404", func(t *testing.T) {
		rule := newTestSamplingRule(t)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeAPIError(w, http.StatusNotFound, "Sampling rule not found")
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).UpdateSamplingRule(context.Background(), "nope", &rule, nil)
		if err == nil {
			t.Fatal("expected an error for a 404 response")
		}
		if !IsNotFound(err) {
			t.Errorf("IsNotFound(err) = false, want true (err = %v)", err)
		}
	})

	t.Run("UpdateSamplingRule sends the dataset query parameter", func(t *testing.T) {
		rule := newTestSamplingRule(t)
		var gotURL *url.URL
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotURL = r.URL
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(rule)
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).UpdateSamplingRule(context.Background(), "sr-123", &rule, Ptr("production"))
		if err != nil {
			t.Fatalf("UpdateSamplingRule failed: %v", err)
		}
		assertEqual(t, "dataset", gotURL.Query().Get("dataset"), "production")
	})

	t.Run("DeleteSamplingRule handles 200", func(t *testing.T) {
		var gotURL *url.URL
		var gotMethod string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotURL = r.URL
			gotMethod = r.Method
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		if err := newTestClient(t, server.URL).DeleteSamplingRule(context.Background(), "sr-123", nil); err != nil {
			t.Fatalf("DeleteSamplingRule failed: %v", err)
		}
		assertEqual(t, "method", gotMethod, http.MethodDelete)
		assertEqual(t, "path", gotURL.Path, "/api/sampling-rules/sr-123")
	})

	t.Run("DeleteSamplingRule handles 204", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		if err := newTestClient(t, server.URL).DeleteSamplingRule(context.Background(), "sr-123", nil); err != nil {
			t.Fatalf("DeleteSamplingRule failed: %v", err)
		}
	})

	t.Run("DeleteSamplingRule handles 404", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeAPIError(w, http.StatusNotFound, "Sampling rule not found")
		}))
		defer server.Close()

		err := newTestClient(t, server.URL).DeleteSamplingRule(context.Background(), "nope", nil)
		if err == nil {
			t.Fatal("expected an error for a 404 response")
		}
		if !IsNotFound(err) {
			t.Errorf("IsNotFound(err) = false, want true (err = %v)", err)
		}
	})

	t.Run("DeleteSamplingRule handles 403", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeAPIError(w, http.StatusForbidden, "insufficient permissions")
		}))
		defer server.Close()

		err := newTestClient(t, server.URL).DeleteSamplingRule(context.Background(), "sr-123", nil)
		if err == nil {
			t.Fatal("expected an error for a 403 response")
		}
		if !IsForbidden(err) {
			t.Errorf("IsForbidden(err) = false, want true (err = %v)", err)
		}
	})

	t.Run("DeleteSamplingRule sends the dataset query parameter", func(t *testing.T) {
		var gotURL *url.URL
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotURL = r.URL
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		if err := newTestClient(t, server.URL).DeleteSamplingRule(context.Background(), "sr-123", Ptr("production")); err != nil {
			t.Fatalf("DeleteSamplingRule failed: %v", err)
		}
		assertEqual(t, "dataset", gotURL.Query().Get("dataset"), "production")
	})

	t.Run("ListSamplingRulesIter iterates every rule", func(t *testing.T) {
		rule := newTestSamplingRule(t)
		second := newTestSamplingRule(t)
		second.Metadata.Name = "keep-slow-checkouts"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(GetSamplingRulesResponse{
				SamplingRules: []SamplingDefinition{rule, second},
			})
		}))
		defer server.Close()

		iter := newTestClient(t, server.URL).ListSamplingRulesIter(context.Background(), nil)
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
		assertEqual(t, "names[0]", names[0], "keep-errors")
		assertEqual(t, "names[1]", names[1], "keep-slow-checkouts")
	})

	t.Run("ListSamplingRulesIter surfaces the list error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeAPIError(w, http.StatusInternalServerError, "boom")
		}))
		defer server.Close()

		iter := newTestClient(t, server.URL).ListSamplingRulesIter(context.Background(), nil)
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

func TestSamplingRules_APINotConfigured(t *testing.T) {
	c, err := NewClient(
		WithOtlpEndpoint(OtlpEncodingJson, "https://ingress.eu-west-1.aws.dash0.com"),
		WithAuthToken("auth_test123"),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	rule := newTestSamplingRule(t)
	ctx := context.Background()

	if _, err := c.ListSamplingRules(ctx, nil); !errors.Is(err, ErrAPINotConfigured) {
		t.Errorf("ListSamplingRules error = %v, want ErrAPINotConfigured", err)
	}
	if _, err := c.GetSamplingRule(ctx, "sr-123", nil); !errors.Is(err, ErrAPINotConfigured) {
		t.Errorf("GetSamplingRule error = %v, want ErrAPINotConfigured", err)
	}
	if _, err := c.CreateSamplingRule(ctx, &rule, nil); !errors.Is(err, ErrAPINotConfigured) {
		t.Errorf("CreateSamplingRule error = %v, want ErrAPINotConfigured", err)
	}
	if _, err := c.UpdateSamplingRule(ctx, "sr-123", &rule, nil); !errors.Is(err, ErrAPINotConfigured) {
		t.Errorf("UpdateSamplingRule error = %v, want ErrAPINotConfigured", err)
	}
	if err := c.DeleteSamplingRule(ctx, "sr-123", nil); !errors.Is(err, ErrAPINotConfigured) {
		t.Errorf("DeleteSamplingRule error = %v, want ErrAPINotConfigured", err)
	}
	iter := c.ListSamplingRulesIter(ctx, nil)
	if !errors.Is(iter.Err(), ErrAPINotConfigured) {
		t.Errorf("ListSamplingRulesIter error = %v, want ErrAPINotConfigured", iter.Err())
	}
}

func TestStripSamplingRuleServerFields(t *testing.T) {
	source := CrdSource("terraform")

	rule := &SamplingDefinition{
		Metadata: SamplingMetadata{
			Name: "keep-this-name",
			Labels: &SamplingLabels{
				Dash0Comid:      Ptr("keep-this-id"),
				Dash0Comversion: Ptr("2"),
				Dash0Comsource:  &source,
				Dash0Comdataset: Ptr("ds"),
				Dash0Comorigin:  Ptr("my-origin"),
				Custom:          &map[string]string{"team": "platform"},
			},
		},
		Spec: SamplingSpec{
			Enabled:   true,
			Display:   &SamplingDisplay{Name: "Keep this display name"},
			RateLimit: &SamplingRateLimit{Rate: 100},
		},
	}

	StripSamplingRuleServerFields(rule)

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
	// same asset. Callers who want it gone call ClearSamplingRuleID.
	if got := GetSamplingRuleID(rule); got != "keep-this-id" {
		t.Errorf("Dash0Comid = %q, want %q (the ID must be preserved)", got, "keep-this-id")
	}
	assertEqual(t, "Metadata.Name", rule.Metadata.Name, "keep-this-name")
	if rule.Metadata.Labels.Custom == nil {
		t.Error("Labels.Custom should be preserved")
	}

	// Enabled is user-set, not server-managed. Clearing it would silently disable
	// the rule on the next update.
	if !rule.Spec.Enabled {
		t.Error("Spec.Enabled should be preserved")
	}
	assertEqual(t, "Spec.Display.Name", GetSamplingRuleName(rule), "Keep this display name")
	if rule.Spec.RateLimit == nil || rule.Spec.RateLimit.Rate != 100 {
		t.Error("Spec.RateLimit should be preserved")
	}
}

func TestStripSamplingRuleServerFields_ZeroValue(t *testing.T) {
	rule := &SamplingDefinition{}
	StripSamplingRuleServerFields(rule) // should not panic
	if rule.Metadata.Labels != nil {
		t.Error("Labels should remain nil")
	}
}

func TestStripSamplingRuleServerFields_Nil(t *testing.T) {
	StripSamplingRuleServerFields(nil) // should not panic
}

func TestGetSamplingRuleName(t *testing.T) {
	tests := []struct {
		name string
		rule *SamplingDefinition
		want string
	}{
		{
			"from display name",
			&SamplingDefinition{
				Spec: SamplingSpec{
					Display: &SamplingDisplay{Name: "Display Name"},
				},
			},
			"Display Name",
		},
		{
			"falls back to metadata name when display is nil",
			&SamplingDefinition{
				Metadata: SamplingMetadata{Name: "meta-name"},
			},
			"meta-name",
		},
		{
			"falls back to metadata name when display name is empty",
			&SamplingDefinition{
				Metadata: SamplingMetadata{Name: "meta-name"},
				Spec: SamplingSpec{
					Display: &SamplingDisplay{Name: ""},
				},
			},
			"meta-name",
		},
		{"nil rule", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetSamplingRuleName(tt.rule); got != tt.want {
				t.Errorf("GetSamplingRuleName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetSamplingRuleID(t *testing.T) {
	tests := []struct {
		name string
		rule *SamplingDefinition
		want string
	}{
		{
			"with ID",
			&SamplingDefinition{
				Metadata: SamplingMetadata{
					Labels: &SamplingLabels{Dash0Comid: Ptr("sr-123")},
				},
			},
			"sr-123",
		},
		{"nil rule", nil, ""},
		{"nil labels", &SamplingDefinition{}, ""},
		{
			"nil ID",
			&SamplingDefinition{
				Metadata: SamplingMetadata{Labels: &SamplingLabels{}},
			},
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetSamplingRuleID(tt.rule); got != tt.want {
				t.Errorf("GetSamplingRuleID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetSamplingRuleID(t *testing.T) {
	rule := &SamplingDefinition{}
	SetSamplingRuleID(rule, "sr-new")
	if rule.Metadata.Labels == nil {
		t.Fatal("expected the labels struct to be initialized")
	}
	assertPtrEqual(t, "Dash0Comid", rule.Metadata.Labels.Dash0Comid, "sr-new")
}

func TestSetSamplingRuleID_Overwrites(t *testing.T) {
	rule := &SamplingDefinition{
		Metadata: SamplingMetadata{
			Labels: &SamplingLabels{Dash0Comid: Ptr("existing-id")},
		},
	}
	SetSamplingRuleID(rule, "sr-new")
	assertPtrEqual(t, "Dash0Comid", rule.Metadata.Labels.Dash0Comid, "sr-new")
}

func TestSetSamplingRuleID_Nil(t *testing.T) {
	SetSamplingRuleID(nil, "sr-new") // should not panic
}

func TestSetSamplingRuleIDIfAbsent(t *testing.T) {
	rule := &SamplingDefinition{}
	SetSamplingRuleIDIfAbsent(rule, "sr-new")
	if rule.Metadata.Labels == nil {
		t.Fatal("expected the labels struct to be initialized")
	}
	assertPtrEqual(t, "Dash0Comid", rule.Metadata.Labels.Dash0Comid, "sr-new")
}

func TestSetSamplingRuleIDIfAbsent_NoOpWhenAlreadySet(t *testing.T) {
	rule := &SamplingDefinition{
		Metadata: SamplingMetadata{
			Labels: &SamplingLabels{Dash0Comid: Ptr("existing-id")},
		},
	}
	SetSamplingRuleIDIfAbsent(rule, "sr-new")
	assertPtrEqual(t, "Dash0Comid", rule.Metadata.Labels.Dash0Comid, "existing-id")
}

func TestSetSamplingRuleIDIfAbsent_Nil(t *testing.T) {
	SetSamplingRuleIDIfAbsent(nil, "sr-new") // should not panic
}

func TestClearSamplingRuleID(t *testing.T) {
	rule := &SamplingDefinition{
		Metadata: SamplingMetadata{
			Labels: &SamplingLabels{Dash0Comid: Ptr("sr-123")},
		},
	}
	ClearSamplingRuleID(rule)
	if rule.Metadata.Labels.Dash0Comid != nil {
		t.Error("Dash0Comid should be nil")
	}
}

func TestClearSamplingRuleID_NilLabels(t *testing.T) {
	ClearSamplingRuleID(&SamplingDefinition{}) // should not panic
}

func TestClearSamplingRuleID_Nil(t *testing.T) {
	ClearSamplingRuleID(nil) // should not panic
}
