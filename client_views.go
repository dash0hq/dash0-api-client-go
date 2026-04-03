package dash0

import (
	"context"
	"fmt"
	"net/http"
)

// ListViews retrieves all views.
func (c *client) ListViews(ctx context.Context, dataset *string) ([]*ViewApiListItem, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
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
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
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
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
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
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
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
	if err := c.requireAPI(); err != nil {
		return err
	}
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

// StripViewServerFields removes server-generated fields from a view definition.
func StripViewServerFields(view *ViewDefinition) {
	if view == nil {
		return
	}
	if view.Metadata.Annotations != nil {
		view.Metadata.Annotations.Dash0ComdeletedAt = nil
	}
	if view.Metadata.Labels != nil {
		view.Metadata.Labels.Dash0Comversion = nil
		view.Metadata.Labels.Dash0Comsource = nil
		view.Metadata.Labels.Dash0Comdataset = nil
		view.Metadata.Labels.Dash0Comorigin = nil
	}
}

// ClearViewID removes the ID from a view definition.
func ClearViewID(view *ViewDefinition) {
	if view == nil {
		return
	}
	if view.Metadata.Labels != nil {
		view.Metadata.Labels.Dash0Comid = nil
	}
}

// GetViewID extracts the ID from a view definition.
func GetViewID(view *ViewDefinition) string {
	if view == nil || view.Metadata.Labels == nil || view.Metadata.Labels.Dash0Comid == nil {
		return ""
	}
	return *view.Metadata.Labels.Dash0Comid
}

// GetViewName extracts the display name from a view definition, falling
// back to metadata.name if the display name is empty.
func GetViewName(view *ViewDefinition) string {
	if view == nil {
		return ""
	}
	if view.Spec.Display.Name != "" {
		return view.Spec.Display.Name
	}
	return view.Metadata.Name
}

// SetViewID sets the dash0.com/id label on a view definition, initializing
// the labels struct if needed.
func SetViewID(view *ViewDefinition, id string) {
	if view == nil {
		return
	}
	if view.Metadata.Labels == nil {
		view.Metadata.Labels = &ViewLabels{}
	}
	view.Metadata.Labels.Dash0Comid = &id
}

// SetViewIDIfAbsent sets the dash0.com/id label on a view definition only if
// it is not already set, initializing the labels struct if needed.
func SetViewIDIfAbsent(view *ViewDefinition, id string) {
	if view == nil {
		return
	}
	if view.Metadata.Labels == nil {
		view.Metadata.Labels = &ViewLabels{}
	}
	if view.Metadata.Labels.Dash0Comid == nil {
		view.Metadata.Labels.Dash0Comid = &id
	}
}
