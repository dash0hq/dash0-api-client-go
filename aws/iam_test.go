package aws

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// mockIAM implements iamAPI for testing.
type mockIAM struct {
	CreateRoleFunc               func(ctx context.Context, params *iam.CreateRoleInput, optFns ...func(*iam.Options)) (*iam.CreateRoleOutput, error)
	GetRoleFunc                  func(ctx context.Context, params *iam.GetRoleInput, optFns ...func(*iam.Options)) (*iam.GetRoleOutput, error)
	DeleteRoleFunc               func(ctx context.Context, params *iam.DeleteRoleInput, optFns ...func(*iam.Options)) (*iam.DeleteRoleOutput, error)
	AttachRolePolicyFunc         func(ctx context.Context, params *iam.AttachRolePolicyInput, optFns ...func(*iam.Options)) (*iam.AttachRolePolicyOutput, error)
	DetachRolePolicyFunc         func(ctx context.Context, params *iam.DetachRolePolicyInput, optFns ...func(*iam.Options)) (*iam.DetachRolePolicyOutput, error)
	PutRolePolicyFunc            func(ctx context.Context, params *iam.PutRolePolicyInput, optFns ...func(*iam.Options)) (*iam.PutRolePolicyOutput, error)
	DeleteRolePolicyFunc         func(ctx context.Context, params *iam.DeleteRolePolicyInput, optFns ...func(*iam.Options)) (*iam.DeleteRolePolicyOutput, error)
	CreatePolicyFunc             func(ctx context.Context, params *iam.CreatePolicyInput, optFns ...func(*iam.Options)) (*iam.CreatePolicyOutput, error)
	DeletePolicyFunc             func(ctx context.Context, params *iam.DeletePolicyInput, optFns ...func(*iam.Options)) (*iam.DeletePolicyOutput, error)
	ListAttachedRolePoliciesFunc func(ctx context.Context, params *iam.ListAttachedRolePoliciesInput, optFns ...func(*iam.Options)) (*iam.ListAttachedRolePoliciesOutput, error)
	ListRolePoliciesFunc         func(ctx context.Context, params *iam.ListRolePoliciesInput, optFns ...func(*iam.Options)) (*iam.ListRolePoliciesOutput, error)
	ListRoleTagsFunc             func(ctx context.Context, params *iam.ListRoleTagsInput, optFns ...func(*iam.Options)) (*iam.ListRoleTagsOutput, error)
	TagRoleFunc                  func(ctx context.Context, params *iam.TagRoleInput, optFns ...func(*iam.Options)) (*iam.TagRoleOutput, error)
	UntagRoleFunc                func(ctx context.Context, params *iam.UntagRoleInput, optFns ...func(*iam.Options)) (*iam.UntagRoleOutput, error)
}

func (m *mockIAM) CreateRole(ctx context.Context, params *iam.CreateRoleInput, optFns ...func(*iam.Options)) (*iam.CreateRoleOutput, error) {
	return m.CreateRoleFunc(ctx, params, optFns...)
}
func (m *mockIAM) GetRole(ctx context.Context, params *iam.GetRoleInput, optFns ...func(*iam.Options)) (*iam.GetRoleOutput, error) {
	return m.GetRoleFunc(ctx, params, optFns...)
}
func (m *mockIAM) DeleteRole(ctx context.Context, params *iam.DeleteRoleInput, optFns ...func(*iam.Options)) (*iam.DeleteRoleOutput, error) {
	return m.DeleteRoleFunc(ctx, params, optFns...)
}
func (m *mockIAM) AttachRolePolicy(ctx context.Context, params *iam.AttachRolePolicyInput, optFns ...func(*iam.Options)) (*iam.AttachRolePolicyOutput, error) {
	return m.AttachRolePolicyFunc(ctx, params, optFns...)
}
func (m *mockIAM) DetachRolePolicy(ctx context.Context, params *iam.DetachRolePolicyInput, optFns ...func(*iam.Options)) (*iam.DetachRolePolicyOutput, error) {
	return m.DetachRolePolicyFunc(ctx, params, optFns...)
}
func (m *mockIAM) PutRolePolicy(ctx context.Context, params *iam.PutRolePolicyInput, optFns ...func(*iam.Options)) (*iam.PutRolePolicyOutput, error) {
	return m.PutRolePolicyFunc(ctx, params, optFns...)
}
func (m *mockIAM) DeleteRolePolicy(ctx context.Context, params *iam.DeleteRolePolicyInput, optFns ...func(*iam.Options)) (*iam.DeleteRolePolicyOutput, error) {
	return m.DeleteRolePolicyFunc(ctx, params, optFns...)
}
func (m *mockIAM) CreatePolicy(ctx context.Context, params *iam.CreatePolicyInput, optFns ...func(*iam.Options)) (*iam.CreatePolicyOutput, error) {
	return m.CreatePolicyFunc(ctx, params, optFns...)
}
func (m *mockIAM) DeletePolicy(ctx context.Context, params *iam.DeletePolicyInput, optFns ...func(*iam.Options)) (*iam.DeletePolicyOutput, error) {
	return m.DeletePolicyFunc(ctx, params, optFns...)
}
func (m *mockIAM) ListAttachedRolePolicies(ctx context.Context, params *iam.ListAttachedRolePoliciesInput, optFns ...func(*iam.Options)) (*iam.ListAttachedRolePoliciesOutput, error) {
	return m.ListAttachedRolePoliciesFunc(ctx, params, optFns...)
}
func (m *mockIAM) ListRolePolicies(ctx context.Context, params *iam.ListRolePoliciesInput, optFns ...func(*iam.Options)) (*iam.ListRolePoliciesOutput, error) {
	return m.ListRolePoliciesFunc(ctx, params, optFns...)
}
func (m *mockIAM) ListRoleTags(ctx context.Context, params *iam.ListRoleTagsInput, optFns ...func(*iam.Options)) (*iam.ListRoleTagsOutput, error) {
	return m.ListRoleTagsFunc(ctx, params, optFns...)
}
func (m *mockIAM) TagRole(ctx context.Context, params *iam.TagRoleInput, optFns ...func(*iam.Options)) (*iam.TagRoleOutput, error) {
	return m.TagRoleFunc(ctx, params, optFns...)
}
func (m *mockIAM) UntagRole(ctx context.Context, params *iam.UntagRoleInput, optFns ...func(*iam.Options)) (*iam.UntagRoleOutput, error) {
	return m.UntagRoleFunc(ctx, params, optFns...)
}

