package dash0

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

const (
	testIntegrationID      = "00000000-0000-0000-0000-000000000001"
	testIntegrationOrigin  = "tf_aws_production"
	testIntegrationRoleArn = "arn:aws:iam::123456789012:role/dash0-read-only"
	testIntegrationStack   = "arn:aws:cloudformation:eu-west-1:123456789012:stack/dash0-onboarding/11111111-1111-1111-1111-111111111111"
)

var testIntegrationTime = time.Date(2026, time.January, 15, 10, 0, 0, 0, time.UTC)

// newAwsIntegrationSpecFixture returns the client-settable part of the AWS
// integration fixture.
func newAwsIntegrationSpecFixture() AwsIntegrationSpec {
	return AwsIntegrationSpec{
		AccountId: "123456789012",
		Dataset:   "default",
		Roles: []AwsIntegrationRole{
			{
				Arn:            testIntegrationRoleArn,
				ExternalId:     "dash0-external-id-0001",
				PermissionType: AwsIntegrationRolePermissionTypeReadOnly,
			},
		},
	}
}

// newAwsIntegrationFixture returns an AWS integration as a caller would send
// it on create.
func newAwsIntegrationFixture() *IntegrationDefinition {
	return NewAwsIntegrationDefinition("aws-production", newAwsIntegrationSpecFixture())
}

// newAwsIntegrationResponseFixture returns an AWS integration as the server
// returns it, with every server-managed field populated.
func newAwsIntegrationResponseFixture() *IntegrationDefinition {
	integration := newAwsIntegrationFixture()
	source := CrdSource("terraform")
	integration.Metadata.Labels = &IntegrationLabels{
		Dash0Comid:      Ptr(testIntegrationID),
		Dash0Comorigin:  Ptr(testIntegrationOrigin),
		Dash0Comversion: Ptr("3"),
		Dash0Comsource:  &source,
	}
	integration.Metadata.Annotations = &IntegrationAnnotations{
		Dash0ComcreatedAt:     Ptr(testIntegrationTime),
		Dash0ComcreatedBy:     Ptr("00000000-0000-0000-0000-0000000000A1"),
		Dash0ComlastUpdatedAt: Ptr(testIntegrationTime),
		Dash0ComlastUpdatedBy: Ptr("00000000-0000-0000-0000-0000000000A1"),
	}
	spec := newAwsIntegrationSpecFixture()
	status := AwsVerificationStatusCompleted
	// Set by the Dash0 CloudFormation template, never by an API caller.
	spec.CloudFormationStackArn = Ptr(testIntegrationStack)
	spec.TemplateVersion = Ptr("1.4.0")
	spec.VerificationStatus = &status
	spec.VerificationError = Ptr("")
	spec.LastVerifiedAt = Ptr(testIntegrationTime)
	spec.IngestAuthTokenId = Ptr(openapi_types.UUID{0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22, 0x22})
	spec.DesiredRegions = &[]string{"eu-west-1"}
	spec.RegionalResources = &[]AwsRegionalResource{
		{
			Region:       "eu-west-1",
			LastSyncedAt: Ptr(testIntegrationTime),
			Firehose:     &AwsRegionalFirehoseDelivery{FirehoseArn: Ptr("arn:aws:firehose:eu-west-1:123456789012:deliverystream/dash0")},
		},
	}
	roleStatus := AwsIntegrationRoleStatusActive
	spec.Roles[0].Status = &roleStatus
	spec.Roles[0].LastAccessTimestamp = Ptr(testIntegrationTime)
	SetAwsIntegrationSpec(integration, spec)
	return integration
}

// nonAwsIntegrationJSON is an integration whose kind the OpenAPI spec does
// not document. The server can return such entries on list and get.
const nonAwsIntegrationJSON = `{
	"kind": "Dash0Integration",
	"metadata": {"name": "gcp-production", "labels": {"dash0.com/id": "00000000-0000-0000-0000-000000000002"}},
	"spec": {
		"enabled": true,
		"display": {"name": "gcp-production"},
		"ai": {"access": "none"},
		"integration": {"kind": "gcp", "spec": {"projectId": "my-project", "verificationStatus": "completed"}}
	}
}`

