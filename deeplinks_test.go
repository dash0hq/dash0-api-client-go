package dash0

import (
	"net/url"
	"testing"
)

func parseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("url.Parse(%q) error: %v", raw, err)
	}
	return u
}

func TestAppBaseURL(t *testing.T) {
	tests := []struct {
		name   string
		apiURL string
		want   string
	}{
		{"regional aws prod", "https://api.us-west-2.aws.dash0.com", "https://app.dash0.com"},
		{"eu region", "https://api.eu-west-1.aws.dash0.com", "https://app.dash0.com"},
		{"dev", "https://api.dash0-dev.com", "https://app.dash0-dev.com"},
		{"custom domain", "https://api.custom.example.io", "https://app.example.io"},
		{"trailing path ignored", "https://api.us-west-2.aws.dash0.com/", "https://app.dash0.com"},
		{"empty", "", ""},
		{"single-label host", "https://localhost", ""},
		{"unparseable", "://nope", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AppBaseURL(tt.apiURL); got != tt.want {
				t.Errorf("AppBaseURL(%q) = %q, want %q", tt.apiURL, got, tt.want)
			}
		})
	}
}

func TestDeeplinkURL(t *testing.T) {
	const api = "https://api.us-west-2.aws.dash0.com"
	tests := []struct {
		name       string
		assetType  DeeplinkAssetType
		id         string
		wantPath   string
		queryParam string
	}{
		{"dashboard", DeeplinkAssetTypeDashboard, "abc-123", "/goto/dashboards", "dashboard_id"},
		{"check rule", DeeplinkAssetTypeCheckRule, "r1", "/goto/alerting/check-rules", "check_rule_id"},
		{"synthetic check", DeeplinkAssetTypeSyntheticCheck, "s1", "/goto/alerting/synthetics", "check_id"},
		{"view default", DeeplinkAssetTypeView, "v1", "/goto/logs", "view_id"},
		{"team", DeeplinkAssetTypeTeam, "t1", "/goto/settings/teams", "team_id"},
		{"member", DeeplinkAssetTypeMember, "m1", "/goto/settings/members", "member_id"},
		{"notification channel", DeeplinkAssetTypeNotificationChannel, "n1", "/goto/settings/notifications", "channel_id"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := parseURL(t, DeeplinkURL(api, tt.assetType, tt.id, nil))
			if u.Scheme != "https" {
				t.Errorf("scheme = %q, want https", u.Scheme)
			}
			if u.Host != "app.dash0.com" {
				t.Errorf("host = %q, want app.dash0.com", u.Host)
			}
			if u.Path != tt.wantPath {
				t.Errorf("path = %q, want %q", u.Path, tt.wantPath)
			}
			if got := u.Query().Get(tt.queryParam); got != tt.id {
				t.Errorf("query %s = %q, want %q", tt.queryParam, got, tt.id)
			}
			if _, ok := u.Query()["dataset"]; ok {
				t.Error("dataset query param should be absent when dataset is nil")
			}
		})
	}
}

func TestDeeplinkURL_WithDataset(t *testing.T) {
	const api = "https://api.us-west-2.aws.dash0.com"
	u := parseURL(t, DeeplinkURL(api, DeeplinkAssetTypeDashboard, "abc-123", Ptr("production")))
	if got := u.Query().Get("dashboard_id"); got != "abc-123" {
		t.Errorf("dashboard_id = %q, want abc-123", got)
	}
	if got := u.Query().Get("dataset"); got != "production" {
		t.Errorf("dataset = %q, want production", got)
	}

	// Empty dataset string is treated as absent.
	u = parseURL(t, DeeplinkURL(api, DeeplinkAssetTypeDashboard, "abc-123", Ptr("")))
	if _, ok := u.Query()["dataset"]; ok {
		t.Error("dataset query param should be absent when dataset is empty string")
	}
}

func TestDeeplinkURL_Empty(t *testing.T) {
	if got := DeeplinkURL("", DeeplinkAssetTypeDashboard, "x", nil); got != "" {
		t.Errorf("empty api url: got %q, want empty", got)
	}
	if got := DeeplinkURL("https://api.us-west-2.aws.dash0.com", DeeplinkAssetType("bogus"), "x", nil); got != "" {
		t.Errorf("unknown asset type: got %q, want empty", got)
	}
}

