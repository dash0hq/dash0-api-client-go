package dash0

import (
	"context"
	"fmt"
	"net/http"
)

// ImportCheckRule imports a check rule.
func (c *client) ImportCheckRule(ctx context.Context, rule *PrometheusAlertRule, dataset *string) (*PrometheusAlertRule, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	params := &PostApiImportCheckRuleParams{
		Dataset: dataset,
	}
	resp, err := c.inner.PostApiImportCheckRuleWithResponse(ctx, params, *rule)
	if err != nil {
		return nil, fmt.Errorf("dash0: import check rule failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200, nil
}

// ImportDashboard imports a dashboard.
func (c *client) ImportDashboard(ctx context.Context, dashboard *DashboardDefinition, dataset *string) (*DashboardDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	params := &PostApiImportDashboardParams{
		Dataset: dataset,
	}
	resp, err := c.inner.PostApiImportDashboardWithResponse(ctx, params, *dashboard)
	if err != nil {
		return nil, fmt.Errorf("dash0: import dashboard failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200, nil
}

// ImportSyntheticCheck imports a synthetic check.
func (c *client) ImportSyntheticCheck(ctx context.Context, check *SyntheticCheckDefinition, dataset *string) (*SyntheticCheckDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	params := &PostApiImportSyntheticCheckParams{
		Dataset: dataset,
	}
	resp, err := c.inner.PostApiImportSyntheticCheckWithResponse(ctx, params, *check)
	if err != nil {
		return nil, fmt.Errorf("dash0: import synthetic check failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200, nil
}

// ImportView imports a view.
func (c *client) ImportView(ctx context.Context, view *ViewDefinition, dataset *string) (*ViewDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	params := &PostApiImportViewParams{
		Dataset: dataset,
	}
	resp, err := c.inner.PostApiImportViewWithResponse(ctx, params, *view)
	if err != nil {
		return nil, fmt.Errorf("dash0: import view failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200, nil
}

// ImportSignalToMetrics creates or updates signal-to-metrics rules in dataset, matching existing rules by their dash0.com/origin label.
// Every rule must carry that label, so that repeated imports of the same rules update them instead of creating copies.
// The import is atomic: when any rule fails validation, the server rejects the whole batch and changes nothing.
// The API accepts at most 500 rules per call.
//
// Unlike the other imports, the dataset is required, so that rules cannot land in the default dataset by accident.
func (c *client) ImportSignalToMetrics(ctx context.Context, rules []*SignalToMetricsDefinition, dataset string) (*SignalToMetricsBulkUpsertResponse, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	if dataset == "" {
		return nil, fmt.Errorf("dash0: import signal-to-metrics requires a dataset")
	}
	items := make([]SignalToMetricsDefinition, len(rules))
	for i, rule := range rules {
		if rule == nil {
			return nil, fmt.Errorf("dash0: import signal-to-metrics requires non-nil rules, but rule %d is nil", i)
		}
		items[i] = *rule
	}
	params := &PutApiImportSignalToMetricsParams{
		Dataset: dataset,
	}
	resp, err := c.inner.PutApiImportSignalToMetricsWithResponse(ctx, params, SignalToMetricsBulkUpsertRequest{Items: items})
	if err != nil {
		return nil, fmt.Errorf("dash0: import signal-to-metrics failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return resp.JSON200, nil
}