func newNonAwsIntegrationFixture(t *testing.T) *IntegrationDefinition {
	t.Helper()
	var integration IntegrationDefinition
	if err := json.Unmarshal([]byte(nonAwsIntegrationJSON), &integration); err != nil {
		t.Fatalf("unmarshal non-aws fixture: %v", err)
	}
	return &integration
}

// Helpers

func TestGetIntegrationID(t *testing.T) {
	tests := []struct {
		name        string
		integration *IntegrationDefinition
		want        string
	}{
		{"nil integration", nil, ""},
		{"nil labels", &IntegrationDefinition{}, ""},
		{"nil ID", &IntegrationDefinition{Metadata: IntegrationMetadata{Labels: &IntegrationLabels{}}}, ""},
		{"with ID", newAwsIntegrationResponseFixture(), testIntegrationID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, "GetIntegrationID", GetIntegrationID(tt.integration), tt.want)
		})
	}
}

func TestGetIntegrationName(t *testing.T) {
	assertEqual(t, "nil", GetIntegrationName(nil), "")
	assertEqual(t, "zero", GetIntegrationName(&IntegrationDefinition{}), "")
	assertEqual(t, "fixture", GetIntegrationName(newAwsIntegrationFixture()), "aws-production")
}

func TestGetIntegrationOrigin(t *testing.T) {
	assertEqual(t, "nil", GetIntegrationOrigin(nil), "")
	assertEqual(t, "nil labels", GetIntegrationOrigin(&IntegrationDefinition{}), "")
	assertEqual(t, "nil origin", GetIntegrationOrigin(&IntegrationDefinition{Metadata: IntegrationMetadata{Labels: &IntegrationLabels{}}}), "")
	assertEqual(t, "fixture", GetIntegrationOrigin(newAwsIntegrationResponseFixture()), testIntegrationOrigin)
}

func TestSetIntegrationID(t *testing.T) {
	t.Run("nil integration does not panic", func(t *testing.T) {
		SetIntegrationID(nil, testIntegrationID)
	})
	t.Run("initializes labels", func(t *testing.T) {
		integration := &IntegrationDefinition{}
		SetIntegrationID(integration, testIntegrationID)
		assertEqual(t, "id", GetIntegrationID(integration), testIntegrationID)
	})
	t.Run("overwrites existing ID", func(t *testing.T) {
		integration := newAwsIntegrationResponseFixture()
		SetIntegrationID(integration, "new-id")
		assertEqual(t, "id", GetIntegrationID(integration), "new-id")
	})
}

func TestSetIntegrationIDIfAbsent(t *testing.T) {
	t.Run("nil integration does not panic", func(t *testing.T) {
		SetIntegrationIDIfAbsent(nil, testIntegrationID)
	})
	t.Run("sets ID when absent", func(t *testing.T) {
		integration := &IntegrationDefinition{}
		SetIntegrationIDIfAbsent(integration, testIntegrationID)
		assertEqual(t, "id", GetIntegrationID(integration), testIntegrationID)
	})
	t.Run("keeps existing ID", func(t *testing.T) {
		integration := newAwsIntegrationResponseFixture()
		SetIntegrationIDIfAbsent(integration, "new-id")
		assertEqual(t, "id", GetIntegrationID(integration), testIntegrationID)
	})
}

func TestClearIntegrationID(t *testing.T) {
	t.Run("nil integration does not panic", func(t *testing.T) {
		ClearIntegrationID(nil)
	})
	t.Run("nil labels does not panic", func(t *testing.T) {
		ClearIntegrationID(&IntegrationDefinition{})
	})
	t.Run("clears ID", func(t *testing.T) {
		integration := newAwsIntegrationResponseFixture()
		ClearIntegrationID(integration)
		assertEqual(t, "id", GetIntegrationID(integration), "")
		assertEqual(t, "origin", GetIntegrationOrigin(integration), testIntegrationOrigin)
	})
}

