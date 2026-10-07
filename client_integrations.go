package dash0

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ListIntegrations retrieves all integrations of the organization.
// Integrations are organization-scoped, so there is no dataset parameter.
// Soft-deleted integrations are not included.
//
// The list can contain integration kinds other than aws even though the
// OpenAPI spec only documents the aws variant. Use [GetIntegrationKind] or
// [GetAwsIntegrationSpec] to tell them apart instead of assuming every entry
// is an AWS integration.
func (c *client) ListIntegrations(ctx context.Context) ([]*IntegrationDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	resp, err := c.inner.GetApiIntegrationsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("dash0: list integrations failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return toPointerSlice(resp.JSON200.Integrations), nil
}

// GetIntegration retrieves an integration by origin or ID.
// A soft-deleted integration is reported as not found.
func (c *client) GetIntegration(ctx context.Context, originOrID string) (*IntegrationDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	resp, err := c.inner.GetApiIntegrationsOriginOrIdWithResponse(ctx, originOrID, nil)
	if err != nil {
		return nil, fmt.Errorf("dash0: get integration failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return resp.JSON200, nil
}

// CreateIntegration creates a new integration via POST /api/integrations.
// Requires the admin role in the organization. The server assigns the
// dash0.com/id label, and for AWS integrations it forces every role's status
// to pending until a backend workflow has verified the role.
func (c *client) CreateIntegration(ctx context.Context, integration *IntegrationDefinition) (*IntegrationDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	if integration == nil {
		return nil, fmt.Errorf("dash0: create integration requires a non-nil integration")
	}
	resp, err := c.inner.PostApiIntegrationsWithResponse(ctx, *integration)
	if err != nil {
		return nil, fmt.Errorf("dash0: create integration failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusCreated {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	var created IntegrationDefinition
	if err := json.Unmarshal(resp.Body, &created); err != nil {
		return nil, fmt.Errorf("dash0: failed to parse integration response: %w", err)
	}
	return &created, nil
}

// UpsertIntegration create-or-replaces an integration via
// PUT /api/integrations/{originOrID}. If an integration with the given origin
// or ID exists, it is replaced; otherwise a new integration is created with
// dash0.com/origin = originOrID. Requires the admin role in the organization.
func (c *client) UpsertIntegration(ctx context.Context, originOrID string, integration *IntegrationDefinition) (*IntegrationDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	if integration == nil {
		return nil, fmt.Errorf("dash0: upsert integration requires a non-nil integration")
	}
	resp, err := c.inner.PutApiIntegrationsOriginOrIdWithResponse(ctx, originOrID, *integration)
	if err != nil {
		return nil, fmt.Errorf("dash0: upsert integration failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusCreated {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	var upserted IntegrationDefinition
	if err := json.Unmarshal(resp.Body, &upserted); err != nil {
		return nil, fmt.Errorf("dash0: failed to parse integration response: %w", err)
	}
	return &upserted, nil
}

// DeleteIntegration soft-deletes an integration by origin or ID. Requires the
// admin role in the organization.
func (c *client) DeleteIntegration(ctx context.Context, originOrID string) error {
	if err := c.requireAPI(); err != nil {
		return err
	}
	resp, err := c.inner.DeleteApiIntegrationsOriginOrIdWithResponse(ctx, originOrID)
	if err != nil {
		return fmt.Errorf("dash0: delete integration failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// ListIntegrationsIter returns an iterator over all integrations. This is a
// convenience wrapper around [client.ListIntegrations] for consistent
// iteration patterns.
func (c *client) ListIntegrationsIter(ctx context.Context) *Iter[IntegrationDefinition] {
	items, err := c.ListIntegrations(ctx)
	if err != nil {
		return newIterWithError[IntegrationDefinition](err)
	}
	return newIter(items, false, nil, nil)
}

// NewAwsIntegrationDefinition builds an integration envelope for an AWS
// integration. The name is used for both metadata.name and
// spec.display.name. The integration is enabled, and AI access defaults to
// [ArtificialIntelligenceAccessNone]; callers that want a different setting
// change spec.ai.access on the returned definition.
func NewAwsIntegrationDefinition(name string, spec AwsIntegrationSpec) *IntegrationDefinition {
	integration := &IntegrationDefinition{
		Kind:     Dash0Integration,
		Metadata: IntegrationMetadata{Name: name},
		Spec: IntegrationSpec{
			Enabled: true,
			Display: IntegrationDisplay{Name: name},
			Ai:      ArtificialIntelligenceSettings{Access: ArtificialIntelligenceAccessNone},
		},
	}
	SetAwsIntegrationSpec(integration, spec)
	return integration
}

// GetIntegrationKind returns the kind of the integration variant carried in
// spec.integration (for example "aws"). Returns the empty string when the
// integration is nil, the variant is unset, or it cannot be decoded.
//
// The server can return kinds that the OpenAPI spec does not document yet, so
// callers must not assume the result is one of the known kinds.
func GetIntegrationKind(integration *IntegrationDefinition) string {
	if integration == nil {
		return ""
	}
	raw, err := integration.Spec.Integration.MarshalJSON()
	if err != nil {
		return ""
	}
	var variant struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(raw, &variant); err != nil {
		return ""
	}
	return variant.Kind
}

// GetAwsIntegrationSpec decodes the AWS variant of an integration. The second
// return value is false when the integration is nil, carries a kind other
// than aws, or the variant cannot be decoded.
//
// The returned spec is a copy: changes to it are not reflected in the
// integration until they are written back with [SetAwsIntegrationSpec].
func GetAwsIntegrationSpec(integration *IntegrationDefinition) (*AwsIntegrationSpec, bool) {
	if GetIntegrationKind(integration) != string(Aws) {
		return nil, false
	}
	aws, err := integration.Spec.Integration.AsAwsIntegration()
	if err != nil {
		return nil, false
	}
	return &aws.Spec, true
}

// SetAwsIntegrationSpec sets spec.integration to the AWS variant carrying the
// given spec, replacing any variant that was set before. No-op when the
// integration is nil.
func SetAwsIntegrationSpec(integration *IntegrationDefinition, spec AwsIntegrationSpec) {
	if integration == nil {
		return
	}
	// Marshaling AwsIntegration cannot fail: every field is a plain value,
	// pointer, or slice of JSON-safe types.
	_ = integration.Spec.Integration.FromAwsIntegration(AwsIntegration{Kind: Aws, Spec: spec})
}

// GetIntegrationID extracts the dash0.com/id label from an integration
// definition. Returns the empty string when the integration is nil, has no
// labels, or the label is unset.
func GetIntegrationID(integration *IntegrationDefinition) string {
	if integration == nil || integration.Metadata.Labels == nil || integration.Metadata.Labels.Dash0Comid == nil {
		return ""
	}
	return *integration.Metadata.Labels.Dash0Comid
}

// GetIntegrationName extracts the metadata.name from an integration
// definition. Returns the empty string when the integration is nil.
func GetIntegrationName(integration *IntegrationDefinition) string {
	if integration == nil {
		return ""
	}
	return integration.Metadata.Name
}

// GetIntegrationOrigin extracts the dash0.com/origin label from an
// integration definition. Returns the empty string when the integration is
// nil, has no labels, or the label is unset.
func GetIntegrationOrigin(integration *IntegrationDefinition) string {
	if integration == nil || integration.Metadata.Labels == nil || integration.Metadata.Labels.Dash0Comorigin == nil {
		return ""
	}
	return *integration.Metadata.Labels.Dash0Comorigin
}

// SetIntegrationID sets the dash0.com/id label on an integration definition,
// initializing the labels struct if needed. No-op when the integration is nil.
func SetIntegrationID(integration *IntegrationDefinition, id string) {
	if integration == nil {
		return
	}
	if integration.Metadata.Labels == nil {
		integration.Metadata.Labels = &IntegrationLabels{}
	}
	integration.Metadata.Labels.Dash0Comid = &id
}

// SetIntegrationIDIfAbsent sets the dash0.com/id label on an integration
// definition only when it is not already set. No-op when the integration is
// nil.
func SetIntegrationIDIfAbsent(integration *IntegrationDefinition, id string) {
	if integration == nil {
		return
	}
	if integration.Metadata.Labels == nil {
		integration.Metadata.Labels = &IntegrationLabels{}
	}
	if integration.Metadata.Labels.Dash0Comid == nil {
		integration.Metadata.Labels.Dash0Comid = &id
	}
}

// ClearIntegrationID removes the dash0.com/id label from an integration
// definition. No-op when the integration is nil or has no labels.
func ClearIntegrationID(integration *IntegrationDefinition) {
	if integration == nil || integration.Metadata.Labels == nil {
		return
	}
	integration.Metadata.Labels.Dash0Comid = nil
}

// StripIntegrationServerFields clears the server-managed fields from an
// integration definition so callers can round-trip it through a write
// endpoint or compare it with the desired state. Cleared fields are:
//
//   - metadata.labels["dash0.com/id"], [".../version"], [".../source"].
//   - metadata.annotations["dash0.com/created-at"], [".../created-by"],
//     [".../last-updated-at"], [".../last-updated-by"].
//   - For aws integrations: spec.integration.spec templateVersion,
//     verificationStatus, verificationError, lastVerifiedAt,
//     ingestAuthTokenId, regionalResources, desiredRegions, and each role's
//     status and lastAccessTimestamp.
//
// dash0.com/origin is preserved because it is client-settable.
// cloudFormationStackArn is preserved too: callers never set it, but the Dash0
// CloudFormation template does, and an update that omits it clears it.
// Integrations of other kinds keep their spec.integration untouched. No-op
// when the integration is nil.
func StripIntegrationServerFields(integration *IntegrationDefinition) {
	if integration == nil {
		return
	}
	if integration.Metadata.Labels != nil {
		integration.Metadata.Labels.Dash0Comid = nil
		integration.Metadata.Labels.Dash0Comversion = nil
		integration.Metadata.Labels.Dash0Comsource = nil
	}
	if integration.Metadata.Annotations != nil {
		integration.Metadata.Annotations.Dash0ComcreatedAt = nil
		integration.Metadata.Annotations.Dash0ComcreatedBy = nil
		integration.Metadata.Annotations.Dash0ComlastUpdatedAt = nil
		integration.Metadata.Annotations.Dash0ComlastUpdatedBy = nil
	}
	spec, ok := GetAwsIntegrationSpec(integration)
	if !ok {
		return
	}
	spec.TemplateVersion = nil
	spec.VerificationStatus = nil
	spec.VerificationError = nil
	spec.LastVerifiedAt = nil
	spec.IngestAuthTokenId = nil
	spec.RegionalResources = nil
	spec.DesiredRegions = nil
	for i := range spec.Roles {
		spec.Roles[i].Status = nil
		spec.Roles[i].LastAccessTimestamp = nil
	}
	SetAwsIntegrationSpec(integration, *spec)
}