func TestDeeplinkURL_EscapesID(t *testing.T) {
	u := parseURL(t, DeeplinkURL("https://api.dash0.com", DeeplinkAssetTypeDashboard, "a b&c", nil))
	if got := u.Query().Get("dashboard_id"); got != "a b&c" {
		t.Errorf("dashboard_id = %q, want %q", got, "a b&c")
	}
}

func TestViewDeeplinkURL(t *testing.T) {
	const api = "https://api.eu-west-1.aws.dash0.com"
	tests := []struct {
		name     string
		viewType ViewType
		wantPath string
	}{
		{"logs", Logs, "/goto/logs"},
		{"spans", Spans, "/goto/traces/explorer"},
		{"metrics", Metrics, "/goto/metrics/explorer"},
		{"services", Services, "/goto/services/map"},
		{"resources", Resources, "/goto/resources/table"},
		{"failed checks", FailedChecks, "/goto/alerting/failed-checks"},
		{"web events", WebEvents, "/goto/web-events/explorer"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := parseURL(t, ViewDeeplinkURL(api, tt.viewType, "view-1", nil))
			if u.Host != "app.dash0.com" {
				t.Errorf("host = %q, want app.dash0.com", u.Host)
			}
			if u.Path != tt.wantPath {
				t.Errorf("path = %q, want %q", u.Path, tt.wantPath)
			}
			if got := u.Query().Get("view_id"); got != "view-1" {
				t.Errorf("view_id = %q, want view-1", got)
			}
			if _, ok := u.Query()["dataset"]; ok {
				t.Error("dataset query param should be absent when dataset is nil")
			}
		})
	}
}

func TestViewDeeplinkURL_WithDataset(t *testing.T) {
	u := parseURL(t, ViewDeeplinkURL("https://api.eu-west-1.aws.dash0.com", Spans, "view-1", Ptr("production")))
	if got := u.Query().Get("view_id"); got != "view-1" {
		t.Errorf("view_id = %q, want view-1", got)
	}
	if got := u.Query().Get("dataset"); got != "production" {
		t.Errorf("dataset = %q, want production", got)
	}
}

func TestViewDeeplinkURL_Empty(t *testing.T) {
	// View types without a dedicated page yield no deep link.
	for _, vt := range []ViewType{Sql, Profiles, AwsLambda, GcpPubsub} {
		if got := ViewDeeplinkURL("https://api.dash0.com", vt, "v1", nil); got != "" {
			t.Errorf("ViewDeeplinkURL(%q) = %q, want empty", vt, got)
		}
	}
	if got := ViewDeeplinkURL("", Logs, "v1", nil); got != "" {
		t.Errorf("empty api url: got %q, want empty", got)
	}
}

func TestFiltersToDeeplinkFilters_Nil(t *testing.T) {
	if got := FiltersToDeeplinkFilters(nil); got != nil {
		t.Errorf("FiltersToDeeplinkFilters(nil) = %v, want nil", got)
	}
}

func TestFiltersToDeeplinkFilters(t *testing.T) {
	var single AttributeFilter_Value
	if err := single.FromAttributeFilterStringValue("checkout"); err != nil {
		t.Fatalf("FromAttributeFilterStringValue: %v", err)
	}

	var first, second AttributeFilter_Values_Item
	if err := first.FromAttributeFilterStringValue("eu"); err != nil {
		t.Fatalf("FromAttributeFilterStringValue: %v", err)
	}
	if err := second.FromAttributeFilterStringValue("us"); err != nil {
		t.Fatalf("FromAttributeFilterStringValue: %v", err)
	}

	filters := FilterCriteria{
		{Key: "service.name", Operator: "is", Value: &single},
		{Key: "region", Operator: "is_one_of", Values: &[]AttributeFilter_Values_Item{first, second}},
		{Key: "error", Operator: "is_set"}, // no value
	}

	got := FiltersToDeeplinkFilters(&filters)
	if len(got) != 3 {
		t.Fatalf("len(got) = %d, want 3", len(got))
	}
	if got[0].Key != "service.name" || got[0].Operator != "is" || got[0].Value != "checkout" {
		t.Errorf("single-value filter = %+v", got[0])
	}
	if got[1].Value != "eu us" {
		t.Errorf("multi-value filter Value = %q, want %q", got[1].Value, "eu us")
	}
	if got[2].Value != "" {
		t.Errorf("no-value filter Value = %q, want empty", got[2].Value)
	}
}

