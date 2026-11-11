package dash0

// AssetMetadata is implemented by every definition type that can be upserted, so a caller can read the identifiers without switching on the kind.
// Each method is nil-safe and returns "" or false on a nil receiver.
type AssetMetadata interface {
	// GetOrigin returns the dash0.com/origin label, or "" when it is unset or the type is not upserted by origin.
	GetOrigin() string
	// GetID returns the asset ID.
	GetID() string
	// GetName returns the asset name.
	GetName() string
	// UpsertKeys reports which identifiers the API upserts the asset by.
	// When both are true, the origin is preferred.
	UpsertKeys() (byOrigin, byID bool)
}

var (
	_ AssetMetadata = (*SloDefinition)(nil)
	_ AssetMetadata = (*TeamDefinitionV1Alpha1)(nil)
	_ AssetMetadata = (*SkillDefinition)(nil)
	_ AssetMetadata = (*GenerativeAiEvaluationDefinition)(nil)
	_ AssetMetadata = (*SignalToMetricsDefinition)(nil)
	_ AssetMetadata = (*SpamFilter)(nil)
	_ AssetMetadata = (*SpamFilterV1Alpha2)(nil)
	_ AssetMetadata = (*NotificationChannelDefinition)(nil)
	_ AssetMetadata = (*TimeSeriesAggregationDefinition)(nil)
	_ AssetMetadata = (*DashboardDefinition)(nil)
	_ AssetMetadata = (*PersesDashboard)(nil)
	_ AssetMetadata = (*PrometheusAlertRule)(nil)
	_ AssetMetadata = (*PrometheusRules)(nil)
	_ AssetMetadata = (*RecordingRule)(nil)
	_ AssetMetadata = (*SyntheticCheckDefinition)(nil)
	_ AssetMetadata = (*ViewDefinition)(nil)
	_ AssetMetadata = (*DatasetDefinition)(nil)
	_ AssetMetadata = (*AwsMetricScrapeConfigDefinition)(nil)
)

func (slo *SloDefinition) GetOrigin() string             { return GetSLOOrigin(slo) }
func (slo *SloDefinition) GetID() string                 { return GetSLOID(slo) }
func (slo *SloDefinition) GetName() string               { return GetSLOName(slo) }
func (*SloDefinition) UpsertKeys() (byOrigin, byID bool) { return true, true }

func (team *TeamDefinitionV1Alpha1) GetOrigin() string            { return GetTeamOrigin(team) }
func (team *TeamDefinitionV1Alpha1) GetID() string                { return GetTeamID(team) }
func (team *TeamDefinitionV1Alpha1) GetName() string              { return GetTeamName(team) }
func (*TeamDefinitionV1Alpha1) UpsertKeys() (byOrigin, byID bool) { return true, true }

func (skill *SkillDefinition) GetOrigin() string           { return GetCustomSkillOrigin(skill) }
func (skill *SkillDefinition) GetID() string               { return GetCustomSkillID(skill) }
func (skill *SkillDefinition) GetName() string             { return GetCustomSkillName(skill) }
func (*SkillDefinition) UpsertKeys() (byOrigin, byID bool) { return true, true }

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

func (rule *SignalToMetricsDefinition) GetOrigin() string            { return GetSignalToMetricsOrigin(rule) }
func (rule *SignalToMetricsDefinition) GetID() string                { return GetSignalToMetricsID(rule) }
func (rule *SignalToMetricsDefinition) GetName() string              { return GetSignalToMetricsName(rule) }
func (*SignalToMetricsDefinition) UpsertKeys() (byOrigin, byID bool) { return true, true }

func (filter *SpamFilter) GetOrigin() string          { return GetSpamFilterOrigin(filter) }
func (filter *SpamFilter) GetID() string              { return GetSpamFilterID(filter) }
func (filter *SpamFilter) GetName() string            { return GetSpamFilterName(filter) }
func (*SpamFilter) UpsertKeys() (byOrigin, byID bool) { return true, true }

func (filter *SpamFilterV1Alpha2) GetOrigin() string          { return GetSpamFilterV1Alpha2Origin(filter) }
func (filter *SpamFilterV1Alpha2) GetID() string              { return GetSpamFilterV1Alpha2ID(filter) }
func (filter *SpamFilterV1Alpha2) GetName() string            { return GetSpamFilterV1Alpha2Name(filter) }
func (*SpamFilterV1Alpha2) UpsertKeys() (byOrigin, byID bool) { return true, true }

func (channel *NotificationChannelDefinition) GetOrigin() string {
	return GetNotificationChannelOrigin(channel)
}
func (channel *NotificationChannelDefinition) GetID() string {
	return GetNotificationChannelID(channel)
}
func (channel *NotificationChannelDefinition) GetName() string {
	return GetNotificationChannelName(channel)
}
func (*NotificationChannelDefinition) UpsertKeys() (byOrigin, byID bool) { return true, false }

