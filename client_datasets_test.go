package dash0

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestDataset(slug string) *DatasetDefinition {
	return &DatasetDefinition{
		ApiVersion: "dash0.com/v1alpha1",
		Kind:       Dash0Dataset,
		Metadata:   DatasetMetadata{Name: "Staging", Labels: &DatasetLabels{Dash0Comid: slug}},
	}
}

func TestListDatasets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertEqual(t, "method", r.Method, http.MethodGet)
		assertEqual(t, "path", r.URL.Path, "/api/datasets")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(DatasetListResponse{*newTestDataset("a-set"), *newTestDataset("b-set")})
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
		_ = json.NewEncoder(w).Encode(DatasetListResponse{*newTestDataset("a-set")})
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

func TestCreateDataset_Shape(t *testing.T) {
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		// The response carries an enum value this client has no constant for.
		_, _ = w.Write([]byte(`{"apiVersion":"dash0.com/v1alpha1","kind":"Dash0Dataset","metadata":{"name":"Staging"},` +
			`"spec":{"privacy":{"ipAddresses":{"endUser":{"storage":"hash"}}}}}`))
	}))
	defer server.Close()

	d := newTestDataset("staging")
	d.Spec = DatasetSpec{
		Retention: &DatasetRetention{Traces: Ptr(N30d)},
		Privacy: &DatasetPrivacy{
			IpAddresses: &DatasetIpAddressPrivacy{EndUser: &DatasetIpAddressStorage{Storage: Ptr(IpAddressStorageStrategyAnonymize)}},
			Redaction:   &DatasetRedaction{CardholderData: &CardholderDataRedactionSettings{Enabled: true}},
		},
		Processing: &DatasetProcessing{SpanEventsToLogs: Ptr(false)},
	}
	created, err := newTestClient(t, server.URL).CreateDataset(context.Background(), d)
	if err != nil {
		t.Fatalf("CreateDataset() error = %v", err)
	}

	assertEqual(t, "apiVersion", fmt.Sprint(got["apiVersion"]), "dash0.com/v1alpha1")
	spec := got["spec"].(map[string]any)
	// Fields left nil stay off the wire, which the server reads as "unchanged".
	if _, ok := spec["telemetryFilters"]; ok {
		t.Errorf("request spec carries telemetryFilters: %v", spec)
	}
	if _, ok := spec["preferred"]; ok {
		t.Errorf("request spec carries preferred: %v", spec)
	}
	assertEqual(t, "retention", fmt.Sprint(spec["retention"].(map[string]any)["traces"]), "30d")
	assertEqual(t, "spanEventsToLogs", fmt.Sprint(spec["processing"].(map[string]any)["spanEventsToLogs"]), "false")
	storage := spec["privacy"].(map[string]any)["ipAddresses"].(map[string]any)["endUser"].(map[string]any)["storage"]
	assertEqual(t, "ip storage", fmt.Sprint(storage), "anonymize")

	assertEqual(t, "ip storage in response", string(*created.Spec.Privacy.IpAddresses.EndUser.Storage), "hash")
}

