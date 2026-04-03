package dash0

import (
	"context"
	"fmt"
	"net/http"
)

// ListSyntheticChecks retrieves all synthetic checks.
func (c *client) ListSyntheticChecks(ctx context.Context, dataset *string) ([]*SyntheticChecksApiListItem, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
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
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
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
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
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
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
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
	if err := c.requireAPI(); err != nil {
		return err
	}
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

// StripSyntheticCheckServerFields removes server-generated fields from a synthetic check definition.
func StripSyntheticCheckServerFields(check *SyntheticCheckDefinition) {
	if check == nil {
		return
	}
	if check.Metadata.Annotations != nil {
		check.Metadata.Annotations.Dash0ComdeletedAt = nil
	}
	if check.Metadata.Labels != nil {
		check.Metadata.Labels.Dash0Comversion = nil
		check.Metadata.Labels.Custom = nil
		check.Metadata.Labels.Dash0Comdataset = nil
		check.Metadata.Labels.Dash0Comorigin = nil
	}
}

// ClearSyntheticCheckID removes the ID from a synthetic check definition.
func ClearSyntheticCheckID(check *SyntheticCheckDefinition) {
	if check == nil {
		return
	}
	if check.Metadata.Labels != nil {
		check.Metadata.Labels.Dash0Comid = nil
	}
}

// GetSyntheticCheckID extracts the ID from a synthetic check definition.
func GetSyntheticCheckID(check *SyntheticCheckDefinition) string {
	if check == nil || check.Metadata.Labels == nil || check.Metadata.Labels.Dash0Comid == nil {
		return ""
	}
	return *check.Metadata.Labels.Dash0Comid
}

// GetSyntheticCheckName extracts the display name from a synthetic check
// definition, falling back to metadata.name if no display name is set.
func GetSyntheticCheckName(check *SyntheticCheckDefinition) string {
	if check == nil {
		return ""
	}
	if check.Spec.Display != nil && check.Spec.Display.Name != "" {
		return check.Spec.Display.Name
	}
	return check.Metadata.Name
}

// SetSyntheticCheckID sets the dash0.com/id label on a synthetic check
// definition, initializing the labels struct if needed.
func SetSyntheticCheckID(check *SyntheticCheckDefinition, id string) {
	if check == nil {
		return
	}
	if check.Metadata.Labels == nil {
		check.Metadata.Labels = &SyntheticCheckLabels{}
	}
	check.Metadata.Labels.Dash0Comid = &id
}

// SetSyntheticCheckIDIfAbsent sets the dash0.com/id label on a synthetic check
// definition only if it is not already set, initializing the labels struct if
// needed.
func SetSyntheticCheckIDIfAbsent(check *SyntheticCheckDefinition, id string) {
	if check == nil {
		return
	}
	if check.Metadata.Labels == nil {
		check.Metadata.Labels = &SyntheticCheckLabels{}
	}
	if check.Metadata.Labels.Dash0Comid == nil {
		check.Metadata.Labels.Dash0Comid = &id
	}
}