// mockSTS implements stsAPI for testing.
type mockSTS struct {
	GetCallerIdentityFunc func(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error)
}

func (m *mockSTS) GetCallerIdentity(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
	return m.GetCallerIdentityFunc(ctx, params, optFns...)
}

var errTest = errors.New("test error")

func noSuchEntity() error {
	return &iamtypes.NoSuchEntityException{Message: aws.String("not found")}
}

func newTestClient(iamMock *mockIAM, stsMock *mockSTS) *IAMClient {
	return &IAMClient{iam: iamMock, sts: stsMock}
}

// defaultMockIAM returns a mockIAM with all methods stubbed to succeed with empty responses.
func defaultMockIAM() *mockIAM {
	return &mockIAM{
		CreateRoleFunc: func(ctx context.Context, params *iam.CreateRoleInput, optFns ...func(*iam.Options)) (*iam.CreateRoleOutput, error) {
			return &iam.CreateRoleOutput{Role: &iamtypes.Role{Arn: aws.String("arn:aws:iam::123:role/test")}}, nil
		},
		GetRoleFunc: func(ctx context.Context, params *iam.GetRoleInput, optFns ...func(*iam.Options)) (*iam.GetRoleOutput, error) {
			return &iam.GetRoleOutput{Role: &iamtypes.Role{Arn: aws.String("arn:aws:iam::123:role/test")}}, nil
		},
		DeleteRoleFunc: func(ctx context.Context, params *iam.DeleteRoleInput, optFns ...func(*iam.Options)) (*iam.DeleteRoleOutput, error) {
			return &iam.DeleteRoleOutput{}, nil
		},
		AttachRolePolicyFunc: func(ctx context.Context, params *iam.AttachRolePolicyInput, optFns ...func(*iam.Options)) (*iam.AttachRolePolicyOutput, error) {
			return &iam.AttachRolePolicyOutput{}, nil
		},
		DetachRolePolicyFunc: func(ctx context.Context, params *iam.DetachRolePolicyInput, optFns ...func(*iam.Options)) (*iam.DetachRolePolicyOutput, error) {
			return &iam.DetachRolePolicyOutput{}, nil
		},
		PutRolePolicyFunc: func(ctx context.Context, params *iam.PutRolePolicyInput, optFns ...func(*iam.Options)) (*iam.PutRolePolicyOutput, error) {
			return &iam.PutRolePolicyOutput{}, nil
		},
		DeleteRolePolicyFunc: func(ctx context.Context, params *iam.DeleteRolePolicyInput, optFns ...func(*iam.Options)) (*iam.DeleteRolePolicyOutput, error) {
			return &iam.DeleteRolePolicyOutput{}, nil
		},
		CreatePolicyFunc: func(ctx context.Context, params *iam.CreatePolicyInput, optFns ...func(*iam.Options)) (*iam.CreatePolicyOutput, error) {
			return &iam.CreatePolicyOutput{Policy: &iamtypes.Policy{Arn: aws.String("arn:aws:iam::123:policy/test")}}, nil
		},
		DeletePolicyFunc: func(ctx context.Context, params *iam.DeletePolicyInput, optFns ...func(*iam.Options)) (*iam.DeletePolicyOutput, error) {
			return &iam.DeletePolicyOutput{}, nil
		},
		ListAttachedRolePoliciesFunc: func(ctx context.Context, params *iam.ListAttachedRolePoliciesInput, optFns ...func(*iam.Options)) (*iam.ListAttachedRolePoliciesOutput, error) {
			return &iam.ListAttachedRolePoliciesOutput{}, nil
		},
		ListRolePoliciesFunc: func(ctx context.Context, params *iam.ListRolePoliciesInput, optFns ...func(*iam.Options)) (*iam.ListRolePoliciesOutput, error) {
			return &iam.ListRolePoliciesOutput{}, nil
		},
		ListRoleTagsFunc: func(ctx context.Context, params *iam.ListRoleTagsInput, optFns ...func(*iam.Options)) (*iam.ListRoleTagsOutput, error) {
			return &iam.ListRoleTagsOutput{}, nil
		},
		TagRoleFunc: func(ctx context.Context, params *iam.TagRoleInput, optFns ...func(*iam.Options)) (*iam.TagRoleOutput, error) {
			return &iam.TagRoleOutput{}, nil
		},
		UntagRoleFunc: func(ctx context.Context, params *iam.UntagRoleInput, optFns ...func(*iam.Options)) (*iam.UntagRoleOutput, error) {
			return &iam.UntagRoleOutput{}, nil
		},
	}
}

