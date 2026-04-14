package aws

import (
	"encoding/json"
	"sort"
	"testing"
)

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
