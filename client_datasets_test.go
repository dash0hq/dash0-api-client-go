package dash0

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestDataset(slug string) *DatasetDefinition {
	return &DatasetDefinition{
		Kind:     Dash0Dataset,
		Metadata: DatasetMetadata{Name: "Staging", Labels: &DatasetLabels{Dash0Comid: slug}},
	}
}

func TestListDatasets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertEqual(t, "method", r.Method, http.MethodGet)
		assertEqual(t, "path", r.URL.Path, "/api/datasets")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(DatasetListResponse{Datasets: []DatasetResponse{*newTestDataset("a-set"), *newTestDataset("b-set")}})
	}))
	defer server.Close()

	got, err := newTestClient(t, server.URL).ListDatasets(context.Background())
	if err != nil {
		t.Fatalf("ListDatasets() error = %v", err)
	}
	if len(got) != 2 || GetDatasetID(got[1]) != "b-set" {
		t.Errorf("ListDatasets() = %+v, want 2 datasets ending in b-set", got)
	}
}

func TestListDatasetsIter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(DatasetListResponse{Datasets: []DatasetResponse{*newTestDataset("a-set")}})
	}))
	defer server.Close()

	it := newTestClient(t, server.URL).ListDatasetsIter(context.Background())
	if !it.Next() || GetDatasetID(it.Current()) != "a-set" {
		t.Fatalf("iterator did not yield a-set, err = %v", it.Err())
	}
	if it.Next() {
		t.Error("iterator yielded more than one dataset")
	}
}

func TestGetDataset(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertEqual(t, "method", r.Method, http.MethodGet)
		assertEqual(t, "path", r.URL.Path, "/api/datasets/staging")
		assertEqual(t, "authorization", r.Header.Get("Authorization"), "Bearer auth_test123")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(newTestDataset("staging"))
	}))
	defer server.Close()

	got, err := newTestClient(t, server.URL).GetDataset(context.Background(), "staging")
	if err != nil {
		t.Fatalf("GetDataset() error = %v", err)
	}
	assertEqual(t, "slug", GetDatasetID(got), "staging")
}

func TestGetDataset_404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeAPIError(w, http.StatusNotFound, "dataset not found")
	}))
	defer server.Close()

	got, err := newTestClient(t, server.URL).GetDataset(context.Background(), "missing")
	if got != nil {
		t.Errorf("GetDataset() = %+v, want nil", got)
	}
	if !IsNotFound(err) {
		t.Errorf("GetDataset() error = %v, want a not found error", err)
	}
}

func TestCreateDataset(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertEqual(t, "method", r.Method, http.MethodPost)
		assertEqual(t, "path", r.URL.Path, "/api/datasets")
		var body DatasetDefinition
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		assertEqual(t, "request slug", GetDatasetID(&body), "staging")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(newTestDataset("staging"))
	}))
	defer server.Close()

	got, err := newTestClient(t, server.URL).CreateDataset(context.Background(), newTestDataset("staging"))
	if err != nil {
		t.Fatalf("CreateDataset() error = %v", err)
	}
	assertEqual(t, "name", GetDatasetName(got), "Staging")
}

func TestCreateDataset_400(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeAPIError(w, http.StatusBadRequest, "slug already in use")
	}))
	defer server.Close()

	_, err := newTestClient(t, server.URL).CreateDataset(context.Background(), newTestDataset("staging"))
	if !IsBadRequest(err) {
		t.Errorf("CreateDataset() error = %v, want a bad request error", err)
	}
}

func TestCreateDataset_Nil(t *testing.T) {
	c := newTestClient(t, "http://127.0.0.1:1")
	if _, err := c.CreateDataset(context.Background(), nil); err == nil {
		t.Error("CreateDataset(nil) error = nil, want an error")
	}
}

func TestUpdateDataset(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertEqual(t, "method", r.Method, http.MethodPut)
		assertEqual(t, "path", r.URL.Path, "/api/datasets/tf-staging")
		var body DatasetDefinition
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		assertEqual(t, "request slug", GetDatasetID(&body), "staging")
		assertEqual(t, "request name", GetDatasetName(&body), "Staging")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(newTestDataset("staging"))
	}))
	defer server.Close()

	got, err := newTestClient(t, server.URL).UpdateDataset(context.Background(), "tf-staging", newTestDataset("staging"))
	if err != nil {
		t.Fatalf("UpdateDataset() error = %v", err)
	}
	assertEqual(t, "slug", GetDatasetID(got), "staging")
}

func TestUpdateDataset_404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeAPIError(w, http.StatusNotFound, "dataset not found")
	}))
	defer server.Close()

	_, err := newTestClient(t, server.URL).UpdateDataset(context.Background(), "missing", newTestDataset("missing"))
	if !IsNotFound(err) {
		t.Errorf("UpdateDataset() error = %v, want a not found error", err)
	}
}

func TestUpdateDataset_409(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeAPIError(w, http.StatusConflict, "version conflict")
	}))
	defer server.Close()

	_, err := newTestClient(t, server.URL).UpdateDataset(context.Background(), "staging", newTestDataset("staging"))
	if !IsConflict(err) {
		t.Errorf("UpdateDataset() error = %v, want a conflict error", err)
	}
}

func TestUpdateDataset_Nil(t *testing.T) {
	c := newTestClient(t, "http://127.0.0.1:1")
	if _, err := c.UpdateDataset(context.Background(), "staging", nil); err == nil {
		t.Error("UpdateDataset(nil) error = nil, want an error")
	}
}