var testParams = RoleParams{
	RoleNamePrefix:    "dash0",
	Dash0AwsAccountID: "111111111111",
	ExternalID:        "ext-123",
	Tags:              map[string]string{"env": "test"},
}

// --- Naming helper tests ---

func TestReadOnlyRoleName(t *testing.T) {
	tests := []struct {
		prefix string
		want   string
	}{
		{"dash0", "dash0-read-only"},
		{"my-org", "my-org-read-only"},
		{"", "-read-only"},
	}
	for _, tt := range tests {
		if got := ReadOnlyRoleName(tt.prefix); got != tt.want {
			t.Errorf("ReadOnlyRoleName(%q) = %q, want %q", tt.prefix, got, tt.want)
		}
	}
}

func TestInstrumentationRoleName(t *testing.T) {
	tests := []struct {
		prefix string
		want   string
	}{
		{"dash0", "dash0-instrumentation"},
		{"my-org", "my-org-instrumentation"},
		{"", "-instrumentation"},
	}
	for _, tt := range tests {
		if got := InstrumentationRoleName(tt.prefix); got != tt.want {
			t.Errorf("InstrumentationRoleName(%q) = %q, want %q", tt.prefix, got, tt.want)
		}
	}
}

func TestInstrumentationPolicyName(t *testing.T) {
	tests := []struct {
		prefix string
		want   string
	}{
		{"dash0", "dash0-lambda-instrumentation"},
		{"my-org", "my-org-lambda-instrumentation"},
		{"", "-lambda-instrumentation"},
	}
	for _, tt := range tests {
		if got := InstrumentationPolicyName(tt.prefix); got != tt.want {
			t.Errorf("InstrumentationPolicyName(%q) = %q, want %q", tt.prefix, got, tt.want)
		}
	}
}

// --- buildTrustPolicy tests ---

func TestBuildTrustPolicy(t *testing.T) {
	accountID := "123456789012"
	externalID := "ext-abc-123"

	policyJSON, err := buildTrustPolicy(accountID, externalID)
	if err != nil {
		t.Fatalf("buildTrustPolicy() error = %v", err)
	}

	var policy map[string]interface{}
	if err := json.Unmarshal([]byte(policyJSON), &policy); err != nil {
		t.Fatalf("failed to unmarshal trust policy: %v", err)
	}

	if policy["Version"] != "2012-10-17" {
		t.Errorf("Version = %v, want %q", policy["Version"], "2012-10-17")
	}

	statements, ok := policy["Statement"].([]interface{})
	if !ok || len(statements) != 1 {
		t.Fatalf("expected 1 statement, got %v", policy["Statement"])
	}

	stmt := statements[0].(map[string]interface{})

	if stmt["Effect"] != "Allow" {
		t.Errorf("Effect = %v, want %q", stmt["Effect"], "Allow")
	}
	if stmt["Action"] != "sts:AssumeRole" {
		t.Errorf("Action = %v, want %q", stmt["Action"], "sts:AssumeRole")
	}

	principal := stmt["Principal"].(map[string]interface{})
	if principal["AWS"] != accountID {
		t.Errorf("Principal.AWS = %v, want %q", principal["AWS"], accountID)
	}

	condition := stmt["Condition"].(map[string]interface{})
	stringEquals := condition["StringEquals"].(map[string]interface{})
	if stringEquals["sts:ExternalId"] != externalID {
		t.Errorf("Condition.StringEquals.sts:ExternalId = %v, want %q", stringEquals["sts:ExternalId"], externalID)
	}
}

