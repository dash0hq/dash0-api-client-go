package dash0

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

// newTestSpamFilterMetadata returns metadata with every server-managed label, a disabled filter, and a custom label.
func newTestSpamFilterMetadata() SpamFilterMetadata {
	source := CrdSource("api")
	return SpamFilterMetadata{
		Name: "Drop noisy health checks",
		Annotations: &SpamFilterAnnotations{
			Dash0Comenabled: Ptr(False),
		},
		Labels: &SpamFilterLabels{
			Dash0Comid:      Ptr("sf-1"),
			Dash0Comsource:  &source,
			Dash0Comorigin:  Ptr("tf_health"),
			Dash0Comdataset: Ptr("default"),
			Custom:          &map[string]string{"team": "platform"},
		},
	}
}

// assertSpamFilterMetadataStripped checks that a stripped filter lost the server-managed labels and kept what the user sets.
func assertSpamFilterMetadataStripped(t *testing.T, metadata SpamFilterMetadata) {
	t.Helper()
	labels := metadata.Labels
	if labels.Dash0Comid != nil {
		t.Error("Dash0Comid should be nil")
	}
	if labels.Dash0Comsource != nil {
		t.Error("Dash0Comsource should be nil")
	}
	if labels.Dash0Comorigin != nil {
		t.Error("Dash0Comorigin should be nil")
	}
	if labels.Dash0Comdataset != nil {
		t.Error("Dash0Comdataset should be nil")
	}
	// A filter sent without the annotation is enabled by default, so dropping it would re-enable a disabled filter.
	assertPtrEqual(t, "Dash0Comenabled", metadata.Annotations.Dash0Comenabled, False)
	if labels.Custom == nil || (*labels.Custom)["team"] != "platform" {
		t.Errorf("Custom = %v, want the team label preserved", labels.Custom)
	}
	assertEqual(t, "Name", metadata.Name, "Drop noisy health checks")
}

func TestStripSpamFilterServerFields(t *testing.T) {
	filter := &SpamFilter{Metadata: newTestSpamFilterMetadata()}
	StripSpamFilterServerFields(filter)
	assertSpamFilterMetadataStripped(t, filter.Metadata)
}

func TestStripSpamFilterServerFields_NilLabels(t *testing.T) {
	filter := &SpamFilter{}
	StripSpamFilterServerFields(filter) // should not panic
	if filter.Metadata.Labels != nil {
		t.Error("Labels should remain nil")
	}
}

func TestStripSpamFilterServerFields_Nil(t *testing.T) {
	StripSpamFilterServerFields(nil) // should not panic
}

func TestClearSpamFilterID(t *testing.T) {
	filter := &SpamFilter{Metadata: SpamFilterMetadata{Labels: &SpamFilterLabels{Dash0Comid: Ptr("sf-1")}}}
	ClearSpamFilterID(filter)
	if filter.Metadata.Labels.Dash0Comid != nil {
		t.Error("Dash0Comid should be nil")
	}
}

func TestClearSpamFilterID_NilLabels(t *testing.T) {
	filter := &SpamFilter{}
	ClearSpamFilterID(filter) // should not panic
}

func TestClearSpamFilterID_NilFilter(t *testing.T) {
	ClearSpamFilterID(nil) // should not panic
}