func (aggregation *TimeSeriesAggregationDefinition) GetOrigin() string {
	return GetTimeSeriesAggregationOrigin(aggregation)
}
func (aggregation *TimeSeriesAggregationDefinition) GetID() string {
	return GetTimeSeriesAggregationID(aggregation)
}
func (aggregation *TimeSeriesAggregationDefinition) GetName() string {
	return GetTimeSeriesAggregationName(aggregation)
}
func (*TimeSeriesAggregationDefinition) UpsertKeys() (byOrigin, byID bool) { return true, false }

// GetOrigin returns "", because DashboardDefinition is not upserted by origin.
func (*DashboardDefinition) GetOrigin() string                 { return "" }
func (dashboard *DashboardDefinition) GetID() string           { return GetDashboardID(dashboard) }
func (dashboard *DashboardDefinition) GetName() string         { return GetDashboardName(dashboard) }
func (*DashboardDefinition) UpsertKeys() (byOrigin, byID bool) { return false, true }

// GetOrigin returns "", because PersesDashboard is not upserted by origin.
func (*PersesDashboard) GetOrigin() string                 { return "" }
func (perses *PersesDashboard) GetID() string              { return GetPersesDashboardID(perses) }
func (perses *PersesDashboard) GetName() string            { return GetPersesDashboardName(perses) }
func (*PersesDashboard) UpsertKeys() (byOrigin, byID bool) { return false, true }

// GetOrigin returns "", because PrometheusAlertRule is not upserted by origin.
func (*PrometheusAlertRule) GetOrigin() string                 { return "" }
func (rule *PrometheusAlertRule) GetID() string                { return GetCheckRuleID(rule) }
func (rule *PrometheusAlertRule) GetName() string              { return GetCheckRuleName(rule) }
func (*PrometheusAlertRule) UpsertKeys() (byOrigin, byID bool) { return false, true }

// GetOrigin returns "", because PrometheusRules is not upserted by origin.
func (*PrometheusRules) GetOrigin() string                 { return "" }
func (rule *PrometheusRules) GetID() string                { return GetPrometheusRuleID(rule) }
func (rule *PrometheusRules) GetName() string              { return GetPrometheusRuleName(rule) }
func (*PrometheusRules) UpsertKeys() (byOrigin, byID bool) { return false, true }

// GetOrigin returns "", because RecordingRule is not upserted by origin.
func (*RecordingRule) GetOrigin() string                 { return "" }
func (rule *RecordingRule) GetID() string                { return GetRecordingRuleID(rule) }
func (rule *RecordingRule) GetName() string              { return GetRecordingRuleName(rule) }
func (*RecordingRule) UpsertKeys() (byOrigin, byID bool) { return false, true }

// GetOrigin returns "", because SyntheticCheckDefinition is not upserted by origin.
func (*SyntheticCheckDefinition) GetOrigin() string                 { return "" }
func (check *SyntheticCheckDefinition) GetID() string               { return GetSyntheticCheckID(check) }
func (check *SyntheticCheckDefinition) GetName() string             { return GetSyntheticCheckName(check) }
func (*SyntheticCheckDefinition) UpsertKeys() (byOrigin, byID bool) { return false, true }

// GetOrigin returns "", because ViewDefinition is not upserted by origin.
func (*ViewDefinition) GetOrigin() string                 { return "" }
func (view *ViewDefinition) GetID() string                { return GetViewID(view) }
func (view *ViewDefinition) GetName() string              { return GetViewName(view) }
func (*ViewDefinition) UpsertKeys() (byOrigin, byID bool) { return false, true }

func (dataset *DatasetDefinition) GetOrigin() string         { return GetDatasetOrigin(dataset) }
func (dataset *DatasetDefinition) GetID() string             { return GetDatasetID(dataset) }
func (dataset *DatasetDefinition) GetName() string           { return GetDatasetName(dataset) }
func (*DatasetDefinition) UpsertKeys() (byOrigin, byID bool) { return false, true }

func (cfg *AwsMetricScrapeConfigDefinition) GetOrigin() string {
	return GetAwsMetricScrapeConfigOrigin(cfg)
}
func (cfg *AwsMetricScrapeConfigDefinition) GetID() string { return GetAwsMetricScrapeConfigID(cfg) }
func (cfg *AwsMetricScrapeConfigDefinition) GetName() string {
	return GetAwsMetricScrapeConfigName(cfg)
}
func (*AwsMetricScrapeConfigDefinition) UpsertKeys() (byOrigin, byID bool) { return false, true }
