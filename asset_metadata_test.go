package dash0

import "testing"

// TestUpsertKeys pins the upsert keys of every AssetMetadata type to the dash0-cli apply behavior.
func TestUpsertKeys(t *testing.T) {
	tests := []struct {
		asset    AssetMetadata
		byOrigin bool
		byID     bool
	}{
		{(*SloDefinition)(nil), true, true},
		{(*TeamDefinitionV1Alpha1)(nil), true, true},
		{(*SkillDefinition)(nil), true, true},
		{(*GenerativeAiEvaluationDefinition)(nil), true, true},
		{(*SignalToMetricsDefinition)(nil), true, true},
		{(*SpamFilter)(nil), true, true},
		{(*SpamFilterV1Alpha2)(nil), true, true},
		{(*NotificationChannelDefinition)(nil), true, false},
		{(*TimeSeriesAggregationDefinition)(nil), true, false},
		{(*DashboardDefinition)(nil), false, true},
		{(*PersesDashboard)(nil), false, true},
		{(*PrometheusAlertRule)(nil), false, true},
		{(*PrometheusRules)(nil), false, true},
		{(*RecordingRule)(nil), false, true},
		{(*SyntheticCheckDefinition)(nil), false, true},
		{(*ViewDefinition)(nil), false, true},
		{(*DatasetDefinition)(nil), false, true},
		{(*AwsMetricScrapeConfigDefinition)(nil), false, true},
	}
	for _, tt := range tests {
		byOrigin, byID := tt.asset.UpsertKeys()
		if byOrigin != tt.byOrigin || byID != tt.byID {
			t.Errorf("%T.UpsertKeys() = (%v, %v), want (%v, %v)", tt.asset, byOrigin, byID, tt.byOrigin, tt.byID)
		}
	}
}