func TestGetSpamFilterID(t *testing.T) {
	tests := []struct {
		name   string
		filter *SpamFilter
		want   string
	}{
		{"nil filter", nil, ""},
		{"nil labels", &SpamFilter{}, ""},
		{"nil ID", &SpamFilter{Metadata: SpamFilterMetadata{Labels: &SpamFilterLabels{}}}, ""},
		{"with ID", &SpamFilter{Metadata: SpamFilterMetadata{Labels: &SpamFilterLabels{Dash0Comid: Ptr("sf-123")}}}, "sf-123"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetSpamFilterID(tt.filter); got != tt.want {
				t.Errorf("GetSpamFilterID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetSpamFilterID(t *testing.T) {
	filter := &SpamFilter{}
	SetSpamFilterID(filter, "new-id")
	if filter.Metadata.Labels == nil || filter.Metadata.Labels.Dash0Comid == nil {
		t.Fatal("expected ID to be set")
	}
	if *filter.Metadata.Labels.Dash0Comid != "new-id" {
		t.Errorf("ID = %q, want %q", *filter.Metadata.Labels.Dash0Comid, "new-id")
	}
}

func TestSetSpamFilterID_Overwrites(t *testing.T) {
	filter := &SpamFilter{
		Metadata: SpamFilterMetadata{Labels: &SpamFilterLabels{Dash0Comid: Ptr("existing-id")}},
	}
	SetSpamFilterID(filter, "new-id")
	if *filter.Metadata.Labels.Dash0Comid != "new-id" {
		t.Errorf("ID = %q, want %q", *filter.Metadata.Labels.Dash0Comid, "new-id")
	}
}

func TestSetSpamFilterID_Nil(t *testing.T) {
	SetSpamFilterID(nil, "new-id") // should not panic
}

func TestSetSpamFilterIDIfAbsent(t *testing.T) {
	filter := &SpamFilter{}
	SetSpamFilterIDIfAbsent(filter, "new-id")
	if filter.Metadata.Labels == nil || filter.Metadata.Labels.Dash0Comid == nil {
		t.Fatal("expected ID to be set")
	}
	if *filter.Metadata.Labels.Dash0Comid != "new-id" {
		t.Errorf("ID = %q, want %q", *filter.Metadata.Labels.Dash0Comid, "new-id")
	}
}

func TestSetSpamFilterIDIfAbsent_NoOpWhenAlreadySet(t *testing.T) {
	filter := &SpamFilter{
		Metadata: SpamFilterMetadata{Labels: &SpamFilterLabels{Dash0Comid: Ptr("existing-id")}},
	}
	SetSpamFilterIDIfAbsent(filter, "new-id")
	if *filter.Metadata.Labels.Dash0Comid != "existing-id" {
		t.Errorf("ID = %q, want %q (should not overwrite)", *filter.Metadata.Labels.Dash0Comid, "existing-id")
	}
}

func TestSetSpamFilterIDIfAbsent_Nil(t *testing.T) {
	SetSpamFilterIDIfAbsent(nil, "new-id") // should not panic
}

func TestGetSpamFilterName(t *testing.T) {
	tests := []struct {
		name   string
		filter *SpamFilter
		want   string
	}{
		{"nil filter", nil, ""},
		{"empty name", &SpamFilter{}, ""},
		{"with name", &SpamFilter{Metadata: SpamFilterMetadata{Name: "Drop noisy health checks"}}, "Drop noisy health checks"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetSpamFilterName(tt.filter); got != tt.want {
				t.Errorf("GetSpamFilterName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetSpamFilterDataset(t *testing.T) {
	tests := []struct {
		name   string
		filter *SpamFilter
		want   string
	}{
		{"nil filter", nil, ""},
		{"nil labels", &SpamFilter{}, ""},
		{"nil dataset", &SpamFilter{Metadata: SpamFilterMetadata{Labels: &SpamFilterLabels{}}}, ""},
		{"with dataset", &SpamFilter{Metadata: SpamFilterMetadata{Labels: &SpamFilterLabels{Dash0Comdataset: Ptr("production")}}}, "production"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetSpamFilterDataset(tt.filter); got != tt.want {
				t.Errorf("GetSpamFilterDataset() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetSpamFilterDataset(t *testing.T) {
	filter := &SpamFilter{}
	SetSpamFilterDataset(filter, "production")
	if filter.Metadata.Labels == nil || filter.Metadata.Labels.Dash0Comdataset == nil {
		t.Fatal("expected dataset to be set")
	}
	if *filter.Metadata.Labels.Dash0Comdataset != "production" {
		t.Errorf("Dataset = %q, want %q", *filter.Metadata.Labels.Dash0Comdataset, "production")
	}
}

func TestSetSpamFilterDataset_Nil(t *testing.T) {
	SetSpamFilterDataset(nil, "production") // should not panic
}

func TestStripSpamFilterV1Alpha2ServerFields(t *testing.T) {
	filter := &SpamFilterV1Alpha2{Metadata: newTestSpamFilterMetadata()}
	StripSpamFilterV1Alpha2ServerFields(filter)
	assertSpamFilterMetadataStripped(t, filter.Metadata)
}

func TestStripSpamFilterV1Alpha2ServerFields_NilLabels(t *testing.T) {
	filter := &SpamFilterV1Alpha2{}
	StripSpamFilterV1Alpha2ServerFields(filter) // should not panic
	if filter.Metadata.Labels != nil {
		t.Error("Labels should remain nil")
	}
}

func TestStripSpamFilterV1Alpha2ServerFields_Nil(t *testing.T) {
	StripSpamFilterV1Alpha2ServerFields(nil) // should not panic
}

func TestClearSpamFilterV1Alpha2ID(t *testing.T) {
	filter := &SpamFilterV1Alpha2{Metadata: SpamFilterMetadata{Labels: &SpamFilterLabels{Dash0Comid: Ptr("sf-1")}}}
	ClearSpamFilterV1Alpha2ID(filter)
	if filter.Metadata.Labels.Dash0Comid != nil {
		t.Error("Dash0Comid should be nil")
	}
}

func TestClearSpamFilterV1Alpha2ID_NilLabels(t *testing.T) {
	ClearSpamFilterV1Alpha2ID(&SpamFilterV1Alpha2{}) // should not panic
}

func TestClearSpamFilterV1Alpha2ID_Nil(t *testing.T) {
	ClearSpamFilterV1Alpha2ID(nil) // should not panic
}

func TestGetSpamFilterV1Alpha2ID(t *testing.T) {
	tests := []struct {
		name   string
		filter *SpamFilterV1Alpha2
		want   string
	}{
		{"nil filter", nil, ""},
		{"nil labels", &SpamFilterV1Alpha2{}, ""},
		{"nil ID", &SpamFilterV1Alpha2{Metadata: SpamFilterMetadata{Labels: &SpamFilterLabels{}}}, ""},
		{"with ID", &SpamFilterV1Alpha2{Metadata: SpamFilterMetadata{Labels: &SpamFilterLabels{Dash0Comid: Ptr("sf-123")}}}, "sf-123"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, "GetSpamFilterV1Alpha2ID()", GetSpamFilterV1Alpha2ID(tt.filter), tt.want)
		})
	}
}

func TestGetSpamFilterV1Alpha2Name(t *testing.T) {
	tests := []struct {
		name   string
		filter *SpamFilterV1Alpha2
		want   string
	}{
		{"nil filter", nil, ""},
		{"empty name", &SpamFilterV1Alpha2{}, ""},
		{"with name", &SpamFilterV1Alpha2{Metadata: SpamFilterMetadata{Name: "Drop noisy health checks"}}, "Drop noisy health checks"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, "GetSpamFilterV1Alpha2Name()", GetSpamFilterV1Alpha2Name(tt.filter), tt.want)
		})
	}
}

func TestGetSpamFilterV1Alpha2Dataset(t *testing.T) {
	tests := []struct {
		name   string
		filter *SpamFilterV1Alpha2
		want   string
	}{
		{"nil filter", nil, ""},
		{"nil labels", &SpamFilterV1Alpha2{}, ""},
		{"nil dataset", &SpamFilterV1Alpha2{Metadata: SpamFilterMetadata{Labels: &SpamFilterLabels{}}}, ""},
		{"with dataset", &SpamFilterV1Alpha2{Metadata: SpamFilterMetadata{Labels: &SpamFilterLabels{Dash0Comdataset: Ptr("production")}}}, "production"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, "GetSpamFilterV1Alpha2Dataset()", GetSpamFilterV1Alpha2Dataset(tt.filter), tt.want)
		})
	}
}

func TestSetSpamFilterV1Alpha2Dataset(t *testing.T) {
	filter := &SpamFilterV1Alpha2{}
	SetSpamFilterV1Alpha2Dataset(filter, "production")
	if filter.Metadata.Labels == nil {
		t.Fatal("expected labels to be initialized")
	}
	assertPtrEqual(t, "Dash0Comdataset", filter.Metadata.Labels.Dash0Comdataset, "production")
}

func TestSetSpamFilterV1Alpha2Dataset_Nil(t *testing.T) {
	SetSpamFilterV1Alpha2Dataset(nil, "production") // should not panic
}

func TestSetSpamFilterV1Alpha2ID(t *testing.T) {
	filter := &SpamFilterV1Alpha2{}
	SetSpamFilterV1Alpha2ID(filter, "new-id")
	if filter.Metadata.Labels == nil {
		t.Fatal("expected labels to be initialized")
	}
	assertPtrEqual(t, "Dash0Comid", filter.Metadata.Labels.Dash0Comid, "new-id")
}

func TestSetSpamFilterV1Alpha2ID_Overwrites(t *testing.T) {
	filter := &SpamFilterV1Alpha2{Metadata: SpamFilterMetadata{Labels: &SpamFilterLabels{Dash0Comid: Ptr("existing-id")}}}
	SetSpamFilterV1Alpha2ID(filter, "new-id")
	assertPtrEqual(t, "Dash0Comid", filter.Metadata.Labels.Dash0Comid, "new-id")
}

func TestSetSpamFilterV1Alpha2ID_Nil(t *testing.T) {
	SetSpamFilterV1Alpha2ID(nil, "new-id") // should not panic
}

func TestSetSpamFilterV1Alpha2IDIfAbsent(t *testing.T) {
	filter := &SpamFilterV1Alpha2{}
	SetSpamFilterV1Alpha2IDIfAbsent(filter, "new-id")
	if filter.Metadata.Labels == nil {
		t.Fatal("expected labels to be initialized")
	}
	assertPtrEqual(t, "Dash0Comid", filter.Metadata.Labels.Dash0Comid, "new-id")
}

func TestSetSpamFilterV1Alpha2IDIfAbsent_NoOpWhenAlreadySet(t *testing.T) {
	filter := &SpamFilterV1Alpha2{Metadata: SpamFilterMetadata{Labels: &SpamFilterLabels{Dash0Comid: Ptr("existing-id")}}}
	SetSpamFilterV1Alpha2IDIfAbsent(filter, "new-id")
	assertPtrEqual(t, "Dash0Comid", filter.Metadata.Labels.Dash0Comid, "existing-id")
}

func TestSetSpamFilterV1Alpha2IDIfAbsent_Nil(t *testing.T) {
	SetSpamFilterV1Alpha2IDIfAbsent(nil, "new-id") // should not panic
}

func newTestSpamFilter() SpamFilter {
	return SpamFilter{
		Kind: Dash0SpamFilter,
		Metadata: SpamFilterMetadata{
			Name: "Drop noisy health checks",
			Labels: &SpamFilterLabels{
				Dash0Comid:      Ptr("sf-123"),
				Dash0Comdataset: Ptr("default"),
			},
			Annotations: &SpamFilterAnnotations{
				Dash0Comenabled: Ptr(True),
			},
		},
		Spec: SpamFilterSpec{
			Contexts: []TelemetryFilterContext{"log"},
			Filter: FilterCriteria{
				{
					Key:      "k8s.namespace.name",
					Operator: "is",
				},
			},
		},
	}
}

func TestSpamFilters_Integration(t *testing.T) {
	t.Run("ListSpamFilters returns filters", func(t *testing.T) {
		filter := newTestSpamFilter()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/spam-filters" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.Method != http.MethodGet {
				t.Errorf("unexpected method: %s", r.Method)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(GetSpamFiltersResponse{SpamFilters: []SpamFilter{filter}})
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		filters, err := client.ListSpamFilters(context.Background(), nil)
		if err != nil {
			t.Fatalf("ListSpamFilters failed: %v", err)
		}

		if len(filters) != 1 {
			t.Fatalf("expected 1 filter, got %d", len(filters))
		}
		if filters[0].Metadata.Name != "Drop noisy health checks" {
			t.Errorf("expected name %q, got %q", "Drop noisy health checks", filters[0].Metadata.Name)
		}
	})

	t.Run("ListSpamFilters returns empty list", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(GetSpamFiltersResponse{SpamFilters: []SpamFilter{}})
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		filters, err := client.ListSpamFilters(context.Background(), nil)
		if err != nil {
			t.Fatalf("ListSpamFilters failed: %v", err)
		}

		if len(filters) != 0 {
			t.Errorf("expected empty list, got %d filters", len(filters))
		}
	})

	t.Run("GetSpamFilter returns filter", func(t *testing.T) {
		filter := newTestSpamFilter()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/spam-filters/sf-123" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.Method != http.MethodGet {
				t.Errorf("unexpected method: %s", r.Method)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(filter)
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		got, err := client.GetSpamFilter(context.Background(), "sf-123", nil)
		if err != nil {
			t.Fatalf("GetSpamFilter failed: %v", err)
		}

		v1, ok := got.(*SpamFilter)
		if !ok {
			t.Fatalf("expected *SpamFilter, got %T", got)
		}
		if v1.Metadata.Name != "Drop noisy health checks" {
			t.Errorf("expected name %q, got %q", "Drop noisy health checks", v1.Metadata.Name)
		}
	})

	t.Run("GetSpamFilter handles 404", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"message": "Spam filter not found",
			})
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, err = client.GetSpamFilter(context.Background(), "nonexistent", nil)
		if err == nil {
			t.Fatal("expected error for 404 response")
		}

		if !IsNotFound(err) {
			t.Errorf("expected IsNotFound to return true")
		}
	})

	t.Run("CreateSpamFilter succeeds with 200", func(t *testing.T) {
		filter := newTestSpamFilter()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/spam-filters" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.Method != http.MethodPost {
				t.Errorf("unexpected method: %s", r.Method)
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("failed to read request body: %v", err)
			}
			var reqFilter SpamFilter
			if err := json.Unmarshal(body, &reqFilter); err != nil {
				t.Fatalf("failed to unmarshal request body: %v", err)
			}
			if reqFilter.Metadata.Name != "Drop noisy health checks" {
				t.Errorf("expected request name %q, got %q", "Drop noisy health checks", reqFilter.Metadata.Name)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(filter)
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		got, err := client.CreateSpamFilter(context.Background(), &filter, nil)
		if err != nil {
			t.Fatalf("CreateSpamFilter failed: %v", err)
		}

		if got.Metadata.Name != "Drop noisy health checks" {
			t.Errorf("expected name %q, got %q", "Drop noisy health checks", got.Metadata.Name)
		}
	})

	t.Run("CreateSpamFilter succeeds with 201", func(t *testing.T) {
		filter := newTestSpamFilter()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("unexpected method: %s", r.Method)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(filter)
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		got, err := client.CreateSpamFilter(context.Background(), &filter, nil)
		if err != nil {
			t.Fatalf("CreateSpamFilter failed: %v", err)
		}

		if got.Metadata.Name != "Drop noisy health checks" {
			t.Errorf("expected name %q, got %q", "Drop noisy health checks", got.Metadata.Name)
		}
		if GetSpamFilterID(got) != "sf-123" {
			t.Errorf("expected ID %q, got %q", "sf-123", GetSpamFilterID(got))
		}
	})

	t.Run("CreateSpamFilter handles error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"message": "Invalid spam filter definition",
			})
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		filter := newTestSpamFilter()
		_, err = client.CreateSpamFilter(context.Background(), &filter, nil)
		if err == nil {
			t.Fatal("expected error for 400 response")
		}

		var apiErr *APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("expected *APIError, got %T", err)
		}
		if apiErr.StatusCode != http.StatusBadRequest {
			t.Errorf("expected status code %d, got %d", http.StatusBadRequest, apiErr.StatusCode)
		}
	})

	t.Run("UpdateSpamFilter succeeds", func(t *testing.T) {
		filter := newTestSpamFilter()
		filter.Metadata.Name = "Drop noisy health checks (updated)"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/spam-filters/sf-123" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.Method != http.MethodPut {
				t.Errorf("unexpected method: %s", r.Method)
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("failed to read request body: %v", err)
			}
			var reqFilter SpamFilter
			if err := json.Unmarshal(body, &reqFilter); err != nil {
				t.Fatalf("failed to unmarshal request body: %v", err)
			}
			if reqFilter.Metadata.Name != "Drop noisy health checks (updated)" {
				t.Errorf("expected request name %q, got %q", "Drop noisy health checks (updated)", reqFilter.Metadata.Name)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(filter)
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		got, err := client.UpdateSpamFilter(context.Background(), "sf-123", &filter, nil)
		if err != nil {
			t.Fatalf("UpdateSpamFilter failed: %v", err)
		}

		if got.Metadata.Name != "Drop noisy health checks (updated)" {
			t.Errorf("expected name %q, got %q", "Drop noisy health checks (updated)", got.Metadata.Name)
		}
	})

	t.Run("DeleteSpamFilter succeeds with 200", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/spam-filters/sf-123" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.Method != http.MethodDelete {
				t.Errorf("unexpected method: %s", r.Method)
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		err = client.DeleteSpamFilter(context.Background(), "sf-123", nil)
		if err != nil {
			t.Fatalf("DeleteSpamFilter failed: %v", err)
		}
	})

	t.Run("DeleteSpamFilter succeeds with 204", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		err = client.DeleteSpamFilter(context.Background(), "sf-123", nil)
		if err != nil {
			t.Fatalf("DeleteSpamFilter failed: %v", err)
		}
	})

	t.Run("DeleteSpamFilter handles error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"message": "Forbidden",
			})
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		err = client.DeleteSpamFilter(context.Background(), "sf-123", nil)
		if err == nil {
			t.Fatal("expected error for 403 response")
		}

		if !IsForbidden(err) {
			t.Errorf("expected IsForbidden to return true")
		}
	})

	t.Run("ListSpamFiltersIter iterates filters", func(t *testing.T) {
		filters := []SpamFilter{
			newTestSpamFilter(),
			{
				Kind:     Dash0SpamFilter,
				Metadata: SpamFilterMetadata{Name: "Drop debug logs"},
				Spec: SpamFilterSpec{
					Contexts: []TelemetryFilterContext{"log"},
					Filter:   FilterCriteria{},
				},
			},
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(GetSpamFiltersResponse{SpamFilters: filters})
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		iter := client.ListSpamFiltersIter(context.Background(), nil)
		var names []string
		for iter.Next() {
			names = append(names, iter.Current().Metadata.Name)
		}
		if iter.Err() != nil {
			t.Fatalf("iterator error: %v", iter.Err())
		}

		if len(names) != 2 {
			t.Fatalf("expected 2 filters, got %d", len(names))
		}
		if names[0] != "Drop noisy health checks" {
			t.Errorf("expected first filter %q, got %q", "Drop noisy health checks", names[0])
		}
		if names[1] != "Drop debug logs" {
			t.Errorf("expected second filter %q, got %q", "Drop debug logs", names[1])
		}
	})

	t.Run("ListSpamFilters passes dataset parameter", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			dataset := r.URL.Query().Get("dataset")
			if dataset != "production" {
				t.Errorf("expected dataset %q, got %q", "production", dataset)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(GetSpamFiltersResponse{SpamFilters: []SpamFilter{}})
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		ds := "production"
		_, err = client.ListSpamFilters(context.Background(), &ds)
		if err != nil {
			t.Fatalf("ListSpamFilters failed: %v", err)
		}
	})

	t.Run("verifies authorization header", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth != "Bearer auth_test123" {
				t.Errorf("unexpected Authorization header: %s", auth)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(GetSpamFiltersResponse{SpamFilters: []SpamFilter{}})
		}))
		defer server.Close()

		client, err := NewClient(
			WithApiUrl(server.URL),
			WithAuthToken("auth_test123"),
		)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, err = client.ListSpamFilters(context.Background(), nil)
		if err != nil {
			t.Fatalf("ListSpamFilters failed: %v", err)
		}
	})

	t.Run("GetSpamFilter returns v1alpha2 typed value when server sends v1alpha2", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"apiVersion": "v1alpha2",
				"kind": "Dash0SpamFilter",
				"metadata": {"name": "v2 filter"},
				"spec": {
					"context": "log",
					"filter": [{"key": "k8s.namespace.name", "operator": "is"}]
				}
			}`))
		}))
		defer server.Close()

		client, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		got, err := client.GetSpamFilter(context.Background(), "sf-v2", nil)
		if err != nil {
			t.Fatalf("GetSpamFilter failed: %v", err)
		}
		v2, ok := got.(*SpamFilterV1Alpha2)
		if !ok {
			t.Fatalf("expected *SpamFilterV1Alpha2, got %T", got)
		}
		if v2.Spec.Context != "log" {
			t.Errorf("expected spec.context %q, got %q", "log", v2.Spec.Context)
		}
		if v2.Metadata.Name != "v2 filter" {
			t.Errorf("expected name %q, got %q", "v2 filter", v2.Metadata.Name)
		}
	})

	t.Run("GetSpamFilter rejects unknown apiVersion", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"apiVersion": "v9alpha9", "kind": "Dash0SpamFilter"}`))
		}))
		defer server.Close()

		client, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, err = client.GetSpamFilter(context.Background(), "sf-bogus", nil)
		if err == nil {
			t.Fatal("expected error for unknown apiVersion")
		}
	})

	t.Run("CreateSpamFilterV1Alpha2 sends v1alpha2 body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("failed to read request body: %v", err)
			}
			var req SpamFilterV1Alpha2
			if err := json.Unmarshal(body, &req); err != nil {
				t.Fatalf("failed to unmarshal request body: %v", err)
			}
			if req.ApiVersion != V1alpha2 {
				t.Errorf("expected request apiVersion v1alpha2, got %q", req.ApiVersion)
			}
			if req.Spec.Context != "log" {
				t.Errorf("expected request spec.context %q, got %q", "log", req.Spec.Context)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write(body)
		}))
		defer server.Close()

		client, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		got, err := client.CreateSpamFilterV1Alpha2(context.Background(), &SpamFilterV1Alpha2{
			ApiVersion: V1alpha2,
			Kind:       SpamFilterDefinitionV1Alpha2KindDash0SpamFilter,
			Metadata:   SpamFilterMetadata{Name: "v2 filter"},
			Spec: SpamFilterSpecV1Alpha2{
				Context: "log",
				Filter:  FilterCriteria{{Key: "k8s.namespace.name", Operator: "is"}},
			},
		}, nil)
		if err != nil {
			t.Fatalf("CreateSpamFilterV1Alpha2 failed: %v", err)
		}
		if got.Spec.Context != "log" {
			t.Errorf("expected returned spec.context %q, got %q", "log", got.Spec.Context)
		}
	})

	t.Run("UpdateSpamFilterV1Alpha2 sends v1alpha2 body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("unexpected method: %s", r.Method)
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("failed to read request body: %v", err)
			}
			var req SpamFilterV1Alpha2
			if err := json.Unmarshal(body, &req); err != nil {
				t.Fatalf("failed to unmarshal request body: %v", err)
			}
			if req.ApiVersion != V1alpha2 {
				t.Errorf("expected request apiVersion v1alpha2, got %q", req.ApiVersion)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
		}))
		defer server.Close()

		client, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		got, err := client.UpdateSpamFilterV1Alpha2(context.Background(), "sf-v2", &SpamFilterV1Alpha2{
			ApiVersion: V1alpha2,
			Kind:       SpamFilterDefinitionV1Alpha2KindDash0SpamFilter,
			Metadata:   SpamFilterMetadata{Name: "v2 filter"},
			Spec: SpamFilterSpecV1Alpha2{
				Context: "span",
				Filter:  FilterCriteria{{Key: "service.name", Operator: "is"}},
			},
		}, nil)
		if err != nil {
			t.Fatalf("UpdateSpamFilterV1Alpha2 failed: %v", err)
		}
		if got.Spec.Context != "span" {
			t.Errorf("expected returned spec.context %q, got %q", "span", got.Spec.Context)
		}
	})

	t.Run("every method sends the dataset query parameter", func(t *testing.T) {
		filter := newTestSpamFilter()
		filterV1Alpha2 := newTestSpamFilterV1Alpha2()
		tests := []struct {
			method   string
			response any
			call     func(c Client, dataset *string) error
		}{
			{"ListSpamFilters", GetSpamFiltersResponse{}, func(c Client, dataset *string) error {
				_, err := c.ListSpamFilters(context.Background(), dataset)
				return err
			}},
			{"ListSpamFilterObjects", GetSpamFiltersResponse{}, func(c Client, dataset *string) error {
				_, err := c.ListSpamFilterObjects(context.Background(), dataset)
				return err
			}},
			{"GetSpamFilter", filter, func(c Client, dataset *string) error {
				_, err := c.GetSpamFilter(context.Background(), "sf-123", dataset)
				return err
			}},
			{"CreateSpamFilter", filter, func(c Client, dataset *string) error {
				_, err := c.CreateSpamFilter(context.Background(), &filter, dataset)
				return err
			}},
			{"UpdateSpamFilter", filter, func(c Client, dataset *string) error {
				_, err := c.UpdateSpamFilter(context.Background(), "sf-123", &filter, dataset)
				return err
			}},
			{"CreateSpamFilterV1Alpha2", filterV1Alpha2, func(c Client, dataset *string) error {
				_, err := c.CreateSpamFilterV1Alpha2(context.Background(), &filterV1Alpha2, dataset)
				return err
			}},
			{"UpdateSpamFilterV1Alpha2", filterV1Alpha2, func(c Client, dataset *string) error {
				_, err := c.UpdateSpamFilterV1Alpha2(context.Background(), "sf-123", &filterV1Alpha2, dataset)
				return err
			}},
			{"DeleteSpamFilter", nil, func(c Client, dataset *string) error {
				return c.DeleteSpamFilter(context.Background(), "sf-123", dataset)
			}},
		}
		for _, tt := range tests {
			t.Run(tt.method, func(t *testing.T) {
				var gotURL *url.URL
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					gotURL = r.URL
					if tt.response == nil {
						w.WriteHeader(http.StatusNoContent)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(tt.response)
				}))
				defer server.Close()

				if err := tt.call(newTestClient(t, server.URL), Ptr("production")); err != nil {
					t.Fatalf("%s failed: %v", tt.method, err)
				}
				assertEqual(t, "dataset", gotURL.Query().Get("dataset"), "production")
			})
		}
	})

	t.Run("create and update reject a nil filter", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("no request should reach the server")
		}))
		defer server.Close()
		c := newTestClient(t, server.URL)

		tests := []struct {
			method string
			call   func() error
		}{
			{"CreateSpamFilter", func() error {
				_, err := c.CreateSpamFilter(context.Background(), nil, nil)
				return err
			}},
			{"UpdateSpamFilter", func() error {
				_, err := c.UpdateSpamFilter(context.Background(), "sf-123", nil, nil)
				return err
			}},
			{"CreateSpamFilterV1Alpha2", func() error {
				_, err := c.CreateSpamFilterV1Alpha2(context.Background(), nil, nil)
				return err
			}},
			{"UpdateSpamFilterV1Alpha2", func() error {
				_, err := c.UpdateSpamFilterV1Alpha2(context.Background(), "sf-123", nil, nil)
				return err
			}},
		}
		for _, tt := range tests {
			t.Run(tt.method, func(t *testing.T) {
				err := tt.call()
				if err == nil {
					t.Fatal("expected an error for a nil filter")
				}
				if !strings.Contains(err.Error(), "requires a non-nil filter") {
					t.Errorf("error = %q, want it to mention the nil filter", err)
				}
			})
		}
	})
}

