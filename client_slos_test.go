package dash0

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newTestSLO builds an SLO definition based on the shared create.yaml fixture:
// an OpenSLO v1 document for a checkout-availability SLO.
func newTestSLO() *SloDefinition {
	return &SloDefinition{
		ApiVersion: "openslo/v1",
		Kind:       "SLO",
		Metadata: SloMetadata{
			Name: "checkout-availability",
			Annotations: &SloAnnotations{
				Dash0Comenabled: Ptr("true"),
				AdditionalProperties: map[string]string{
					"dash0.com/display-name": "Checkout availability",
				},
			},
		},
		Spec: SloSpec{
			Description:     Ptr("99 percent of checkout HTTP requests succeed over a rolling 28-day window."),
			Service:         Ptr("checkout"),
			BudgetingMethod: "Occurrences",
			Objectives: []SloObjective{
				{DisplayName: Ptr("99% availability"), Target: Ptr(float32(0.99))},
			},
		},
	}
}

func TestStripSLOServerFields(t *testing.T) {
	createdAt := time.Now()
	updatedAt := time.Now()
	deletedAt := time.Now()
	version := "2"
	dataset := "default"
	origin := "my-origin"

	slo := &SloDefinition{
		Metadata: SloMetadata{
			Annotations: &SloAnnotations{
				Dash0ComcreatedAt: &createdAt,
				Dash0ComupdatedAt: &updatedAt,
				Dash0ComdeletedAt: &deletedAt,
			},
			Labels: &SloLabels{
				Dash0Comversion: &version,
				Dash0Comdataset: &dataset,
				Dash0Comorigin:  &origin,
				Dash0Comsource:  Ptr(Api),
				Dash0Comid:      Ptr("keep-this"),
			},
		},
	}

	StripSLOServerFields(slo)

	if slo.Metadata.Annotations.Dash0ComcreatedAt != nil {
		t.Error("Dash0ComcreatedAt should be nil")
	}
	if slo.Metadata.Annotations.Dash0ComupdatedAt != nil {
		t.Error("Dash0ComupdatedAt should be nil")
	}
	if slo.Metadata.Annotations.Dash0ComdeletedAt != nil {
		t.Error("Dash0ComdeletedAt should be nil")
	}
	if slo.Metadata.Labels.Dash0Comversion != nil {
		t.Error("Dash0Comversion should be nil")
	}
	if slo.Metadata.Labels.Dash0Comdataset != nil {
		t.Error("Dash0Comdataset should be nil")
	}
	if slo.Metadata.Labels.Dash0Comorigin != nil {
		t.Error("Dash0Comorigin should be nil")
	}
	if slo.Metadata.Labels.Dash0Comsource != nil {
		t.Error("Dash0Comsource should be nil")
	}
	if slo.Metadata.Labels.Dash0Comid == nil || *slo.Metadata.Labels.Dash0Comid != "keep-this" {
		t.Error("Dash0Comid should be preserved")
	}
}

func TestStripSLOServerFields_Nil(t *testing.T) {
	StripSLOServerFields(nil) // should not panic
}

func TestStripSLOServerFields_NilLabels(t *testing.T) {
	slo := &SloDefinition{}
	StripSLOServerFields(slo) // should not panic
	if slo.Metadata.Labels != nil {
		t.Error("Labels should remain nil")
	}
	if slo.Metadata.Annotations != nil {
		t.Error("Annotations should remain nil")
	}
}

func TestClearSLOID(t *testing.T) {
	slo := &SloDefinition{Metadata: SloMetadata{Labels: &SloLabels{Dash0Comid: Ptr("slo-1")}}}
	ClearSLOID(slo)
	if slo.Metadata.Labels.Dash0Comid != nil {
		t.Error("Dash0Comid should be nil")
	}
}

func TestClearSLOID_Nil(t *testing.T) {
	ClearSLOID(nil)              // should not panic
	ClearSLOID(&SloDefinition{}) // should not panic
}

