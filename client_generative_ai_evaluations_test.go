package dash0

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

const testGenerativeAiEvaluationID = "eval_01k9z3d4q5r6s7t8u9v0w1x2y3"

// newTestGenerativeAiEvaluation builds a boolean-judge evaluation as a caller would write it.
func newTestGenerativeAiEvaluation(t *testing.T) *GenerativeAiEvaluationDefinition {
	t.Helper()
	var evaluation GenerativeAiEvaluationDefinition
	err := json.Unmarshal([]byte(`{
		"kind": "Dash0GenerativeAiEvaluation",
		"metadata": {
			"name": "answer-relevance",
			"labels": {"dash0.com/origin": "tf_answer-relevance"},
			"annotations": {"dash0.com/folder-path": "/checkout"}
		},
		"spec": {
			"display": {"name": "Answer relevance"},
			"enabled": true,
			"judge": {
				"model": "standard",
				"output": {"kind": "boolean"},
				"systemPrompt": "You judge whether an answer is relevant.",
				"userPrompt": "Is the answer relevant to the question?"
			},
			"match": {"filters": [{"key": "gen_ai.operation.name", "operator": "is_set"}]},
			"sample": {"rate": 0.25}
		}
	}`), &evaluation)
	if err != nil {
		t.Fatalf("failed to build test evaluation: %v", err)
	}
	return &evaluation
}

func TestStripGenerativeAiEvaluationServerFields(t *testing.T) {
	now := time.Now()
	evaluation := &GenerativeAiEvaluationDefinition{
		Metadata: GenerativeAiEvaluationMetadata{
			Name: "answer-relevance",
			Annotations: &GenerativeAiEvaluationGroupAnnotations{
				Dash0ComcreatedAt:  &now,
				Dash0ComcreatedBy:  Ptr("user_1"),
				Dash0ComupdatedAt:  &now,
				Dash0ComdeletedAt:  &now,
				Dash0ComfolderPath: Ptr("/checkout"),
				Dash0Comsharing:    Ptr("team:team_01abc"),
			},
			Labels: &GenerativeAiEvaluationLabels{
				Dash0Comid:      Ptr(testGenerativeAiEvaluationID),
				Dash0Comversion: Ptr("3"),
				Dash0Comsource:  Ptr(Terraform),
				Dash0Comdataset: Ptr("default"),
				Dash0Comorigin:  Ptr("tf_answer-relevance"),
			},
		},
	}

	StripGenerativeAiEvaluationServerFields(evaluation)

	annotations := evaluation.Metadata.Annotations
	if annotations.Dash0ComcreatedAt != nil || annotations.Dash0ComcreatedBy != nil || annotations.Dash0ComupdatedAt != nil || annotations.Dash0ComdeletedAt != nil {
		t.Errorf("server-set annotations survived the strip: %+v", annotations)
	}
	labels := evaluation.Metadata.Labels
	if labels.Dash0Comid != nil {
		t.Error("Dash0Comid should be nil")
	}
	// The version must go, or a later write fails with 400 once the evaluation changes.
	if labels.Dash0Comversion != nil {
		t.Error("Dash0Comversion should be nil")
	}
	if labels.Dash0Comsource != nil {
		t.Error("Dash0Comsource should be nil")
	}
	if labels.Dash0Comdataset != nil {
		t.Error("Dash0Comdataset should be nil")
	}
	// The caller sets these, so they must survive.
	assertPtrEqual(t, "origin", labels.Dash0Comorigin, "tf_answer-relevance")
	assertPtrEqual(t, "folder path", annotations.Dash0ComfolderPath, "/checkout")
	assertPtrEqual(t, "sharing", annotations.Dash0Comsharing, "team:team_01abc")
	assertEqual(t, "name", evaluation.Metadata.Name, "answer-relevance")
}