func TestCreateDataset_Slug(t *testing.T) {
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		_, _ = w.Write([]byte(`{"apiVersion":"dash0.com/v1alpha1","kind":"Dash0Dataset","metadata":{"name":"Staging"},"spec":{"slug":"stg"}}`))
	}))
	defer server.Close()

	d := &DatasetDefinition{ApiVersion: "dash0.com/v1alpha1", Kind: Dash0Dataset, Metadata: DatasetMetadata{Name: "Staging"}}
	d.Spec.Slug = "stg"
	created, err := newTestClient(t, server.URL).CreateDataset(context.Background(), d)
	if err != nil {
		t.Fatalf("CreateDataset() error = %v", err)
	}

	assertEqual(t, "request slug", fmt.Sprint(got["spec"].(map[string]any)["slug"]), "stg")
	if _, ok := got["metadata"].(map[string]any)["labels"]; ok {
		t.Errorf("request carries labels: %v", got["metadata"])
	}
	assertEqual(t, "response slug", created.Spec.Slug, "stg")
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
		Spec: DatasetSpec{PermittedActions: &actions},
	}

	StripDatasetServerFields(d)

	if d.Metadata.Annotations != nil || d.Metadata.Labels.Dash0Comversion != nil ||
		d.Metadata.Labels.Dash0Comsource != nil || d.Spec.PermittedActions != nil {
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

func TestDatasetHelpers_SlugSources(t *testing.T) {
	onlySpec := func() *DatasetDefinition { return &DatasetDefinition{Spec: DatasetSpec{Slug: "from-spec"}} }
	onlyLabel := func() *DatasetDefinition {
		return &DatasetDefinition{Metadata: DatasetMetadata{Labels: &DatasetLabels{Dash0Comid: "from-label"}}}
	}
	both := func() *DatasetDefinition {
		return &DatasetDefinition{Spec: DatasetSpec{Slug: "staging"}, Metadata: DatasetMetadata{Labels: &DatasetLabels{Dash0Comid: "staging"}}}
	}
	for name, tc := range map[string]struct {
		dataset func() *DatasetDefinition
		want    string
	}{
		"only spec.slug": {onlySpec, "from-spec"},
		"only label":     {onlyLabel, "from-label"},
		"both":           {both, "staging"},
	} {
		t.Run(name, func(t *testing.T) {
			d := tc.dataset()
			assertEqual(t, "get", GetDatasetID(d), tc.want)

			SetDatasetIDIfAbsent(d, "other")
			assertEqual(t, "after IfAbsent", GetDatasetID(d), tc.want)
			if d.Metadata.Labels != nil && d.Metadata.Labels.Dash0Comid != "" && d.Metadata.Labels.Dash0Comid != tc.want {
				t.Errorf("IfAbsent wrote a label that does not match the slug: %q", d.Metadata.Labels.Dash0Comid)
			}

			SetDatasetID(d, "renamed")
			assertEqual(t, "spec.slug after Set", d.Spec.Slug, "renamed")
			assertEqual(t, "label after Set", d.Metadata.Labels.Dash0Comid, "renamed")

			ClearDatasetID(d)
			assertEqual(t, "spec.slug after Clear", d.Spec.Slug, "")
			assertEqual(t, "label after Clear", d.Metadata.Labels.Dash0Comid, "")
		})
	}
}

// A copy must not carry the source slug, or the PUT addresses the source dataset instead of creating a new one.
func TestDataset_CopyFlow(t *testing.T) {
	var sent map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"apiVersion":"dash0.com/v1alpha1","kind":"Dash0Dataset",` +
				`"metadata":{"name":"Staging","labels":{"dash0.com/id":"staging","dash0.com/origin":"tf-staging","dash0.com/version":"3"}},` +
				`"spec":{"slug":"staging"}}`))
			return
		}
		assertEqual(t, "path", r.URL.Path, "/api/datasets/tf-staging-copy")
		if err := json.NewDecoder(r.Body).Decode(&sent); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(newTestDataset("staging-copy"))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	d, err := c.GetDataset(context.Background(), "tf-staging")
	if err != nil {
		t.Fatalf("GetDataset() error = %v", err)
	}
	StripDatasetServerFields(d)
	ClearDatasetID(d)
	if _, err := c.UpdateDataset(context.Background(), "tf-staging-copy", d); err != nil {
		t.Fatalf("UpdateDataset() error = %v", err)
	}

	if _, ok := sent["spec"].(map[string]any)["slug"]; ok {
		t.Errorf("copy request carries spec.slug: %v", sent["spec"])
	}
	if _, ok := sent["metadata"].(map[string]any)["labels"].(map[string]any)["dash0.com/id"]; ok {
		t.Errorf("copy request carries dash0.com/id: %v", sent["metadata"])
	}
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
