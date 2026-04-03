package dash0

import (
	"context"
	"fmt"
	"net/http"
)

// ListDashboards retrieves all dashboards.
func (c *client) ListDashboards(ctx context.Context, dataset *string) ([]*DashboardApiListItem, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
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
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
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
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
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
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
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
	if err := c.requireAPI(); err != nil {
		return err
	}
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

// StripDashboardServerFields removes server-generated metadata fields from a dashboard definition.
func StripDashboardServerFields(dashboard *DashboardDefinition) {
	if dashboard.Metadata.Annotations != nil {
		dashboard.Metadata.Annotations.Dash0ComdeletedAt = nil
	}
	dashboard.Metadata.CreatedAt = nil
	dashboard.Metadata.UpdatedAt = nil
	dashboard.Metadata.Version = nil
	if dashboard.Metadata.Dash0Extensions != nil {
		dashboard.Metadata.Dash0Extensions.Dataset = nil
	}
}

// ClearDashboardID removes the ID from a dashboard definition.
func ClearDashboardID(dashboard *DashboardDefinition) {
	if dashboard.Metadata.Dash0Extensions != nil {
		dashboard.Metadata.Dash0Extensions.Id = nil
	}
}

// GetDashboardID extracts the ID from a dashboard definition.
func GetDashboardID(dashboard *DashboardDefinition) string {
	if dashboard.Metadata.Dash0Extensions != nil && dashboard.Metadata.Dash0Extensions.Id != nil && *dashboard.Metadata.Dash0Extensions.Id != "" {
		return *dashboard.Metadata.Dash0Extensions.Id
	}
	return ""
}

// GetDashboardName extracts the display name from a dashboard definition.
func GetDashboardName(dashboard *DashboardDefinition) string {
	if dashboard == nil || dashboard.Spec == nil {
		return ""
	}
	display, ok := dashboard.Spec["display"].(map[string]any)
	if !ok {
		return ""
	}
	name, ok := display["name"].(string)
	if !ok {
		return ""
	}
	return name
}

// SetDashboardID sets the ID on a dashboard definition, initializing the
// dash0Extensions struct if needed.
func SetDashboardID(dashboard *DashboardDefinition, id string) {
	if dashboard.Metadata.Dash0Extensions == nil {
		dashboard.Metadata.Dash0Extensions = &DashboardMetadataExtensions{}
	}
	dashboard.Metadata.Dash0Extensions.Id = &id
}

// SetDashboardIDIfAbsent sets the ID on a dashboard definition only if it is
// not already set, initializing the dash0Extensions struct if needed.
func SetDashboardIDIfAbsent(dashboard *DashboardDefinition, id string) {
	if dashboard.Metadata.Dash0Extensions == nil {
		dashboard.Metadata.Dash0Extensions = &DashboardMetadataExtensions{}
	}
	if dashboard.Metadata.Dash0Extensions.Id == nil {
		dashboard.Metadata.Dash0Extensions.Id = &id
	}
}