func TestStripGenerativeAiEvaluationServerFields_Nil(t *testing.T) {
	StripGenerativeAiEvaluationServerFields(nil) // should not panic
	evaluation := &GenerativeAiEvaluationDefinition{}
	StripGenerativeAiEvaluationServerFields(evaluation) // should not panic
	if evaluation.Metadata.Labels != nil || evaluation.Metadata.Annotations != nil {
		t.Error("strip should not create labels or annotations")
	}
}

func TestGenerativeAiEvaluationIDHelpers(t *testing.T) {
	evaluation := &GenerativeAiEvaluationDefinition{}
	SetGenerativeAiEvaluationIDIfAbsent(evaluation, "first")
	SetGenerativeAiEvaluationIDIfAbsent(evaluation, "second")
	assertEqual(t, "ID after IfAbsent", GetGenerativeAiEvaluationID(evaluation), "first")
	SetGenerativeAiEvaluationID(evaluation, "third")
	assertEqual(t, "ID after Set", GetGenerativeAiEvaluationID(evaluation), "third")
	ClearGenerativeAiEvaluationID(evaluation)
	assertEqual(t, "ID after Clear", GetGenerativeAiEvaluationID(evaluation), "")
}

func TestGenerativeAiEvaluationHelpers_Nil(t *testing.T) {
	// None of these may panic on a nil evaluation or on one without labels.
	for _, evaluation := range []*GenerativeAiEvaluationDefinition{nil, {}} {
		assertEqual(t, "ID", GetGenerativeAiEvaluationID(evaluation), "")
		assertEqual(t, "name", GetGenerativeAiEvaluationName(evaluation), "")
		assertEqual(t, "origin", GetGenerativeAiEvaluationOrigin(evaluation), "")
		assertEqual(t, "dataset", GetGenerativeAiEvaluationDataset(evaluation), "")
		ClearGenerativeAiEvaluationID(evaluation)
	}
	SetGenerativeAiEvaluationID(nil, "id")
	SetGenerativeAiEvaluationIDIfAbsent(nil, "id")
	SetGenerativeAiEvaluationDataset(nil, "default")
}

func TestGetGenerativeAiEvaluationName(t *testing.T) {
	evaluation := &GenerativeAiEvaluationDefinition{Metadata: GenerativeAiEvaluationMetadata{Name: "answer-relevance"}}
	assertEqual(t, "fallback name", GetGenerativeAiEvaluationName(evaluation), "answer-relevance")
	evaluation.Spec.Display.Name = "Answer relevance"
	assertEqual(t, "display name", GetGenerativeAiEvaluationName(evaluation), "Answer relevance")
}

func TestGenerativeAiEvaluationDatasetHelpers(t *testing.T) {
	evaluation := &GenerativeAiEvaluationDefinition{}
	SetGenerativeAiEvaluationDataset(evaluation, "production")
	assertEqual(t, "dataset", GetGenerativeAiEvaluationDataset(evaluation), "production")
}

func TestListGenerativeAiEvaluations(t *testing.T) {
	evaluation := newTestGenerativeAiEvaluation(t)
	var gotQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/generative-ai-evaluations" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(GenerativeAiEvaluationListResponse{
			GenerativeAiEvaluations: []GenerativeAiEvaluationResponse{*evaluation},
		})
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	evaluations, err := c.ListGenerativeAiEvaluations(context.Background(), Ptr("production"), WithOriginPrefix("tf_"))
	if err != nil {
		t.Fatalf("ListGenerativeAiEvaluations failed: %v", err)
	}
	assertEqual(t, "dataset", gotQuery.Get("dataset"), "production")
	assertEqual(t, "originPrefix", gotQuery.Get("originPrefix"), "tf_")
	if len(evaluations) != 1 {
		t.Fatalf("expected 1 evaluation, got %d", len(evaluations))
	}
	assertEqual(t, "name", GetGenerativeAiEvaluationName(evaluations[0]), "Answer relevance")

	iter := c.ListGenerativeAiEvaluationsIter(context.Background(), nil)
	count := 0
	for iter.Next() {
		count++
	}
	if err := iter.Err(); err != nil || count != 1 {
		t.Errorf("iterator yielded %d evaluations with error %v, want 1 and no error", count, err)
	}
	if _, set := gotQuery["originPrefix"]; set {
		t.Error("originPrefix sent without WithOriginPrefix")
	}
}

