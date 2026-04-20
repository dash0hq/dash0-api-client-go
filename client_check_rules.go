package dash0

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ListCheckRules retrieves all check rules.
func (c *client) ListCheckRules(ctx context.Context, dataset *string) ([]*PrometheusAlertRuleApiListItem, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
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
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
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
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	params := &PostApiAlertingCheckRulesParams{
		Dataset: dataset,
	}
	resp, err := c.inner.PostApiAlertingCheckRulesWithResponse(ctx, params, *rule)
	if err != nil {
		return nil, fmt.Errorf("dash0: create check rule failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusCreated {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	var created PrometheusAlertRule
	if err := json.Unmarshal(resp.Body, &created); err != nil {
		return nil, fmt.Errorf("dash0: failed to parse check rule response: %w", err)
	}
	return &created, nil
}

// UpdateCheckRule updates an existing check rule.
func (c *client) UpdateCheckRule(ctx context.Context, originOrID string, rule *PrometheusAlertRule, dataset *string) (*PrometheusAlertRule, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
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
	if err := c.requireAPI(); err != nil {
		return err
	}
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

// StripCheckRuleServerFields removes server-generated fields from a check rule definition.
func StripCheckRuleServerFields(rule *PrometheusAlertRule) {
	if rule == nil {
		return
	}
	rule.Dataset = nil
	if rule.Labels != nil {
		delete(*rule.Labels, LabelOrigin)
	}
}

// ClearCheckRuleID removes the ID from a check rule definition.
func ClearCheckRuleID(rule *PrometheusAlertRule) {
	if rule == nil {
		return
	}
	rule.Id = nil
}

// GetCheckRuleDataset extracts the dataset from a check rule definition.
func GetCheckRuleDataset(rule *PrometheusAlertRule) string {
	if rule == nil || rule.Dataset == nil {
		return ""
	}
	return *rule.Dataset
}

// SetCheckRuleDataset sets the dataset on a check rule definition.
func SetCheckRuleDataset(rule *PrometheusAlertRule, dataset string) {
	if rule == nil {
		return
	}
	ds := Dataset(dataset)
	rule.Dataset = &ds
}

// GetCheckRuleID extracts the ID from a check rule definition.
func GetCheckRuleID(rule *PrometheusAlertRule) string {
	if rule == nil || rule.Id == nil {
		return ""
	}
	return *rule.Id
}

// GetCheckRuleName extracts the name from a check rule definition.
func GetCheckRuleName(rule *PrometheusAlertRule) string {
	if rule == nil {
		return ""
	}
	return rule.Name
}

// SetCheckRuleID sets the ID on a check rule definition.
func SetCheckRuleID(rule *PrometheusAlertRule, id string) {
	if rule == nil {
		return
	}
	rule.Id = &id
}

// SetCheckRuleIDIfAbsent sets the ID on a check rule definition only if it is
// not already set.
func SetCheckRuleIDIfAbsent(rule *PrometheusAlertRule, id string) {
	if rule == nil {
		return
	}
	if rule.Id == nil {
		rule.Id = &id
	}
}
