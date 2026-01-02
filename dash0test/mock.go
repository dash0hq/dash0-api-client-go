package dash0test

import (
	"context"

	"github.com/dash0hq/dash0-api-client-go"
)

// MockClient is a configurable mock implementation of dash0.Client.
// Set the function fields to customize behavior for each test.
//
// Example:
//
//	mock := &dash0test.MockClient{
//	    ListDashboardsFunc: func(ctx context.Context, dataset *string) ([]*dash0.DashboardApiListItem, error) {
//	        return []*dash0.DashboardApiListItem{{Id: dash0.Ptr("test-id")}}, nil
//	    },
//	}
//	svc := NewMyService(mock) // accepts dash0.Client interface
type MockClient struct {
	// Dashboards
	ListDashboardsFunc     func(ctx context.Context, dataset *string) ([]*dash0.DashboardApiListItem, error)
	GetDashboardFunc       func(ctx context.Context, originOrID string, dataset *string) (*dash0.DashboardDefinition, error)
	CreateDashboardFunc    func(ctx context.Context, dashboard *dash0.DashboardDefinition, dataset *string) (*dash0.DashboardDefinition, error)
	UpdateDashboardFunc    func(ctx context.Context, originOrID string, dashboard *dash0.DashboardDefinition, dataset *string) (*dash0.DashboardDefinition, error)
	DeleteDashboardFunc    func(ctx context.Context, originOrID string, dataset *string) error
	ListDashboardsIterFunc func(ctx context.Context, dataset *string) *dash0.Iter[dash0.DashboardApiListItem]

	// Check Rules
	ListCheckRulesFunc     func(ctx context.Context, dataset *string) ([]*dash0.PrometheusAlertRuleApiListItem, error)
	GetCheckRuleFunc       func(ctx context.Context, originOrID string, dataset *string) (*dash0.PrometheusAlertRule, error)
	CreateCheckRuleFunc    func(ctx context.Context, rule *dash0.PrometheusAlertRule, dataset *string) (*dash0.PrometheusAlertRule, error)
	UpdateCheckRuleFunc    func(ctx context.Context, originOrID string, rule *dash0.PrometheusAlertRule, dataset *string) (*dash0.PrometheusAlertRule, error)
	DeleteCheckRuleFunc    func(ctx context.Context, originOrID string, dataset *string) error
	ListCheckRulesIterFunc func(ctx context.Context, dataset *string) *dash0.Iter[dash0.PrometheusAlertRuleApiListItem]

	// Synthetic Checks
	ListSyntheticChecksFunc     func(ctx context.Context, dataset *string) ([]*dash0.SyntheticChecksApiListItem, error)
	GetSyntheticCheckFunc       func(ctx context.Context, originOrID string, dataset *string) (*dash0.SyntheticCheckDefinition, error)
	CreateSyntheticCheckFunc    func(ctx context.Context, check *dash0.SyntheticCheckDefinition, dataset *string) (*dash0.SyntheticCheckDefinition, error)
	UpdateSyntheticCheckFunc    func(ctx context.Context, originOrID string, check *dash0.SyntheticCheckDefinition, dataset *string) (*dash0.SyntheticCheckDefinition, error)
	DeleteSyntheticCheckFunc    func(ctx context.Context, originOrID string, dataset *string) error
	ListSyntheticChecksIterFunc func(ctx context.Context, dataset *string) *dash0.Iter[dash0.SyntheticChecksApiListItem]

	// Views
	ListViewsFunc     func(ctx context.Context, dataset *string) ([]*dash0.ViewApiListItem, error)
	GetViewFunc       func(ctx context.Context, originOrID string, dataset *string) (*dash0.ViewDefinition, error)
	CreateViewFunc    func(ctx context.Context, view *dash0.ViewDefinition, dataset *string) (*dash0.ViewDefinition, error)
	UpdateViewFunc    func(ctx context.Context, originOrID string, view *dash0.ViewDefinition, dataset *string) (*dash0.ViewDefinition, error)
	DeleteViewFunc    func(ctx context.Context, originOrID string, dataset *string) error
	ListViewsIterFunc func(ctx context.Context, dataset *string) *dash0.Iter[dash0.ViewApiListItem]

	// Sampling Rules
	ListSamplingRulesFunc     func(ctx context.Context, dataset *string) ([]*dash0.SamplingDefinition, error)
	GetSamplingRuleFunc       func(ctx context.Context, originOrID string, dataset *string) (*dash0.SamplingDefinition, error)
	CreateSamplingRuleFunc    func(ctx context.Context, rule *dash0.SamplingDefinition, dataset *string) (*dash0.SamplingDefinition, error)
	UpdateSamplingRuleFunc    func(ctx context.Context, originOrID string, rule *dash0.SamplingDefinition, dataset *string) (*dash0.SamplingDefinition, error)
	DeleteSamplingRuleFunc    func(ctx context.Context, originOrID string, dataset *string) error
	ListSamplingRulesIterFunc func(ctx context.Context, dataset *string) *dash0.Iter[dash0.SamplingDefinition]

	// Spans
	GetSpansFunc     func(ctx context.Context, request *dash0.GetSpansRequest) (*dash0.GetSpansResponse, error)
	GetSpansIterFunc func(ctx context.Context, request *dash0.GetSpansRequest) *dash0.Iter[dash0.ResourceSpans]

	// Logs
	GetLogRecordsFunc     func(ctx context.Context, request *dash0.GetLogRecordsRequest) (*dash0.GetLogRecordsResponse, error)
	GetLogRecordsIterFunc func(ctx context.Context, request *dash0.GetLogRecordsRequest) *dash0.Iter[dash0.ResourceLogs]

	// Import
	ImportCheckRuleFunc      func(ctx context.Context, rule *dash0.PostApiImportCheckRuleJSONRequestBody, dataset *string) (*dash0.PrometheusAlertRule, error)
	ImportDashboardFunc      func(ctx context.Context, dashboard *dash0.PostApiImportDashboardJSONRequestBody, dataset *string) (*dash0.DashboardDefinition, error)
	ImportSyntheticCheckFunc func(ctx context.Context, check *dash0.PostApiImportSyntheticCheckJSONRequestBody, dataset *string) (*dash0.SyntheticCheckDefinition, error)
	ImportViewFunc           func(ctx context.Context, view *dash0.PostApiImportViewJSONRequestBody, dataset *string) (*dash0.ViewDefinition, error)

	// Inner
	InnerFunc func() *dash0.ClientWithResponses
}