func TestGetIntegrationKind(t *testing.T) {
	assertEqual(t, "nil", GetIntegrationKind(nil), "")
	assertEqual(t, "unset variant", GetIntegrationKind(&IntegrationDefinition{}), "")
	assertEqual(t, "aws", GetIntegrationKind(newAwsIntegrationFixture()), "aws")
	assertEqual(t, "non-aws", GetIntegrationKind(newNonAwsIntegrationFixture(t)), "gcp")
}

func TestGetAwsIntegrationSpec(t *testing.T) {
	t.Run("nil integration", func(t *testing.T) {
		if spec, ok := GetAwsIntegrationSpec(nil); ok || spec != nil {
			t.Errorf("expected (nil, false), got (%v, %v)", spec, ok)
		}
	})
	t.Run("unset variant", func(t *testing.T) {
		if _, ok := GetAwsIntegrationSpec(&IntegrationDefinition{}); ok {
			t.Error("expected false for an unset variant")
		}
	})
	t.Run("non-aws kind", func(t *testing.T) {
		if _, ok := GetAwsIntegrationSpec(newNonAwsIntegrationFixture(t)); ok {
			t.Error("expected false for a non-aws kind")
		}
	})
	t.Run("aws kind", func(t *testing.T) {
		spec, ok := GetAwsIntegrationSpec(newAwsIntegrationResponseFixture())
		if !ok {
			t.Fatal("expected true for an aws kind")
		}
		assertEqual(t, "accountId", spec.AccountId, "123456789012")
		assertEqual(t, "dataset", spec.Dataset, "default")
		if len(spec.Roles) != 1 {
			t.Fatalf("expected 1 role, got %d", len(spec.Roles))
		}
		assertEqual(t, "roles[0].arn", spec.Roles[0].Arn, testIntegrationRoleArn)
		assertEqual(t, "roles[0].permissionType", string(spec.Roles[0].PermissionType), "read_only")
		assertPtrEqual(t, "roles[0].status", spec.Roles[0].Status, AwsIntegrationRoleStatusActive)
	})
	t.Run("returned spec is a copy", func(t *testing.T) {
		integration := newAwsIntegrationFixture()
		spec, _ := GetAwsIntegrationSpec(integration)
		spec.AccountId = "999999999999"
		again, _ := GetAwsIntegrationSpec(integration)
		assertEqual(t, "accountId", again.AccountId, "123456789012")
	})
}

func TestSetAwsIntegrationSpec(t *testing.T) {
	t.Run("nil integration does not panic", func(t *testing.T) {
		SetAwsIntegrationSpec(nil, newAwsIntegrationSpecFixture())
	})
	t.Run("replaces a non-aws variant", func(t *testing.T) {
		integration := newNonAwsIntegrationFixture(t)
		SetAwsIntegrationSpec(integration, newAwsIntegrationSpecFixture())
		assertEqual(t, "kind", GetIntegrationKind(integration), "aws")
		spec, ok := GetAwsIntegrationSpec(integration)
		if !ok {
			t.Fatal("expected the aws variant")
		}
		assertEqual(t, "accountId", spec.AccountId, "123456789012")
	})
}

