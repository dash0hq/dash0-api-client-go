package dash0

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// SpamFilter is a convenience alias for SpamFilterDefinition.
type SpamFilter = SpamFilterDefinition

// ListSpamFilters retrieves all spam filters.
func (c *client) ListSpamFilters(ctx context.Context, dataset *string) ([]*SpamFilter, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	params := &GetApiSpamFiltersParams{
		Dataset: dataset,
	}
	resp, err := c.inner.GetApiSpamFiltersWithResponse(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("dash0: list spam filters failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return toPointerSlice(resp.JSON200.SpamFilters), nil
}

// GetSpamFilter retrieves a spam filter by origin or ID.
func (c *client) GetSpamFilter(ctx context.Context, originOrID string, dataset *string) (*SpamFilter, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	params := &GetApiSpamFiltersOriginOrIdParams{
		Dataset: dataset,
	}
	resp, err := c.inner.GetApiSpamFiltersOriginOrIdWithResponse(ctx, originOrID, params)
	if err != nil {
		return nil, fmt.Errorf("dash0: get spam filter failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200, nil
}

// CreateSpamFilter creates a new spam filter.
func (c *client) CreateSpamFilter(ctx context.Context, filter *SpamFilter, dataset *string) (*SpamFilter, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	params := &PostApiSpamFiltersParams{
		Dataset: dataset,
	}
	resp, err := c.inner.PostApiSpamFiltersWithResponse(ctx, params, *filter)
	if err != nil {
		return nil, fmt.Errorf("dash0: create spam filter failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusCreated {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	var created SpamFilter
	if err := json.Unmarshal(resp.Body, &created); err != nil {
		return nil, fmt.Errorf("dash0: failed to parse spam filter response: %w", err)
	}
	return &created, nil
}

// UpdateSpamFilter updates an existing spam filter.
func (c *client) UpdateSpamFilter(ctx context.Context, originOrID string, filter *SpamFilter, dataset *string) (*SpamFilter, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	params := &PutApiSpamFiltersOriginOrIdParams{
		Dataset: dataset,
	}
	resp, err := c.inner.PutApiSpamFiltersOriginOrIdWithResponse(ctx, originOrID, params, *filter)
	if err != nil {
		return nil, fmt.Errorf("dash0: update spam filter failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200, nil
}

// DeleteSpamFilter deletes a spam filter by origin or ID.
func (c *client) DeleteSpamFilter(ctx context.Context, originOrID string, dataset *string) error {
	if err := c.requireAPI(); err != nil {
		return err
	}
	params := &DeleteApiSpamFiltersOriginOrIdParams{
		Dataset: dataset,
	}
	resp, err := c.inner.DeleteApiSpamFiltersOriginOrIdWithResponse(ctx, originOrID, params)
	if err != nil {
		return fmt.Errorf("dash0: delete spam filter failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// ListSpamFiltersIter returns an iterator over all spam filters.
// This is a convenience wrapper around ListSpamFilters for consistent iteration patterns.
func (c *client) ListSpamFiltersIter(ctx context.Context, dataset *string) *Iter[SpamFilter] {
	items, err := c.ListSpamFilters(ctx, dataset)
	if err != nil {
		return newIterWithError[SpamFilter](err)
	}
	return newIter(items, false, nil, nil)
}

// StripSpamFilterServerFields removes server-generated fields from a
// spam filter definition.
func StripSpamFilterServerFields(filter *SpamFilter) {
	if filter == nil {
		return
	}
	if filter.Metadata.Annotations != nil {
		filter.Metadata.Annotations.Dash0Comenabled = nil
	}
	if filter.Metadata.Labels != nil {
		filter.Metadata.Labels.Dash0Comid = nil
		filter.Metadata.Labels.Dash0Comsource = nil
	}
}

// ClearSpamFilterID removes the ID from a spam filter definition.
func ClearSpamFilterID(filter *SpamFilter) {
	if filter == nil {
		return
	}
	if filter.Metadata.Labels != nil {
		filter.Metadata.Labels.Dash0Comid = nil
	}
}

// GetSpamFilterID extracts the ID from a spam filter definition.
func GetSpamFilterID(filter *SpamFilter) string {
	if filter == nil || filter.Metadata.Labels == nil || filter.Metadata.Labels.Dash0Comid == nil {
		return ""
	}
	return *filter.Metadata.Labels.Dash0Comid
}

// GetSpamFilterName extracts the display name from a spam filter definition.
func GetSpamFilterName(filter *SpamFilter) string {
	if filter == nil {
		return ""
	}
	return filter.Metadata.Name
}

// GetSpamFilterDataset extracts the dataset label from a spam filter definition.
func GetSpamFilterDataset(filter *SpamFilter) string {
	if filter == nil || filter.Metadata.Labels == nil || filter.Metadata.Labels.Dash0Comdataset == nil {
		return ""
	}
	return *filter.Metadata.Labels.Dash0Comdataset
}

// SetSpamFilterDataset sets the dash0.com/dataset label on a spam filter
// definition, initializing the labels struct if needed.
func SetSpamFilterDataset(filter *SpamFilter, dataset string) {
	if filter == nil {
		return
	}
	if filter.Metadata.Labels == nil {
		filter.Metadata.Labels = &SpamFilterLabels{}
	}
	filter.Metadata.Labels.Dash0Comdataset = &dataset
}

// SetSpamFilterID sets the dash0.com/id label on a spam filter definition,
// initializing the labels struct if needed.
func SetSpamFilterID(filter *SpamFilter, id string) {
	if filter == nil {
		return
	}
	if filter.Metadata.Labels == nil {
		filter.Metadata.Labels = &SpamFilterLabels{}
	}
	filter.Metadata.Labels.Dash0Comid = &id
}

// SetSpamFilterIDIfAbsent sets the dash0.com/id label on a spam filter
// definition only if it is not already set, initializing the labels struct
// if needed.
func SetSpamFilterIDIfAbsent(filter *SpamFilter, id string) {
	if filter == nil {
		return
	}
	if filter.Metadata.Labels == nil {
		filter.Metadata.Labels = &SpamFilterLabels{}
	}
	if filter.Metadata.Labels.Dash0Comid == nil {
		filter.Metadata.Labels.Dash0Comid = &id
	}
}
