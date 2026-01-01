package dash0

import (
	"context"
	"fmt"
	"net/http"
)

// ImportCheckRule imports a check rule
func (c *client) ImportCheckRule(ctx context.Context, rule *PostApiImportCheckRuleJSONRequestBody, dataset *string) (*PrometheusAlertRule, error) {
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

// ImportDashboard imports a dashboard
func (c *client) ImportDashboard(ctx context.Context, dashboard *PostApiImportDashboardJSONRequestBody, dataset *string) (*DashboardDefinition, error) {
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
func (c *client) ImportSyntheticCheck(ctx context.Context, check *PostApiImportSyntheticCheckJSONRequestBody, dataset *string) (*SyntheticCheckDefinition, error) {
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
func (c *client) ImportView(ctx context.Context, view *PostApiImportViewJSONRequestBody, dataset *string) (*ViewDefinition, error) {
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