func TestNewAwsIntegrationDefinition(t *testing.T) {
	integration := NewAwsIntegrationDefinition("aws-production", newAwsIntegrationSpecFixture())
	assertEqual(t, "kind", string(integration.Kind), "Dash0Integration")
	assertEqual(t, "metadata.name", integration.Metadata.Name, "aws-production")
	assertEqual(t, "spec.display.name", integration.Spec.Display.Name, "aws-production")
	assertEqual(t, "spec.ai.access", string(integration.Spec.Ai.Access), "none")
	if !integration.Spec.Enabled {
		t.Error("expected spec.enabled to be true")
	}

	raw, err := json.Marshal(integration)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var wire map[string]any
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	variant := wire["spec"].(map[string]any)["integration"].(map[string]any)
	assertEqual(t, "spec.integration.kind", variant["kind"].(string), "aws")
	awsSpec := variant["spec"].(map[string]any)
	assertEqual(t, "spec.integration.spec.accountId", awsSpec["accountId"].(string), "123456789012")
	for _, key := range []string{"templateVersion", "verificationStatus", "lastVerifiedAt", "ingestAuthTokenId", "regionalResources", "desiredRegions"} {
		if _, present := awsSpec[key]; present {
			t.Errorf("expected %s to be omitted from the wire shape", key)
		}
	}
}

func TestStripIntegrationServerFields(t *testing.T) {
	t.Run("nil integration does not panic", func(t *testing.T) {
		StripIntegrationServerFields(nil)
	})
	t.Run("zero-value integration does not panic", func(t *testing.T) {
		StripIntegrationServerFields(&IntegrationDefinition{})
	})
	t.Run("full aws response", func(t *testing.T) {
		integration := newAwsIntegrationResponseFixture()
		StripIntegrationServerFields(integration)

		labels := integration.Metadata.Labels
		if labels.Dash0Comid != nil || labels.Dash0Comversion != nil || labels.Dash0Comsource != nil {
			t.Errorf("expected id, version, and source labels to be cleared, got %+v", labels)
		}
		assertPtrEqual(t, "dash0.com/origin", labels.Dash0Comorigin, testIntegrationOrigin)

		annotations := integration.Metadata.Annotations
		if annotations.Dash0ComcreatedAt != nil || annotations.Dash0ComcreatedBy != nil ||
			annotations.Dash0ComlastUpdatedAt != nil || annotations.Dash0ComlastUpdatedBy != nil {
			t.Errorf("expected all annotations to be cleared, got %+v", annotations)
		}

		spec, ok := GetAwsIntegrationSpec(integration)
		if !ok {
			t.Fatal("expected the aws variant to survive stripping")
		}
		if spec.TemplateVersion != nil || spec.VerificationStatus != nil || spec.VerificationError != nil ||
			spec.LastVerifiedAt != nil || spec.IngestAuthTokenId != nil || spec.RegionalResources != nil ||
			spec.DesiredRegions != nil {
			t.Errorf("expected server-managed AWS fields to be cleared, got %+v", spec)
		}
		if spec.Roles[0].Status != nil || spec.Roles[0].LastAccessTimestamp != nil {
			t.Errorf("expected role status and lastAccessTimestamp to be cleared, got %+v", spec.Roles[0])
		}

		// The stack ARN is preserved because an update without it clears it.
		assertPtrEqual(t, "cloudFormationStackArn", spec.CloudFormationStackArn, testIntegrationStack)

		// Client-settable fields are preserved, so the stripped response equals
		// what the caller sent on create plus the preserved origin and stack ARN.
		want := newAwsIntegrationFixture()
		want.Metadata.Labels = &IntegrationLabels{Dash0Comorigin: Ptr(testIntegrationOrigin)}
		wantSpec, _ := GetAwsIntegrationSpec(want)
		wantSpec.CloudFormationStackArn = Ptr(testIntegrationStack)
		SetAwsIntegrationSpec(want, *wantSpec)
		want.Metadata.Annotations = &IntegrationAnnotations{}
		gotJSON, _ := json.Marshal(integration)
		wantJSON, _ := json.Marshal(want)
		if string(gotJSON) != string(wantJSON) {
			t.Errorf("stripped integration mismatch\n got: %s\nwant: %s", gotJSON, wantJSON)
		}
	})
	t.Run("non-aws kind keeps its variant untouched", func(t *testing.T) {
		integration := newNonAwsIntegrationFixture(t)
		before, _ := integration.Spec.Integration.MarshalJSON()
		StripIntegrationServerFields(integration)
		after, _ := integration.Spec.Integration.MarshalJSON()
		if string(before) != string(after) {
			t.Errorf("expected non-aws variant to be untouched\nbefore: %s\n after: %s", before, after)
		}
		assertEqual(t, "id", GetIntegrationID(integration), "")
	})
}

