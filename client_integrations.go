package dash0

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const integrationErrMsg = "dash0: %s integration failed: %w"

// GetIntegration retrieves an integration by origin or ID.
func (c *client) GetIntegration(ctx context.Context, originOrID string, dataset *string) (*IntegrationDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	reqURL := c.integrationURL(originOrID, dataset)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf(integrationErrMsg, "get", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(integrationErrMsg, "get", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf(integrationErrMsg, "get", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, newAPIErrorWithBody(resp, body)
	}

	var def IntegrationDefinition
	if err := json.Unmarshal(body, &def); err != nil {
		return nil, fmt.Errorf(integrationErrMsg, "get", err)
	}
	return &def, nil
}

// CreateIntegration creates a new integration (upsert via PUT).
func (c *client) CreateIntegration(ctx context.Context, integration *IntegrationDefinition, dataset *string) (*IntegrationDefinition, error) {
	return c.upsertIntegration(ctx, GetIntegrationOrigin(integration), integration, dataset, "create")
}

// UpdateIntegration updates an existing integration (upsert via PUT).
func (c *client) UpdateIntegration(ctx context.Context, originOrID string, integration *IntegrationDefinition, dataset *string) (*IntegrationDefinition, error) {
	return c.upsertIntegration(ctx, originOrID, integration, dataset, "update")
}

func (c *client) upsertIntegration(ctx context.Context, originOrID string, integration *IntegrationDefinition, dataset *string, operation string) (*IntegrationDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	reqURL := c.integrationURL(originOrID, dataset)
	bodyBytes, err := json.Marshal(integration)
	if err != nil {
		return nil, fmt.Errorf(integrationErrMsg, operation, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, reqURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf(integrationErrMsg, operation, err)
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(integrationErrMsg, operation, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf(integrationErrMsg, operation, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, newAPIErrorWithBody(resp, body)
	}

	var def IntegrationDefinition
	if err := json.Unmarshal(body, &def); err != nil {
		return nil, fmt.Errorf(integrationErrMsg, operation, err)
	}
	return &def, nil
}

// DeleteIntegration deletes an integration by origin or ID.
func (c *client) DeleteIntegration(ctx context.Context, originOrID string, dataset *string) error {
	if err := c.requireAPI(); err != nil {
		return err
	}
	reqURL := c.integrationURL(originOrID, dataset)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, reqURL, nil)
	if err != nil {
		return fmt.Errorf(integrationErrMsg, "delete", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf(integrationErrMsg, "delete", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf(integrationErrMsg, "delete", err)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return newAPIErrorWithBody(resp, body)
	}
	return nil
}

// integrationURL builds the full URL for an integration API request.
func (c *client) integrationURL(originOrID string, dataset *string) string {
	u := c.config.apiUrl + "/api/integrations/" + url.PathEscape(originOrID)
	if dataset != nil && *dataset != "" {
		u += "?dataset=" + url.QueryEscape(*dataset)
	}
	return u
}

// setHeaders sets common headers on an outgoing request.
func (c *client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.config.authToken)
	req.Header.Set("User-Agent", c.config.userAgent)
}

// StripIntegrationServerFields removes server-generated metadata fields from an integration definition.
func StripIntegrationServerFields(def *IntegrationDefinition) {
	if def == nil {
		return
	}
	def.Metadata.CreatedAt = nil
	def.Metadata.UpdatedAt = nil
	def.Metadata.Version = nil
	if def.Metadata.Annotations != nil {
		def.Metadata.Annotations.CreatedAt = nil
		def.Metadata.Annotations.LastUpdatedAt = nil
	}
}

// GetIntegrationID extracts the ID from an integration definition's labels.
func GetIntegrationID(def *IntegrationDefinition) string {
	if def == nil || def.Metadata.Labels == nil {
		return ""
	}
	return def.Metadata.Labels[LabelID]
}

// GetIntegrationOrigin extracts the origin from an integration definition's labels.
func GetIntegrationOrigin(def *IntegrationDefinition) string {
	if def == nil || def.Metadata.Labels == nil {
		return ""
	}
	return def.Metadata.Labels[LabelOrigin]
}

// GetIntegrationName extracts the display name from an integration definition.
func GetIntegrationName(def *IntegrationDefinition) string {
	if def == nil {
		return ""
	}
	return def.Spec.Display.Name
}

// SetIntegrationID sets the ID label on an integration definition, initializing
// the labels map if needed.
func SetIntegrationID(def *IntegrationDefinition, id string) {
	if def == nil {
		return
	}
	if def.Metadata.Labels == nil {
		def.Metadata.Labels = map[string]string{}
	}
	def.Metadata.Labels[LabelID] = id
}

// SetIntegrationIDIfAbsent sets the ID label on an integration definition only
// if it is not already set, initializing the labels map if needed.
func SetIntegrationIDIfAbsent(def *IntegrationDefinition, id string) {
	if def == nil {
		return
	}
	if def.Metadata.Labels == nil {
		def.Metadata.Labels = map[string]string{}
	}
	if _, ok := def.Metadata.Labels[LabelID]; !ok {
		def.Metadata.Labels[LabelID] = id
	}
}

// ClearIntegrationID removes the ID label from an integration definition.
func ClearIntegrationID(def *IntegrationDefinition) {
	if def == nil || def.Metadata.Labels == nil {
		return
	}
	delete(def.Metadata.Labels, LabelID)
}
