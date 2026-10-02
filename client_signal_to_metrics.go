package dash0

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ListSignalToMetrics retrieves all signal-to-metrics rules.
// Pass [WithOriginPrefix] to restrict the result to rules whose origin starts with a given prefix.
//
// The endpoint is offset-paginated; ListSignalToMetrics follows pages until the server reports no more results.
func (c *client) ListSignalToMetrics(ctx context.Context, dataset *string, opts ...ListOption) ([]*SignalToMetricsDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	options := NewListOptions(opts...)
	var all []SignalToMetricsDefinition
	for {
		params := &GetApiSignalToMetricsParams{
			Dataset:      dataset,
			OriginPrefix: options.OriginPrefix,
		}
		if len(all) > 0 {
			params.Offset = Ptr(len(all))
		}
		resp, err := c.inner.GetApiSignalToMetricsWithResponse(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("dash0: list signal-to-metrics failed: %w", err)
		}
		if resp.StatusCode() != http.StatusOK {
			return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
		}
		if resp.JSON200 == nil {
			return nil, fmt.Errorf("dash0: unexpected nil response")
		}
		page := resp.JSON200.SignalToMetrics
		all = append(all, page...)
		if !BoolValue(resp.JSON200.HasMore) || len(page) == 0 {
			break
		}
	}
	return toPointerSlice(all), nil
}

