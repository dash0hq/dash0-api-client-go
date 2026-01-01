package dash0

import (
	"context"
	"fmt"
	"net/http"
)

// ListSyntheticChecks retrieves all synthetic checks.
func (c *client) ListSyntheticChecks(ctx context.Context, dataset *string) ([]*SyntheticChecksApiListItem, error) {
	params := &GetApiSyntheticChecksParams{
		Dataset: dataset,
	}
	resp, err := c.inner.GetApiSyntheticChecksWithResponse(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("dash0: list synthetic checks failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return toPointerSlice(*resp.JSON200), nil
}

// GetSyntheticCheck retrieves a synthetic check by origin or ID.
func (c *client) GetSyntheticCheck(ctx context.Context, originOrID string, dataset *string) (*SyntheticCheckDefinition, error) {
	params := &GetApiSyntheticChecksOriginOrIdParams{
		Dataset: dataset,
	}
	resp, err := c.inner.GetApiSyntheticChecksOriginOrIdWithResponse(ctx, originOrID, params)
	if err != nil {
		return nil, fmt.Errorf("dash0: get synthetic check failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200, nil
}

// CreateSyntheticCheck creates a new synthetic check.
func (c *client) CreateSyntheticCheck(ctx context.Context, check *SyntheticCheckDefinition, dataset *string) (*SyntheticCheckDefinition, error) {
	params := &PostApiSyntheticChecksParams{
		Dataset: dataset,
	}
	resp, err := c.inner.PostApiSyntheticChecksWithResponse(ctx, params, *check)
	if err != nil {
		return nil, fmt.Errorf("dash0: create synthetic check failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200, nil
}

// UpdateSyntheticCheck updates an existing synthetic check.
func (c *client) UpdateSyntheticCheck(ctx context.Context, originOrID string, check *SyntheticCheckDefinition, dataset *string) (*SyntheticCheckDefinition, error) {
	params := &PutApiSyntheticChecksOriginOrIdParams{
		Dataset: dataset,
	}
	resp, err := c.inner.PutApiSyntheticChecksOriginOrIdWithResponse(ctx, originOrID, params, *check)
	if err != nil {
		return nil, fmt.Errorf("dash0: update synthetic check failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200, nil
}

// DeleteSyntheticCheck deletes a synthetic check by origin or ID.
func (c *client) DeleteSyntheticCheck(ctx context.Context, originOrID string, dataset *string) error {
	params := &DeleteApiSyntheticChecksOriginOrIdParams{
		Dataset: dataset,
	}
	resp, err := c.inner.DeleteApiSyntheticChecksOriginOrIdWithResponse(ctx, originOrID, params)
	if err != nil {
		return fmt.Errorf("dash0: delete synthetic check failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// ListSyntheticChecksIter returns an iterator over all synthetic checks.
// This is a convenience wrapper around ListSyntheticChecks for consistent iteration patterns.
func (c *client) ListSyntheticChecksIter(ctx context.Context, dataset *string) *Iter[SyntheticChecksApiListItem] {
	items, err := c.ListSyntheticChecks(ctx, dataset)
	if err != nil {
		return newIterWithError[SyntheticChecksApiListItem](err)
	}
	return newIter(items, false, nil, nil)
}
