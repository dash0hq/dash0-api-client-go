package dash0

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestStripViewServerFields(t *testing.T) {
	deletedAt := time.Now()
	version := "2"
	source := CrdSource("terraform")
	dataset := "ds"
	origin := "my-origin"

	v := &ViewDefinition{
		Metadata: ViewMetadata{
			Annotations: &ViewAnnotations{Dash0ComdeletedAt: &deletedAt},
			Labels: &ViewLabels{
				Dash0Comversion: &version,
				Dash0Comsource:  &source,
				Dash0Comdataset: &dataset,
				Dash0Comorigin:  &origin,
				Dash0Comid:      Ptr("keep-this"),
			},
		},
	}

	StripViewServerFields(v)

	if v.Metadata.Annotations.Dash0ComdeletedAt != nil {
		t.Error("Dash0ComdeletedAt should be nil")
	}
	if v.Metadata.Labels.Dash0Comversion != nil {
		t.Error("Dash0Comversion should be nil")
	}
	if v.Metadata.Labels.Dash0Comsource != nil {
		t.Error("Dash0Comsource should be nil")
	}
	if v.Metadata.Labels.Dash0Comdataset != nil {
		t.Error("Dash0Comdataset should be nil")
	}
	if v.Metadata.Labels.Dash0Comorigin != nil {
		t.Error("Dash0Comorigin should be nil")
	}
	if v.Metadata.Labels.Dash0Comid == nil || *v.Metadata.Labels.Dash0Comid != "keep-this" {
		t.Error("Dash0Comid should be preserved")
	}
}

func TestStripViewServerFields_NilLabels(t *testing.T) {
	v := &ViewDefinition{}
	StripViewServerFields(v) // should not panic
	if v.Metadata.Labels != nil {
		t.Error("Labels should remain nil")
	}
}

func TestClearViewID(t *testing.T) {
	v := &ViewDefinition{Metadata: ViewMetadata{Labels: &ViewLabels{Dash0Comid: Ptr("v-1")}}}
	ClearViewID(v)
	if v.Metadata.Labels.Dash0Comid != nil {
		t.Error("Dash0Comid should be nil")
	}
}

func TestClearViewID_NilLabels(t *testing.T) {
	v := &ViewDefinition{}
	ClearViewID(v) // should not panic
}

func TestGetViewDataset(t *testing.T) {
	tests := []struct {
		name string
		view *ViewDefinition
		want string
	}{
		{"nil view", nil, ""},
		{"nil labels", &ViewDefinition{}, ""},
		{"nil dataset", &ViewDefinition{Metadata: ViewMetadata{Labels: &ViewLabels{}}}, ""},
		{"with dataset", &ViewDefinition{Metadata: ViewMetadata{Labels: &ViewLabels{Dash0Comdataset: Ptr("production")}}}, "production"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetViewDataset(tt.view); got != tt.want {
				t.Errorf("GetViewDataset() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetViewDataset(t *testing.T) {
	view := &ViewDefinition{}
	SetViewDataset(view, "production")
	if view.Metadata.Labels == nil || view.Metadata.Labels.Dash0Comdataset == nil {
		t.Fatal("expected dataset to be set")
	}
	if *view.Metadata.Labels.Dash0Comdataset != "production" {
		t.Errorf("Dataset = %q, want %q", *view.Metadata.Labels.Dash0Comdataset, "production")
	}
}

func TestSetViewDataset_Nil(t *testing.T) {
	SetViewDataset(nil, "production") // should not panic
}

func TestGetViewID(t *testing.T) {
	tests := []struct {
		name string
		view *ViewDefinition
		want string
	}{
		{
			"with ID",
			&ViewDefinition{Metadata: ViewMetadata{Labels: &ViewLabels{Dash0Comid: Ptr("v-123")}}},
			"v-123",
		},
		{"nil view", nil, ""},
		{"nil labels", &ViewDefinition{}, ""},
		{"nil ID", &ViewDefinition{Metadata: ViewMetadata{Labels: &ViewLabels{}}}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetViewID(tt.view); got != tt.want {
				t.Errorf("GetViewID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetViewID(t *testing.T) {
	v := &ViewDefinition{}
	SetViewID(v, "new-id")
	if v.Metadata.Labels == nil || v.Metadata.Labels.Dash0Comid == nil {
		t.Fatal("expected ID to be set")
	}
	if *v.Metadata.Labels.Dash0Comid != "new-id" {
		t.Errorf("ID = %q, want %q", *v.Metadata.Labels.Dash0Comid, "new-id")
	}
}

func TestSetViewID_Overwrites(t *testing.T) {
	v := &ViewDefinition{
		Metadata: ViewMetadata{Labels: &ViewLabels{Dash0Comid: Ptr("existing-id")}},
	}
	SetViewID(v, "new-id")
	if *v.Metadata.Labels.Dash0Comid != "new-id" {
		t.Errorf("ID = %q, want %q", *v.Metadata.Labels.Dash0Comid, "new-id")
	}
}

func TestSetViewIDIfAbsent(t *testing.T) {
	v := &ViewDefinition{}
	SetViewIDIfAbsent(v, "new-id")
	if v.Metadata.Labels == nil || v.Metadata.Labels.Dash0Comid == nil {
		t.Fatal("expected ID to be set")
	}
	if *v.Metadata.Labels.Dash0Comid != "new-id" {
		t.Errorf("ID = %q, want %q", *v.Metadata.Labels.Dash0Comid, "new-id")
	}
}

func TestSetViewIDIfAbsent_NoOpWhenAlreadySet(t *testing.T) {
	v := &ViewDefinition{
		Metadata: ViewMetadata{Labels: &ViewLabels{Dash0Comid: Ptr("existing-id")}},
	}
	SetViewIDIfAbsent(v, "new-id")
	if *v.Metadata.Labels.Dash0Comid != "existing-id" {
		t.Errorf("ID = %q, want %q (should not overwrite)", *v.Metadata.Labels.Dash0Comid, "existing-id")
	}
}

func TestGetViewName(t *testing.T) {
	tests := []struct {
		name string
		view *ViewDefinition
		want string
	}{
		{
			"from display name",
			&ViewDefinition{Spec: ViewSpec{Display: ViewDisplay{Name: "Display Name"}}},
			"Display Name",
		},
		{
			"falls back to metadata name",
			&ViewDefinition{Metadata: ViewMetadata{Name: "meta-name"}},
			"meta-name",
		},
		{"nil view", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetViewName(tt.view); got != tt.want {
				t.Errorf("GetViewName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCreateView_201(t *testing.T) {
	view := ViewDefinition{
		Metadata: ViewMetadata{
			Labels: &ViewLabels{
				Dash0Comid: Ptr("vw-123"),
			},
		},
		Spec: ViewSpec{
			Display: ViewDisplay{Name: "Test View"},
		},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(view)
	}))
	defer server.Close()

	client, err := NewClient(
		WithApiUrl(server.URL),
		WithAuthToken("auth_test123"),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	got, err := client.CreateView(context.Background(), &view, nil)
	if err != nil {
		t.Fatalf("CreateView failed: %v", err)
	}

	if GetViewName(got) != "Test View" {
		t.Errorf("expected name %q, got %q", "Test View", GetViewName(got))
	}
	if GetViewID(got) != "vw-123" {
		t.Errorf("expected ID %q, got %q", "vw-123", GetViewID(got))
	}
}
