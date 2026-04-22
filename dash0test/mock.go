package dash0test

import (
	"context"

	"github.com/dash0hq/dash0-api-client-go"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
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

	// Members
	ListMembersFunc     func(ctx context.Context) ([]*dash0.MemberDefinition, error)
	InviteMemberFunc    func(ctx context.Context, request *dash0.InviteMemberRequest) error
	DeleteMemberFunc    func(ctx context.Context, memberID string) error
	ListMembersIterFunc func(ctx context.Context) *dash0.Iter[dash0.MemberDefinition]

	// Teams
	ListTeamsFunc         func(ctx context.Context) ([]*dash0.TeamsListItem, error)
	CreateTeamFunc        func(ctx context.Context, team *dash0.TeamDefinition) (*dash0.TeamDefinition, error)
	GetTeamFunc           func(ctx context.Context, originOrID string) (*dash0.GetTeamResponse, error)
	DeleteTeamFunc        func(ctx context.Context, originOrID string) error
	UpdateTeamDisplayFunc func(ctx context.Context, originOrID string, display *dash0.TeamDisplay) error
	AddTeamMembersFunc    func(ctx context.Context, originOrID string, request *dash0.AddTeamMembersRequest) error
	RemoveTeamMemberFunc  func(ctx context.Context, originOrID string, memberID string) error
	ListTeamsIterFunc     func(ctx context.Context) *dash0.Iter[dash0.TeamsListItem]

	// Recording Rules
	ListRecordingRulesFunc     func(ctx context.Context, dataset *string) ([]*dash0.RecordingRule, error)
	GetRecordingRuleFunc       func(ctx context.Context, originOrID string, dataset *string) (*dash0.RecordingRule, error)
	CreateRecordingRuleFunc    func(ctx context.Context, rule *dash0.RecordingRule, dataset *string) (*dash0.RecordingRule, error)
	UpdateRecordingRuleFunc    func(ctx context.Context, originOrID string, rule *dash0.RecordingRule, dataset *string) (*dash0.RecordingRule, error)
	DeleteRecordingRuleFunc    func(ctx context.Context, originOrID string, dataset *string) error
	ListRecordingRulesIterFunc func(ctx context.Context, dataset *string) *dash0.Iter[dash0.RecordingRule]

	// Notification Channels
	ListNotificationChannelsFunc     func(ctx context.Context) ([]*dash0.NotificationChannelDefinition, error)
	GetNotificationChannelFunc       func(ctx context.Context, originOrID string) (*dash0.NotificationChannelDefinition, error)
	CreateNotificationChannelFunc    func(ctx context.Context, channel *dash0.NotificationChannelDefinition) (*dash0.NotificationChannelDefinition, error)
	UpdateNotificationChannelFunc    func(ctx context.Context, originOrID string, channel *dash0.NotificationChannelDefinition) (*dash0.NotificationChannelDefinition, error)
	DeleteNotificationChannelFunc    func(ctx context.Context, originOrID string) error
	ListNotificationChannelsIterFunc func(ctx context.Context) *dash0.Iter[dash0.NotificationChannelDefinition]

	// Spans
	GetSpansFunc     func(ctx context.Context, request *dash0.GetSpansRequest) (*dash0.GetSpansResponse, error)
	GetSpansIterFunc func(ctx context.Context, request *dash0.GetSpansRequest) *dash0.Iter[dash0.ResourceSpans]

	// Logs
	GetLogRecordsFunc     func(ctx context.Context, request *dash0.GetLogRecordsRequest) (*dash0.GetLogRecordsResponse, error)
	GetLogRecordsIterFunc func(ctx context.Context, request *dash0.GetLogRecordsRequest) *dash0.Iter[dash0.ResourceLogs]

	// Import
	ImportCheckRuleFunc      func(ctx context.Context, rule *dash0.PrometheusAlertRule, dataset *string) (*dash0.PrometheusAlertRule, error)
	ImportDashboardFunc      func(ctx context.Context, dashboard *dash0.DashboardDefinition, dataset *string) (*dash0.DashboardDefinition, error)
	ImportSyntheticCheckFunc func(ctx context.Context, check *dash0.SyntheticCheckDefinition, dataset *string) (*dash0.SyntheticCheckDefinition, error)
	ImportViewFunc           func(ctx context.Context, view *dash0.ViewDefinition, dataset *string) (*dash0.ViewDefinition, error)

	// OTLP
	SendLogsFunc    func(ctx context.Context, logs plog.Logs, dataset *string) error
	SendMetricsFunc func(ctx context.Context, metrics pmetric.Metrics, dataset *string) error
	SendTracesFunc  func(ctx context.Context, traces ptrace.Traces, dataset *string) error
	CloseFunc       func(ctx context.Context) error

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

// Members

func (m *MockClient) ListMembers(ctx context.Context) ([]*dash0.MemberDefinition, error) {
	if m.ListMembersFunc != nil {
		return m.ListMembersFunc(ctx)
	}
	return nil, nil
}

func (m *MockClient) InviteMember(ctx context.Context, request *dash0.InviteMemberRequest) error {
	if m.InviteMemberFunc != nil {
		return m.InviteMemberFunc(ctx, request)
	}
	return nil
}

func (m *MockClient) DeleteMember(ctx context.Context, memberID string) error {
	if m.DeleteMemberFunc != nil {
		return m.DeleteMemberFunc(ctx, memberID)
	}
	return nil
}

func (m *MockClient) ListMembersIter(ctx context.Context) *dash0.Iter[dash0.MemberDefinition] {
	if m.ListMembersIterFunc != nil {
		return m.ListMembersIterFunc(ctx)
	}
	return nil
}

// Teams

func (m *MockClient) ListTeams(ctx context.Context) ([]*dash0.TeamsListItem, error) {
	if m.ListTeamsFunc != nil {
		return m.ListTeamsFunc(ctx)
	}
	return nil, nil
}

func (m *MockClient) CreateTeam(ctx context.Context, team *dash0.TeamDefinition) (*dash0.TeamDefinition, error) {
	if m.CreateTeamFunc != nil {
		return m.CreateTeamFunc(ctx, team)
	}
	return nil, nil
}

func (m *MockClient) GetTeam(ctx context.Context, originOrID string) (*dash0.GetTeamResponse, error) {
	if m.GetTeamFunc != nil {
		return m.GetTeamFunc(ctx, originOrID)
	}
	return nil, nil
}

func (m *MockClient) DeleteTeam(ctx context.Context, originOrID string) error {
	if m.DeleteTeamFunc != nil {
		return m.DeleteTeamFunc(ctx, originOrID)
	}
	return nil
}

func (m *MockClient) UpdateTeamDisplay(ctx context.Context, originOrID string, display *dash0.TeamDisplay) error {
	if m.UpdateTeamDisplayFunc != nil {
		return m.UpdateTeamDisplayFunc(ctx, originOrID, display)
	}
	return nil
}

func (m *MockClient) AddTeamMembers(ctx context.Context, originOrID string, request *dash0.AddTeamMembersRequest) error {
	if m.AddTeamMembersFunc != nil {
		return m.AddTeamMembersFunc(ctx, originOrID, request)
	}
	return nil
}

func (m *MockClient) RemoveTeamMember(ctx context.Context, originOrID string, memberID string) error {
	if m.RemoveTeamMemberFunc != nil {
		return m.RemoveTeamMemberFunc(ctx, originOrID, memberID)
	}
	return nil
}

func (m *MockClient) ListTeamsIter(ctx context.Context) *dash0.Iter[dash0.TeamsListItem] {
	if m.ListTeamsIterFunc != nil {
		return m.ListTeamsIterFunc(ctx)
	}
	return nil
}

// Recording Rules

func (m *MockClient) ListRecordingRules(ctx context.Context, dataset *string) ([]*dash0.RecordingRule, error) {
	if m.ListRecordingRulesFunc != nil {
		return m.ListRecordingRulesFunc(ctx, dataset)
	}
	return nil, nil
}

func (m *MockClient) GetRecordingRule(ctx context.Context, originOrID string, dataset *string) (*dash0.RecordingRule, error) {
	if m.GetRecordingRuleFunc != nil {
		return m.GetRecordingRuleFunc(ctx, originOrID, dataset)
	}
	return nil, nil
}

func (m *MockClient) CreateRecordingRule(ctx context.Context, rule *dash0.RecordingRule, dataset *string) (*dash0.RecordingRule, error) {
	if m.CreateRecordingRuleFunc != nil {
		return m.CreateRecordingRuleFunc(ctx, rule, dataset)
	}
	return nil, nil
}

func (m *MockClient) UpdateRecordingRule(ctx context.Context, originOrID string, rule *dash0.RecordingRule, dataset *string) (*dash0.RecordingRule, error) {
	if m.UpdateRecordingRuleFunc != nil {
		return m.UpdateRecordingRuleFunc(ctx, originOrID, rule, dataset)
	}
	return nil, nil
}

func (m *MockClient) DeleteRecordingRule(ctx context.Context, originOrID string, dataset *string) error {
	if m.DeleteRecordingRuleFunc != nil {
		return m.DeleteRecordingRuleFunc(ctx, originOrID, dataset)
	}
	return nil
}

func (m *MockClient) ListRecordingRulesIter(ctx context.Context, dataset *string) *dash0.Iter[dash0.RecordingRule] {
	if m.ListRecordingRulesIterFunc != nil {
		return m.ListRecordingRulesIterFunc(ctx, dataset)
	}
	return nil
}

// Notification Channels

func (m *MockClient) ListNotificationChannels(ctx context.Context) ([]*dash0.NotificationChannelDefinition, error) {
	if m.ListNotificationChannelsFunc != nil {
		return m.ListNotificationChannelsFunc(ctx)
	}
	return nil, nil
}

func (m *MockClient) GetNotificationChannel(ctx context.Context, originOrID string) (*dash0.NotificationChannelDefinition, error) {
	if m.GetNotificationChannelFunc != nil {
		return m.GetNotificationChannelFunc(ctx, originOrID)
	}
	return nil, nil
}

func (m *MockClient) CreateNotificationChannel(ctx context.Context, channel *dash0.NotificationChannelDefinition) (*dash0.NotificationChannelDefinition, error) {
	if m.CreateNotificationChannelFunc != nil {
		return m.CreateNotificationChannelFunc(ctx, channel)
	}
	return nil, nil
}

func (m *MockClient) UpdateNotificationChannel(ctx context.Context, originOrID string, channel *dash0.NotificationChannelDefinition) (*dash0.NotificationChannelDefinition, error) {
	if m.UpdateNotificationChannelFunc != nil {
		return m.UpdateNotificationChannelFunc(ctx, originOrID, channel)
	}
	return nil, nil
}

func (m *MockClient) DeleteNotificationChannel(ctx context.Context, originOrID string) error {
	if m.DeleteNotificationChannelFunc != nil {
		return m.DeleteNotificationChannelFunc(ctx, originOrID)
	}
	return nil
}

func (m *MockClient) ListNotificationChannelsIter(ctx context.Context) *dash0.Iter[dash0.NotificationChannelDefinition] {
	if m.ListNotificationChannelsIterFunc != nil {
		return m.ListNotificationChannelsIterFunc(ctx)
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

func (m *MockClient) ImportCheckRule(ctx context.Context, rule *dash0.PrometheusAlertRule, dataset *string) (*dash0.PrometheusAlertRule, error) {
	if m.ImportCheckRuleFunc != nil {
		return m.ImportCheckRuleFunc(ctx, rule, dataset)
	}
	return nil, nil
}

func (m *MockClient) ImportDashboard(ctx context.Context, dashboard *dash0.DashboardDefinition, dataset *string) (*dash0.DashboardDefinition, error) {
	if m.ImportDashboardFunc != nil {
		return m.ImportDashboardFunc(ctx, dashboard, dataset)
	}
	return nil, nil
}

func (m *MockClient) ImportSyntheticCheck(ctx context.Context, check *dash0.SyntheticCheckDefinition, dataset *string) (*dash0.SyntheticCheckDefinition, error) {
	if m.ImportSyntheticCheckFunc != nil {
		return m.ImportSyntheticCheckFunc(ctx, check, dataset)
	}
	return nil, nil
}

func (m *MockClient) ImportView(ctx context.Context, view *dash0.ViewDefinition, dataset *string) (*dash0.ViewDefinition, error) {
	if m.ImportViewFunc != nil {
		return m.ImportViewFunc(ctx, view, dataset)
	}
	return nil, nil
}

// OTLP

func (m *MockClient) SendTraces(ctx context.Context, traces ptrace.Traces, dataset *string) error {
	if m.SendTracesFunc != nil {
		return m.SendTracesFunc(ctx, traces, dataset)
	}
	return nil
}

func (m *MockClient) SendMetrics(ctx context.Context, metrics pmetric.Metrics, dataset *string) error {
	if m.SendMetricsFunc != nil {
		return m.SendMetricsFunc(ctx, metrics, dataset)
	}
	return nil
}

func (m *MockClient) SendLogs(ctx context.Context, logs plog.Logs, dataset *string) error {
	if m.SendLogsFunc != nil {
		return m.SendLogsFunc(ctx, logs, dataset)
	}
	return nil
}

func (m *MockClient) Close(ctx context.Context) error {
	if m.CloseFunc != nil {
		return m.CloseFunc(ctx)
	}
	return nil
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