// --- convertTags tests ---

func TestConvertTags(t *testing.T) {
	t.Run("nil map", func(t *testing.T) {
		tags := convertTags(nil)
		if len(tags) != 0 {
			t.Errorf("convertTags(nil) returned %d tags, want 0", len(tags))
		}
	})

	t.Run("empty map", func(t *testing.T) {
		tags := convertTags(map[string]string{})
		if len(tags) != 0 {
			t.Errorf("convertTags(empty) returned %d tags, want 0", len(tags))
		}
	})

	t.Run("multiple tags", func(t *testing.T) {
		input := map[string]string{
			"env":     "prod",
			"team":    "platform",
			"project": "dash0",
		}
		tags := convertTags(input)
		if len(tags) != 3 {
			t.Fatalf("convertTags() returned %d tags, want 3", len(tags))
		}

		// Sort by key for deterministic comparison.
		sort.Slice(tags, func(i, j int) bool {
			return *tags[i].Key < *tags[j].Key
		})

		expectedKeys := []string{"env", "project", "team"}
		expectedValues := []string{"prod", "dash0", "platform"}
		for i, tag := range tags {
			if *tag.Key != expectedKeys[i] {
				t.Errorf("tag[%d].Key = %q, want %q", i, *tag.Key, expectedKeys[i])
			}
			if *tag.Value != expectedValues[i] {
				t.Errorf("tag[%d].Value = %q, want %q", i, *tag.Value, expectedValues[i])
			}
		}
	})
}

// --- GetCallerAccountID tests ---

