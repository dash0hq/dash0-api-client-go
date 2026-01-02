package dash0

import (
	"context"
	"fmt"
	"net/http"
)

// ListSamplingRules retrieves all sampling rules.
func (c *client) ListSamplingRules(ctx context.Context, dataset *string) ([]*SamplingDefinition, error) {
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
	params := &PostApiSamplingRulesParams{
		Dataset: dataset,
	}
	resp, err := c.inner.PostApiSamplingRulesWithResponse(ctx, params, *rule)
	if err != nil {
		return nil, fmt.Errorf("dash0: create sampling rule failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200, nil
}

// UpdateSamplingRule updates an existing sampling rule.
func (c *client) UpdateSamplingRule(ctx context.Context, originOrID string, rule *SamplingDefinition, dataset *string) (*SamplingDefinition, error) {
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
