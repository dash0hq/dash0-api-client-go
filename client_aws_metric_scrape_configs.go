package dash0

import (
	"context"
	"fmt"
	"net/http"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

// GetAwsMetricScrapeConfig retrieves the CloudWatch metric scrape configuration of an AWS integration.
// The API returns a document for every integration; one with nothing configured has an empty spec.namespaces list.
func (c *client) GetAwsMetricScrapeConfig(ctx context.Context, integrationID string) (*AwsMetricScrapeConfigDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	id, err := parseIntegrationID(integrationID)
	if err != nil {
		return nil, err
	}
	resp, err := c.inner.GetApiAwsMetricScrapeConfigsIntegrationIdWithResponse(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("dash0: get AWS metric scrape config failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return resp.JSON200, nil
}

// UpdateAwsMetricScrapeConfig replaces the CloudWatch metric scrape configuration of an AWS integration.
// It returns the stored configuration.
func (c *client) UpdateAwsMetricScrapeConfig(ctx context.Context, integrationID string, cfg *AwsMetricScrapeConfigDefinition) (*AwsMetricScrapeConfigDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, fmt.Errorf("dash0: AWS metric scrape config is required")
	}
	id, err := parseIntegrationID(integrationID)
	if err != nil {
		return nil, err
	}
	resp, err := c.inner.PutApiAwsMetricScrapeConfigsIntegrationIdWithResponse(ctx, id, withEmptyAwsMetricScrapeConfigLists(cfg))
	if err != nil {
		return nil, fmt.Errorf("dash0: update AWS metric scrape config failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return resp.JSON200, nil
}

// DeleteAwsMetricScrapeConfig clears the CloudWatch metric scrape configuration of an AWS integration.
// The integration itself is not deleted.
func (c *client) DeleteAwsMetricScrapeConfig(ctx context.Context, integrationID string) error {
	if err := c.requireAPI(); err != nil {
		return err
	}
	id, err := parseIntegrationID(integrationID)
	if err != nil {
		return err
	}
	resp, err := c.inner.DeleteApiAwsMetricScrapeConfigsIntegrationIdWithResponse(ctx, id)
	if err != nil {
		return fmt.Errorf("dash0: delete AWS metric scrape config failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// withEmptyAwsMetricScrapeConfigLists returns a copy of cfg whose nil namespace and metric lists are empty.
// The spec types both as non-nullable arrays, so they must be sent as [] rather than null.
func withEmptyAwsMetricScrapeConfigLists(cfg *AwsMetricScrapeConfigDefinition) AwsMetricScrapeConfigDefinition {
	body := *cfg
	body.Spec.Namespaces = make([]AwsMetricScrapeConfigNamespace, len(cfg.Spec.Namespaces))
	for i, ns := range cfg.Spec.Namespaces {
		if ns.Metrics == nil {
			ns.Metrics = []string{}
		}
		body.Spec.Namespaces[i] = ns
	}
	return body
}

func parseIntegrationID(integrationID string) (openapi_types.UUID, error) {
	var id openapi_types.UUID
	if err := id.UnmarshalText([]byte(integrationID)); err != nil {
		return id, fmt.Errorf("dash0: integration ID %q is not a valid UUID: %w", integrationID, err)
	}
	return id, nil
}

// StripAwsMetricScrapeConfigServerFields removes server-managed fields from an AWS metric scrape config.
// It removes metadata.name, the dash0.com/id, dash0.com/dataset, and dash0.com/source labels, and the created-at and updated-at annotations.
// The dash0.com/origin label is kept, because clients may set it.
func StripAwsMetricScrapeConfigServerFields(cfg *AwsMetricScrapeConfigDefinition) {
	if cfg == nil {
		return
	}
	cfg.Metadata.Name = nil
	cfg.Metadata.Annotations = nil
	if labels := cfg.Metadata.Labels; labels != nil {
		labels.Dash0Comid = nil
		labels.Dash0Comdataset = nil
		labels.Dash0Comsource = nil
		if labels.Dash0Comorigin == nil {
			cfg.Metadata.Labels = nil
		}
	}
}

// GetAwsMetricScrapeConfigID returns the dash0.com/id label of an AWS metric scrape config, or "" if absent.
// The ID is the ID of the AWS integration the config belongs to.
// There are no Set or Clear ID helpers, because the integration ID is taken from the request path and cannot be chosen by clients.
func GetAwsMetricScrapeConfigID(cfg *AwsMetricScrapeConfigDefinition) string {
	if cfg == nil || cfg.Metadata.Labels == nil || cfg.Metadata.Labels.Dash0Comid == nil {
		return ""
	}
	return *cfg.Metadata.Labels.Dash0Comid
}

// GetAwsMetricScrapeConfigName returns the metadata.name of an AWS metric scrape config, or "" if absent.
// The server sets it to the display name of the AWS integration and ignores it on write.
func GetAwsMetricScrapeConfigName(cfg *AwsMetricScrapeConfigDefinition) string {
	if cfg == nil || cfg.Metadata.Name == nil {
		return ""
	}
	return *cfg.Metadata.Name
}

// GetAwsMetricScrapeConfigOrigin returns the dash0.com/origin label of an AWS metric scrape config, or "" if absent.
func GetAwsMetricScrapeConfigOrigin(cfg *AwsMetricScrapeConfigDefinition) string {
	if cfg == nil || cfg.Metadata.Labels == nil || cfg.Metadata.Labels.Dash0Comorigin == nil {
		return ""
	}
	return *cfg.Metadata.Labels.Dash0Comorigin
}

// SetAwsMetricScrapeConfigOrigin sets the dash0.com/origin label on an AWS metric scrape config.
// It initializes the labels struct if needed.
func SetAwsMetricScrapeConfigOrigin(cfg *AwsMetricScrapeConfigDefinition, origin string) {
	if cfg == nil {
		return
	}
	if cfg.Metadata.Labels == nil {
		cfg.Metadata.Labels = &AwsMetricScrapeConfigLabels{}
	}
	cfg.Metadata.Labels.Dash0Comorigin = &origin
}
