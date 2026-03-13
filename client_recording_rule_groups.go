package dash0

import (
	"context"
	"fmt"
	"net/http"
)

// ListRecordingRuleGroups retrieves all recording rule groups.
func (c *client) ListRecordingRuleGroups(ctx context.Context, dataset *string) ([]*RecordingRuleGroupDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	params := &GetApiRecordingRuleGroupsParams{
		Dataset: dataset,
	}
	resp, err := c.inner.GetApiRecordingRuleGroupsWithResponse(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("dash0: list recording rule groups failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return toPointerSlice(resp.JSON200.RecordingRuleGroups), nil
}

// GetRecordingRuleGroup retrieves a recording rule group by origin or ID.
func (c *client) GetRecordingRuleGroup(ctx context.Context, originOrID string, dataset *string) (*RecordingRuleGroupDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	params := &GetApiRecordingRuleGroupsOriginOrIdParams{
		Dataset: dataset,
	}
	resp, err := c.inner.GetApiRecordingRuleGroupsOriginOrIdWithResponse(ctx, originOrID, params)
	if err != nil {
		return nil, fmt.Errorf("dash0: get recording rule group failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200, nil
}

// CreateRecordingRuleGroup creates a new recording rule group.
func (c *client) CreateRecordingRuleGroup(ctx context.Context, group *RecordingRuleGroupDefinition) (*RecordingRuleGroupDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	resp, err := c.inner.PostApiRecordingRuleGroupsWithResponse(ctx, *group)
	if err != nil {
		return nil, fmt.Errorf("dash0: create recording rule group failed: %w", err)
	}
	if resp.StatusCode() != http.StatusCreated {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return resp.JSON201, nil
}

// UpdateRecordingRuleGroup updates an existing recording rule group.
func (c *client) UpdateRecordingRuleGroup(ctx context.Context, originOrID string, group *RecordingRuleGroupDefinition) (*RecordingRuleGroupDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	resp, err := c.inner.PutApiRecordingRuleGroupsOriginOrIdWithResponse(ctx, originOrID, *group)
	if err != nil {
		return nil, fmt.Errorf("dash0: update recording rule group failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200, nil
}

// DeleteRecordingRuleGroup deletes a recording rule group by origin or ID.
func (c *client) DeleteRecordingRuleGroup(ctx context.Context, originOrID string, dataset *string) error {
	if err := c.requireAPI(); err != nil {
		return err
	}
	params := &DeleteApiRecordingRuleGroupsOriginOrIdParams{
		Dataset: dataset,
	}
	resp, err := c.inner.DeleteApiRecordingRuleGroupsOriginOrIdWithResponse(ctx, originOrID, params)
	if err != nil {
		return fmt.Errorf("dash0: delete recording rule group failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// ListRecordingRuleGroupsIter returns an iterator over all recording rule groups.
// This is a convenience wrapper around ListRecordingRuleGroups for consistent iteration patterns.
func (c *client) ListRecordingRuleGroupsIter(ctx context.Context, dataset *string) *Iter[RecordingRuleGroupDefinition] {
	items, err := c.ListRecordingRuleGroups(ctx, dataset)
	if err != nil {
		return newIterWithError[RecordingRuleGroupDefinition](err)
	}
	return newIter(items, false, nil, nil)
}
