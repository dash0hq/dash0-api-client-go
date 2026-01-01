package dash0

import (
	"context"
	"fmt"
	"net/http"
)

// ListViews retrieves all views.
func (c *client) ListViews(ctx context.Context, dataset *string) ([]*ViewApiListItem, error) {
	params := &GetApiViewsParams{
		Dataset: dataset,
	}
	resp, err := c.inner.GetApiViewsWithResponse(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("dash0: list views failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return toPointerSlice(*resp.JSON200), nil
}

// GetView retrieves a view by origin or ID.
func (c *client) GetView(ctx context.Context, originOrID string, dataset *string) (*ViewDefinition, error) {
	params := &GetApiViewsOriginOrIdParams{
		Dataset: dataset,
	}
	resp, err := c.inner.GetApiViewsOriginOrIdWithResponse(ctx, originOrID, params)
	if err != nil {
		return nil, fmt.Errorf("dash0: get view failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200, nil
}

// CreateView creates a new view.
func (c *client) CreateView(ctx context.Context, view *ViewDefinition, dataset *string) (*ViewDefinition, error) {
	params := &PostApiViewsParams{
		Dataset: dataset,
	}
	resp, err := c.inner.PostApiViewsWithResponse(ctx, params, *view)
	if err != nil {
		return nil, fmt.Errorf("dash0: create view failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200, nil
}

// UpdateView updates an existing view.
func (c *client) UpdateView(ctx context.Context, originOrID string, view *ViewDefinition, dataset *string) (*ViewDefinition, error) {
	params := &PutApiViewsOriginOrIdParams{
		Dataset: dataset,
	}
	resp, err := c.inner.PutApiViewsOriginOrIdWithResponse(ctx, originOrID, params, *view)
	if err != nil {
		return nil, fmt.Errorf("dash0: update view failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200, nil
}

// DeleteView deletes a view by origin or ID.
func (c *client) DeleteView(ctx context.Context, originOrID string, dataset *string) error {
	params := &DeleteApiViewsOriginOrIdParams{
		Dataset: dataset,
	}
	resp, err := c.inner.DeleteApiViewsOriginOrIdWithResponse(ctx, originOrID, params)
	if err != nil {
		return fmt.Errorf("dash0: delete view failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// ListViewsIter returns an iterator over all views.
// This is a convenience wrapper around ListViews for consistent iteration patterns.
func (c *client) ListViewsIter(ctx context.Context, dataset *string) *Iter[ViewApiListItem] {
	items, err := c.ListViews(ctx, dataset)
	if err != nil {
		return newIterWithError[ViewApiListItem](err)
	}
	return newIter(items, false, nil, nil)
}
