package dash0

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ListTimeSeriesAggregations retrieves all time series aggregations.
func (c *client) ListTimeSeriesAggregations(ctx context.Context, dataset *string) ([]*TimeSeriesAggregationDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	params := &GetApiTimeSeriesAggregationsParams{
		Dataset: dataset,
	}
	resp, err := c.inner.GetApiTimeSeriesAggregationsWithResponse(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("dash0: list time series aggregations failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return toPointerSlice(resp.JSON200.TimeSeriesAggregations), nil
}

// GetTimeSeriesAggregation retrieves a time series aggregation by origin or ID.
func (c *client) GetTimeSeriesAggregation(ctx context.Context, originOrID string, dataset *string) (*TimeSeriesAggregationDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	params := &GetApiTimeSeriesAggregationsOriginOrIdParams{
		Dataset: dataset,
	}
	resp, err := c.inner.GetApiTimeSeriesAggregationsOriginOrIdWithResponse(ctx, originOrID, params)
	if err != nil {
		return nil, fmt.Errorf("dash0: get time series aggregation failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return resp.JSON200, nil
}

// CreateTimeSeriesAggregation creates a new time series aggregation.
func (c *client) CreateTimeSeriesAggregation(ctx context.Context, aggregation *TimeSeriesAggregationDefinition, dataset *string) (*TimeSeriesAggregationDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	params := &PostApiTimeSeriesAggregationsParams{
		Dataset: dataset,
	}
	resp, err := c.inner.PostApiTimeSeriesAggregationsWithResponse(ctx, params, *aggregation)
	if err != nil {
		return nil, fmt.Errorf("dash0: create time series aggregation failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusCreated {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	var created TimeSeriesAggregationDefinition
	if err := json.Unmarshal(resp.Body, &created); err != nil {
		return nil, fmt.Errorf("dash0: failed to parse time series aggregation response: %w", err)
	}
	return &created, nil
}

// UpdateTimeSeriesAggregation updates an existing time series aggregation.
func (c *client) UpdateTimeSeriesAggregation(ctx context.Context, originOrID string, aggregation *TimeSeriesAggregationDefinition, dataset *string) (*TimeSeriesAggregationDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	params := &PutApiTimeSeriesAggregationsOriginOrIdParams{
		Dataset: dataset,
	}
	resp, err := c.inner.PutApiTimeSeriesAggregationsOriginOrIdWithResponse(ctx, originOrID, params, *aggregation)
	if err != nil {
		return nil, fmt.Errorf("dash0: update time series aggregation failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return resp.JSON200, nil
}

// DeleteTimeSeriesAggregation deletes a time series aggregation by origin or ID.
func (c *client) DeleteTimeSeriesAggregation(ctx context.Context, originOrID string, dataset *string) error {
	if err := c.requireAPI(); err != nil {
		return err
	}
	params := &DeleteApiTimeSeriesAggregationsOriginOrIdParams{
		Dataset: dataset,
	}
	resp, err := c.inner.DeleteApiTimeSeriesAggregationsOriginOrIdWithResponse(ctx, originOrID, params)
	if err != nil {
		return fmt.Errorf("dash0: delete time series aggregation failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// ListTimeSeriesAggregationsIter returns an iterator over all time series aggregations.
// This is a convenience wrapper around ListTimeSeriesAggregations for consistent iteration patterns.
func (c *client) ListTimeSeriesAggregationsIter(ctx context.Context, dataset *string) *Iter[TimeSeriesAggregationDefinition] {
	items, err := c.ListTimeSeriesAggregations(ctx, dataset)
	if err != nil {
		return newIterWithError[TimeSeriesAggregationDefinition](err)
	}
	return newIter(items, false, nil, nil)
}

// StripTimeSeriesAggregationServerFields removes server-generated fields from a time series aggregation definition.
//
// All three annotation timestamps are server-assigned and are cleared.
// The dash0.com/id label is preserved, so a stripped definition still addresses the same aggregation on update.
// Callers that want a definition suitable for creating a new aggregation should also call [ClearTimeSeriesAggregationID].
func StripTimeSeriesAggregationServerFields(aggregation *TimeSeriesAggregationDefinition) {
	if aggregation == nil {
		return
	}
	if aggregation.Metadata.Annotations != nil {
		aggregation.Metadata.Annotations.Dash0ComcreatedAt = nil
		aggregation.Metadata.Annotations.Dash0ComupdatedAt = nil
		aggregation.Metadata.Annotations.Dash0ComdeletedAt = nil
	}
	if aggregation.Metadata.Labels != nil {
		aggregation.Metadata.Labels.Dash0Comversion = nil
		aggregation.Metadata.Labels.Dash0Comsource = nil
		aggregation.Metadata.Labels.Dash0Comdataset = nil
		aggregation.Metadata.Labels.Dash0Comorigin = nil
	}
}

// ClearTimeSeriesAggregationID removes the ID from a time series aggregation definition.
func ClearTimeSeriesAggregationID(aggregation *TimeSeriesAggregationDefinition) {
	if aggregation == nil {
		return
	}
	if aggregation.Metadata.Labels != nil {
		aggregation.Metadata.Labels.Dash0Comid = nil
	}
}

// GetTimeSeriesAggregationDataset extracts the dataset from a time series aggregation definition.
func GetTimeSeriesAggregationDataset(aggregation *TimeSeriesAggregationDefinition) string {
	if aggregation == nil || aggregation.Metadata.Labels == nil || aggregation.Metadata.Labels.Dash0Comdataset == nil {
		return ""
	}
	return *aggregation.Metadata.Labels.Dash0Comdataset
}

// SetTimeSeriesAggregationDataset sets the dash0.com/dataset label on a time series aggregation definition, initializing the labels struct if needed.
func SetTimeSeriesAggregationDataset(aggregation *TimeSeriesAggregationDefinition, dataset string) {
	if aggregation == nil {
		return
	}
	if aggregation.Metadata.Labels == nil {
		aggregation.Metadata.Labels = &TimeSeriesAggregationLabels{}
	}
	aggregation.Metadata.Labels.Dash0Comdataset = &dataset
}

// GetTimeSeriesAggregationID extracts the ID from a time series aggregation definition.
func GetTimeSeriesAggregationID(aggregation *TimeSeriesAggregationDefinition) string {
	if aggregation == nil || aggregation.Metadata.Labels == nil || aggregation.Metadata.Labels.Dash0Comid == nil {
		return ""
	}
	return *aggregation.Metadata.Labels.Dash0Comid
}

// GetTimeSeriesAggregationName extracts the display name from a time series aggregation definition.
// It falls back to metadata.name when the display block is absent or its name is empty.
func GetTimeSeriesAggregationName(aggregation *TimeSeriesAggregationDefinition) string {
	if aggregation == nil {
		return ""
	}
	if aggregation.Spec.Display != nil && aggregation.Spec.Display.Name != "" {
		return aggregation.Spec.Display.Name
	}
	return aggregation.Metadata.Name
}

// SetTimeSeriesAggregationID sets the dash0.com/id label on a time series aggregation definition, initializing the labels struct if needed.
func SetTimeSeriesAggregationID(aggregation *TimeSeriesAggregationDefinition, id string) {
	if aggregation == nil {
		return
	}
	if aggregation.Metadata.Labels == nil {
		aggregation.Metadata.Labels = &TimeSeriesAggregationLabels{}
	}
	aggregation.Metadata.Labels.Dash0Comid = &id
}

// SetTimeSeriesAggregationIDIfAbsent sets the dash0.com/id label on a time series aggregation definition only if it is not already set.
// It initializes the labels struct if needed.
func SetTimeSeriesAggregationIDIfAbsent(aggregation *TimeSeriesAggregationDefinition, id string) {
	if aggregation == nil {
		return
	}
	if aggregation.Metadata.Labels == nil {
		aggregation.Metadata.Labels = &TimeSeriesAggregationLabels{}
	}
	if aggregation.Metadata.Labels.Dash0Comid == nil {
		aggregation.Metadata.Labels.Dash0Comid = &id
	}
}
