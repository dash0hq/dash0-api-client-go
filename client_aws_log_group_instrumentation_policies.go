package dash0

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

// GetAwsLogGroupInstrumentationPolicy retrieves the log-group instrumentation policy of an AWS integration.
// It returns a not-found [APIError] when the integration never had a policy.
// A deleted policy is still returned, with the dash0.com/deleted-at annotation set; see [IsAwsLogGroupInstrumentationPolicyDeleted].
func (c *client) GetAwsLogGroupInstrumentationPolicy(ctx context.Context, integrationID string) (*AwsLogGroupInstrumentationPolicy, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	id, err := parseIntegrationID(integrationID)
	if err != nil {
		return nil, err
	}
	resp, err := c.inner.GetApiAwsLogGroupInstrumentationPoliciesIntegrationIdWithResponse(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("dash0: get AWS log-group instrumentation policy failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return resp.JSON200, nil
}

// UpdateAwsLogGroupInstrumentationPolicy creates or replaces the log-group instrumentation policy of an AWS integration, and recreates a deleted one.
// It returns the stored policy.
// When the dash0.com/version label is set, the API rejects the write if the policy changed since that revision; check with [IsSupersededVersion].
// Without the label, the write replaces whatever is stored.
// Writing a dash0.com/origin that differs from the one stored on an existing policy returns a conflict [APIError].
func (c *client) UpdateAwsLogGroupInstrumentationPolicy(ctx context.Context, integrationID string, policy *AwsLogGroupInstrumentationPolicy) (*AwsLogGroupInstrumentationPolicy, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	if policy == nil {
		return nil, fmt.Errorf("dash0: AWS log-group instrumentation policy is required")
	}
	id, err := parseIntegrationID(integrationID)
	if err != nil {
		return nil, err
	}
	resp, err := c.inner.PutApiAwsLogGroupInstrumentationPoliciesIntegrationIdWithResponse(ctx, id, withEmptyAwsLogGroupPolicyLists(policy))
	if err != nil {
		return nil, fmt.Errorf("dash0: update AWS log-group instrumentation policy failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return resp.JSON200, nil
}

// DeleteAwsLogGroupInstrumentationPolicy deletes the log-group instrumentation policy of an AWS integration and returns the deleted policy.
// disposition decides what happens to log groups the policy forwards:
// [AwsLogGroupPolicyDispositionKeep] keeps forwarding them, and [AwsLogGroupPolicyDispositionRemove] removes their subscription filters.
// Manually selected log groups keep forwarding either way.
// When expectedRevision is not nil, the API rejects the delete if the policy changed since that revision.
//
// A server-error [APIError] with Type "reconciliation_dispatch_failed" means the deletion was saved but cleanup did not start.
// Read the policy, and if it is still deleted with the same disposition, repeat the delete with its current revision.
// The client retries a server error once by default, and a repeated delete with the same disposition is accepted, so without expectedRevision that retry usually recovers on its own.
// With expectedRevision, the retry instead fails with [IsSupersededVersion], because the saved deletion advanced the revision; read the policy to tell the two cases apart.
func (c *client) DeleteAwsLogGroupInstrumentationPolicy(ctx context.Context, integrationID string, disposition AwsLogGroupPolicyDisposition, expectedRevision *int64) (*AwsLogGroupInstrumentationPolicy, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	if !disposition.Valid() {
		return nil, fmt.Errorf("dash0: deletion disposition must be %q or %q, got %q", AwsLogGroupPolicyDispositionKeep, AwsLogGroupPolicyDispositionRemove, disposition)
	}
	id, err := parseIntegrationID(integrationID)
	if err != nil {
		return nil, err
	}
	params := &DeleteApiAwsLogGroupInstrumentationPoliciesIntegrationIdParams{Disposition: disposition, ExpectedRevision: expectedRevision}
	resp, err := c.inner.DeleteApiAwsLogGroupInstrumentationPoliciesIntegrationIdWithResponse(ctx, id, params)
	if err != nil {
		return nil, fmt.Errorf("dash0: delete AWS log-group instrumentation policy failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return resp.JSON200, nil
}

// withEmptyAwsLogGroupPolicyLists returns a copy of policy whose nil rule and condition lists are empty.
// The spec types both as non-nullable arrays, so they must be sent as [] rather than null.
func withEmptyAwsLogGroupPolicyLists(policy *AwsLogGroupInstrumentationPolicy) AwsLogGroupInstrumentationPolicy {
	body := *policy
	body.Spec.Rules = make([]AwsLogGroupPolicyRule, len(policy.Spec.Rules))
	for i, rule := range policy.Spec.Rules {
		if rule.Conditions == nil {
			rule.Conditions = []AwsLogGroupPolicyCondition{}
		}
		body.Spec.Rules[i] = rule
	}
	return body
}

// StripAwsLogGroupInstrumentationPolicyServerFields removes server-managed fields from an AWS log-group instrumentation policy.
// It removes metadata.name, the dash0.com/id, dash0.com/dataset, dash0.com/source, and dash0.com/version labels, and all annotations.
// The dash0.com/origin label is kept, because clients may set it.
// A stripped copy sent to [Client.UpdateAwsLogGroupInstrumentationPolicy] skips the revision check.
func StripAwsLogGroupInstrumentationPolicyServerFields(policy *AwsLogGroupInstrumentationPolicy) {
	if policy == nil {
		return
	}
	policy.Metadata.Name = nil
	policy.Metadata.Annotations = nil
	if labels := policy.Metadata.Labels; labels != nil {
		labels.Dash0Comid = nil
		labels.Dash0Comdataset = nil
		labels.Dash0Comsource = nil
		labels.Dash0Comversion = nil
		if labels.Dash0Comorigin == nil {
			policy.Metadata.Labels = nil
		}
	}
}

// GetAwsLogGroupInstrumentationPolicyID returns the dash0.com/id label of an AWS log-group instrumentation policy, or "" if absent.
// The ID is the ID of the AWS integration the policy belongs to.
// There are no Set or Clear ID helpers, because the integration ID is taken from the request path and cannot be chosen by clients.
func GetAwsLogGroupInstrumentationPolicyID(policy *AwsLogGroupInstrumentationPolicy) string {
	if policy == nil || policy.Metadata.Labels == nil || policy.Metadata.Labels.Dash0Comid == nil {
		return ""
	}
	return *policy.Metadata.Labels.Dash0Comid
}

// GetAwsLogGroupInstrumentationPolicyName returns the metadata.name of an AWS log-group instrumentation policy, or "" if absent.
// The server sets it to the integration ID and ignores it on write.
func GetAwsLogGroupInstrumentationPolicyName(policy *AwsLogGroupInstrumentationPolicy) string {
	if policy == nil || policy.Metadata.Name == nil {
		return ""
	}
	return *policy.Metadata.Name
}

// GetAwsLogGroupInstrumentationPolicyOrigin returns the dash0.com/origin label of an AWS log-group instrumentation policy, or "" if absent.
func GetAwsLogGroupInstrumentationPolicyOrigin(policy *AwsLogGroupInstrumentationPolicy) string {
	if policy == nil || policy.Metadata.Labels == nil || policy.Metadata.Labels.Dash0Comorigin == nil {
		return ""
	}
	return *policy.Metadata.Labels.Dash0Comorigin
}

// SetAwsLogGroupInstrumentationPolicyOrigin sets the dash0.com/origin label on an AWS log-group instrumentation policy.
// It initializes the labels struct if needed.
func SetAwsLogGroupInstrumentationPolicyOrigin(policy *AwsLogGroupInstrumentationPolicy, origin string) {
	if policy == nil {
		return
	}
	if policy.Metadata.Labels == nil {
		policy.Metadata.Labels = &AwsLogGroupPolicyLabels{}
	}
	policy.Metadata.Labels.Dash0Comorigin = &origin
}

// GetAwsLogGroupInstrumentationPolicyRevision returns the dash0.com/version label of an AWS log-group instrumentation policy as a number.
// ok is false when the label is absent or not a number.
func GetAwsLogGroupInstrumentationPolicyRevision(policy *AwsLogGroupInstrumentationPolicy) (revision int64, ok bool) {
	if policy == nil || policy.Metadata.Labels == nil || policy.Metadata.Labels.Dash0Comversion == nil {
		return 0, false
	}
	revision, err := strconv.ParseInt(*policy.Metadata.Labels.Dash0Comversion, 10, 64)
	if err != nil {
		return 0, false
	}
	return revision, true
}

// SetAwsLogGroupInstrumentationPolicyRevision sets the dash0.com/version label, so that [Client.UpdateAwsLogGroupInstrumentationPolicy] fails with [IsSupersededVersion] if the policy changed since that revision.
// Zero means the caller expects no policy to exist yet.
// It initializes the labels struct if needed.
func SetAwsLogGroupInstrumentationPolicyRevision(policy *AwsLogGroupInstrumentationPolicy, revision int64) {
	if policy == nil {
		return
	}
	if policy.Metadata.Labels == nil {
		policy.Metadata.Labels = &AwsLogGroupPolicyLabels{}
	}
	version := strconv.FormatInt(revision, 10)
	policy.Metadata.Labels.Dash0Comversion = &version
}

// IsAwsLogGroupInstrumentationPolicyDeleted reports whether the policy carries the dash0.com/deleted-at annotation.
// The API keeps a deleted policy so that its revisions keep increasing when it is recreated.
func IsAwsLogGroupInstrumentationPolicyDeleted(policy *AwsLogGroupInstrumentationPolicy) bool {
	return policy != nil && policy.Metadata.Annotations != nil && policy.Metadata.Annotations.Dash0ComdeletedAt != nil
}

var _ AssetMetadata = (*AwsLogGroupInstrumentationPolicy)(nil)

func (policy *AwsLogGroupInstrumentationPolicy) GetOrigin() string {
	return GetAwsLogGroupInstrumentationPolicyOrigin(policy)
}
func (policy *AwsLogGroupInstrumentationPolicy) GetID() string {
	return GetAwsLogGroupInstrumentationPolicyID(policy)
}
func (policy *AwsLogGroupInstrumentationPolicy) GetName() string {
	return GetAwsLogGroupInstrumentationPolicyName(policy)
}
func (*AwsLogGroupInstrumentationPolicy) UpsertKeys() (byOrigin, byID bool) { return false, true }