// Dashboards

func (m *MockClient) ListDashboards(ctx context.Context, dataset *string) ([]*dash0.DashboardApiListItem, error) {
	if m.ListDashboardsFunc != nil {
		return m.ListDashboardsFunc(ctx, dataset)
	}
	return nil, nil
}

func (m *MockClient) GetDashboard(ctx context.Context, originOrID string, dataset *string) (*dash0.DashboardDefinition, error) {
	if m.GetDashboardFunc != nil {
		return m.GetDashboardFunc(ctx, originOrID, dataset)
	}
	return nil, nil
}

func (m *MockClient) CreateDashboard(ctx context.Context, dashboard *dash0.DashboardDefinition, dataset *string) (*dash0.DashboardDefinition, error) {
	if m.CreateDashboardFunc != nil {
		return m.CreateDashboardFunc(ctx, dashboard, dataset)
	}
	return nil, nil
}

func (m *MockClient) UpdateDashboard(ctx context.Context, originOrID string, dashboard *dash0.DashboardDefinition, dataset *string) (*dash0.DashboardDefinition, error) {
	if m.UpdateDashboardFunc != nil {
		return m.UpdateDashboardFunc(ctx, originOrID, dashboard, dataset)
	}
	return nil, nil
}

func (m *MockClient) DeleteDashboard(ctx context.Context, originOrID string, dataset *string) error {
	if m.DeleteDashboardFunc != nil {
		return m.DeleteDashboardFunc(ctx, originOrID, dataset)
	}
	return nil
}

func (m *MockClient) ListDashboardsIter(ctx context.Context, dataset *string) *dash0.Iter[dash0.DashboardApiListItem] {
	if m.ListDashboardsIterFunc != nil {
		return m.ListDashboardsIterFunc(ctx, dataset)
	}
	return nil
}

// Check Rules

func (m *MockClient) ListCheckRules(ctx context.Context, dataset *string) ([]*dash0.PrometheusAlertRuleApiListItem, error) {
	if m.ListCheckRulesFunc != nil {
		return m.ListCheckRulesFunc(ctx, dataset)
	}
	return nil, nil
}

func (m *MockClient) GetCheckRule(ctx context.Context, originOrID string, dataset *string) (*dash0.PrometheusAlertRule, error) {
	if m.GetCheckRuleFunc != nil {
		return m.GetCheckRuleFunc(ctx, originOrID, dataset)
	}
	return nil, nil
}

func (m *MockClient) CreateCheckRule(ctx context.Context, rule *dash0.PrometheusAlertRule, dataset *string) (*dash0.PrometheusAlertRule, error) {
	if m.CreateCheckRuleFunc != nil {
		return m.CreateCheckRuleFunc(ctx, rule, dataset)
	}
	return nil, nil
}