func TestGetSLODataset(t *testing.T) {
	tests := []struct {
		name string
		slo  *SloDefinition
		want string
	}{
		{"nil slo", nil, ""},
		{"nil labels", &SloDefinition{}, ""},
		{"nil dataset", &SloDefinition{Metadata: SloMetadata{Labels: &SloLabels{}}}, ""},
		{"with dataset", &SloDefinition{Metadata: SloMetadata{Labels: &SloLabels{Dash0Comdataset: Ptr("default")}}}, "default"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetSLODataset(tt.slo); got != tt.want {
				t.Errorf("GetSLODataset() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetSLODataset(t *testing.T) {
	slo := &SloDefinition{}
	SetSLODataset(slo, "default")
	if slo.Metadata.Labels == nil || slo.Metadata.Labels.Dash0Comdataset == nil {
		t.Fatal("expected dataset to be set")
	}
	if *slo.Metadata.Labels.Dash0Comdataset != "default" {
		t.Errorf("Dataset = %q, want %q", *slo.Metadata.Labels.Dash0Comdataset, "default")
	}
}

func TestSetSLODataset_Nil(t *testing.T) {
	SetSLODataset(nil, "default") // should not panic
}

func TestGetSLOID(t *testing.T) {
	tests := []struct {
		name string
		slo  *SloDefinition
		want string
	}{
		{
			"with ID",
			&SloDefinition{Metadata: SloMetadata{Labels: &SloLabels{Dash0Comid: Ptr("00000000-0000-0000-0000-000000000001")}}},
			"00000000-0000-0000-0000-000000000001",
		},
		{"nil slo", nil, ""},
		{"nil labels", &SloDefinition{}, ""},
		{"nil ID", &SloDefinition{Metadata: SloMetadata{Labels: &SloLabels{}}}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetSLOID(tt.slo); got != tt.want {
				t.Errorf("GetSLOID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetSLOID(t *testing.T) {
	slo := &SloDefinition{}
	SetSLOID(slo, "new-id")
	if slo.Metadata.Labels == nil || slo.Metadata.Labels.Dash0Comid == nil {
		t.Fatal("expected ID to be set")
	}
	if *slo.Metadata.Labels.Dash0Comid != "new-id" {
		t.Errorf("ID = %q, want %q", *slo.Metadata.Labels.Dash0Comid, "new-id")
	}
}

func TestSetSLOID_Overwrites(t *testing.T) {
	slo := &SloDefinition{
		Metadata: SloMetadata{Labels: &SloLabels{Dash0Comid: Ptr("existing-id")}},
	}
	SetSLOID(slo, "new-id")
	if *slo.Metadata.Labels.Dash0Comid != "new-id" {
		t.Errorf("ID = %q, want %q", *slo.Metadata.Labels.Dash0Comid, "new-id")
	}
}

func TestSetSLOID_Nil(t *testing.T) {
	SetSLOID(nil, "new-id") // should not panic
}

func TestSetSLOIDIfAbsent(t *testing.T) {
	slo := &SloDefinition{}
	SetSLOIDIfAbsent(slo, "new-id")
	if slo.Metadata.Labels == nil || slo.Metadata.Labels.Dash0Comid == nil {
		t.Fatal("expected ID to be set")
	}
	if *slo.Metadata.Labels.Dash0Comid != "new-id" {
		t.Errorf("ID = %q, want %q", *slo.Metadata.Labels.Dash0Comid, "new-id")
	}
}

func TestSetSLOIDIfAbsent_NoOpWhenAlreadySet(t *testing.T) {
	slo := &SloDefinition{
		Metadata: SloMetadata{Labels: &SloLabels{Dash0Comid: Ptr("existing-id")}},
	}
	SetSLOIDIfAbsent(slo, "new-id")
	if *slo.Metadata.Labels.Dash0Comid != "existing-id" {
		t.Errorf("ID = %q, want %q (should not overwrite)", *slo.Metadata.Labels.Dash0Comid, "existing-id")
	}
}

func TestSetSLOIDIfAbsent_Nil(t *testing.T) {
	SetSLOIDIfAbsent(nil, "new-id") // should not panic
}

func TestGetSLOName(t *testing.T) {
	tests := []struct {
		name string
		slo  *SloDefinition
		want string
	}{
		{
			"from display-name annotation",
			&SloDefinition{Metadata: SloMetadata{
				Name:        "checkout-availability",
				Annotations: &SloAnnotations{AdditionalProperties: map[string]string{"dash0.com/display-name": "Checkout availability"}},
			}},
			"Checkout availability",
		},
		{
			"falls back to metadata name when annotation empty",
			&SloDefinition{Metadata: SloMetadata{
				Name:        "checkout-availability",
				Annotations: &SloAnnotations{AdditionalProperties: map[string]string{"dash0.com/display-name": ""}},
			}},
			"checkout-availability",
		},
		{
			"falls back to metadata name when no annotations",
			&SloDefinition{Metadata: SloMetadata{Name: "checkout-availability"}},
			"checkout-availability",
		},
		{"nil slo", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetSLOName(tt.slo); got != tt.want {
				t.Errorf("GetSLOName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestListSLOs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/slos" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("dataset"); got != "default" {
			t.Errorf("dataset query = %q, want default", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]SloDefinition{*newTestSLO()})
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	slos, err := client.ListSLOs(context.Background(), Ptr("default"))
	if err != nil {
		t.Fatalf("ListSLOs failed: %v", err)
	}
	if len(slos) != 1 {
		t.Fatalf("expected 1 SLO, got %d", len(slos))
	}
	if GetSLOName(slos[0]) != "Checkout availability" {
		t.Errorf("expected name %q, got %q", "Checkout availability", GetSLOName(slos[0]))
	}
}

func TestListSLOs_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	if _, err := client.ListSLOs(context.Background(), nil); err == nil {
		t.Fatal("expected error for 403 response")
	}
}

func TestGetSLO(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/slos/00000000-0000-0000-0000-000000000001" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		slo := newTestSLO()
		SetSLOID(slo, "00000000-0000-0000-0000-000000000001")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(slo)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	slo, err := client.GetSLO(context.Background(), "00000000-0000-0000-0000-000000000001", Ptr("default"))
	if err != nil {
		t.Fatalf("GetSLO failed: %v", err)
	}
	if GetSLOID(slo) != "00000000-0000-0000-0000-000000000001" {
		t.Errorf("unexpected ID: %q", GetSLOID(slo))
	}
}

func TestGetSLO_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	if _, err := client.GetSLO(context.Background(), "missing", nil); err == nil {
		t.Fatal("expected error for 404 response")
	}
}

func TestCreateSLO_200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		slo := newTestSLO()
		SetSLOID(slo, "00000000-0000-0000-0000-000000000001")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(slo)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	created, err := client.CreateSLO(context.Background(), newTestSLO(), Ptr("default"))
	if err != nil {
		t.Fatalf("CreateSLO failed: %v", err)
	}
	if GetSLOID(created) != "00000000-0000-0000-0000-000000000001" {
		t.Errorf("unexpected ID: %q", GetSLOID(created))
	}
	if GetSLOName(created) != "Checkout availability" {
		t.Errorf("unexpected name: %q", GetSLOName(created))
	}
}

// TestCreateSLO_201 covers the 201 status code with a body that is parsed from
// the raw response (JSON200 is nil on non-200 status codes).
func TestCreateSLO_201(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		slo := newTestSLO()
		SetSLOID(slo, "00000000-0000-0000-0000-000000000001")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(slo)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	created, err := client.CreateSLO(context.Background(), newTestSLO(), nil)
	if err != nil {
		t.Fatalf("CreateSLO failed: %v", err)
	}
	if GetSLOID(created) != "00000000-0000-0000-0000-000000000001" {
		t.Errorf("unexpected ID: %q", GetSLOID(created))
	}
}

func TestCreateSLO_BadRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	if _, err := client.CreateSLO(context.Background(), newTestSLO(), nil); err == nil {
		t.Fatal("expected error for 400 response")
	}
}

func TestUpdateSLO(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/slos/00000000-0000-0000-0000-000000000001" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		// Respond with the updated fixture: target 0.995, version 2.
		updated := newTestSLO()
		updated.Spec.Description = Ptr("99.5 percent of checkout HTTP requests succeed over a rolling 28-day window.")
		updated.Spec.Objectives = []SloObjective{{DisplayName: Ptr("99.5% availability"), Target: Ptr(float32(0.995))}}
		SetSLOID(updated, "00000000-0000-0000-0000-000000000001")
		updated.Metadata.Labels.Dash0Comversion = Ptr("2")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(updated)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	in := newTestSLO()
	SetSLOID(in, "00000000-0000-0000-0000-000000000001")
	updated, err := client.UpdateSLO(context.Background(), "00000000-0000-0000-0000-000000000001", in, Ptr("default"))
	if err != nil {
		t.Fatalf("UpdateSLO failed: %v", err)
	}
	if updated.Spec.Objectives[0].Target == nil || *updated.Spec.Objectives[0].Target != 0.995 {
		t.Errorf("expected target 0.995, got %v", updated.Spec.Objectives[0].Target)
	}
}

func TestDeleteSLO_200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("unexpected method: %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	if err := client.DeleteSLO(context.Background(), "00000000-0000-0000-0000-000000000001", Ptr("default")); err != nil {
		t.Fatalf("DeleteSLO failed: %v", err)
	}
}

func TestDeleteSLO_204(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	if err := client.DeleteSLO(context.Background(), "id", nil); err != nil {
		t.Fatalf("DeleteSLO failed: %v", err)
	}
}

func TestDeleteSLO_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	if err := client.DeleteSLO(context.Background(), "missing", nil); err == nil {
		t.Fatal("expected error for 404 response")
	}
}

func TestListSLOsIter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]SloDefinition{*newTestSLO()})
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	iter := client.ListSLOsIter(context.Background(), Ptr("default"))
	count := 0
	for iter.Next() {
		count++
	}
	if err := iter.Err(); err != nil {
		t.Fatalf("iterator error: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 SLO, got %d", count)
	}
}
