package dash0

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStripSpamFilterServerFields(t *testing.T) {
	source := CrdSource("api")
	filter := &SpamFilter{
		Metadata: SpamFilterMetadata{
			Annotations: &SpamFilterAnnotations{
				Dash0Comenabled: Ptr(True),
			},
			Labels: &SpamFilterLabels{
				Dash0Comid:      Ptr("sf-1"),
				Dash0Comsource:  &source,
				Dash0Comorigin:  Ptr("terraform"),
				Dash0Comdataset: Ptr("default"),
			},
		},
	}

	StripSpamFilterServerFields(filter)

	if filter.Metadata.Annotations.Dash0Comenabled != nil {
		t.Error("Dash0Comenabled should be nil")
	}
	if filter.Metadata.Labels.Dash0Comid != nil {
		t.Error("Dash0Comid should be nil")
	}
	if filter.Metadata.Labels.Dash0Comsource != nil {
		t.Error("Dash0Comsource should be nil")
	}
	if filter.Metadata.Labels.Dash0Comorigin == nil || *filter.Metadata.Labels.Dash0Comorigin != "terraform" {
		t.Error("Dash0Comorigin should be preserved")
	}
	if filter.Metadata.Labels.Dash0Comdataset == nil || *filter.Metadata.Labels.Dash0Comdataset != "default" {
		t.Error("Dash0Comdataset should be preserved")
	}
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

		apiErr, ok := err.(*APIError)
		if !ok {
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
}