func (m *MockClient) UpdateCheckRule(ctx context.Context, originOrID string, rule *dash0.PrometheusAlertRule, dataset *string) (*dash0.PrometheusAlertRule, error) {
	if m.UpdateCheckRuleFunc != nil {
		return m.UpdateCheckRuleFunc(ctx, originOrID, rule, dataset)
	}
	return nil, nil
}

func (m *MockClient) DeleteCheckRule(ctx context.Context, originOrID string, dataset *string) error {
	if m.DeleteCheckRuleFunc != nil {
		return m.DeleteCheckRuleFunc(ctx, originOrID, dataset)
	}
	return nil
}

func (m *MockClient) ListCheckRulesIter(ctx context.Context, dataset *string) *dash0.Iter[dash0.PrometheusAlertRuleApiListItem] {
	if m.ListCheckRulesIterFunc != nil {
		return m.ListCheckRulesIterFunc(ctx, dataset)
	}
	return nil
}

// Synthetic Checks

func (m *MockClient) ListSyntheticChecks(ctx context.Context, dataset *string) ([]*dash0.SyntheticChecksApiListItem, error) {
	if m.ListSyntheticChecksFunc != nil {
		return m.ListSyntheticChecksFunc(ctx, dataset)
	}
	return nil, nil
}

func (m *MockClient) GetSyntheticCheck(ctx context.Context, originOrID string, dataset *string) (*dash0.SyntheticCheckDefinition, error) {
	if m.GetSyntheticCheckFunc != nil {
		return m.GetSyntheticCheckFunc(ctx, originOrID, dataset)
	}
	return nil, nil
}

func (m *MockClient) CreateSyntheticCheck(ctx context.Context, check *dash0.SyntheticCheckDefinition, dataset *string) (*dash0.SyntheticCheckDefinition, error) {
	if m.CreateSyntheticCheckFunc != nil {
		return m.CreateSyntheticCheckFunc(ctx, check, dataset)
	}
	return nil, nil
}

func (m *MockClient) UpdateSyntheticCheck(ctx context.Context, originOrID string, check *dash0.SyntheticCheckDefinition, dataset *string) (*dash0.SyntheticCheckDefinition, error) {
	if m.UpdateSyntheticCheckFunc != nil {
		return m.UpdateSyntheticCheckFunc(ctx, originOrID, check, dataset)
	}
	return nil, nil
}

func (m *MockClient) DeleteSyntheticCheck(ctx context.Context, originOrID string, dataset *string) error {
	if m.DeleteSyntheticCheckFunc != nil {
		return m.DeleteSyntheticCheckFunc(ctx, originOrID, dataset)
	}
	return nil
}

func (m *MockClient) ListSyntheticChecksIter(ctx context.Context, dataset *string) *dash0.Iter[dash0.SyntheticChecksApiListItem] {
	if m.ListSyntheticChecksIterFunc != nil {
		return m.ListSyntheticChecksIterFunc(ctx, dataset)
	}
	return nil
}

// Views

func (m *MockClient) ListViews(ctx context.Context, dataset *string) ([]*dash0.ViewApiListItem, error) {
	if m.ListViewsFunc != nil {
		return m.ListViewsFunc(ctx, dataset)
	}
	return nil, nil
}

func (m *MockClient) GetView(ctx context.Context, originOrID string, dataset *string) (*dash0.ViewDefinition, error) {
	if m.GetViewFunc != nil {
		return m.GetViewFunc(ctx, originOrID, dataset)
	}
	return nil, nil
}

func (m *MockClient) CreateView(ctx context.Context, view *dash0.ViewDefinition, dataset *string) (*dash0.ViewDefinition, error) {
	if m.CreateViewFunc != nil {
		return m.CreateViewFunc(ctx, view, dataset)
	}
	return nil, nil
}

func (m *MockClient) UpdateView(ctx context.Context, originOrID string, view *dash0.ViewDefinition, dataset *string) (*dash0.ViewDefinition, error) {
	if m.UpdateViewFunc != nil {
		return m.UpdateViewFunc(ctx, originOrID, view, dataset)
	}
	return nil, nil
}

func (m *MockClient) DeleteView(ctx context.Context, originOrID string, dataset *string) error {
	if m.DeleteViewFunc != nil {
		return m.DeleteViewFunc(ctx, originOrID, dataset)
	}
	return nil
}

func (m *MockClient) ListViewsIter(ctx context.Context, dataset *string) *dash0.Iter[dash0.ViewApiListItem] {
	if m.ListViewsIterFunc != nil {
		return m.ListViewsIterFunc(ctx, dataset)
	}
	return nil
}

// Sampling Rules

