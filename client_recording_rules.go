package dash0

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// RecordingRule is a PrometheusRule CRD document containing recording rule groups.
type RecordingRule = generatedPrometheusRule

// ListRecordingRules retrieves all recording rules.
func (c *client) ListRecordingRules(ctx context.Context, dataset *string) ([]*RecordingRule, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	params := &GetApiRecordingRulesParams{
		Dataset: dataset,
	}
	resp, err := c.inner.GetApiRecordingRulesWithResponse(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("dash0: list recording rules failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return toPointerSlice(*resp.JSON200), nil
}

// GetRecordingRule retrieves a recording rule by origin or ID.
func (c *client) GetRecordingRule(ctx context.Context, originOrID string, dataset *string) (*RecordingRule, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	params := &GetApiRecordingRulesOriginOrIdParams{
		Dataset: dataset,
	}
	resp, err := c.inner.GetApiRecordingRulesOriginOrIdWithResponse(ctx, originOrID, params)
	if err != nil {
		return nil, fmt.Errorf("dash0: get recording rule failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200, nil
}

// CreateRecordingRule creates a new recording rule.
func (c *client) CreateRecordingRule(ctx context.Context, rule *RecordingRule, dataset *string) (*RecordingRule, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	_ = dataset // dataset is part of the rule body metadata, not a query param
	resp, err := c.inner.PostApiRecordingRulesWithResponse(ctx, *rule)
	if err != nil {
		return nil, fmt.Errorf("dash0: create recording rule failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusCreated {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	var created RecordingRule
	if err := json.Unmarshal(resp.Body, &created); err != nil {
		return nil, fmt.Errorf("dash0: failed to parse recording rule response: %w", err)
	}
	return &created, nil
}

// UpdateRecordingRule updates an existing recording rule.
func (c *client) UpdateRecordingRule(ctx context.Context, originOrID string, rule *RecordingRule, dataset *string) (*RecordingRule, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	_ = dataset // dataset is part of the rule body metadata, not a query param
	resp, err := c.inner.PutApiRecordingRulesOriginOrIdWithResponse(ctx, originOrID, *rule)
	if err != nil {
		return nil, fmt.Errorf("dash0: update recording rule failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusCreated {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	var updated RecordingRule
	if err := json.Unmarshal(resp.Body, &updated); err != nil {
		return nil, fmt.Errorf("dash0: failed to parse recording rule response: %w", err)
	}
	return &updated, nil
}

// DeleteRecordingRule deletes a recording rule by origin or ID.
func (c *client) DeleteRecordingRule(ctx context.Context, originOrID string, dataset *string) error {
	if err := c.requireAPI(); err != nil {
		return err
	}
	params := &DeleteApiRecordingRulesOriginOrIdParams{
		Dataset: dataset,
	}
	resp, err := c.inner.DeleteApiRecordingRulesOriginOrIdWithResponse(ctx, originOrID, params)
	if err != nil {
		return fmt.Errorf("dash0: delete recording rule failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// ListRecordingRulesIter returns an iterator over all recording rules.
// This is a convenience wrapper around ListRecordingRules for consistent iteration patterns.
func (c *client) ListRecordingRulesIter(ctx context.Context, dataset *string) *Iter[RecordingRule] {
	items, err := c.ListRecordingRules(ctx, dataset)
	if err != nil {
		return newIterWithError[RecordingRule](err)
	}
	return newIter(items, false, nil, nil)
}

// StripRecordingRuleServerFields removes server-generated fields from a recording rule definition.
func StripRecordingRuleServerFields(rule *RecordingRule) {
	if rule == nil {
		return
	}
	if rule.Metadata.Labels != nil {
		delete(*rule.Metadata.Labels, LabelID)
		delete(*rule.Metadata.Labels, LabelOrigin)
		delete(*rule.Metadata.Labels, LabelDataset)
		delete(*rule.Metadata.Labels, "dash0.com/version")
		delete(*rule.Metadata.Labels, "dash0.com/source")
	}
	if rule.Metadata.Annotations != nil {
		delete(*rule.Metadata.Annotations, "dash0.com/created-at")
		delete(*rule.Metadata.Annotations, "dash0.com/updated-at")
		delete(*rule.Metadata.Annotations, "dash0.com/deleted-at")
		delete(*rule.Metadata.Annotations, "dash0.com/first-evaluation-at")
	}
}

// GetRecordingRuleID extracts the ID from a recording rule definition.
func GetRecordingRuleID(rule *RecordingRule) string {
	if rule == nil || rule.Metadata.Labels == nil {
		return ""
	}
	return (*rule.Metadata.Labels)[LabelID]
}

// SetRecordingRuleID sets the ID on a recording rule definition.
func SetRecordingRuleID(rule *RecordingRule, id string) {
	if rule == nil {
		return
	}
	if rule.Metadata.Labels == nil {
		m := map[string]string{}
		rule.Metadata.Labels = &m
	}
	(*rule.Metadata.Labels)[LabelID] = id
}

// SetRecordingRuleIDIfAbsent sets the ID on a recording rule definition only if it is
// not already set.
func SetRecordingRuleIDIfAbsent(rule *RecordingRule, id string) {
	if rule == nil {
		return
	}
	if rule.Metadata.Labels == nil {
		m := map[string]string{}
		rule.Metadata.Labels = &m
	}
	if _, ok := (*rule.Metadata.Labels)[LabelID]; !ok {
		(*rule.Metadata.Labels)[LabelID] = id
	}
}

// ClearRecordingRuleID removes the ID from a recording rule definition.
func ClearRecordingRuleID(rule *RecordingRule) {
	if rule == nil {
		return
	}
	if rule.Metadata.Labels != nil {
		delete(*rule.Metadata.Labels, LabelID)
	}
}

// GetRecordingRuleName extracts the name from a recording rule definition.
func GetRecordingRuleName(rule *RecordingRule) string {
	if rule == nil {
		return ""
	}
	return rule.Metadata.Name
}

// GetRecordingRuleDataset extracts the dataset from a recording rule definition.
func GetRecordingRuleDataset(rule *RecordingRule) string {
	if rule == nil || rule.Metadata.Labels == nil {
		return ""
	}
	return (*rule.Metadata.Labels)[LabelDataset]
}

// SetRecordingRuleDataset sets the dataset on a recording rule definition.
func SetRecordingRuleDataset(rule *RecordingRule, dataset string) {
	if rule == nil {
		return
	}
	if rule.Metadata.Labels == nil {
		m := map[string]string{}
		rule.Metadata.Labels = &m
	}
	(*rule.Metadata.Labels)[LabelDataset] = dataset
}
