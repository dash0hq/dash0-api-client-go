package dash0

import "time"

// IntegrationDefinition represents an integration resource.
type IntegrationDefinition struct {
	Kind     string              `json:"kind"`
	Metadata IntegrationMetadata `json:"metadata"`
	Spec     IntegrationSpec     `json:"spec"`
}

// IntegrationMetadata holds the metadata for an integration resource.
type IntegrationMetadata struct {
	Name        string                  `json:"name"`
	Labels      map[string]string       `json:"labels,omitempty"`
	Annotations *IntegrationAnnotations `json:"annotations,omitempty"`
	CreatedAt   *time.Time              `json:"createdAt,omitempty"`
	UpdatedAt   *time.Time              `json:"updatedAt,omitempty"`
	Version     *int64                  `json:"version,omitempty"`
}

// IntegrationAnnotations holds the annotation fields for an integration resource.
type IntegrationAnnotations struct {
	CreatedAt     *string `json:"dash0.com/created-at,omitempty"`
	LastUpdatedAt *string `json:"dash0.com/last-updated-at,omitempty"`
}

// IntegrationSpec holds the top-level spec for an integration resource.
type IntegrationSpec struct {
	Enabled     bool               `json:"enabled"`
	Display     IntegrationDisplay `json:"display"`
	AI          IntegrationAI      `json:"ai"`
	Integration IntegrationInner   `json:"integration"`
}

// IntegrationDisplay holds the display configuration for an integration.
type IntegrationDisplay struct {
	Name string `json:"name"`
}

// IntegrationAI holds the AI configuration for an integration.
type IntegrationAI struct {
	Access string `json:"access"`
}

// IntegrationInner wraps the kind-specific integration spec.
// Spec is map[string]any because integrations can be different kinds
// (aws, gcp, etc.) with different specs.
type IntegrationInner struct {
	Kind string         `json:"kind"`
	Spec map[string]any `json:"spec"`
}

// AwsIntegrationSpec is the spec for kind "aws".
type AwsIntegrationSpec struct {
	Dataset   string               `json:"dataset"`
	AccountID string               `json:"accountId"`
	Roles     []AwsIntegrationRole `json:"roles"`
}

// AwsIntegrationRole represents a single IAM role in an AWS integration.
type AwsIntegrationRole struct {
	Arn            string `json:"arn"`
	ExternalID     string `json:"externalId"`
	PermissionType string `json:"permissionType"`
	Status         string `json:"status,omitempty"`
}
