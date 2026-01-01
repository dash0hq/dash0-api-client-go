package dash0

import (
	"context"
	"fmt"
	"net/http"
)

// ListCheckRules retrieves all check rules.
func (c *client) ListCheckRules(ctx context.Context, dataset *string) ([]*PrometheusAlertRuleApiListItem, error) {
	params := &GetApiAlertingCheckRulesParams{
		Dataset: dataset,
	}
	resp, err := c.inner.GetApiAlertingCheckRulesWithResponse(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("dash0: list check rules failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return toPointerSlice(*resp.JSON200), nil
}

// GetCheckRule retrieves a check rule by origin or ID.
func (c *client) GetCheckRule(ctx context.Context, originOrID string, dataset *string) (*PrometheusAlertRule, error) {
	params := &GetApiAlertingCheckRulesOriginOrIdParams{
		Dataset: dataset,
	}
	resp, err := c.inner.GetApiAlertingCheckRulesOriginOrIdWithResponse(ctx, originOrID, params)
	if err != nil {
		return nil, fmt.Errorf("dash0: get check rule failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200, nil
}

// CreateCheckRule creates a new check rule.
func (c *client) CreateCheckRule(ctx context.Context, rule *PrometheusAlertRule, dataset *string) (*PrometheusAlertRule, error) {
	params := &PostApiAlertingCheckRulesParams{
		Dataset: dataset,
	}
	resp, err := c.inner.PostApiAlertingCheckRulesWithResponse(ctx, params, *rule)
	if err != nil {
		return nil, fmt.Errorf("dash0: create check rule failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200, nil
}

// UpdateCheckRule updates an existing check rule.
func (c *client) UpdateCheckRule(ctx context.Context, originOrID string, rule *PrometheusAlertRule, dataset *string) (*PrometheusAlertRule, error) {
	params := &PutApiAlertingCheckRulesOriginOrIdParams{
		Dataset: dataset,
	}
	resp, err := c.inner.PutApiAlertingCheckRulesOriginOrIdWithResponse(ctx, originOrID, params, *rule)
	if err != nil {
		return nil, fmt.Errorf("dash0: update check rule failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200, nil
}

// DeleteCheckRule deletes a check rule by origin or ID.
func (c *client) DeleteCheckRule(ctx context.Context, originOrID string, dataset *string) error {
	params := &DeleteApiAlertingCheckRulesOriginOrIdParams{
		Dataset: dataset,
	}
	resp, err := c.inner.DeleteApiAlertingCheckRulesOriginOrIdWithResponse(ctx, originOrID, params)
	if err != nil {
		return fmt.Errorf("dash0: delete check rule failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// ListCheckRulesIter returns an iterator over all check rules.
// This is a convenience wrapper around ListCheckRules for consistent iteration patterns.
func (c *client) ListCheckRulesIter(ctx context.Context, dataset *string) *Iter[PrometheusAlertRuleApiListItem] {
	items, err := c.ListCheckRules(ctx, dataset)
	if err != nil {
		return newIterWithError[PrometheusAlertRuleApiListItem](err)
	}
	return newIter(items, false, nil, nil)
}