func (m *MockClient) ListSamplingRules(ctx context.Context, dataset *string) ([]*dash0.SamplingDefinition, error) {
	if m.ListSamplingRulesFunc != nil {
		return m.ListSamplingRulesFunc(ctx, dataset)
	}
	return nil, nil
}

func (m *MockClient) GetSamplingRule(ctx context.Context, originOrID string, dataset *string) (*dash0.SamplingDefinition, error) {
	if m.GetSamplingRuleFunc != nil {
		return m.GetSamplingRuleFunc(ctx, originOrID, dataset)
	}
	return nil, nil
}

func (m *MockClient) CreateSamplingRule(ctx context.Context, rule *dash0.SamplingDefinition, dataset *string) (*dash0.SamplingDefinition, error) {
	if m.CreateSamplingRuleFunc != nil {
		return m.CreateSamplingRuleFunc(ctx, rule, dataset)
	}
	return nil, nil
}

func (m *MockClient) UpdateSamplingRule(ctx context.Context, originOrID string, rule *dash0.SamplingDefinition, dataset *string) (*dash0.SamplingDefinition, error) {
	if m.UpdateSamplingRuleFunc != nil {
		return m.UpdateSamplingRuleFunc(ctx, originOrID, rule, dataset)
	}
	return nil, nil
}

func (m *MockClient) DeleteSamplingRule(ctx context.Context, originOrID string, dataset *string) error {
	if m.DeleteSamplingRuleFunc != nil {
		return m.DeleteSamplingRuleFunc(ctx, originOrID, dataset)
	}
	return nil
}

func (m *MockClient) ListSamplingRulesIter(ctx context.Context, dataset *string) *dash0.Iter[dash0.SamplingDefinition] {
	if m.ListSamplingRulesIterFunc != nil {
		return m.ListSamplingRulesIterFunc(ctx, dataset)
	}
	return nil
}

// Spans

func (m *MockClient) GetSpans(ctx context.Context, request *dash0.GetSpansRequest) (*dash0.GetSpansResponse, error) {
	if m.GetSpansFunc != nil {
		return m.GetSpansFunc(ctx, request)
	}
	return nil, nil
}

func (m *MockClient) GetSpansIter(ctx context.Context, request *dash0.GetSpansRequest) *dash0.Iter[dash0.ResourceSpans] {
	if m.GetSpansIterFunc != nil {
		return m.GetSpansIterFunc(ctx, request)
	}
	return nil
}

// Logs

func (m *MockClient) GetLogRecords(ctx context.Context, request *dash0.GetLogRecordsRequest) (*dash0.GetLogRecordsResponse, error) {
	if m.GetLogRecordsFunc != nil {
		return m.GetLogRecordsFunc(ctx, request)
	}
	return nil, nil
}

func (m *MockClient) GetLogRecordsIter(ctx context.Context, request *dash0.GetLogRecordsRequest) *dash0.Iter[dash0.ResourceLogs] {
	if m.GetLogRecordsIterFunc != nil {
		return m.GetLogRecordsIterFunc(ctx, request)
	}
	return nil
}

// Import

func (m *MockClient) ImportCheckRule(ctx context.Context, rule *dash0.PostApiImportCheckRuleJSONRequestBody, dataset *string) (*dash0.PrometheusAlertRule, error) {
	if m.ImportCheckRuleFunc != nil {
		return m.ImportCheckRuleFunc(ctx, rule, dataset)
	}
	return nil, nil
}

func (m *MockClient) ImportDashboard(ctx context.Context, dashboard *dash0.PostApiImportDashboardJSONRequestBody, dataset *string) (*dash0.DashboardDefinition, error) {
	if m.ImportDashboardFunc != nil {
		return m.ImportDashboardFunc(ctx, dashboard, dataset)
	}
	return nil, nil
}

func (m *MockClient) ImportSyntheticCheck(ctx context.Context, check *dash0.PostApiImportSyntheticCheckJSONRequestBody, dataset *string) (*dash0.SyntheticCheckDefinition, error) {
	if m.ImportSyntheticCheckFunc != nil {
		return m.ImportSyntheticCheckFunc(ctx, check, dataset)
	}
	return nil, nil
}

func (m *MockClient) ImportView(ctx context.Context, view *dash0.PostApiImportViewJSONRequestBody, dataset *string) (*dash0.ViewDefinition, error) {
	if m.ImportViewFunc != nil {
		return m.ImportViewFunc(ctx, view, dataset)
	}
	return nil, nil
}

// Inner

func (m *MockClient) Inner() *dash0.ClientWithResponses {
	if m.InnerFunc != nil {
		return m.InnerFunc()
	}
	return nil
}

// Compile-time check that MockClient implements dash0.Client.
var _ dash0.Client = (*MockClient)(nil)
