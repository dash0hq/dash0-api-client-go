package dash0

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ListAgenticWorkflows retrieves all Agent0 automations.
func (c *client) ListAgenticWorkflows(ctx context.Context, dataset *string) ([]*AgenticWorkflowDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	params := &GetApiAgenticWorkflowsParams{
		Dataset: dataset,
	}
	resp, err := c.inner.GetApiAgenticWorkflowsWithResponse(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("dash0: list automations failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return toPointerSlice(resp.JSON200.AgenticWorkflows), nil
}

// GetAgenticWorkflow retrieves an Agent0 automation by origin or ID.
func (c *client) GetAgenticWorkflow(ctx context.Context, originOrID string, dataset *string) (*AgenticWorkflowDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	params := &GetApiAgenticWorkflowsOriginOrIdParams{
		Dataset: dataset,
	}
	resp, err := c.inner.GetApiAgenticWorkflowsOriginOrIdWithResponse(ctx, originOrID, params)
	if err != nil {
		return nil, fmt.Errorf("dash0: get automation failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200, nil
}

// CreateAgenticWorkflow creates a new Agent0 automation.
func (c *client) CreateAgenticWorkflow(ctx context.Context, workflow *AgenticWorkflowDefinition, dataset *string) (*AgenticWorkflowDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	if workflow == nil {
		return nil, fmt.Errorf("dash0: create automation requires a non-nil automation")
	}
	params := &PostApiAgenticWorkflowsParams{
		Dataset: dataset,
	}
	resp, err := c.inner.PostApiAgenticWorkflowsWithResponse(ctx, params, *workflow)
	if err != nil {
		return nil, fmt.Errorf("dash0: create automation failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusCreated {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	var created AgenticWorkflowDefinition
	if err := json.Unmarshal(resp.Body, &created); err != nil {
		return nil, fmt.Errorf("dash0: failed to parse automation response: %w", err)
	}
	return &created, nil
}

// UpdateAgenticWorkflow updates an existing Agent0 automation (create-or-replace).
func (c *client) UpdateAgenticWorkflow(ctx context.Context, originOrID string, workflow *AgenticWorkflowDefinition, dataset *string) (*AgenticWorkflowDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	if workflow == nil {
		return nil, fmt.Errorf("dash0: update automation requires a non-nil automation")
	}
	params := &PutApiAgenticWorkflowsOriginOrIdParams{
		Dataset: dataset,
	}
	resp, err := c.inner.PutApiAgenticWorkflowsOriginOrIdWithResponse(ctx, originOrID, params, *workflow)
	if err != nil {
		return nil, fmt.Errorf("dash0: update automation failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	var updated AgenticWorkflowDefinition
	if err := json.Unmarshal(resp.Body, &updated); err != nil {
		return nil, fmt.Errorf("dash0: failed to parse automation response: %w", err)
	}
	return &updated, nil
}

// DeleteAgenticWorkflow deletes an Agent0 automation by origin or ID.
func (c *client) DeleteAgenticWorkflow(ctx context.Context, originOrID string, dataset *string) error {
	if err := c.requireAPI(); err != nil {
		return err
	}
	params := &DeleteApiAgenticWorkflowsOriginOrIdParams{
		Dataset: dataset,
	}
	resp, err := c.inner.DeleteApiAgenticWorkflowsOriginOrIdWithResponse(ctx, originOrID, params)
	if err != nil {
		return fmt.Errorf("dash0: delete automation failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// ListAgenticWorkflowsIter returns an iterator over all Agent0 automations.
// This is a convenience wrapper around ListAgenticWorkflows for consistent
// iteration patterns.
func (c *client) ListAgenticWorkflowsIter(ctx context.Context, dataset *string) *Iter[AgenticWorkflowDefinition] {
	items, err := c.ListAgenticWorkflows(ctx, dataset)
	if err != nil {
		return newIterWithError[AgenticWorkflowDefinition](err)
	}
	return newIter(items, false, nil, nil)
}

// StripAgenticWorkflowServerFields removes server-generated fields from an
// automation definition so callers can round-trip an automation through a
// write endpoint without accidentally shipping stale server-set metadata.
//
// This includes dash0.com/id: like SLO ids, an automation's id is assigned
// by the server (`agentic_workflow_<ulid>`) and cannot be chosen by a
// caller, so it is server-managed metadata stripped here alongside
// version/source. dash0.com/origin is also stripped: it travels as the
// `originOrId` path parameter on write, not in the request body, matching
// StripSLOServerFields.
//
// The plaintext webhook secret
// (dash0.com/webhook-secret) and the webhook id (dash0.com/webhook-id) are
// also stripped: both are returned by the server and are never accepted on
// write.
//
// User-defined (non-dash0.com/*) labels and annotations are preserved.
//
// Callers that need the id must read it (via [GetAgenticWorkflowID]) before
// calling this.
func StripAgenticWorkflowServerFields(workflow *AgenticWorkflowDefinition) {
	if workflow == nil {
		return
	}
	if workflow.Metadata.Labels != nil {
		workflow.Metadata.Labels.Dash0Comid = nil
		workflow.Metadata.Labels.Dash0Comversion = nil
		workflow.Metadata.Labels.Dash0Comorigin = nil
		workflow.Metadata.Labels.Dash0Comdataset = nil
		workflow.Metadata.Labels.Dash0Comsource = nil
		workflow.Metadata.Labels.Dash0ComwebhookId = nil
	}
	if workflow.Metadata.Annotations != nil {
		workflow.Metadata.Annotations.Dash0ComcreatedAt = nil
		workflow.Metadata.Annotations.Dash0ComcreatedBy = nil
		workflow.Metadata.Annotations.Dash0ComdeletedAt = nil
		workflow.Metadata.Annotations.Dash0ComlastUpdatedAt = nil
		workflow.Metadata.Annotations.Dash0ComlastUpdatedBy = nil
		workflow.Metadata.Annotations.Dash0ComwebhookSecret = nil
	}
	workflow.Spec.PermittedActions = nil
}

// GetAgenticWorkflowID extracts the dash0.com/id label from an automation
// definition. Returns the empty string when the automation is nil, has no
// labels, or the label is unset.
func GetAgenticWorkflowID(workflow *AgenticWorkflowDefinition) string {
	if workflow == nil || workflow.Metadata.Labels == nil || workflow.Metadata.Labels.Dash0Comid == nil {
		return ""
	}
	return *workflow.Metadata.Labels.Dash0Comid
}

// GetAgenticWorkflowOrigin extracts the dash0.com/origin label from an
// automation definition. Returns the empty string when the automation is
// nil, has no labels, or the label is unset.
//
// An automation's id is server-assigned, so origin is the only identifier a
// hand-authored document can pin, and it doubles as the upsert key.
func GetAgenticWorkflowOrigin(workflow *AgenticWorkflowDefinition) string {
	if workflow == nil || workflow.Metadata.Labels == nil || workflow.Metadata.Labels.Dash0Comorigin == nil {
		return ""
	}
	return *workflow.Metadata.Labels.Dash0Comorigin
}

// GetAgenticWorkflowName extracts the human-facing name from an automation
// definition, reading spec.display.name. Returns the empty string when the
// automation is nil.
func GetAgenticWorkflowName(workflow *AgenticWorkflowDefinition) string {
	if workflow == nil {
		return ""
	}
	return workflow.Spec.Display.Name
}

// GetAgenticWorkflowDataset extracts the dash0.com/dataset label from an
// automation definition. Returns the empty string when the automation is
// nil, has no labels, or the label is unset.
func GetAgenticWorkflowDataset(workflow *AgenticWorkflowDefinition) string {
	if workflow == nil || workflow.Metadata.Labels == nil || workflow.Metadata.Labels.Dash0Comdataset == nil {
		return ""
	}
	return *workflow.Metadata.Labels.Dash0Comdataset
}

// SetAgenticWorkflowDataset sets the dash0.com/dataset label on an
// automation definition, initializing the labels struct if needed. No-op
// when the automation is nil.
func SetAgenticWorkflowDataset(workflow *AgenticWorkflowDefinition, dataset string) {
	if workflow == nil {
		return
	}
	if workflow.Metadata.Labels == nil {
		workflow.Metadata.Labels = &AgenticWorkflowLabels{}
	}
	workflow.Metadata.Labels.Dash0Comdataset = &dataset
}

// SetAgenticWorkflowID sets the dash0.com/id label on an automation
// definition, initializing the labels struct if needed. No-op when the
// automation is nil.
func SetAgenticWorkflowID(workflow *AgenticWorkflowDefinition, id string) {
	if workflow == nil {
		return
	}
	if workflow.Metadata.Labels == nil {
		workflow.Metadata.Labels = &AgenticWorkflowLabels{}
	}
	workflow.Metadata.Labels.Dash0Comid = &id
}

// SetAgenticWorkflowIDIfAbsent sets the dash0.com/id label on an automation
// definition only if it is not already set, initializing the labels struct
// if needed. No-op when the automation is nil.
func SetAgenticWorkflowIDIfAbsent(workflow *AgenticWorkflowDefinition, id string) {
	if workflow == nil {
		return
	}
	if workflow.Metadata.Labels == nil {
		workflow.Metadata.Labels = &AgenticWorkflowLabels{}
	}
	if workflow.Metadata.Labels.Dash0Comid == nil {
		workflow.Metadata.Labels.Dash0Comid = &id
	}
}

// ClearAgenticWorkflowID removes the dash0.com/id label from an automation
// definition. No-op when the automation is nil or has no labels.
func ClearAgenticWorkflowID(workflow *AgenticWorkflowDefinition) {
	if workflow == nil || workflow.Metadata.Labels == nil {
		return
	}
	workflow.Metadata.Labels.Dash0Comid = nil
}
