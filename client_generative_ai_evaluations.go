package dash0

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ListGenerativeAiEvaluations retrieves the generative AI evaluations of a dataset.
// The result leaves out deleted evaluations.
// Pass [WithOriginPrefix] to restrict the result to evaluations whose origin starts with a given prefix.
// Any other [ListOption] filter returns an error, because the endpoint does not support it.
func (c *client) ListGenerativeAiEvaluations(ctx context.Context, dataset *string, opts ...ListOption) ([]*GenerativeAiEvaluationDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	options := NewListOptions(opts...)
	if err := options.requireOnly("list generative AI evaluations", filterOriginPrefix); err != nil {
		return nil, err
	}
	params := &GetApiGenerativeAiEvaluationsParams{
		Dataset:      dataset,
		OriginPrefix: options.OriginPrefix,
	}
	resp, err := c.inner.GetApiGenerativeAiEvaluationsWithResponse(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("dash0: list generative AI evaluations failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return toPointerSlice(resp.JSON200.GenerativeAiEvaluations), nil
}

// GetGenerativeAiEvaluation retrieves a generative AI evaluation by origin or ID.
// A deleted evaluation is still returned, with the dash0.com/deleted-at annotation set.
func (c *client) GetGenerativeAiEvaluation(ctx context.Context, originOrID string, dataset *string) (*GenerativeAiEvaluationDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	params := &GetApiGenerativeAiEvaluationsOriginOrIdParams{
		Dataset: dataset,
	}
	resp, err := c.inner.GetApiGenerativeAiEvaluationsOriginOrIdWithResponse(ctx, originOrID, params)
	if err != nil {
		return nil, fmt.Errorf("dash0: get generative AI evaluation failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return resp.JSON200, nil
}

// CreateGenerativeAiEvaluation creates a generative AI evaluation.
// The server assigns the dash0.com/id label and rejects a request that sets it.
// When an API token without a user sends no dash0.com/origin, the server sets one.
func (c *client) CreateGenerativeAiEvaluation(ctx context.Context, evaluation *GenerativeAiEvaluationDefinition, dataset *string) (*GenerativeAiEvaluationDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	if evaluation == nil {
		return nil, fmt.Errorf("dash0: create generative AI evaluation failed: evaluation is nil")
	}
	params := &PostApiGenerativeAiEvaluationsParams{
		Dataset: dataset,
	}
	resp, err := c.inner.PostApiGenerativeAiEvaluationsWithResponse(ctx, params, *evaluation)
	if err != nil {
		return nil, fmt.Errorf("dash0: create generative AI evaluation failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusCreated {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	var created GenerativeAiEvaluationDefinition
	if err := json.Unmarshal(resp.Body, &created); err != nil {
		return nil, fmt.Errorf("dash0: failed to parse generative AI evaluation response: %w", err)
	}
	return &created, nil
}

// UpdateGenerativeAiEvaluation creates or replaces a generative AI evaluation by origin or ID.
// An unknown origin creates the evaluation with that origin, and the origin of a deleted evaluation restores it.
// When the dash0.com/version label is set, the server rejects the write if the evaluation changed since that version.
// Without it, the write overwrites unconditionally.
// The server rejects a change to metadata.name or to the judge's output kind.
func (c *client) UpdateGenerativeAiEvaluation(ctx context.Context, originOrID string, evaluation *GenerativeAiEvaluationDefinition, dataset *string) (*GenerativeAiEvaluationDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	if evaluation == nil {
		return nil, fmt.Errorf("dash0: update generative AI evaluation failed: evaluation is nil")
	}
	params := &PutApiGenerativeAiEvaluationsOriginOrIdParams{
		Dataset: dataset,
	}
	resp, err := c.inner.PutApiGenerativeAiEvaluationsOriginOrIdWithResponse(ctx, originOrID, params, *evaluation)
	if err != nil {
		return nil, fmt.Errorf("dash0: update generative AI evaluation failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return resp.JSON200, nil
}

// DeleteGenerativeAiEvaluation deletes a generative AI evaluation by origin or ID.
// Deleting an evaluation that is already deleted returns a not-found error.
func (c *client) DeleteGenerativeAiEvaluation(ctx context.Context, originOrID string, dataset *string) error {
	if err := c.requireAPI(); err != nil {
		return err
	}
	params := &DeleteApiGenerativeAiEvaluationsOriginOrIdParams{
		Dataset: dataset,
	}
	resp, err := c.inner.DeleteApiGenerativeAiEvaluationsOriginOrIdWithResponse(ctx, originOrID, params)
	if err != nil {
		return fmt.Errorf("dash0: delete generative AI evaluation failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// ListGenerativeAiEvaluationsIter returns an iterator over the generative AI evaluations of a dataset.
// This is a convenience wrapper around ListGenerativeAiEvaluations for consistent iteration patterns.
func (c *client) ListGenerativeAiEvaluationsIter(ctx context.Context, dataset *string, opts ...ListOption) *Iter[GenerativeAiEvaluationDefinition] {
	items, err := c.ListGenerativeAiEvaluations(ctx, dataset, opts...)
	if err != nil {
		return newIterWithError[GenerativeAiEvaluationDefinition](err)
	}
	return newIter(items, false, nil, nil)
}

// StripGenerativeAiEvaluationServerFields removes the fields the server sets from a generative AI evaluation, so the result can be sent back on a write.
// It removes the dash0.com/id, dash0.com/version, dash0.com/source, and dash0.com/dataset labels, and the created-at, created-by, updated-at, and deleted-at annotations.
// The server assigns the id and rejects a create that sets it, so callers that need it read it with [GetGenerativeAiEvaluationID] first.
// Without the version, an update overwrites unconditionally.
//
// It keeps the dash0.com/origin label, which a caller sets on create and which is the upsert key for an update.
// It also keeps the dash0.com/folder-path and dash0.com/sharing annotations and the whole spec.
func StripGenerativeAiEvaluationServerFields(evaluation *GenerativeAiEvaluationDefinition) {
	if evaluation == nil {
		return
	}
	if evaluation.Metadata.Annotations != nil {
		evaluation.Metadata.Annotations.Dash0ComcreatedAt = nil
		evaluation.Metadata.Annotations.Dash0ComcreatedBy = nil
		evaluation.Metadata.Annotations.Dash0ComupdatedAt = nil
		evaluation.Metadata.Annotations.Dash0ComdeletedAt = nil
	}
	if evaluation.Metadata.Labels != nil {
		evaluation.Metadata.Labels.Dash0Comid = nil
		evaluation.Metadata.Labels.Dash0Comversion = nil
		evaluation.Metadata.Labels.Dash0Comsource = nil
		evaluation.Metadata.Labels.Dash0Comdataset = nil
	}
}

// ClearGenerativeAiEvaluationID removes the dash0.com/id label from a generative AI evaluation.
func ClearGenerativeAiEvaluationID(evaluation *GenerativeAiEvaluationDefinition) {
	if evaluation == nil {
		return
	}
	if evaluation.Metadata.Labels != nil {
		evaluation.Metadata.Labels.Dash0Comid = nil
	}
}

// GetGenerativeAiEvaluationID extracts the dash0.com/id label from a generative AI evaluation.
func GetGenerativeAiEvaluationID(evaluation *GenerativeAiEvaluationDefinition) string {
	if evaluation == nil || evaluation.Metadata.Labels == nil || evaluation.Metadata.Labels.Dash0Comid == nil {
		return ""
	}
	return *evaluation.Metadata.Labels.Dash0Comid
}

// SetGenerativeAiEvaluationID sets the dash0.com/id label on a generative AI evaluation, initializing the labels struct if needed.
func SetGenerativeAiEvaluationID(evaluation *GenerativeAiEvaluationDefinition, id string) {
	if evaluation == nil {
		return
	}
	if evaluation.Metadata.Labels == nil {
		evaluation.Metadata.Labels = &GenerativeAiEvaluationLabels{}
	}
	evaluation.Metadata.Labels.Dash0Comid = &id
}

// SetGenerativeAiEvaluationIDIfAbsent sets the dash0.com/id label on a generative AI evaluation only if it is not already set.
func SetGenerativeAiEvaluationIDIfAbsent(evaluation *GenerativeAiEvaluationDefinition, id string) {
	if evaluation == nil {
		return
	}
	if evaluation.Metadata.Labels == nil {
		evaluation.Metadata.Labels = &GenerativeAiEvaluationLabels{}
	}
	if evaluation.Metadata.Labels.Dash0Comid == nil {
		evaluation.Metadata.Labels.Dash0Comid = &id
	}
}

// GetGenerativeAiEvaluationName returns the display name of a generative AI evaluation from spec.display.name.
// It falls back to metadata.name when the display name is empty.
func GetGenerativeAiEvaluationName(evaluation *GenerativeAiEvaluationDefinition) string {
	if evaluation == nil {
		return ""
	}
	if evaluation.Spec.Display.Name != "" {
		return evaluation.Spec.Display.Name
	}
	return evaluation.Metadata.Name
}

// GetGenerativeAiEvaluationOrigin extracts the dash0.com/origin label from a generative AI evaluation.
// The origin is the upsert key for [Client.UpdateGenerativeAiEvaluation].
func GetGenerativeAiEvaluationOrigin(evaluation *GenerativeAiEvaluationDefinition) string {
	if evaluation == nil || evaluation.Metadata.Labels == nil || evaluation.Metadata.Labels.Dash0Comorigin == nil {
		return ""
	}
	return *evaluation.Metadata.Labels.Dash0Comorigin
}

// GetGenerativeAiEvaluationDataset extracts the dash0.com/dataset label from a generative AI evaluation.
func GetGenerativeAiEvaluationDataset(evaluation *GenerativeAiEvaluationDefinition) string {
	if evaluation == nil || evaluation.Metadata.Labels == nil || evaluation.Metadata.Labels.Dash0Comdataset == nil {
		return ""
	}
	return *evaluation.Metadata.Labels.Dash0Comdataset
}

// SetGenerativeAiEvaluationDataset sets the dash0.com/dataset label on a generative AI evaluation, initializing the labels struct if needed.
func SetGenerativeAiEvaluationDataset(evaluation *GenerativeAiEvaluationDefinition, dataset string) {
	if evaluation == nil {
		return
	}
	if evaluation.Metadata.Labels == nil {
		evaluation.Metadata.Labels = &GenerativeAiEvaluationLabels{}
	}
	evaluation.Metadata.Labels.Dash0Comdataset = &dataset
}

var _ AssetMetadata = (*GenerativeAiEvaluationDefinition)(nil)

func (evaluation *GenerativeAiEvaluationDefinition) GetOrigin() string {
	return GetGenerativeAiEvaluationOrigin(evaluation)
}
func (evaluation *GenerativeAiEvaluationDefinition) GetID() string {
	return GetGenerativeAiEvaluationID(evaluation)
}
func (evaluation *GenerativeAiEvaluationDefinition) GetName() string {
	return GetGenerativeAiEvaluationName(evaluation)
}
func (*GenerativeAiEvaluationDefinition) UpsertKeys() (byOrigin, byID bool) { return true, true }