func newTestSpamFilterV1Alpha2() SpamFilterV1Alpha2 {
	return SpamFilterV1Alpha2{
		ApiVersion: V1alpha2,
		Kind:       SpamFilterDefinitionV1Alpha2KindDash0SpamFilter,
		Metadata:   SpamFilterMetadata{Name: "v2 filter"},
		Spec: SpamFilterSpecV1Alpha2{
			Context: "log",
			Filter:  FilterCriteria{{Key: "k8s.namespace.name", Operator: "is"}},
		},
	}
}

func TestSpamFilters_APINotConfigured(t *testing.T) {
	c, err := NewClient(
		WithOtlpEndpoint(OtlpEncodingJson, "https://ingress.eu-west-1.aws.dash0.com"),
		WithAuthToken("auth_test123"),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	ctx := context.Background()

	if _, err := c.ListSpamFilterObjects(ctx, nil); !errors.Is(err, ErrAPINotConfigured) {
		t.Errorf("ListSpamFilterObjects error = %v, want ErrAPINotConfigured", err)
	}
}

func TestSpamFilters_Integration_OperatorApiVersion(t *testing.T) {
	t.Run("ListSpamFilterObjects accepts operator.dash0.com group", func(t *testing.T) {
		// Mimic the wire shape the Dash0 Kubernetes operator emits: TypeMeta
		// from operator.dash0.com/v1alpha1 plus the standard v1alpha1 spec.
		operatorBody := `{
			"spamFilters": [
				{
					"apiVersion": "operator.dash0.com/v1alpha1",
					"kind": "Dash0SpamFilter",
					"metadata": {"name": "from-operator"},
					"spec": {
						"contexts": ["log"],
						"filter": [{"key": "http.target", "operator": "ends_with", "value": "/healthz"}]
					}
				},
				{
					"apiVersion": "v1alpha2",
					"kind": "Dash0SpamFilter",
					"metadata": {"name": "from-cli"},
					"spec": {
						"context": "log",
						"filter": [{"key": "service.name", "operator": "is", "value": "noisy"}]
					}
				}
			]
		}`

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(operatorBody))
		}))
		defer server.Close()

		c, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		got, err := c.ListSpamFilterObjects(context.Background(), nil)
		if err != nil {
			t.Fatalf("ListSpamFilterObjects failed: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("expected 2 filters, got %d", len(got))
		}
		v1, ok := got[0].(*SpamFilter)
		if !ok {
			t.Fatalf("expected *SpamFilter for operator item, got %T", got[0])
		}
		if v1.Metadata.Name != "from-operator" {
			t.Errorf("expected name %q, got %q", "from-operator", v1.Metadata.Name)
		}
		v2, ok := got[1].(*SpamFilterV1Alpha2)
		if !ok {
			t.Fatalf("expected *SpamFilterV1Alpha2 for CLI item, got %T", got[1])
		}
		if v2.Spec.Context != "log" {
			t.Errorf("expected spec.context %q, got %q", "log", v2.Spec.Context)
		}
	})

	t.Run("GetSpamFilter accepts operator.dash0.com group", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"apiVersion": "operator.dash0.com/v1alpha1",
				"kind": "Dash0SpamFilter",
				"metadata": {"name": "from-operator"},
				"spec": {
					"contexts": ["log"],
					"filter": [{"key": "http.target", "operator": "ends_with", "value": "/healthz"}]
				}
			}`))
		}))
		defer server.Close()

		c, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		got, err := c.GetSpamFilter(context.Background(), "from-operator", nil)
		if err != nil {
			t.Fatalf("GetSpamFilter failed: %v", err)
		}
		v1, ok := got.(*SpamFilter)
		if !ok {
			t.Fatalf("expected *SpamFilter, got %T", got)
		}
		if v1.Metadata.Name != "from-operator" {
			t.Errorf("expected name %q, got %q", "from-operator", v1.Metadata.Name)
		}
	})

	t.Run("ListSpamFilterObjects rejects foreign apiGroup", func(t *testing.T) {
		body := `{"spamFilters":[{"apiVersion":"monitoring.coreos.com/v1","kind":"PrometheusRule","metadata":{"name":"x"},"spec":{"contexts":["log"],"filter":[]}}]}`
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(body))
		}))
		defer server.Close()

		c, err := NewClient(WithApiUrl(server.URL), WithAuthToken("auth_test123"))
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, err = c.ListSpamFilterObjects(context.Background(), nil)
		if err == nil {
			t.Fatal("expected error for foreign apiGroup, got nil")
		}
	})
}

// TestLiveAPI_SpamFilters runs spam filter round trips against a real Dash0 API.
// It is skipped unless DASH0_API_URL and DASH0_AUTH_TOKEN are set.
// The filters it creates are disabled, so they drop no telemetry, and it deletes them when it finishes.
func TestLiveAPI_SpamFilters(t *testing.T) {
	apiUrl := os.Getenv("DASH0_API_URL")
	authToken := os.Getenv("DASH0_AUTH_TOKEN")
	if apiUrl == "" || authToken == "" {
		t.Skip("Skipping live API tests: DASH0_API_URL and DASH0_AUTH_TOKEN must be set")
	}
	client, err := NewClient(WithApiUrl(apiUrl), WithAuthToken(authToken))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx := context.Background()
	dataset := Ptr("default")
	stamp := time.Now().UnixNano()
	originV1Alpha1 := fmt.Sprintf("live-spam-filter-v1alpha1-%d", stamp)
	originV1Alpha2 := fmt.Sprintf("live-spam-filter-v1alpha2-%d", stamp)
	t.Cleanup(func() {
		for _, origin := range []string{originV1Alpha1, originV1Alpha2} {
			if err := client.DeleteSpamFilter(context.Background(), origin, dataset); err != nil && !IsNotFound(err) {
				t.Logf("cleanup of %s failed: %v", origin, err)
			}
		}
	})
	disabled := func() *SpamFilterAnnotations {
		return &SpamFilterAnnotations{Dash0Comenabled: Ptr(False)}
	}
	match := FilterCriteria{{Key: "dash0.live.test.never.set", Operator: "is_set"}}

	t.Run("a disabled v1alpha1 filter stays disabled through get, strip, and update", func(t *testing.T) {
		filter := &SpamFilter{
			Kind:     Dash0SpamFilter,
			Metadata: SpamFilterMetadata{Name: originV1Alpha1, Annotations: disabled()},
			Spec:     SpamFilterSpec{Contexts: []TelemetryFilterContext{TelemetryFilterContextLog}, Filter: match},
		}
		if _, err := client.UpdateSpamFilter(ctx, originV1Alpha1, filter, dataset); err != nil {
			t.Fatalf("UpdateSpamFilter as upsert failed: %v", err)
		}

		got := getLiveSpamFilter[*SpamFilter](t, client, originV1Alpha1, dataset)
		assertLiveSpamFilterDisabled(t, "after create", got.Metadata)

		StripSpamFilterServerFields(got)
		got.Metadata.Name = originV1Alpha1 + "-renamed"
		if _, err := client.UpdateSpamFilter(ctx, originV1Alpha1, got, dataset); err != nil {
			t.Fatalf("UpdateSpamFilter after strip failed: %v", err)
		}

		got = getLiveSpamFilter[*SpamFilter](t, client, originV1Alpha1, dataset)
		assertEqual(t, "name after update", got.Metadata.Name, originV1Alpha1+"-renamed")
		assertLiveSpamFilterDisabled(t, "after strip and update", got.Metadata)
		// Strip clears the origin from the body, and the server takes it from the path instead.
		assertEqual(t, "origin after update", liveSpamFilterOrigin(got.Metadata), originV1Alpha1)
	})

	t.Run("a disabled v1alpha2 filter keeps its version and stays disabled", func(t *testing.T) {
		filter := &SpamFilterV1Alpha2{
			ApiVersion: V1alpha2,
			Kind:       SpamFilterDefinitionV1Alpha2KindDash0SpamFilter,
			Metadata:   SpamFilterMetadata{Name: originV1Alpha2, Annotations: disabled()},
			Spec:       SpamFilterSpecV1Alpha2{Context: TelemetryFilterContextSpan, Filter: match},
		}
		if _, err := client.UpdateSpamFilterV1Alpha2(ctx, originV1Alpha2, filter, dataset); err != nil {
			t.Fatalf("UpdateSpamFilterV1Alpha2 as upsert failed: %v", err)
		}

		got := getLiveSpamFilter[*SpamFilterV1Alpha2](t, client, originV1Alpha2, dataset)
		StripSpamFilterV1Alpha2ServerFields(got)
		if _, err := client.UpdateSpamFilterV1Alpha2(ctx, originV1Alpha2, got, dataset); err != nil {
			t.Fatalf("UpdateSpamFilterV1Alpha2 after strip failed: %v", err)
		}

		filters, err := client.ListSpamFilterObjects(ctx, dataset)
		if err != nil {
			t.Fatalf("ListSpamFilterObjects failed: %v", err)
		}
		var listed *SpamFilterV1Alpha2
		for _, obj := range filters {
			if filter, ok := obj.(*SpamFilterV1Alpha2); ok && liveSpamFilterOrigin(filter.Metadata) == originV1Alpha2 {
				listed = filter
			}
		}
		if listed == nil {
			t.Fatalf("ListSpamFilterObjects did not return %s as a v1alpha2 filter", originV1Alpha2)
		}
		assertEqual(t, "spec.context", string(listed.Spec.Context), string(TelemetryFilterContextSpan))
		assertLiveSpamFilterDisabled(t, "in the list after strip and update", listed.Metadata)
	})
}

// getLiveSpamFilter gets the filter at origin and fails the test unless the server returned it as T.
func getLiveSpamFilter[T SpamFilterObject](t *testing.T, client Client, origin string, dataset *string) T {
	t.Helper()
	obj, err := client.GetSpamFilter(context.Background(), origin, dataset)
	if err != nil {
		t.Fatalf("GetSpamFilter(%s) failed: %v", origin, err)
	}
	filter, ok := obj.(T)
	if !ok {
		t.Fatalf("GetSpamFilter(%s) returned %T, want %T", origin, obj, filter)
	}
	return filter
}

func assertLiveSpamFilterDisabled(t *testing.T, when string, metadata SpamFilterMetadata) {
	t.Helper()
	if metadata.Annotations == nil || metadata.Annotations.Dash0Comenabled == nil {
		t.Fatalf("%s: dash0.com/enabled is missing, so the server treats the filter as enabled", when)
	}
	assertEqual(t, "dash0.com/enabled "+when, string(*metadata.Annotations.Dash0Comenabled), string(False))
}

func liveSpamFilterOrigin(metadata SpamFilterMetadata) string {
	if metadata.Labels == nil {
		return ""
	}
	return StringValue(metadata.Labels.Dash0Comorigin)
}
