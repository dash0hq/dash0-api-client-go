package dash0

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ListDatasets retrieves all datasets in the organization.
func (c *client) ListDatasets(ctx context.Context) ([]*DatasetDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	resp, err := c.inner.GetApiDatasetsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("dash0: list datasets failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return toPointerSlice(*resp.JSON200), nil
}

// GetDataset retrieves a dataset by origin or slug.
func (c *client) GetDataset(ctx context.Context, originOrID string) (*DatasetDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	resp, err := c.inner.GetApiDatasetsOriginOrIdWithResponse(ctx, originOrID)
	if err != nil {
		return nil, fmt.Errorf("dash0: get dataset failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return resp.JSON200, nil
}

// CreateDataset creates a new dataset.
// The slug comes from spec.slug or the dash0.com/id label, which must match when both are set, and cannot be changed afterwards.
// When neither is set, the server derives the slug from metadata.name.
// The apiVersion field is required and must be "dash0.com/v1alpha1".
// The server ignores spec.permissions on create, so set permissions with a later [Client.UpdateDataset].
func (c *client) CreateDataset(ctx context.Context, dataset *DatasetDefinition) (*DatasetDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	if dataset == nil {
		return nil, fmt.Errorf("dash0: create dataset requires a non-nil dataset")
	}
	resp, err := c.inner.PostApiDatasetsWithResponse(ctx, *dataset)
	if err != nil {
		return nil, fmt.Errorf("dash0: create dataset failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusCreated {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	var created DatasetDefinition
	if err := json.Unmarshal(resp.Body, &created); err != nil {
		return nil, fmt.Errorf("dash0: failed to parse dataset response: %w", err)
	}
	return &created, nil
}

// UpdateDataset updates a dataset by origin or slug.
// For every token except a Dash0 UI session, the server creates the dataset when none matches.
// A mistyped origin or slug therefore creates a new dataset instead of returning not found.
// The apiVersion field is required and must be "dash0.com/v1alpha1".
// Omitting any spec field leaves its stored value unchanged.
func (c *client) UpdateDataset(ctx context.Context, originOrID string, dataset *DatasetDefinition) (*DatasetDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	if dataset == nil {
		return nil, fmt.Errorf("dash0: update dataset requires a non-nil dataset")
	}
	resp, err := c.inner.PutApiDatasetsOriginOrIdWithResponse(ctx, originOrID, *dataset)
	if err != nil {
		return nil, fmt.Errorf("dash0: update dataset failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusCreated {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	var updated DatasetDefinition
	if err := json.Unmarshal(resp.Body, &updated); err != nil {
		return nil, fmt.Errorf("dash0: failed to parse dataset response: %w", err)
	}
	return &updated, nil
}

// DeleteDataset deletes a dataset by origin or slug.
// The delete is soft: creating a dataset with the same slug within the grace period restores it.
// The server also answers 200 for an unknown or already-deleted dataset, so a nil error does not prove the dataset existed.
// The default dataset cannot be deleted and returns a 400 error.
func (c *client) DeleteDataset(ctx context.Context, originOrID string) error {
	if err := c.requireAPI(); err != nil {
		return err
	}
	resp, err := c.inner.DeleteApiDatasetsOriginOrIdWithResponse(ctx, originOrID)
	if err != nil {
		return fmt.Errorf("dash0: delete dataset failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// ListDatasetsIter returns an iterator over all datasets.
// This is a convenience wrapper around ListDatasets for consistent iteration patterns.
func (c *client) ListDatasetsIter(ctx context.Context) *Iter[DatasetDefinition] {
	items, err := c.ListDatasets(ctx)
	if err != nil {
		return newIterWithError[DatasetDefinition](err)
	}
	return newIter(items, false, nil, nil)
}

// StripDatasetServerFields removes server-generated fields from a dataset definition.
// It also clears the dash0.com/version label, so a stripped copy sent to [Client.UpdateDataset] skips the version check.
// It keeps dash0.com/origin, which is client-settable and identifies the dataset on update, so a copy created in the same organization needs a new origin.
func StripDatasetServerFields(dataset *DatasetDefinition) {
	if dataset == nil {
		return
	}
	dataset.Metadata.Annotations = nil
	if dataset.Metadata.Labels != nil {
		dataset.Metadata.Labels.Dash0Comversion = nil
		dataset.Metadata.Labels.Dash0Comsource = nil
	}
	dataset.Spec.PermittedActions = nil
}

// GetDatasetID extracts the slug from a dataset definition.
// It prefers spec.slug and falls back to the dash0.com/id label.
func GetDatasetID(dataset *DatasetDefinition) string {
	if dataset == nil {
		return ""
	}
	if dataset.Spec.Slug != "" {
		return dataset.Spec.Slug
	}
	if dataset.Metadata.Labels == nil {
		return ""
	}
	return dataset.Metadata.Labels.Dash0Comid
}

// GetDatasetName extracts the name from a dataset definition.
func GetDatasetName(dataset *DatasetDefinition) string {
	if dataset == nil {
		return ""
	}
	return dataset.Metadata.Name
}

// SetDatasetID sets the slug on a dataset definition, in both spec.slug and the dash0.com/id label, so the two always match.
// It initializes the labels struct if needed.
func SetDatasetID(dataset *DatasetDefinition, id string) {
	if dataset == nil {
		return
	}
	if dataset.Metadata.Labels == nil {
		dataset.Metadata.Labels = &DatasetLabels{}
	}
	dataset.Spec.Slug = id
	dataset.Metadata.Labels.Dash0Comid = id
}

// SetDatasetIDIfAbsent sets the slug on a dataset definition, like [SetDatasetID], only if neither spec.slug nor the dash0.com/id label is set.
func SetDatasetIDIfAbsent(dataset *DatasetDefinition, id string) {
	if GetDatasetID(dataset) == "" {
		SetDatasetID(dataset, id)
	}
}

// ClearDatasetID empties spec.slug and the dash0.com/id label on a dataset definition, which drops the slug from the request.
// A create then derives the slug from metadata.name, and an update of a missing dataset takes it from the path.
func ClearDatasetID(dataset *DatasetDefinition) {
	if dataset == nil {
		return
	}
	dataset.Spec.Slug = ""
	if dataset.Metadata.Labels != nil {
		dataset.Metadata.Labels.Dash0Comid = ""
	}
}
