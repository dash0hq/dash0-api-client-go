package dash0

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// UnmarshalJSON decodes OTLP numbers as json.Number: float64 would round integers above 2^53.
// encoding/json and sigs.k8s.io/yaml write them back as numbers; go.yaml.in/yaml/v3 writes them as quoted strings.
func (e *TelemetryTransformationRuleExample) UnmarshalJSON(data []byte) error {
	type plain TelemetryTransformationRuleExample
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	return decoder.Decode((*plain)(e))
}

// ListTelemetryTransformationRules retrieves rules without their stored examples.
// Use GetTelemetryTransformationRule before a read-modify-write update to preserve examples.
func (c *client) ListTelemetryTransformationRules(ctx context.Context, dataset *string) ([]*TelemetryTransformationRuleDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	params := &GetApiTelemetryTransformationRulesParams{Dataset: dataset}
	resp, err := c.inner.GetApiTelemetryTransformationRulesWithResponse(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("dash0: list telemetry transformation rules failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return toPointerSlice(resp.JSON200.TelemetryTransformationRules), nil
}

// GetTelemetryTransformationRule retrieves a complete rule, including stored examples.
func (c *client) GetTelemetryTransformationRule(ctx context.Context, originOrID string, dataset *string) (*TelemetryTransformationRuleDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	params := &GetApiTelemetryTransformationRulesOriginOrIdParams{Dataset: dataset}
	resp, err := c.inner.GetApiTelemetryTransformationRulesOriginOrIdWithResponse(ctx, originOrID, params)
	if err != nil {
		return nil, fmt.Errorf("dash0: get telemetry transformation rule failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return resp.JSON200, nil
}

// CreateTelemetryTransformationRule creates a rule without replaying an ambiguous POST outcome.
func (c *client) CreateTelemetryTransformationRule(ctx context.Context, rule *TelemetryTransformationRuleDefinition, dataset *string) (*TelemetryTransformationRuleDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	if rule == nil {
		return nil, fmt.Errorf("dash0: create telemetry transformation rule requires a non-nil rule")
	}
	params := &PostApiTelemetryTransformationRulesParams{Dataset: dataset}
	resp, err := c.inner.PostApiTelemetryTransformationRulesWithResponse(ctx, params, withEmptyTelemetryTransformationRuleLists(rule))
	if err != nil {
		return nil, fmt.Errorf("dash0: create telemetry transformation rule failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return resp.JSON200, nil
}

// UpdateTelemetryTransformationRule replaces an existing rule without fetching or merging it.
// Omitting examples removes stored examples; a supplied updated-at concurrency token is sent unchanged.
func (c *client) UpdateTelemetryTransformationRule(ctx context.Context, originOrID string, rule *TelemetryTransformationRuleDefinition, dataset *string) (*TelemetryTransformationRuleDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	if rule == nil {
		return nil, fmt.Errorf("dash0: update telemetry transformation rule requires a non-nil rule")
	}
	params := &PutApiTelemetryTransformationRulesOriginOrIdParams{Dataset: dataset}
	resp, err := c.inner.PutApiTelemetryTransformationRulesOriginOrIdWithResponse(ctx, originOrID, params, withEmptyTelemetryTransformationRuleLists(rule))
	if err != nil {
		return nil, fmt.Errorf("dash0: update telemetry transformation rule failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return resp.JSON200, nil
}

// DeleteTelemetryTransformationRule deletes a rule by origin or ID.
// Pass the rule's dash0.com/updated-at as updatedAt to delete only if the rule has not changed since; a stale value fails with superseded_version.
// A nil updatedAt deletes unconditionally.
func (c *client) DeleteTelemetryTransformationRule(ctx context.Context, originOrID string, updatedAt *time.Time, dataset *string) error {
	if err := c.requireAPI(); err != nil {
		return err
	}
	params := &DeleteApiTelemetryTransformationRulesOriginOrIdParams{Dataset: dataset, UpdatedAt: updatedAt}
	resp, err := c.inner.DeleteApiTelemetryTransformationRulesOriginOrIdWithResponse(ctx, originOrID, params)
	if err != nil {
		return fmt.Errorf("dash0: delete telemetry transformation rule failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// withEmptyTelemetryTransformationRuleLists returns a copy of rule whose nil transformation and statement lists are empty.
// The spec types both as non-nullable arrays, so they must be sent as [] rather than null.
func withEmptyTelemetryTransformationRuleLists(rule *TelemetryTransformationRuleDefinition) TelemetryTransformationRuleDefinition {
	body := *rule
	body.Spec.Transformations = make([]TelemetryTransformationRuleTransformation, len(rule.Spec.Transformations))
	for i, transformation := range rule.Spec.Transformations {
		if transformation.Statements == nil {
			transformation.Statements = OTTLStatements{}
		}
		body.Spec.Transformations[i] = transformation
	}
	return body
}

// ListTelemetryTransformationRulesIter iterates over rules without stored examples.
func (c *client) ListTelemetryTransformationRulesIter(ctx context.Context, dataset *string) *Iter[TelemetryTransformationRuleDefinition] {
	items, err := c.ListTelemetryTransformationRules(ctx, dataset)
	if err != nil {
		return newIterWithError[TelemetryTransformationRuleDefinition](err)
	}
	return newIter(items, false, nil, nil)
}

// StripTelemetryTransformationRuleServerFields prepares a fetched rule for an update.
// It keeps identity, origin, dataset, and the update concurrency token.
// To create a copy, also call ClearTelemetryTransformationRuleID and remove or change dash0.com/origin: POST rejects an origin that already exists with 409.
func StripTelemetryTransformationRuleServerFields(rule *TelemetryTransformationRuleDefinition) {
	if rule == nil {
		return
	}
	if rule.Metadata.Labels != nil {
		rule.Metadata.Labels.Dash0Comsource = nil
	}
	if rule.Metadata.Annotations != nil {
		rule.Metadata.Annotations.Dash0ComcreatedAt = nil
		rule.Metadata.Annotations.Dash0ComdeletedAt = nil
	}
}

// ClearTelemetryTransformationRuleID removes the ID without changing the origin or dataset.
func ClearTelemetryTransformationRuleID(rule *TelemetryTransformationRuleDefinition) {
	if rule == nil || rule.Metadata.Labels == nil {
		return
	}
	rule.Metadata.Labels.Dash0Comid = nil
}

// GetTelemetryTransformationRuleID returns an empty string when the ID is absent.
func GetTelemetryTransformationRuleID(rule *TelemetryTransformationRuleDefinition) string {
	if rule == nil || rule.Metadata.Labels == nil || rule.Metadata.Labels.Dash0Comid == nil {
		return ""
	}
	return *rule.Metadata.Labels.Dash0Comid
}

// GetTelemetryTransformationRuleOrigin returns an empty string when the origin is absent.
func GetTelemetryTransformationRuleOrigin(rule *TelemetryTransformationRuleDefinition) string {
	if rule == nil || rule.Metadata.Labels == nil || rule.Metadata.Labels.Dash0Comorigin == nil {
		return ""
	}
	return *rule.Metadata.Labels.Dash0Comorigin
}

// GetTelemetryTransformationRuleName falls back to metadata.name when the display name is absent or empty.
func GetTelemetryTransformationRuleName(rule *TelemetryTransformationRuleDefinition) string {
	if rule == nil {
		return ""
	}
	if rule.Spec.Display != nil && rule.Spec.Display.Name != "" {
		return rule.Spec.Display.Name
	}
	return rule.Metadata.Name
}

// SetTelemetryTransformationRuleID initializes missing labels and overwrites the ID.
func SetTelemetryTransformationRuleID(rule *TelemetryTransformationRuleDefinition, id string) {
	if rule == nil {
		return
	}
	if rule.Metadata.Labels == nil {
		rule.Metadata.Labels = &TelemetryTransformationRuleLabels{}
	}
	rule.Metadata.Labels.Dash0Comid = &id
}

// SetTelemetryTransformationRuleIDIfAbsent preserves an existing ID.
func SetTelemetryTransformationRuleIDIfAbsent(rule *TelemetryTransformationRuleDefinition, id string) {
	if rule == nil {
		return
	}
	if rule.Metadata.Labels == nil {
		rule.Metadata.Labels = &TelemetryTransformationRuleLabels{}
	}
	if rule.Metadata.Labels.Dash0Comid == nil {
		rule.Metadata.Labels.Dash0Comid = &id
	}
}

// GetTelemetryTransformationRuleDataset returns an empty string when no dataset label is present.
func GetTelemetryTransformationRuleDataset(rule *TelemetryTransformationRuleDefinition) string {
	if rule == nil || rule.Metadata.Labels == nil || rule.Metadata.Labels.Dash0Comdataset == nil {
		return ""
	}
	return *rule.Metadata.Labels.Dash0Comdataset
}

// SetTelemetryTransformationRuleDataset initializes missing labels and overwrites the dataset.
func SetTelemetryTransformationRuleDataset(rule *TelemetryTransformationRuleDefinition, dataset string) {
	if rule == nil {
		return
	}
	if rule.Metadata.Labels == nil {
		rule.Metadata.Labels = &TelemetryTransformationRuleLabels{}
	}
	rule.Metadata.Labels.Dash0Comdataset = &dataset
}

var _ AssetMetadata = (*TelemetryTransformationRuleDefinition)(nil)

func (rule *TelemetryTransformationRuleDefinition) GetOrigin() string {
	return GetTelemetryTransformationRuleOrigin(rule)
}
func (rule *TelemetryTransformationRuleDefinition) GetID() string {
	return GetTelemetryTransformationRuleID(rule)
}
func (rule *TelemetryTransformationRuleDefinition) GetName() string {
	return GetTelemetryTransformationRuleName(rule)
}
func (*TelemetryTransformationRuleDefinition) UpsertKeys() (byOrigin, byID bool) { return true, true }