// GetSignalToMetrics retrieves a signal-to-metrics rule by origin or ID.
func (c *client) GetSignalToMetrics(ctx context.Context, originOrID string, dataset *string) (*SignalToMetricsDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	params := &GetApiSignalToMetricsOriginOrIdParams{
		Dataset: dataset,
	}
	resp, err := c.inner.GetApiSignalToMetricsOriginOrIdWithResponse(ctx, originOrID, params)
	if err != nil {
		return nil, fmt.Errorf("dash0: get signal-to-metrics failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200, nil
}

// CreateSignalToMetrics creates a new signal-to-metrics rule.
func (c *client) CreateSignalToMetrics(ctx context.Context, rule *SignalToMetricsDefinition, dataset *string) (*SignalToMetricsDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	if rule == nil {
		return nil, fmt.Errorf("dash0: create signal-to-metrics requires a non-nil rule")
	}
	params := &PostApiSignalToMetricsParams{
		Dataset: dataset,
	}
	resp, err := c.inner.PostApiSignalToMetricsWithResponse(ctx, params, *rule)
	if err != nil {
		return nil, fmt.Errorf("dash0: create signal-to-metrics failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusCreated {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	var created SignalToMetricsDefinition
	if err := json.Unmarshal(resp.Body, &created); err != nil {
		return nil, fmt.Errorf("dash0: failed to parse signal-to-metrics response: %w", err)
	}
	return &created, nil
}

// UpdateSignalToMetrics updates an existing signal-to-metrics rule.
func (c *client) UpdateSignalToMetrics(ctx context.Context, originOrID string, rule *SignalToMetricsDefinition, dataset *string) (*SignalToMetricsDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	if rule == nil {
		return nil, fmt.Errorf("dash0: update signal-to-metrics requires a non-nil rule")
	}
	params := &PutApiSignalToMetricsOriginOrIdParams{
		Dataset: dataset,
	}
	resp, err := c.inner.PutApiSignalToMetricsOriginOrIdWithResponse(ctx, originOrID, params, *rule)
	if err != nil {
		return nil, fmt.Errorf("dash0: update signal-to-metrics failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200, nil
}

// DeleteSignalToMetrics deletes a signal-to-metrics rule by origin or ID.
func (c *client) DeleteSignalToMetrics(ctx context.Context, originOrID string, dataset *string) error {
	if err := c.requireAPI(); err != nil {
		return err
	}
	params := &DeleteApiSignalToMetricsOriginOrIdParams{
		Dataset: dataset,
	}
	resp, err := c.inner.DeleteApiSignalToMetricsOriginOrIdWithResponse(ctx, originOrID, params)
	if err != nil {
		return fmt.Errorf("dash0: delete signal-to-metrics failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// ListSignalToMetricsIter returns an iterator over all signal-to-metrics rules.
// This is a convenience wrapper around ListSignalToMetrics for consistent iteration patterns.
func (c *client) ListSignalToMetricsIter(ctx context.Context, dataset *string, opts ...ListOption) *Iter[SignalToMetricsDefinition] {
	items, err := c.ListSignalToMetrics(ctx, dataset, opts...)
	if err != nil {
		return newIterWithError[SignalToMetricsDefinition](err)
	}
	return newIter(items, false, nil, nil)
}

// StripSignalToMetricsServerFields removes server-generated fields from a signal-to-metrics rule.
//
// All three annotation timestamps are server-assigned and are cleared.
// The dash0.com/id label is preserved, so a stripped rule still addresses the same rule on update.
// Callers that want a definition suitable for creating a new rule should also call [ClearSignalToMetricsID].
func StripSignalToMetricsServerFields(rule *SignalToMetricsDefinition) {
	if rule == nil {
		return
	}
	if rule.Metadata.Annotations != nil {
		rule.Metadata.Annotations.Dash0ComcreatedAt = nil
		rule.Metadata.Annotations.Dash0ComupdatedAt = nil
		rule.Metadata.Annotations.Dash0ComdeletedAt = nil
	}
	if rule.Metadata.Labels != nil {
		rule.Metadata.Labels.Dash0Comversion = nil
		rule.Metadata.Labels.Dash0Comsource = nil
		rule.Metadata.Labels.Dash0Comdataset = nil
		rule.Metadata.Labels.Dash0Comorigin = nil
	}
}

// ClearSignalToMetricsID removes the ID from a signal-to-metrics rule.
func ClearSignalToMetricsID(rule *SignalToMetricsDefinition) {
	if rule == nil {
		return
	}
	if rule.Metadata.Labels != nil {
		rule.Metadata.Labels.Dash0Comid = nil
	}
}

// GetSignalToMetricsDataset extracts the dataset from a signal-to-metrics rule.
func GetSignalToMetricsDataset(rule *SignalToMetricsDefinition) string {
	if rule == nil || rule.Metadata.Labels == nil || rule.Metadata.Labels.Dash0Comdataset == nil {
		return ""
	}
	return *rule.Metadata.Labels.Dash0Comdataset
}

// SetSignalToMetricsDataset sets the dash0.com/dataset label on a signal-to-metrics rule, initializing the labels struct if needed.
func SetSignalToMetricsDataset(rule *SignalToMetricsDefinition, dataset string) {
	if rule == nil {
		return
	}
	if rule.Metadata.Labels == nil {
		rule.Metadata.Labels = &SignalToMetricsLabels{}
	}
	rule.Metadata.Labels.Dash0Comdataset = &dataset
}

// GetSignalToMetricsID extracts the ID from a signal-to-metrics rule.
func GetSignalToMetricsID(rule *SignalToMetricsDefinition) string {
	if rule == nil || rule.Metadata.Labels == nil || rule.Metadata.Labels.Dash0Comid == nil {
		return ""
	}
	return *rule.Metadata.Labels.Dash0Comid
}

// GetSignalToMetricsName extracts the display name from a signal-to-metrics rule.
// It falls back to metadata.name when the display name is empty.
func GetSignalToMetricsName(rule *SignalToMetricsDefinition) string {
	if rule == nil {
		return ""
	}
	if rule.Spec.Display.Name != "" {
		return rule.Spec.Display.Name
	}
	return rule.Metadata.Name
}

// SetSignalToMetricsID sets the dash0.com/id label on a signal-to-metrics rule, initializing the labels struct if needed.
func SetSignalToMetricsID(rule *SignalToMetricsDefinition, id string) {
	if rule == nil {
		return
	}
	if rule.Metadata.Labels == nil {
		rule.Metadata.Labels = &SignalToMetricsLabels{}
	}
	rule.Metadata.Labels.Dash0Comid = &id
}

// SetSignalToMetricsIDIfAbsent sets the dash0.com/id label on a signal-to-metrics rule only if it is not already set.
// It initializes the labels struct if needed.
func SetSignalToMetricsIDIfAbsent(rule *SignalToMetricsDefinition, id string) {
	if rule == nil {
		return
	}
	if rule.Metadata.Labels == nil {
		rule.Metadata.Labels = &SignalToMetricsLabels{}
	}
	if rule.Metadata.Labels.Dash0Comid == nil {
		rule.Metadata.Labels.Dash0Comid = &id
	}
}
