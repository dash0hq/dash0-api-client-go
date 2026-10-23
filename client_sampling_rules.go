package dash0

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ListSamplingRules retrieves all sampling rules.
func (c *client) ListSamplingRules(ctx context.Context, dataset *string) ([]*SamplingDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	params := &GetApiSamplingRulesParams{
		Dataset: dataset,
	}
	resp, err := c.inner.GetApiSamplingRulesWithResponse(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("dash0: list sampling rules failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return toPointerSlice(resp.JSON200.SamplingRules), nil
}

// GetSamplingRule retrieves a sampling rule by origin or ID.
func (c *client) GetSamplingRule(ctx context.Context, originOrID string, dataset *string) (*SamplingDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	params := &GetApiSamplingRulesOriginOrIdParams{
		Dataset: dataset,
	}
	resp, err := c.inner.GetApiSamplingRulesOriginOrIdWithResponse(ctx, originOrID, params)
	if err != nil {
		return nil, fmt.Errorf("dash0: get sampling rule failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200, nil
}

// CreateSamplingRule creates a new sampling rule.
func (c *client) CreateSamplingRule(ctx context.Context, rule *SamplingDefinition, dataset *string) (*SamplingDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	if rule == nil {
		return nil, fmt.Errorf("dash0: create sampling rule requires a non-nil rule")
	}
	params := &PostApiSamplingRulesParams{
		Dataset: dataset,
	}
	resp, err := c.inner.PostApiSamplingRulesWithResponse(ctx, params, *rule)
	if err != nil {
		return nil, fmt.Errorf("dash0: create sampling rule failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusCreated {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	var created SamplingDefinition
	if err := json.Unmarshal(resp.Body, &created); err != nil {
		return nil, fmt.Errorf("dash0: failed to parse sampling rule response: %w", err)
	}
	return &created, nil
}

// UpdateSamplingRule updates an existing sampling rule.
func (c *client) UpdateSamplingRule(ctx context.Context, originOrID string, rule *SamplingDefinition, dataset *string) (*SamplingDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	if rule == nil {
		return nil, fmt.Errorf("dash0: update sampling rule requires a non-nil rule")
	}
	params := &PutApiSamplingRulesOriginOrIdParams{
		Dataset: dataset,
	}
	resp, err := c.inner.PutApiSamplingRulesOriginOrIdWithResponse(ctx, originOrID, params, *rule)
	if err != nil {
		return nil, fmt.Errorf("dash0: update sampling rule failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200, nil
}

// DeleteSamplingRule deletes a sampling rule by origin or ID.
func (c *client) DeleteSamplingRule(ctx context.Context, originOrID string, dataset *string) error {
	if err := c.requireAPI(); err != nil {
		return err
	}
	params := &DeleteApiSamplingRulesOriginOrIdParams{
		Dataset: dataset,
	}
	resp, err := c.inner.DeleteApiSamplingRulesOriginOrIdWithResponse(ctx, originOrID, params)
	if err != nil {
		return fmt.Errorf("dash0: delete sampling rule failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// ListSamplingRulesIter returns an iterator over all sampling rules.
// This is a convenience wrapper around ListSamplingRules for consistent iteration patterns.
func (c *client) ListSamplingRulesIter(ctx context.Context, dataset *string) *Iter[SamplingDefinition] {
	items, err := c.ListSamplingRules(ctx, dataset)
	if err != nil {
		return newIterWithError[SamplingDefinition](err)
	}
	return newIter(items, false, nil, nil)
}

// StripSamplingRuleServerFields removes server-generated fields from a sampling rule.
//
// Sampling rules carry no annotations, so only the server-managed labels are cleared.
// The dash0.com/id label is preserved, so a stripped rule still addresses the same rule on update.
// User-set fields such as spec.enabled and the custom labels are left untouched.
// Callers that want a definition suitable for creating a new rule should also call [ClearSamplingRuleID].
func StripSamplingRuleServerFields(rule *SamplingDefinition) {
	if rule == nil {
		return
	}
	if rule.Metadata.Labels != nil {
		rule.Metadata.Labels.Dash0Comversion = nil
		rule.Metadata.Labels.Dash0Comsource = nil
		rule.Metadata.Labels.Dash0Comdataset = nil
		rule.Metadata.Labels.Dash0Comorigin = nil
	}
}

// ClearSamplingRuleID removes the ID from a sampling rule.
func ClearSamplingRuleID(rule *SamplingDefinition) {
	if rule == nil {
		return
	}
	if rule.Metadata.Labels != nil {
		rule.Metadata.Labels.Dash0Comid = nil
	}
}

// GetSamplingRuleID extracts the ID from a sampling rule.
func GetSamplingRuleID(rule *SamplingDefinition) string {
	if rule == nil || rule.Metadata.Labels == nil || rule.Metadata.Labels.Dash0Comid == nil {
		return ""
	}
	return *rule.Metadata.Labels.Dash0Comid
}

// GetSamplingRuleName extracts the display name from a sampling rule.
// It falls back to metadata.name when the display block is absent or its name is empty.
func GetSamplingRuleName(rule *SamplingDefinition) string {
	if rule == nil {
		return ""
	}
	if rule.Spec.Display != nil && rule.Spec.Display.Name != "" {
		return rule.Spec.Display.Name
	}
	return rule.Metadata.Name
}

// SetSamplingRuleID sets the dash0.com/id label on a sampling rule, initializing the labels struct if needed.
func SetSamplingRuleID(rule *SamplingDefinition, id string) {
	if rule == nil {
		return
	}
	if rule.Metadata.Labels == nil {
		rule.Metadata.Labels = &SamplingLabels{}
	}
	rule.Metadata.Labels.Dash0Comid = &id
}

// SetSamplingRuleIDIfAbsent sets the dash0.com/id label on a sampling rule only if it is not already set.
// It initializes the labels struct if needed.
func SetSamplingRuleIDIfAbsent(rule *SamplingDefinition, id string) {
	if rule == nil {
		return
	}
	if rule.Metadata.Labels == nil {
		rule.Metadata.Labels = &SamplingLabels{}
	}
	if rule.Metadata.Labels.Dash0Comid == nil {
		rule.Metadata.Labels.Dash0Comid = &id
	}
}
