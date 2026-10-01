package dash0

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newTestAgenticWorkflow builds an automation definition modeled on a real
// schedule-triggered Agent0 automation: a daily digest with a single
// scheduled trigger, a no-network sandbox, and a standing prompt.
func newTestAgenticWorkflow() *AgenticWorkflowDefinition {
	var trigger AgenticWorkflowTrigger
	if err := trigger.FromScheduledAgenticWorkflowTrigger(ScheduledAgenticWorkflowTrigger{
		Kind: "schedule",
		Spec: ScheduledAgenticWorkflowTriggerSpec{
			Cron: "0 9 * * 1-5",
		},
	}); err != nil {
		panic(err)
	}
	return &AgenticWorkflowDefinition{
		Kind: Dash0AgenticWorkflow,
		Metadata: AgenticWorkflowMetadata{
			Name: "daily-digest",
		},
		Spec: AgenticWorkflowSpec{
			Display: AgenticWorkflowDisplay{
				Name:        "Daily digest",
				Description: Ptr("Summarizes overnight incidents every weekday morning."),
			},
			Enabled: true,
			Prompt: AgenticWorkflowPrompt{
				User: "Summarize overnight incidents and post a digest.",
			},
			Sandbox: AgenticWorkflowSandbox{
				NetworkLevel: "no_network",
			},
			Triggers: []AgenticWorkflowTrigger{trigger},
		},
	}
}

func TestStripAgenticWorkflowServerFields(t *testing.T) {
	createdAt := time.Now()
	updatedAt := time.Now()
	deletedAt := time.Now()
	createdBy := "member_1"
	updatedBy := "member_2"
	version := "3"
	dataset := "default"
	origin := "my-origin"
	webhookID := "wh_1"
	webhookSecret := "shh"

	workflow := &AgenticWorkflowDefinition{
		Metadata: AgenticWorkflowMetadata{
			Annotations: &AgenticWorkflowAnnotations{
				Dash0ComcreatedAt:     &createdAt,
				Dash0ComcreatedBy:     &createdBy,
				Dash0ComdeletedAt:     &deletedAt,
				Dash0ComlastUpdatedAt: &updatedAt,
				Dash0ComlastUpdatedBy: &updatedBy,
				Dash0ComwebhookSecret: &webhookSecret,
			},
			Labels: &AgenticWorkflowLabels{
				Dash0Comversion:   &version,
				Dash0Comdataset:   &dataset,
				Dash0Comorigin:    &origin,
				Dash0Comsource:    Ptr(Api),
				Dash0Comid:        Ptr("agentic_workflow_01k5vpx97efdnrkqan15b41k84"),
				Dash0ComwebhookId: &webhookID,
			},
		},
		Spec: AgenticWorkflowSpec{
			PermittedActions: &[]AgenticWorkflowAction{AgenticWorkflowRead},
		},
	}

	StripAgenticWorkflowServerFields(workflow)

	if workflow.Metadata.Annotations.Dash0ComcreatedAt != nil {
		t.Error("Dash0ComcreatedAt should be nil")
	}
	if workflow.Metadata.Annotations.Dash0ComcreatedBy != nil {
		t.Error("Dash0ComcreatedBy should be nil")
	}
	if workflow.Metadata.Annotations.Dash0ComdeletedAt != nil {
		t.Error("Dash0ComdeletedAt should be nil")
	}
	if workflow.Metadata.Annotations.Dash0ComlastUpdatedAt != nil {
		t.Error("Dash0ComlastUpdatedAt should be nil")
	}
	if workflow.Metadata.Annotations.Dash0ComlastUpdatedBy != nil {
		t.Error("Dash0ComlastUpdatedBy should be nil")
	}
	if workflow.Metadata.Annotations.Dash0ComwebhookSecret != nil {
		t.Error("Dash0ComwebhookSecret should be nil")
	}
	if workflow.Metadata.Labels.Dash0Comversion != nil {
		t.Error("Dash0Comversion should be nil")
	}
	if workflow.Metadata.Labels.Dash0Comdataset != nil {
		t.Error("Dash0Comdataset should be nil")
	}
	if workflow.Metadata.Labels.Dash0Comorigin != nil {
		t.Error("Dash0Comorigin should be nil")
	}
	if workflow.Metadata.Labels.Dash0Comsource != nil {
		t.Error("Dash0Comsource should be nil")
	}
	if workflow.Metadata.Labels.Dash0ComwebhookId != nil {
		t.Error("Dash0ComwebhookId should be nil")
	}
	// Automation ids are server-assigned (`agentic_workflow_<ulid>`), not
	// client-settable, so the id is server-managed metadata and is stripped
	// — matching SLOs. Callers that need it read it via
	// GetAgenticWorkflowID before stripping.
	if workflow.Metadata.Labels.Dash0Comid != nil {
		t.Error("Dash0Comid should be nil")
	}
	if workflow.Spec.PermittedActions != nil {
		t.Error("PermittedActions should be nil")
	}
}