func TestListGenerativeAiEvaluations_Forbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeAPIError(w, http.StatusForbidden, "forbidden")
	}))
	defer server.Close()

	_, err := newTestClient(t, server.URL).ListGenerativeAiEvaluations(context.Background(), nil)
	if !IsForbidden(err) {
		t.Errorf("expected a forbidden error, got %v", err)
	}
}

func TestListGenerativeAiEvaluations_RejectsUnsupportedFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("no request should reach the server")
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	if _, err := c.ListGenerativeAiEvaluations(context.Background(), nil, WithEnabled(true)); err == nil {
		t.Error("ListGenerativeAiEvaluations with the enabled filter succeeded, want an error")
	}
	if err := c.ListGenerativeAiEvaluationsIter(context.Background(), nil, WithEnabled(true)).Err(); err == nil {
		t.Error("ListGenerativeAiEvaluationsIter with the enabled filter succeeded, want an error")
	}
}

func TestGetGenerativeAiEvaluation_Deleted(t *testing.T) {
	// The API returns a deleted evaluation with 200 and dash0.com/deleted-at set.
	deleted := newTestGenerativeAiEvaluation(t)
	deletedAt := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	deleted.Metadata.Annotations.Dash0ComdeletedAt = &deletedAt
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/generative-ai-evaluations/tf_answer-relevance" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("dataset"); got != "production" {
			t.Errorf("dataset query = %q, want production", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(deleted)
	}))
	defer server.Close()

	evaluation, err := newTestClient(t, server.URL).GetGenerativeAiEvaluation(context.Background(), "tf_answer-relevance", Ptr("production"))
	if err != nil {
		t.Fatalf("GetGenerativeAiEvaluation failed: %v", err)
	}
	if evaluation.Metadata.Annotations.Dash0ComdeletedAt == nil {
		t.Error("expected dash0.com/deleted-at to be set")
	}
}

func TestGetGenerativeAiEvaluation_NotJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	evaluation, err := newTestClient(t, server.URL).GetGenerativeAiEvaluation(context.Background(), "tf_answer-relevance", nil)
	if err == nil {
		t.Errorf("expected an error for a 200 response that is not JSON, got %+v", evaluation)
	}
}

func TestGetGenerativeAiEvaluation_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeAPIError(w, http.StatusNotFound, "not found")
	}))
	defer server.Close()

	_, err := newTestClient(t, server.URL).GetGenerativeAiEvaluation(context.Background(), "missing", nil)
	if !IsNotFound(err) {
		t.Errorf("expected a not-found error, got %v", err)
	}
}

func TestCreateGenerativeAiEvaluation(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusCreated} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/api/generative-ai-evaluations" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				if got := r.URL.Query().Get("dataset"); got != "production" {
					t.Errorf("dataset query = %q, want production", got)
				}
				var sent GenerativeAiEvaluationDefinition
				if err := json.NewDecoder(r.Body).Decode(&sent); err != nil {
					t.Errorf("failed to decode request body: %v", err)
					return
				}
				if GetGenerativeAiEvaluationOrigin(&sent) != "tf_answer-relevance" {
					t.Errorf("origin = %q, want tf_answer-relevance", GetGenerativeAiEvaluationOrigin(&sent))
				}
				SetGenerativeAiEvaluationID(&sent, testGenerativeAiEvaluationID)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				_ = json.NewEncoder(w).Encode(sent)
			}))
			defer server.Close()

			created, err := newTestClient(t, server.URL).CreateGenerativeAiEvaluation(context.Background(), newTestGenerativeAiEvaluation(t), Ptr("production"))
			if err != nil {
				t.Fatalf("CreateGenerativeAiEvaluation failed: %v", err)
			}
			assertEqual(t, "ID", GetGenerativeAiEvaluationID(created), testGenerativeAiEvaluationID)
		})
	}
}