func TestLogsExplorerURL(t *testing.T) {
	filters := []DeeplinkFilter{{Key: "service.name", Operator: "is", Value: "checkout"}}
	u := parseURL(t, LogsExplorerURL("https://api.us-west-2.aws.dash0.com", filters, "now-1h", "now", Ptr("production")))
	if u.Host != "app.dash0.com" {
		t.Errorf("host = %q, want app.dash0.com", u.Host)
	}
	if u.Path != "/goto/logs" {
		t.Errorf("path = %q, want /goto/logs", u.Path)
	}
	q := u.Query()
	if q.Get("from") != "now-1h" {
		t.Errorf("from = %q, want now-1h", q.Get("from"))
	}
	if q.Get("to") != "now" {
		t.Errorf("to = %q, want now", q.Get("to"))
	}
	if q.Get("dataset") != "production" {
		t.Errorf("dataset = %q, want production", q.Get("dataset"))
	}
	if f := q.Get("filter"); len(f) == 0 || f[0] != '[' {
		t.Errorf("filter = %q, want a JSON array", f)
	}
}

func TestFailedChecksExplorerURL(t *testing.T) {
	filters := []DeeplinkFilter{{Key: "priority", Operator: "is", Value: "p1"}}
	u := parseURL(t, FailedChecksExplorerURL("https://api.us-west-2.aws.dash0.com", filters, "now-1h", "now", Ptr("production")))
	if u.Host != "app.dash0.com" {
		t.Errorf("host = %q, want app.dash0.com", u.Host)
	}
	if u.Path != "/goto/alerting/failed-checks" {
		t.Errorf("path = %q, want /goto/alerting/failed-checks", u.Path)
	}
	q := u.Query()
	if q.Get("from") != "now-1h" {
		t.Errorf("from = %q, want now-1h", q.Get("from"))
	}
	if q.Get("to") != "now" {
		t.Errorf("to = %q, want now", q.Get("to"))
	}
	if q.Get("dataset") != "production" {
		t.Errorf("dataset = %q, want production", q.Get("dataset"))
	}
	if f := q.Get("filter"); len(f) == 0 || f[0] != '[' {
		t.Errorf("filter = %q, want a JSON array", f)
	}
}

func TestTracesExplorerURL(t *testing.T) {
	u := parseURL(t, TracesExplorerURL("https://api.dash0.com", "trace-abc", nil))
	if u.Path != "/goto/traces/explorer" {
		t.Errorf("path = %q, want /goto/traces/explorer", u.Path)
	}
	if got := u.Query().Get("trace_id"); got != "trace-abc" {
		t.Errorf("trace_id = %q, want trace-abc", got)
	}
	if _, ok := u.Query()["dataset"]; ok {
		t.Error("dataset query param should be absent when dataset is nil")
	}
}

func TestExplorerURL_EmptyAPIURL(t *testing.T) {
	if got := LogsExplorerURL("", nil, "now-1h", "now", nil); got != "" {
		t.Errorf("LogsExplorerURL empty api url = %q, want empty", got)
	}
	if got := SpansExplorerURL("", nil, "now-1h", "now", nil); got != "" {
		t.Errorf("SpansExplorerURL empty api url = %q, want empty", got)
	}
	if got := FailedChecksExplorerURL("", nil, "now-1h", "now", nil); got != "" {
		t.Errorf("FailedChecksExplorerURL empty api url = %q, want empty", got)
	}
	if got := TracesExplorerURL("", "t", nil); got != "" {
		t.Errorf("TracesExplorerURL empty api url = %q, want empty", got)
	}
}