func TestStripDatasetServerFields(t *testing.T) {
	now := time.Now()
	version := "3"
	source := CrdSource("api")
	origin := "tf-staging"
	actions := []DatasetAction{"dataset:read"}
	d := &DatasetDefinition{
		Metadata: DatasetMetadata{
			Annotations: &DatasetAnnotations{Dash0ComcreatedAt: &now},
			Labels: &DatasetLabels{
				Dash0Comid:      "staging",
				Dash0Comorigin:  &origin,
				Dash0Comversion: &version,
				Dash0Comsource:  &source,
			},
		},
		Spec: DatasetSpec{PermittedActions: &actions, TelemetryFilters: &[]TelemetryFilter{{}}},
	}

	StripDatasetServerFields(d)

	if d.Metadata.Annotations != nil || d.Metadata.Labels.Dash0Comversion != nil ||
		d.Metadata.Labels.Dash0Comsource != nil || d.Spec.PermittedActions != nil ||
		d.Spec.TelemetryFilters != nil {
		t.Errorf("server fields were not stripped: %+v", d)
	}
	assertEqual(t, "slug", d.Metadata.Labels.Dash0Comid, "staging")
	assertPtrEqual(t, "origin", d.Metadata.Labels.Dash0Comorigin, "tf-staging")
}

func TestDatasetHelpers(t *testing.T) {
	d := &DatasetDefinition{Metadata: DatasetMetadata{Name: "Staging"}}
	assertEqual(t, "id before set", GetDatasetID(d), "")

	SetDatasetIDIfAbsent(d, "first")
	SetDatasetIDIfAbsent(d, "second")
	assertEqual(t, "id after IfAbsent", GetDatasetID(d), "first")

	SetDatasetID(d, "third")
	assertEqual(t, "id after Set", GetDatasetID(d), "third")

	ClearDatasetID(d)
	assertEqual(t, "id after Clear", GetDatasetID(d), "")
	assertEqual(t, "name", GetDatasetName(d), "Staging")
}

func TestDatasetHelpers_Nil(t *testing.T) {
	StripDatasetServerFields(nil)
	SetDatasetID(nil, "x")
	SetDatasetIDIfAbsent(nil, "x")
	ClearDatasetID(nil)
	ClearDatasetID(&DatasetDefinition{})
	assertEqual(t, "id", GetDatasetID(nil), "")
	assertEqual(t, "name", GetDatasetName(nil), "")
}

func TestCreateDataset_201(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(newTestDataset("staging"))
	}))
	defer server.Close()

	got, err := newTestClient(t, server.URL).CreateDataset(context.Background(), newTestDataset("staging"))
	if err != nil {
		t.Fatalf("CreateDataset() error = %v", err)
	}
	assertEqual(t, "slug", GetDatasetID(got), "staging")
}

func TestListDatasets_403(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeAPIError(w, http.StatusForbidden, "no permission")
	}))
	defer server.Close()

	_, err := newTestClient(t, server.URL).ListDatasets(context.Background())
	if !IsForbidden(err) {
		t.Errorf("ListDatasets() error = %v, want a forbidden error", err)
	}
}

func TestUpdateDataset_403(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeAPIError(w, http.StatusForbidden, "no permission")
	}))
	defer server.Close()

	_, err := newTestClient(t, server.URL).UpdateDataset(context.Background(), "staging", newTestDataset("staging"))
	if !IsForbidden(err) {
		t.Errorf("UpdateDataset() error = %v, want a forbidden error", err)
	}
}

func TestGetDataset_EmptyBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	got, err := newTestClient(t, server.URL).GetDataset(context.Background(), "staging")
	if err == nil || got != nil {
		t.Errorf("GetDataset() = %+v, %v, want nil and an error", got, err)
	}
}

func TestDeleteDataset(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertEqual(t, "method", r.Method, http.MethodDelete)
		assertEqual(t, "path", r.URL.Path, "/api/datasets/staging")
		assertEqual(t, "authorization", r.Header.Get("Authorization"), "Bearer auth_test123")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	if err := newTestClient(t, server.URL).DeleteDataset(context.Background(), "staging"); err != nil {
		t.Fatalf("DeleteDataset() error = %v", err)
	}
}

func TestDeleteDataset_204(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	if err := newTestClient(t, server.URL).DeleteDataset(context.Background(), "staging"); err != nil {
		t.Fatalf("DeleteDataset() error = %v", err)
	}
}

// The server answers 200 for an unknown or already-deleted dataset, so this is not an error.
func TestDeleteDataset_Unknown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertEqual(t, "path", r.URL.Path, "/api/datasets/does-not-exist")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	if err := newTestClient(t, server.URL).DeleteDataset(context.Background(), "does-not-exist"); err != nil {
		t.Fatalf("DeleteDataset() error = %v, want nil", err)
	}
}

func TestDeleteDataset_Errors(t *testing.T) {
	tests := []struct {
		name   string
		status int
		check  func(error) bool
	}{
		{"400 for the default dataset", http.StatusBadRequest, IsBadRequest},
		{"401", http.StatusUnauthorized, IsUnauthorized},
		{"403", http.StatusForbidden, IsForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assertEqual(t, "path", r.URL.Path, "/api/datasets/default")
				writeAPIError(w, tt.status, "rejected")
			}))
			defer server.Close()

			err := newTestClient(t, server.URL).DeleteDataset(context.Background(), "default")
			if !tt.check(err) {
				t.Errorf("DeleteDataset() error = %v, want status %d", err, tt.status)
			}
		})
	}
}