func TestGetCallerAccountID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := newTestClient(defaultMockIAM(), &mockSTS{
			GetCallerIdentityFunc: func(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
				return &sts.GetCallerIdentityOutput{Account: aws.String("123456789012")}, nil
			},
		})
		id, err := client.GetCallerAccountID(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != "123456789012" {
			t.Errorf("got %q, want %q", id, "123456789012")
		}
	})

	t.Run("error", func(t *testing.T) {
		client := newTestClient(defaultMockIAM(), &mockSTS{
			GetCallerIdentityFunc: func(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
				return nil, errTest
			},
		})
		_, err := client.GetCallerAccountID(context.Background())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

// --- CreateReadOnlyRole tests ---

func TestCreateReadOnlyRole(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := defaultMockIAM()
		mock.CreateRoleFunc = func(ctx context.Context, params *iam.CreateRoleInput, optFns ...func(*iam.Options)) (*iam.CreateRoleOutput, error) {
			return &iam.CreateRoleOutput{Role: &iamtypes.Role{
				Arn: aws.String("arn:aws:iam::123:role/dash0-read-only"),
			}}, nil
		}
		client := newTestClient(mock, nil)

		info, err := client.CreateReadOnlyRole(context.Background(), testParams)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.RoleName != "dash0-read-only" {
			t.Errorf("RoleName = %q, want %q", info.RoleName, "dash0-read-only")
		}
		if info.RoleArn != "arn:aws:iam::123:role/dash0-read-only" {
			t.Errorf("RoleArn = %q, want %q", info.RoleArn, "arn:aws:iam::123:role/dash0-read-only")
		}
	})

	t.Run("fails when role already exists", func(t *testing.T) {
		mock := defaultMockIAM()
		mock.CreateRoleFunc = func(ctx context.Context, params *iam.CreateRoleInput, optFns ...func(*iam.Options)) (*iam.CreateRoleOutput, error) {
			return nil, &iamtypes.EntityAlreadyExistsException{Message: aws.String("already exists")}
		}
		client := newTestClient(mock, nil)

		_, err := client.CreateReadOnlyRole(context.Background(), testParams)
		if err == nil {
			t.Fatal("expected error when role already exists, got nil")
		}
	})

	t.Run("create role fails with other error", func(t *testing.T) {
		mock := defaultMockIAM()
		mock.CreateRoleFunc = func(ctx context.Context, params *iam.CreateRoleInput, optFns ...func(*iam.Options)) (*iam.CreateRoleOutput, error) {
			return nil, errTest
		}
		client := newTestClient(mock, nil)

		_, err := client.CreateReadOnlyRole(context.Background(), testParams)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("attach ViewOnlyAccess fails triggers cleanup", func(t *testing.T) {
		cleanupCalled := false
		mock := defaultMockIAM()
		mock.AttachRolePolicyFunc = func(ctx context.Context, params *iam.AttachRolePolicyInput, optFns ...func(*iam.Options)) (*iam.AttachRolePolicyOutput, error) {
			return nil, errTest
		}
		origDeleteRole := mock.DeleteRoleFunc
		mock.DeleteRoleFunc = func(ctx context.Context, params *iam.DeleteRoleInput, optFns ...func(*iam.Options)) (*iam.DeleteRoleOutput, error) {
			cleanupCalled = true
			return origDeleteRole(ctx, params, optFns...)
		}
		client := newTestClient(mock, nil)

		_, err := client.CreateReadOnlyRole(context.Background(), testParams)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !cleanupCalled {
			t.Error("expected cleanup to be called")
		}
	})

	t.Run("put inline policy fails triggers cleanup", func(t *testing.T) {
		cleanupCalled := false
		mock := defaultMockIAM()
		mock.PutRolePolicyFunc = func(ctx context.Context, params *iam.PutRolePolicyInput, optFns ...func(*iam.Options)) (*iam.PutRolePolicyOutput, error) {
			return nil, errTest
		}
		origDeleteRole := mock.DeleteRoleFunc
		mock.DeleteRoleFunc = func(ctx context.Context, params *iam.DeleteRoleInput, optFns ...func(*iam.Options)) (*iam.DeleteRoleOutput, error) {
			cleanupCalled = true
			return origDeleteRole(ctx, params, optFns...)
		}
		client := newTestClient(mock, nil)

		_, err := client.CreateReadOnlyRole(context.Background(), testParams)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !cleanupCalled {
			t.Error("expected cleanup to be called")
		}
	})
}

// --- CreateInstrumentationRole tests ---

func TestCreateInstrumentationRole(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := defaultMockIAM()
		mock.CreateRoleFunc = func(ctx context.Context, params *iam.CreateRoleInput, optFns ...func(*iam.Options)) (*iam.CreateRoleOutput, error) {
			return &iam.CreateRoleOutput{Role: &iamtypes.Role{
				Arn: aws.String("arn:aws:iam::123:role/dash0-instrumentation"),
			}}, nil
		}
		client := newTestClient(mock, nil)

		info, err := client.CreateInstrumentationRole(context.Background(), testParams)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.RoleName != "dash0-instrumentation" {
			t.Errorf("RoleName = %q, want %q", info.RoleName, "dash0-instrumentation")
		}
	})

	t.Run("fails when role already exists", func(t *testing.T) {
		mock := defaultMockIAM()
		mock.CreateRoleFunc = func(ctx context.Context, params *iam.CreateRoleInput, optFns ...func(*iam.Options)) (*iam.CreateRoleOutput, error) {
			return nil, &iamtypes.EntityAlreadyExistsException{Message: aws.String("already exists")}
		}
		client := newTestClient(mock, nil)

		_, err := client.CreateInstrumentationRole(context.Background(), testParams)
		if err == nil {
			t.Fatal("expected error when role already exists, got nil")
		}
	})

	t.Run("create policy fails triggers cleanup", func(t *testing.T) {
		cleanupCalled := false
		mock := defaultMockIAM()
		mock.CreatePolicyFunc = func(ctx context.Context, params *iam.CreatePolicyInput, optFns ...func(*iam.Options)) (*iam.CreatePolicyOutput, error) {
			return nil, errTest
		}
		origDeleteRole := mock.DeleteRoleFunc
		mock.DeleteRoleFunc = func(ctx context.Context, params *iam.DeleteRoleInput, optFns ...func(*iam.Options)) (*iam.DeleteRoleOutput, error) {
			cleanupCalled = true
			return origDeleteRole(ctx, params, optFns...)
		}
		client := newTestClient(mock, nil)

		_, err := client.CreateInstrumentationRole(context.Background(), testParams)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !cleanupCalled {
			t.Error("expected role cleanup to be called")
		}
	})

	t.Run("attach policy fails triggers cleanup of both policy and role", func(t *testing.T) {
		roleDeleted := false
		policyDeleted := false
		mock := defaultMockIAM()
		mock.AttachRolePolicyFunc = func(ctx context.Context, params *iam.AttachRolePolicyInput, optFns ...func(*iam.Options)) (*iam.AttachRolePolicyOutput, error) {
			return nil, errTest
		}
		mock.DeleteRoleFunc = func(ctx context.Context, params *iam.DeleteRoleInput, optFns ...func(*iam.Options)) (*iam.DeleteRoleOutput, error) {
			roleDeleted = true
			return &iam.DeleteRoleOutput{}, nil
		}
		mock.DeletePolicyFunc = func(ctx context.Context, params *iam.DeletePolicyInput, optFns ...func(*iam.Options)) (*iam.DeletePolicyOutput, error) {
			policyDeleted = true
			return &iam.DeletePolicyOutput{}, nil
		}
		client := newTestClient(mock, nil)

		_, err := client.CreateInstrumentationRole(context.Background(), testParams)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !roleDeleted {
			t.Error("expected role cleanup to be called")
		}
		if !policyDeleted {
			t.Error("expected policy cleanup to be called")
		}
	})
}

