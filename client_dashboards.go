package dash0

import (
	"context"
	"fmt"
	"net/http"
)

// ListDashboards retrieves all dashboards.
func (c *client) ListDashboards(ctx context.Context, dataset *string) ([]*DashboardApiListItem, error) {
	params := &GetApiDashboardsParams{
		Dataset: dataset,
	}
	resp, err := c.inner.GetApiDashboardsWithResponse(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("dash0: list dashboards failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return toPointerSlice(*resp.JSON200), nil
}

// GetDashboard retrieves a dashboard by origin or ID.
func (c *client) GetDashboard(ctx context.Context, originOrID string, dataset *string) (*DashboardDefinition, error) {
	params := &GetApiDashboardsOriginOrIdParams{
		Dataset: dataset,
	}
	resp, err := c.inner.GetApiDashboardsOriginOrIdWithResponse(ctx, originOrID, params)
	if err != nil {
		return nil, fmt.Errorf("dash0: get dashboard failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200, nil
}

// CreateDashboard creates a new dashboard.
func (c *client) CreateDashboard(ctx context.Context, dashboard *DashboardDefinition, dataset *string) (*DashboardDefinition, error) {
	params := &PostApiDashboardsParams{
		Dataset: dataset,
	}
	resp, err := c.inner.PostApiDashboardsWithResponse(ctx, params, *dashboard)
	if err != nil {
		return nil, fmt.Errorf("dash0: create dashboard failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200, nil
}

// UpdateDashboard updates an existing dashboard.
func (c *client) UpdateDashboard(ctx context.Context, originOrID string, dashboard *DashboardDefinition, dataset *string) (*DashboardDefinition, error) {
	params := &PutApiDashboardsOriginOrIdParams{
		Dataset: dataset,
	}
	resp, err := c.inner.PutApiDashboardsOriginOrIdWithResponse(ctx, originOrID, params, *dashboard)
	if err != nil {
		return nil, fmt.Errorf("dash0: update dashboard failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200, nil
}

// DeleteDashboard deletes a dashboard by origin or ID.
func (c *client) DeleteDashboard(ctx context.Context, originOrID string, dataset *string) error {
	params := &DeleteApiDashboardsOriginOrIdParams{
		Dataset: dataset,
	}
	resp, err := c.inner.DeleteApiDashboardsOriginOrIdWithResponse(ctx, originOrID, params)
	if err != nil {
		return fmt.Errorf("dash0: delete dashboard failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// ListDashboardsIter returns an iterator over all dashboards.
// This is a convenience wrapper around ListDashboards for consistent iteration patterns.
func (c *client) ListDashboardsIter(ctx context.Context, dataset *string) *Iter[DashboardApiListItem] {
	items, err := c.ListDashboards(ctx, dataset)
	if err != nil {
		return newIterWithError[DashboardApiListItem](err)
	}
	return newIter(items, false, nil, nil)
}
