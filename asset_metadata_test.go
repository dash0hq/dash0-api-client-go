package dash0

import (
	"encoding/json"
	"fmt"
	"testing"
)

// assetMetadataDocument sets the origin, the id, and the name in every location the definition types read them from.
// Each type ignores the fields it does not have.
const assetMetadataDocument = `{
	"id": "test-id",
	"name": "test-name",
	"labels": {"dash0.com/origin": "test-origin"},
	"metadata": {
		"name": "test-name",
		"labels": {"dash0.com/origin": "test-origin", "dash0.com/id": "test-id"},
		"dash0Extensions": {"id": "test-id"}
	},
	"spec": {"display": {"name": "test-name"}, "slug": "test-id"}
}`

// TestAssetMetadata pins what every AssetMetadata type returns, and its upsert keys to the dash0-cli apply behavior.
func TestAssetMetadata(t *testing.T) {
	tests := []struct {
		asset    AssetMetadata
		origin   string
		byOrigin bool
		byID     bool
	}{
		{&SloDefinition{}, "test-origin", true, true},
		{&TeamDefinitionV1Alpha1{}, "test-origin", true, true},
		{&SkillDefinition{}, "test-origin", true, true},
		{&GenerativeAiEvaluationDefinition{}, "test-origin", true, true},
		{&SignalToMetricsDefinition{}, "test-origin", true, true},
		{&SpamFilter{}, "test-origin", true, true},
		{&SpamFilterV1Alpha2{}, "test-origin", true, true},
		{&NotificationChannelDefinition{}, "test-origin", true, false},
		{&TimeSeriesAggregationDefinition{}, "test-origin", true, false},
		{&DashboardDefinition{}, "", false, true},
		{&PersesDashboard{}, "", false, true},
		{&PrometheusAlertRule{}, "test-origin", false, true},
		{&PrometheusRules{}, "", false, true},
		{&RecordingRule{}, "", false, true},
		{&SyntheticCheckDefinition{}, "test-origin", false, true},
		{&ViewDefinition{}, "test-origin", false, true},
		{&DatasetDefinition{}, "test-origin", false, true},
		{&AwsMetricScrapeConfigDefinition{}, "test-origin", false, true},
	}
	for _, tt := range tests {
		if err := json.Unmarshal([]byte(assetMetadataDocument), tt.asset); err != nil {
			t.Errorf("%T: unmarshal: %v", tt.asset, err)
			continue
		}
		typeName := fmt.Sprintf("%T", tt.asset)
		assertEqual(t, typeName+" origin", tt.asset.GetOrigin(), tt.origin)
		assertEqual(t, typeName+" id", tt.asset.GetID(), "test-id")
		assertEqual(t, typeName+" name", tt.asset.GetName(), "test-name")
		byOrigin, byID := tt.asset.UpsertKeys()
		if byOrigin != tt.byOrigin || byID != tt.byID {
			t.Errorf("%T.UpsertKeys() = (%v, %v), want (%v, %v)", tt.asset, byOrigin, byID, tt.byOrigin, tt.byID)
		}
	}
}
