package dash0

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// sloDisplayNameAnnotation is the annotation key that carries the SLO display
// name. It lives in the annotations map rather than as a typed field, so it is
// read and written through SloAnnotations.Get/Set (AdditionalProperties).
const sloDisplayNameAnnotation = "dash0.com/display-name"

// ListSLOs retrieves all SLOs.
func (c *client) ListSLOs(ctx context.Context, dataset *string) ([]*SloDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	params := &GetApiSlosParams{
		Dataset: dataset,
	}
	resp, err := c.inner.GetApiSlosWithResponse(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("dash0: list SLOs failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return toPointerSlice(*resp.JSON200), nil
}

// GetSLO retrieves an SLO by origin or ID.
func (c *client) GetSLO(ctx context.Context, originOrID string, dataset *string) (*SloDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	params := &GetApiSlosOriginOrIdParams{
		Dataset: dataset,
	}
	resp, err := c.inner.GetApiSlosOriginOrIdWithResponse(ctx, originOrID, params)
	if err != nil {
		return nil, fmt.Errorf("dash0: get SLO failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200, nil
}

// CreateSLO creates a new SLO.
func (c *client) CreateSLO(ctx context.Context, slo *SloDefinition, dataset *string) (*SloDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	params := &PostApiSlosParams{
		Dataset: dataset,
	}
	resp, err := c.inner.PostApiSlosWithResponse(ctx, params, *slo)
	if err != nil {
		return nil, fmt.Errorf("dash0: create SLO failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusCreated {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	var created SloDefinition
	if err := json.Unmarshal(resp.Body, &created); err != nil {
		return nil, fmt.Errorf("dash0: failed to parse SLO response: %w", err)
	}
	return &created, nil
}

// UpdateSLO updates an existing SLO (create-or-replace).
func (c *client) UpdateSLO(ctx context.Context, originOrID string, slo *SloDefinition, dataset *string) (*SloDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	params := &PutApiSlosOriginOrIdParams{
		Dataset: dataset,
	}
	resp, err := c.inner.PutApiSlosOriginOrIdWithResponse(ctx, originOrID, params, *slo)
	if err != nil {
		return nil, fmt.Errorf("dash0: update SLO failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200, nil
}

// DeleteSLO deletes an SLO by origin or ID.
func (c *client) DeleteSLO(ctx context.Context, originOrID string, dataset *string) error {
	if err := c.requireAPI(); err != nil {
		return err
	}
	params := &DeleteApiSlosOriginOrIdParams{
		Dataset: dataset,
	}
	resp, err := c.inner.DeleteApiSlosOriginOrIdWithResponse(ctx, originOrID, params)
	if err != nil {
		return fmt.Errorf("dash0: delete SLO failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// ListSLOsIter returns an iterator over all SLOs.
// This is a convenience wrapper around ListSLOs for consistent iteration patterns.
func (c *client) ListSLOsIter(ctx context.Context, dataset *string) *Iter[SloDefinition] {
	items, err := c.ListSLOs(ctx, dataset)
	if err != nil {
		return newIterWithError[SloDefinition](err)
	}
	return newIter(items, false, nil, nil)
}

// StripSLOServerFields removes server-generated fields from an SLO definition.
func StripSLOServerFields(slo *SloDefinition) {
	if slo == nil {
		return
	}
	if slo.Metadata.Annotations != nil {
		slo.Metadata.Annotations.Dash0ComcreatedAt = nil
		slo.Metadata.Annotations.Dash0ComupdatedAt = nil
		slo.Metadata.Annotations.Dash0ComdeletedAt = nil
	}
	if slo.Metadata.Labels != nil {
		slo.Metadata.Labels.Dash0Comversion = nil
		slo.Metadata.Labels.Dash0Comorigin = nil
		slo.Metadata.Labels.Dash0Comdataset = nil
		slo.Metadata.Labels.Dash0Comsource = nil
	}
}

// ClearSLOID removes the ID from an SLO definition.
func ClearSLOID(slo *SloDefinition) {
	if slo == nil {
		return
	}
	if slo.Metadata.Labels != nil {
		slo.Metadata.Labels.Dash0Comid = nil
	}
}

// GetSLODataset extracts the dataset from an SLO definition.
func GetSLODataset(slo *SloDefinition) string {
	if slo == nil || slo.Metadata.Labels == nil || slo.Metadata.Labels.Dash0Comdataset == nil {
		return ""
	}
	return *slo.Metadata.Labels.Dash0Comdataset
}

// SetSLODataset sets the dataset on an SLO definition, initializing the labels
// struct if needed.
func SetSLODataset(slo *SloDefinition, dataset string) {
	if slo == nil {
		return
	}
	if slo.Metadata.Labels == nil {
		slo.Metadata.Labels = &SloLabels{}
	}
	slo.Metadata.Labels.Dash0Comdataset = &dataset
}

// GetSLOID extracts the ID from an SLO definition.
func GetSLOID(slo *SloDefinition) string {
	if slo == nil || slo.Metadata.Labels == nil || slo.Metadata.Labels.Dash0Comid == nil {
		return ""
	}
	return *slo.Metadata.Labels.Dash0Comid
}

// GetSLOName extracts the display name from an SLO definition, reading the
// dash0.com/display-name annotation and falling back to metadata.name when it
// is not set.
func GetSLOName(slo *SloDefinition) string {
	if slo == nil {
		return ""
	}
	if slo.Metadata.Annotations != nil {
		if name, ok := slo.Metadata.Annotations.Get(sloDisplayNameAnnotation); ok && name != "" {
			return name
		}
	}
	return slo.Metadata.Name
}

// SetSLOID sets the dash0.com/id label on an SLO definition, initializing the
// labels struct if needed.
func SetSLOID(slo *SloDefinition, id string) {
	if slo == nil {
		return
	}
	if slo.Metadata.Labels == nil {
		slo.Metadata.Labels = &SloLabels{}
	}
	slo.Metadata.Labels.Dash0Comid = &id
}

// SetSLOIDIfAbsent sets the dash0.com/id label on an SLO definition only if it
// is not already set, initializing the labels struct if needed.
func SetSLOIDIfAbsent(slo *SloDefinition, id string) {
	if slo == nil {
		return
	}
	if slo.Metadata.Labels == nil {
		slo.Metadata.Labels = &SloLabels{}
	}
	if slo.Metadata.Labels.Dash0Comid == nil {
		slo.Metadata.Labels.Dash0Comid = &id
	}
}