func TestCreateGenerativeAiEvaluation_Errors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeAPIError(w, http.StatusBadRequest, "duplicate origin")
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	_, err := c.CreateGenerativeAiEvaluation(context.Background(), newTestGenerativeAiEvaluation(t), nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("expected a 400 APIError, got %v", err)
	}
	if _, err := c.CreateGenerativeAiEvaluation(context.Background(), nil, nil); err == nil {
		t.Error("expected an error for a nil evaluation")
	}
}

func TestUpdateGenerativeAiEvaluation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/generative-ai-evaluations/tf_answer-relevance" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("dataset"); got != "production" {
			t.Errorf("dataset query = %q, want production", got)
		}
		body, _ := io.ReadAll(r.Body)
		var sent GenerativeAiEvaluationDefinition
		if err := json.Unmarshal(body, &sent); err != nil {
			t.Errorf("failed to decode request body: %v", err)
			return
		}
		if sent.Metadata.Labels == nil {
			t.Errorf("request body has no labels")
			return
		}
		if sent.Metadata.Labels.Dash0Comversion == nil || *sent.Metadata.Labels.Dash0Comversion != "2" {
			t.Errorf("version = %v, want 2", sent.Metadata.Labels.Dash0Comversion)
		}
		sent.Metadata.Labels.Dash0Comversion = Ptr("3")
		sent.Spec.Sample.Rate = 0.5
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(sent)
	}))
	defer server.Close()

	in := newTestGenerativeAiEvaluation(t)
	in.Metadata.Labels.Dash0Comversion = Ptr("2")
	updated, err := newTestClient(t, server.URL).UpdateGenerativeAiEvaluation(context.Background(), "tf_answer-relevance", in, Ptr("production"))
	if err != nil {
		t.Fatalf("UpdateGenerativeAiEvaluation failed: %v", err)
	}
	assertPtrEqual(t, "version", updated.Metadata.Labels.Dash0Comversion, "3")
	if updated.Spec.Sample.Rate != 0.5 {
		t.Errorf("sample rate = %v, want 0.5", updated.Spec.Sample.Rate)
	}
}

func TestUpdateGenerativeAiEvaluation_StaleVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeAPIError(w, http.StatusBadRequest, "version conflict")
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	_, err := c.UpdateGenerativeAiEvaluation(context.Background(), "tf_answer-relevance", newTestGenerativeAiEvaluation(t), nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("expected a 400 APIError, got %v", err)
	}
	if _, err := c.UpdateGenerativeAiEvaluation(context.Background(), "tf_answer-relevance", nil, nil); err == nil {
		t.Error("expected an error for a nil evaluation")
	}
}

func TestDeleteGenerativeAiEvaluation(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusNoContent} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodDelete || r.URL.Path != "/api/generative-ai-evaluations/"+testGenerativeAiEvaluationID {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				if got := r.URL.Query().Get("dataset"); got != "production" {
					t.Errorf("dataset query = %q, want production", got)
				}
				w.WriteHeader(status)
			}))
			defer server.Close()

			if err := newTestClient(t, server.URL).DeleteGenerativeAiEvaluation(context.Background(), testGenerativeAiEvaluationID, Ptr("production")); err != nil {
				t.Fatalf("DeleteGenerativeAiEvaluation failed: %v", err)
			}
		})
	}
}

func TestDeleteGenerativeAiEvaluation_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeAPIError(w, http.StatusNotFound, "not found")
	}))
	defer server.Close()

	err := newTestClient(t, server.URL).DeleteGenerativeAiEvaluation(context.Background(), "missing", nil)
	if !IsNotFound(err) {
		t.Errorf("expected a not-found error, got %v", err)
	}
}