func TestIntegration_MinimalWireShape(t *testing.T) {
	// Only the required fields; every omitempty field is unset.
	const minimal = `{
		"kind": "Dash0Integration",
		"metadata": {"name": "aws-minimal"},
		"spec": {
			"enabled": false,
			"display": {"name": "aws-minimal"},
			"ai": {"access": "read_only"},
			"integration": {"kind": "aws", "spec": {"accountId": "123456789012", "dataset": "default", "roles": []}}
		}
	}`
	var integration IntegrationDefinition
	if err := json.Unmarshal([]byte(minimal), &integration); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	assertEqual(t, "id", GetIntegrationID(&integration), "")
	assertEqual(t, "origin", GetIntegrationOrigin(&integration), "")
	assertEqual(t, "name", GetIntegrationName(&integration), "aws-minimal")
	spec, ok := GetAwsIntegrationSpec(&integration)
	if !ok {
		t.Fatal("expected the aws variant")
	}
	if spec.CloudFormationStackArn != nil || spec.VerificationStatus != nil || len(spec.Roles) != 0 {
		t.Errorf("expected optional fields to be unset, got %+v", spec)
	}

	StripIntegrationServerFields(&integration)
	if integration.Metadata.Labels != nil || integration.Metadata.Annotations != nil {
		t.Error("expected strip not to allocate labels or annotations")
	}
	raw, err := json.Marshal(&integration)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var wire map[string]any
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	metadata := wire["metadata"].(map[string]any)
	if _, present := metadata["labels"]; present {
		t.Error("expected metadata.labels to be omitted")
	}
	if _, present := metadata["annotations"]; present {
		t.Error("expected metadata.annotations to be omitted")
	}
}

// Client methods