func TestStripAgenticWorkflowServerFields_Nil(t *testing.T) {
	StripAgenticWorkflowServerFields(nil) // should not panic
}

func TestStripAgenticWorkflowServerFields_NilLabels(t *testing.T) {
	workflow := &AgenticWorkflowDefinition{}
	StripAgenticWorkflowServerFields(workflow) // should not panic
	if workflow.Metadata.Labels != nil {
		t.Error("Labels should remain nil")
	}
	if workflow.Metadata.Annotations != nil {
		t.Error("Annotations should remain nil")
	}
}

func TestClearAgenticWorkflowID(t *testing.T) {
	workflow := &AgenticWorkflowDefinition{Metadata: AgenticWorkflowMetadata{Labels: &AgenticWorkflowLabels{Dash0Comid: Ptr("wf-1")}}}
	ClearAgenticWorkflowID(workflow)
	if workflow.Metadata.Labels.Dash0Comid != nil {
		t.Error("Dash0Comid should be nil")
	}
}

func TestClearAgenticWorkflowID_Nil(t *testing.T) {
	ClearAgenticWorkflowID(nil)                          // should not panic
	ClearAgenticWorkflowID(&AgenticWorkflowDefinition{}) // should not panic
}

func TestGetAgenticWorkflowDataset(t *testing.T) {
	tests := []struct {
		name     string
		workflow *AgenticWorkflowDefinition
		want     string
	}{
		{"nil workflow", nil, ""},
		{"nil labels", &AgenticWorkflowDefinition{}, ""},
		{"nil dataset", &AgenticWorkflowDefinition{Metadata: AgenticWorkflowMetadata{Labels: &AgenticWorkflowLabels{}}}, ""},
		{"with dataset", &AgenticWorkflowDefinition{Metadata: AgenticWorkflowMetadata{Labels: &AgenticWorkflowLabels{Dash0Comdataset: Ptr("default")}}}, "default"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetAgenticWorkflowDataset(tt.workflow); got != tt.want {
				t.Errorf("GetAgenticWorkflowDataset() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetAgenticWorkflowDataset(t *testing.T) {
	workflow := &AgenticWorkflowDefinition{}
	SetAgenticWorkflowDataset(workflow, "default")
	if workflow.Metadata.Labels == nil || workflow.Metadata.Labels.Dash0Comdataset == nil {
		t.Fatal("expected dataset to be set")
	}
	if *workflow.Metadata.Labels.Dash0Comdataset != "default" {
		t.Errorf("Dataset = %q, want %q", *workflow.Metadata.Labels.Dash0Comdataset, "default")
	}
}

func TestSetAgenticWorkflowDataset_Nil(t *testing.T) {
	SetAgenticWorkflowDataset(nil, "default") // should not panic
}

func TestGetAgenticWorkflowID(t *testing.T) {
	tests := []struct {
		name     string
		workflow *AgenticWorkflowDefinition
		want     string
	}{
		{
			"with ID",
			&AgenticWorkflowDefinition{Metadata: AgenticWorkflowMetadata{Labels: &AgenticWorkflowLabels{Dash0Comid: Ptr("agentic_workflow_01k5vpx97efdnrkqan15b41k84")}}},
			"agentic_workflow_01k5vpx97efdnrkqan15b41k84",
		},
		{"nil workflow", nil, ""},
		{"nil labels", &AgenticWorkflowDefinition{}, ""},
		{"nil ID", &AgenticWorkflowDefinition{Metadata: AgenticWorkflowMetadata{Labels: &AgenticWorkflowLabels{}}}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetAgenticWorkflowID(tt.workflow); got != tt.want {
				t.Errorf("GetAgenticWorkflowID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetAgenticWorkflowID(t *testing.T) {
	workflow := &AgenticWorkflowDefinition{}
	SetAgenticWorkflowID(workflow, "new-id")
	if workflow.Metadata.Labels == nil || workflow.Metadata.Labels.Dash0Comid == nil {
		t.Fatal("expected ID to be set")
	}
	if *workflow.Metadata.Labels.Dash0Comid != "new-id" {
		t.Errorf("ID = %q, want %q", *workflow.Metadata.Labels.Dash0Comid, "new-id")
	}
}

func TestSetAgenticWorkflowID_Overwrites(t *testing.T) {
	workflow := &AgenticWorkflowDefinition{
		Metadata: AgenticWorkflowMetadata{Labels: &AgenticWorkflowLabels{Dash0Comid: Ptr("existing-id")}},
	}
	SetAgenticWorkflowID(workflow, "new-id")
	if *workflow.Metadata.Labels.Dash0Comid != "new-id" {
		t.Errorf("ID = %q, want %q", *workflow.Metadata.Labels.Dash0Comid, "new-id")
	}
}

func TestSetAgenticWorkflowID_Nil(t *testing.T) {
	SetAgenticWorkflowID(nil, "new-id") // should not panic
}

func TestSetAgenticWorkflowIDIfAbsent(t *testing.T) {
	workflow := &AgenticWorkflowDefinition{}
	SetAgenticWorkflowIDIfAbsent(workflow, "new-id")
	if workflow.Metadata.Labels == nil || workflow.Metadata.Labels.Dash0Comid == nil {
		t.Fatal("expected ID to be set")
	}
	if *workflow.Metadata.Labels.Dash0Comid != "new-id" {
		t.Errorf("ID = %q, want %q", *workflow.Metadata.Labels.Dash0Comid, "new-id")
	}
}

func TestSetAgenticWorkflowIDIfAbsent_NoOpWhenAlreadySet(t *testing.T) {
	workflow := &AgenticWorkflowDefinition{
		Metadata: AgenticWorkflowMetadata{Labels: &AgenticWorkflowLabels{Dash0Comid: Ptr("existing-id")}},
	}
	SetAgenticWorkflowIDIfAbsent(workflow, "new-id")
	if *workflow.Metadata.Labels.Dash0Comid != "existing-id" {
		t.Errorf("ID = %q, want %q (should not overwrite)", *workflow.Metadata.Labels.Dash0Comid, "existing-id")
	}
}

func TestSetAgenticWorkflowIDIfAbsent_Nil(t *testing.T) {
	SetAgenticWorkflowIDIfAbsent(nil, "new-id") // should not panic
}

func TestGetAgenticWorkflowOrigin(t *testing.T) {
	tests := []struct {
		name     string
		workflow *AgenticWorkflowDefinition
		want     string
	}{
		{
			"with origin",
			&AgenticWorkflowDefinition{Metadata: AgenticWorkflowMetadata{Labels: &AgenticWorkflowLabels{Dash0Comorigin: Ptr("dash0-cli")}}},
			"dash0-cli",
		},
		{"nil workflow", nil, ""},
		{"nil labels", &AgenticWorkflowDefinition{}, ""},
		{"nil origin", &AgenticWorkflowDefinition{Metadata: AgenticWorkflowMetadata{Labels: &AgenticWorkflowLabels{}}}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetAgenticWorkflowOrigin(tt.workflow); got != tt.want {
				t.Errorf("GetAgenticWorkflowOrigin() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetAgenticWorkflowName(t *testing.T) {
	tests := []struct {
		name     string
		workflow *AgenticWorkflowDefinition
		want     string
	}{
		{
			"from spec.display.name",
			&AgenticWorkflowDefinition{Spec: AgenticWorkflowSpec{Display: AgenticWorkflowDisplay{Name: "Daily digest"}}},
			"Daily digest",
		},
		{"nil workflow", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetAgenticWorkflowName(tt.workflow); got != tt.want {
				t.Errorf("GetAgenticWorkflowName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestListAgenticWorkflows(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/agentic-workflows" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("dataset"); got != "default" {
			t.Errorf("dataset query = %q, want default", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(GetAgenticWorkflowsResponse{
			AgenticWorkflows: []AgenticWorkflowDefinition{*newTestAgenticWorkflow()},
		})
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	workflows, err := client.ListAgenticWorkflows(context.Background(), Ptr("default"))
	if err != nil {
		t.Fatalf("ListAgenticWorkflows failed: %v", err)
	}
	if len(workflows) != 1 {
		t.Fatalf("expected 1 automation, got %d", len(workflows))
	}
	if GetAgenticWorkflowName(workflows[0]) != "Daily digest" {
		t.Errorf("expected name %q, got %q", "Daily digest", GetAgenticWorkflowName(workflows[0]))
	}
}

func TestListAgenticWorkflows_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	if _, err := client.ListAgenticWorkflows(context.Background(), nil); err == nil {
		t.Fatal("expected error for 403 response")
	}
}

func TestGetAgenticWorkflow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/agentic-workflows/agentic_workflow_01k5vpx97efdnrkqan15b41k84" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		workflow := newTestAgenticWorkflow()
		SetAgenticWorkflowID(workflow, "agentic_workflow_01k5vpx97efdnrkqan15b41k84")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(workflow)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	workflow, err := client.GetAgenticWorkflow(context.Background(), "agentic_workflow_01k5vpx97efdnrkqan15b41k84", Ptr("default"))
	if err != nil {
		t.Fatalf("GetAgenticWorkflow failed: %v", err)
	}
	if GetAgenticWorkflowID(workflow) != "agentic_workflow_01k5vpx97efdnrkqan15b41k84" {
		t.Errorf("unexpected ID: %q", GetAgenticWorkflowID(workflow))
	}
}

func TestGetAgenticWorkflow_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	if _, err := client.GetAgenticWorkflow(context.Background(), "missing", nil); err == nil {
		t.Fatal("expected error for 404 response")
	}
}

func TestCreateAgenticWorkflow_200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		workflow := newTestAgenticWorkflow()
		SetAgenticWorkflowID(workflow, "agentic_workflow_01k5vpx97efdnrkqan15b41k84")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(workflow)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	created, err := client.CreateAgenticWorkflow(context.Background(), newTestAgenticWorkflow(), Ptr("default"))
	if err != nil {
		t.Fatalf("CreateAgenticWorkflow failed: %v", err)
	}
	if GetAgenticWorkflowID(created) != "agentic_workflow_01k5vpx97efdnrkqan15b41k84" {
		t.Errorf("unexpected ID: %q", GetAgenticWorkflowID(created))
	}
	if GetAgenticWorkflowName(created) != "Daily digest" {
		t.Errorf("unexpected name: %q", GetAgenticWorkflowName(created))
	}
}

// TestCreateAgenticWorkflow_201 covers the 201 status code with a body that
// is parsed from the raw response (JSON200 is nil on non-200 status codes).
func TestCreateAgenticWorkflow_201(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		workflow := newTestAgenticWorkflow()
		SetAgenticWorkflowID(workflow, "agentic_workflow_01k5vpx97efdnrkqan15b41k84")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(workflow)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	created, err := client.CreateAgenticWorkflow(context.Background(), newTestAgenticWorkflow(), nil)
	if err != nil {
		t.Fatalf("CreateAgenticWorkflow failed: %v", err)
	}
	if GetAgenticWorkflowID(created) != "agentic_workflow_01k5vpx97efdnrkqan15b41k84" {
		t.Errorf("unexpected ID: %q", GetAgenticWorkflowID(created))
	}
}

func TestCreateAgenticWorkflow_Nil(t *testing.T) {
	client := newTestClient(t, "http://example.invalid")
	if _, err := client.CreateAgenticWorkflow(context.Background(), nil, nil); err == nil {
		t.Fatal("expected error for nil automation")
	}
}

func TestCreateAgenticWorkflow_Forbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	if _, err := client.CreateAgenticWorkflow(context.Background(), newTestAgenticWorkflow(), nil); err == nil {
		t.Fatal("expected error for 403 response")
	}
}

func TestUpdateAgenticWorkflow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/agentic-workflows/agentic_workflow_01k5vpx97efdnrkqan15b41k84" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		updated := newTestAgenticWorkflow()
		updated.Spec.Enabled = false
		SetAgenticWorkflowID(updated, "agentic_workflow_01k5vpx97efdnrkqan15b41k84")
		updated.Metadata.Labels.Dash0Comversion = Ptr("2")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(updated)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	in := newTestAgenticWorkflow()
	SetAgenticWorkflowID(in, "agentic_workflow_01k5vpx97efdnrkqan15b41k84")
	updated, err := client.UpdateAgenticWorkflow(context.Background(), "agentic_workflow_01k5vpx97efdnrkqan15b41k84", in, Ptr("default"))
	if err != nil {
		t.Fatalf("UpdateAgenticWorkflow failed: %v", err)
	}
	if updated.Spec.Enabled {
		t.Error("expected Enabled to be false after update")
	}
}

func TestUpdateAgenticWorkflow_Nil(t *testing.T) {
	client := newTestClient(t, "http://example.invalid")
	if _, err := client.UpdateAgenticWorkflow(context.Background(), "id", nil, nil); err == nil {
		t.Fatal("expected error for nil automation")
	}
}

func TestUpdateAgenticWorkflow_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	if _, err := client.UpdateAgenticWorkflow(context.Background(), "missing", newTestAgenticWorkflow(), nil); err == nil {
		t.Fatal("expected error for 404 response")
	}
}

func TestDeleteAgenticWorkflow_200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("unexpected method: %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	if err := client.DeleteAgenticWorkflow(context.Background(), "agentic_workflow_01k5vpx97efdnrkqan15b41k84", Ptr("default")); err != nil {
		t.Fatalf("DeleteAgenticWorkflow failed: %v", err)
	}
}

func TestDeleteAgenticWorkflow_204(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	if err := client.DeleteAgenticWorkflow(context.Background(), "id", nil); err != nil {
		t.Fatalf("DeleteAgenticWorkflow failed: %v", err)
	}
}

func TestDeleteAgenticWorkflow_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	if err := client.DeleteAgenticWorkflow(context.Background(), "missing", nil); err == nil {
		t.Fatal("expected error for 404 response")
	}
}

func TestListAgenticWorkflowsIter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(GetAgenticWorkflowsResponse{
			AgenticWorkflows: []AgenticWorkflowDefinition{*newTestAgenticWorkflow()},
		})
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	iter := client.ListAgenticWorkflowsIter(context.Background(), Ptr("default"))
	count := 0
	for iter.Next() {
		count++
	}
	if err := iter.Err(); err != nil {
		t.Fatalf("iterator error: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 automation, got %d", count)
	}
}