// --- ReadRole tests ---

func TestReadRole(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := defaultMockIAM()
		mock.GetRoleFunc = func(ctx context.Context, params *iam.GetRoleInput, optFns ...func(*iam.Options)) (*iam.GetRoleOutput, error) {
			return &iam.GetRoleOutput{Role: &iamtypes.Role{
				Arn: aws.String("arn:aws:iam::123:role/my-role"),
			}}, nil
		}
		client := newTestClient(mock, nil)

		info, err := client.ReadRole(context.Background(), "my-role")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.RoleName != "my-role" {
			t.Errorf("RoleName = %q, want %q", info.RoleName, "my-role")
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock := defaultMockIAM()
		mock.GetRoleFunc = func(ctx context.Context, params *iam.GetRoleInput, optFns ...func(*iam.Options)) (*iam.GetRoleOutput, error) {
			return nil, noSuchEntity()
		}
		client := newTestClient(mock, nil)

		_, err := client.ReadRole(context.Background(), "missing-role")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

// --- DeleteReadOnlyRole tests ---

func TestDeleteReadOnlyRole(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := newTestClient(defaultMockIAM(), nil)
		err := client.DeleteReadOnlyRole(context.Background(), "dash0")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("idempotent when role does not exist", func(t *testing.T) {
		mock := defaultMockIAM()
		mock.GetRoleFunc = func(ctx context.Context, params *iam.GetRoleInput, optFns ...func(*iam.Options)) (*iam.GetRoleOutput, error) {
			return nil, noSuchEntity()
		}
		deleteCalled := false
		mock.DeleteRoleFunc = func(ctx context.Context, params *iam.DeleteRoleInput, optFns ...func(*iam.Options)) (*iam.DeleteRoleOutput, error) {
			deleteCalled = true
			return &iam.DeleteRoleOutput{}, nil
		}
		client := newTestClient(mock, nil)

		err := client.DeleteReadOnlyRole(context.Background(), "dash0")
		if err != nil {
			t.Fatalf("expected nil for idempotent delete, got: %v", err)
		}
		if deleteCalled {
			t.Error("delete should not be attempted when role does not exist")
		}
	})

	t.Run("fails when GetRole returns real error", func(t *testing.T) {
		mock := defaultMockIAM()
		mock.GetRoleFunc = func(ctx context.Context, params *iam.GetRoleInput, optFns ...func(*iam.Options)) (*iam.GetRoleOutput, error) {
			return nil, errTest
		}
		client := newTestClient(mock, nil)

		err := client.DeleteReadOnlyRole(context.Background(), "dash0")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest in chain, got: %v", err)
		}
	})

	t.Run("fails when delete inline policy returns real error", func(t *testing.T) {
		mock := defaultMockIAM()
		mock.DeleteRolePolicyFunc = func(ctx context.Context, params *iam.DeleteRolePolicyInput, optFns ...func(*iam.Options)) (*iam.DeleteRolePolicyOutput, error) {
			return nil, errTest
		}
		client := newTestClient(mock, nil)

		err := client.DeleteReadOnlyRole(context.Background(), "dash0")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest in chain, got: %v", err)
		}
	})

	t.Run("fails when detach managed policy returns real error", func(t *testing.T) {
		mock := defaultMockIAM()
		mock.DetachRolePolicyFunc = func(ctx context.Context, params *iam.DetachRolePolicyInput, optFns ...func(*iam.Options)) (*iam.DetachRolePolicyOutput, error) {
			return nil, errTest
		}
		client := newTestClient(mock, nil)

		err := client.DeleteReadOnlyRole(context.Background(), "dash0")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest in chain, got: %v", err)
		}
	})

	t.Run("fails when delete role returns real error", func(t *testing.T) {
		mock := defaultMockIAM()
		mock.DeleteRoleFunc = func(ctx context.Context, params *iam.DeleteRoleInput, optFns ...func(*iam.Options)) (*iam.DeleteRoleOutput, error) {
			return nil, errTest
		}
		client := newTestClient(mock, nil)

		err := client.DeleteReadOnlyRole(context.Background(), "dash0")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest in chain, got: %v", err)
		}
	})

	t.Run("recovers from partial delete where inline policy already removed", func(t *testing.T) {
		mock := defaultMockIAM()
		mock.DeleteRolePolicyFunc = func(ctx context.Context, params *iam.DeleteRolePolicyInput, optFns ...func(*iam.Options)) (*iam.DeleteRolePolicyOutput, error) {
			return nil, noSuchEntity()
		}
		client := newTestClient(mock, nil)

		err := client.DeleteReadOnlyRole(context.Background(), "dash0")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("recovers from partial delete where managed policy already detached", func(t *testing.T) {
		mock := defaultMockIAM()
		mock.DeleteRolePolicyFunc = func(ctx context.Context, params *iam.DeleteRolePolicyInput, optFns ...func(*iam.Options)) (*iam.DeleteRolePolicyOutput, error) {
			return nil, noSuchEntity()
		}
		mock.DetachRolePolicyFunc = func(ctx context.Context, params *iam.DetachRolePolicyInput, optFns ...func(*iam.Options)) (*iam.DetachRolePolicyOutput, error) {
			return nil, noSuchEntity()
		}
		client := newTestClient(mock, nil)

		err := client.DeleteReadOnlyRole(context.Background(), "dash0")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

// --- DeleteInstrumentationRole tests ---

func TestDeleteInstrumentationRole(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := newTestClient(defaultMockIAM(), nil)
		err := client.DeleteInstrumentationRole(context.Background(), "dash0", "123456789012")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("idempotent when role does not exist", func(t *testing.T) {
		mock := defaultMockIAM()
		mock.GetRoleFunc = func(ctx context.Context, params *iam.GetRoleInput, optFns ...func(*iam.Options)) (*iam.GetRoleOutput, error) {
			return nil, noSuchEntity()
		}
		deleteCalled := false
		mock.DeleteRoleFunc = func(ctx context.Context, params *iam.DeleteRoleInput, optFns ...func(*iam.Options)) (*iam.DeleteRoleOutput, error) {
			deleteCalled = true
			return &iam.DeleteRoleOutput{}, nil
		}
		client := newTestClient(mock, nil)

		err := client.DeleteInstrumentationRole(context.Background(), "dash0", "123456789012")
		if err != nil {
			t.Fatalf("expected nil for idempotent delete, got: %v", err)
		}
		if deleteCalled {
			t.Error("delete should not be attempted when role does not exist")
		}
	})

	t.Run("fails when GetRole returns real error", func(t *testing.T) {
		mock := defaultMockIAM()
		mock.GetRoleFunc = func(ctx context.Context, params *iam.GetRoleInput, optFns ...func(*iam.Options)) (*iam.GetRoleOutput, error) {
			return nil, errTest
		}
		client := newTestClient(mock, nil)

		err := client.DeleteInstrumentationRole(context.Background(), "dash0", "123456789012")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest in chain, got: %v", err)
		}
	})

	t.Run("fails when detach policy returns real error", func(t *testing.T) {
		mock := defaultMockIAM()
		mock.DetachRolePolicyFunc = func(ctx context.Context, params *iam.DetachRolePolicyInput, optFns ...func(*iam.Options)) (*iam.DetachRolePolicyOutput, error) {
			return nil, errTest
		}
		client := newTestClient(mock, nil)

		err := client.DeleteInstrumentationRole(context.Background(), "dash0", "123456789012")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest in chain, got: %v", err)
		}
	})

	t.Run("fails when delete policy returns real error", func(t *testing.T) {
		mock := defaultMockIAM()
		mock.DeletePolicyFunc = func(ctx context.Context, params *iam.DeletePolicyInput, optFns ...func(*iam.Options)) (*iam.DeletePolicyOutput, error) {
			return nil, errTest
		}
		client := newTestClient(mock, nil)

		err := client.DeleteInstrumentationRole(context.Background(), "dash0", "123456789012")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest in chain, got: %v", err)
		}
	})

	t.Run("fails when delete role returns real error", func(t *testing.T) {
		mock := defaultMockIAM()
		mock.DeleteRoleFunc = func(ctx context.Context, params *iam.DeleteRoleInput, optFns ...func(*iam.Options)) (*iam.DeleteRoleOutput, error) {
			return nil, errTest
		}
		client := newTestClient(mock, nil)

		err := client.DeleteInstrumentationRole(context.Background(), "dash0", "123456789012")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, errTest) {
			t.Errorf("expected errTest in chain, got: %v", err)
		}
	})

	t.Run("recovers from partial delete where policy already detached and deleted", func(t *testing.T) {
		mock := defaultMockIAM()
		mock.DetachRolePolicyFunc = func(ctx context.Context, params *iam.DetachRolePolicyInput, optFns ...func(*iam.Options)) (*iam.DetachRolePolicyOutput, error) {
			return nil, noSuchEntity()
		}
		mock.DeletePolicyFunc = func(ctx context.Context, params *iam.DeletePolicyInput, optFns ...func(*iam.Options)) (*iam.DeletePolicyOutput, error) {
			return nil, noSuchEntity()
		}
		client := newTestClient(mock, nil)

		err := client.DeleteInstrumentationRole(context.Background(), "dash0", "123456789012")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

// --- UpdateRoleTags tests ---

func TestUpdateRoleTags(t *testing.T) {
	t.Run("success with existing tags", func(t *testing.T) {
		untagCalled := false
		tagCalled := false
		mock := defaultMockIAM()
		mock.ListRoleTagsFunc = func(ctx context.Context, params *iam.ListRoleTagsInput, optFns ...func(*iam.Options)) (*iam.ListRoleTagsOutput, error) {
			return &iam.ListRoleTagsOutput{Tags: []iamtypes.Tag{
				{Key: aws.String("old"), Value: aws.String("value")},
			}}, nil
		}
		mock.UntagRoleFunc = func(ctx context.Context, params *iam.UntagRoleInput, optFns ...func(*iam.Options)) (*iam.UntagRoleOutput, error) {
			untagCalled = true
			if len(params.TagKeys) != 1 || params.TagKeys[0] != "old" {
				t.Errorf("untag keys = %v, want [old]", params.TagKeys)
			}
			return &iam.UntagRoleOutput{}, nil
		}
		mock.TagRoleFunc = func(ctx context.Context, params *iam.TagRoleInput, optFns ...func(*iam.Options)) (*iam.TagRoleOutput, error) {
			tagCalled = true
			return &iam.TagRoleOutput{}, nil
		}
		client := newTestClient(mock, nil)

		err := client.UpdateRoleTags(context.Background(), "my-role", map[string]string{"new": "val"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !untagCalled {
			t.Error("expected untag to be called")
		}
		if !tagCalled {
			t.Error("expected tag to be called")
		}
	})

	t.Run("success with no existing tags", func(t *testing.T) {
		mock := defaultMockIAM()
		untagCalled := false
		mock.UntagRoleFunc = func(ctx context.Context, params *iam.UntagRoleInput, optFns ...func(*iam.Options)) (*iam.UntagRoleOutput, error) {
			untagCalled = true
			return &iam.UntagRoleOutput{}, nil
		}
		client := newTestClient(mock, nil)

		err := client.UpdateRoleTags(context.Background(), "my-role", map[string]string{"new": "val"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if untagCalled {
			t.Error("untag should not be called when there are no existing tags")
		}
	})

	t.Run("fails when list tags errors", func(t *testing.T) {
		mock := defaultMockIAM()
		mock.ListRoleTagsFunc = func(ctx context.Context, params *iam.ListRoleTagsInput, optFns ...func(*iam.Options)) (*iam.ListRoleTagsOutput, error) {
			return nil, errTest
		}
		client := newTestClient(mock, nil)

		err := client.UpdateRoleTags(context.Background(), "my-role", map[string]string{"k": "v"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("fails when untag errors", func(t *testing.T) {
		mock := defaultMockIAM()
		mock.ListRoleTagsFunc = func(ctx context.Context, params *iam.ListRoleTagsInput, optFns ...func(*iam.Options)) (*iam.ListRoleTagsOutput, error) {
			return &iam.ListRoleTagsOutput{Tags: []iamtypes.Tag{
				{Key: aws.String("old"), Value: aws.String("value")},
			}}, nil
		}
		mock.UntagRoleFunc = func(ctx context.Context, params *iam.UntagRoleInput, optFns ...func(*iam.Options)) (*iam.UntagRoleOutput, error) {
			return nil, errTest
		}
		client := newTestClient(mock, nil)

		err := client.UpdateRoleTags(context.Background(), "my-role", map[string]string{"k": "v"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("fails when tag errors", func(t *testing.T) {
		mock := defaultMockIAM()
		mock.TagRoleFunc = func(ctx context.Context, params *iam.TagRoleInput, optFns ...func(*iam.Options)) (*iam.TagRoleOutput, error) {
			return nil, errTest
		}
		client := newTestClient(mock, nil)

		err := client.UpdateRoleTags(context.Background(), "my-role", map[string]string{"k": "v"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

// --- isNotFound tests ---

func TestIsNotFound(t *testing.T) {
	if !isNotFound(noSuchEntity()) {
		t.Error("expected isNotFound to return true for NoSuchEntityException")
	}
	if isNotFound(errTest) {
		t.Error("expected isNotFound to return false for generic error")
	}
	if isNotFound(nil) {
		t.Error("expected isNotFound to return false for nil")
	}
}