func TestIntegrations_Integration(t *testing.T) {
	t.Run("ListIntegrations returns integrations including non-aws kinds", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/integrations" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.Method != http.MethodGet {
				t.Errorf("unexpected method: %s", r.Method)
			}
			if r.URL.Query().Has("dataset") {
				t.Errorf("unexpected dataset query parameter: %s", r.URL.RawQuery)
			}
			aws, _ := json.Marshal(newAwsIntegrationResponseFixture())
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"integrations":[` + string(aws) + `,` + nonAwsIntegrationJSON + `]}`))
		}))
		defer server.Close()

		got, err := newTestClient(t, server.URL).ListIntegrations(context.Background())
		if err != nil {
			t.Fatalf("ListIntegrations failed: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("expected 2 integrations, got %d", len(got))
		}
		assertEqual(t, "items[0].id", GetIntegrationID(got[0]), testIntegrationID)
		assertEqual(t, "items[0].kind", GetIntegrationKind(got[0]), "aws")
		assertEqual(t, "items[1].kind", GetIntegrationKind(got[1]), "gcp")
		if _, ok := GetAwsIntegrationSpec(got[1]); ok {
			t.Error("expected the non-aws entry not to decode as aws")
		}
	})

	t.Run("ListIntegrationsIter iterates over all integrations", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/integrations" || r.Method != http.MethodGet {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(GetIntegrationsResponse{
				Integrations: []IntegrationDefinition{*newAwsIntegrationResponseFixture()},
			})
		}))
		defer server.Close()

		iter := newTestClient(t, server.URL).ListIntegrationsIter(context.Background())
		var ids []string
		for iter.Next() {
			ids = append(ids, GetIntegrationID(iter.Current()))
		}
		if err := iter.Err(); err != nil {
			t.Fatalf("iterator failed: %v", err)
		}
		if len(ids) != 1 || ids[0] != testIntegrationID {
			t.Errorf("unexpected ids: %v", ids)
		}
	})

	t.Run("ListIntegrationsIter surfaces errors", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeAPIError(w, http.StatusForbidden, "forbidden")
		}))
		defer server.Close()

		iter := newTestClient(t, server.URL).ListIntegrationsIter(context.Background())
		if iter.Next() {
			t.Fatal("expected no items")
		}
		if !IsForbidden(iter.Err()) {
			t.Errorf("expected a forbidden error, got %v", iter.Err())
		}
	})

	t.Run("GetIntegration returns the envelope", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/integrations/"+testIntegrationOrigin {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.Method != http.MethodGet {
				t.Errorf("unexpected method: %s", r.Method)
			}
			if r.URL.RawQuery != "" {
				t.Errorf("unexpected query: %s", r.URL.RawQuery)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(newAwsIntegrationResponseFixture())
		}))
		defer server.Close()

		got, err := newTestClient(t, server.URL).GetIntegration(context.Background(), testIntegrationOrigin)
		if err != nil {
			t.Fatalf("GetIntegration failed: %v", err)
		}
		assertEqual(t, "id", GetIntegrationID(got), testIntegrationID)
		if got.Metadata.Annotations == nil || got.Metadata.Annotations.Dash0ComcreatedAt == nil ||
			!got.Metadata.Annotations.Dash0ComcreatedAt.Equal(testIntegrationTime) {
			t.Errorf("unexpected created-at annotation: %+v", got.Metadata.Annotations)
		}
		spec, ok := GetAwsIntegrationSpec(got)
		if !ok {
			t.Fatal("expected the aws variant")
		}
		assertPtrEqual(t, "verificationStatus", spec.VerificationStatus, AwsVerificationStatusCompleted)
	})

	t.Run("GetIntegration returns a non-aws kind without error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(nonAwsIntegrationJSON))
		}))
		defer server.Close()

		got, err := newTestClient(t, server.URL).GetIntegration(context.Background(), "gcp-production")
		if err != nil {
			t.Fatalf("GetIntegration failed: %v", err)
		}
		assertEqual(t, "kind", GetIntegrationKind(got), "gcp")
	})

	t.Run("GetIntegration maps 404 to IsNotFound", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeAPIError(w, http.StatusNotFound, "integration not found")
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).GetIntegration(context.Background(), "missing")
		if !IsNotFound(err) {
			t.Errorf("expected IsNotFound, got %v", err)
		}
	})

	t.Run("CreateIntegration POSTs the envelope", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/integrations" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.Method != http.MethodPost {
				t.Errorf("unexpected method: %s", r.Method)
			}
			var req IntegrationDefinition
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			spec, ok := GetAwsIntegrationSpec(&req)
			if !ok {
				t.Fatal("expected an aws request body")
			}
			assertEqual(t, "accountId", spec.AccountId, "123456789012")
			assertEqual(t, "roles[0].externalId", spec.Roles[0].ExternalId, "dash0-external-id-0001")
			if spec.Roles[0].Status != nil {
				t.Errorf("expected no role status on the request, got %v", *spec.Roles[0].Status)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(newAwsIntegrationResponseFixture())
		}))
		defer server.Close()

		got, err := newTestClient(t, server.URL).CreateIntegration(context.Background(), newAwsIntegrationFixture())
		if err != nil {
			t.Fatalf("CreateIntegration failed: %v", err)
		}
		assertEqual(t, "id", GetIntegrationID(got), testIntegrationID)
	})

	t.Run("CreateIntegration accepts 201", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(newAwsIntegrationResponseFixture())
		}))
		defer server.Close()

		got, err := newTestClient(t, server.URL).CreateIntegration(context.Background(), newAwsIntegrationFixture())
		if err != nil {
			t.Fatalf("CreateIntegration failed: %v", err)
		}
		assertEqual(t, "id", GetIntegrationID(got), testIntegrationID)
	})

	t.Run("CreateIntegration rejects nil integration", func(t *testing.T) {
		if _, err := newTestClient(t, "http://example.invalid").CreateIntegration(context.Background(), nil); err == nil {
			t.Fatal("expected error for nil integration")
		}
	})

	t.Run("CreateIntegration maps 403 to IsForbidden", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeAPIError(w, http.StatusForbidden, "admin access required")
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).CreateIntegration(context.Background(), newAwsIntegrationFixture())
		if !IsForbidden(err) {
			t.Errorf("expected IsForbidden, got %v", err)
		}
	})

	t.Run("CreateIntegration maps 400 to IsBadRequest", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeAPIError(w, http.StatusBadRequest, "accountId must have 12 digits")
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).CreateIntegration(context.Background(), newAwsIntegrationFixture())
		if !IsBadRequest(err) {
			t.Errorf("expected IsBadRequest, got %v", err)
		}
	})

	t.Run("UpsertIntegration PUTs to /api/integrations/{originOrId}", func(t *testing.T) {
		var gotPath, gotMethod string
		var gotBody []byte
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			gotMethod = r.Method
			gotBody, _ = io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(newAwsIntegrationResponseFixture())
		}))
		defer server.Close()

		integration := newAwsIntegrationFixture()
		got, err := newTestClient(t, server.URL).UpsertIntegration(context.Background(), testIntegrationOrigin, integration)
		if err != nil {
			t.Fatalf("UpsertIntegration failed: %v", err)
		}
		assertEqual(t, "path", gotPath, "/api/integrations/"+testIntegrationOrigin)
		assertEqual(t, "method", gotMethod, http.MethodPut)
		wantBody, _ := json.Marshal(integration)
		var gotWire, wantWire any
		if err := json.Unmarshal(gotBody, &gotWire); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}
		_ = json.Unmarshal(wantBody, &wantWire)
		gotNormalized, _ := json.Marshal(gotWire)
		wantNormalized, _ := json.Marshal(wantWire)
		if string(gotNormalized) != string(wantNormalized) {
			t.Errorf("unexpected PUT body\n got: %s\nwant: %s", gotNormalized, wantNormalized)
		}
		assertEqual(t, "id", GetIntegrationID(got), testIntegrationID)
	})

	t.Run("UpsertIntegration rejects nil integration", func(t *testing.T) {
		if _, err := newTestClient(t, "http://example.invalid").UpsertIntegration(context.Background(), "x", nil); err == nil {
			t.Fatal("expected error for nil integration")
		}
	})

	t.Run("UpsertIntegration maps 403 to IsForbidden", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeAPIError(w, http.StatusForbidden, "admin access required")
		}))
		defer server.Close()

		_, err := newTestClient(t, server.URL).UpsertIntegration(context.Background(), testIntegrationOrigin, newAwsIntegrationFixture())
		if !IsForbidden(err) {
			t.Errorf("expected IsForbidden, got %v", err)
		}
	})

	t.Run("DeleteIntegration DELETEs /api/integrations/{originOrId}", func(t *testing.T) {
		for _, status := range []int{http.StatusOK, http.StatusNoContent} {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/integrations/"+testIntegrationID {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}
				if r.Method != http.MethodDelete {
					t.Errorf("unexpected method: %s", r.Method)
				}
				w.WriteHeader(status)
			}))
			if err := newTestClient(t, server.URL).DeleteIntegration(context.Background(), testIntegrationID); err != nil {
				t.Errorf("DeleteIntegration with status %d failed: %v", status, err)
			}
			server.Close()
		}
	})

	t.Run("DeleteIntegration maps 404 to IsNotFound", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeAPIError(w, http.StatusNotFound, "integration not found")
		}))
		defer server.Close()

		if err := newTestClient(t, server.URL).DeleteIntegration(context.Background(), "missing"); !IsNotFound(err) {
			t.Errorf("expected IsNotFound, got %v", err)
		}
	})
}
