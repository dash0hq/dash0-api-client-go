package dash0

import (
	"net/url"
	"testing"
)

const testBaseURL = "https://app.dash0.com"

func TestAppBaseURL(t *testing.T) {
	tests := []struct {
		name   string
		apiUrl string
		want   string
	}{
		{"eu-west", "https://api.eu-west-1.aws.dash0.com", "https://app.dash0.com"},
		{"us-west", "https://api.us-west-2.aws.dash0.com", "https://app.dash0.com"},
		{"dev", "https://api.eu-west-1.aws.dash0-dev.com", "https://app.dash0-dev.com"},
		{"empty", "", ""},
		{"single label", "https://localhost", ""},
		{"invalid URL", "not-a-url", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AppBaseURL(tt.apiUrl)
			if got != tt.want {
				t.Errorf("AppBaseURL(%q) = %q, want %q", tt.apiUrl, got, tt.want)
			}
		})
	}
}

func TestDeeplinkURL(t *testing.T) {
	tests := []struct {
		name      string
		baseURL   string
		assetType DeeplinkType
		assetID   string
		wantPath  string
		wantQuery url.Values
		wantEmpty bool
	}{
		{
			name:      "dashboard",
			baseURL:   testBaseURL,
			assetType: DeeplinkTypeDashboard,
			assetID:   "abc-123",
			wantPath:  "/goto/dashboards",
			wantQuery: url.Values{"dashboard_id": {"abc-123"}},
		},
		{
			name:      "check rule",
			baseURL:   testBaseURL,
			assetType: DeeplinkTypeCheckRule,
			assetID:   "rule-1",
			wantPath:  "/goto/alerting/check-rules",
			wantQuery: url.Values{"check_rule_id": {"rule-1"}},
		},
		{
			name:      "synthetic check",
			baseURL:   testBaseURL,
			assetType: DeeplinkTypeSyntheticCheck,
			assetID:   "check-1",
			wantPath:  "/goto/alerting/synthetics",
			wantQuery: url.Values{"check_id": {"check-1"}},
		},
		{
			name:      "team",
			baseURL:   testBaseURL,
			assetType: DeeplinkTypeTeam,
			assetID:   "team-1",
			wantPath:  "/goto/settings/teams",
			wantQuery: url.Values{"team_id": {"team-1"}},
		},
		{
			name:      "member",
			baseURL:   testBaseURL,
			assetType: DeeplinkTypeMember,
			assetID:   "member-1",
			wantPath:  "/goto/settings/members",
			wantQuery: url.Values{"member_id": {"member-1"}},
		},
		{
			name:      "ID with special characters",
			baseURL:   testBaseURL,
			assetType: DeeplinkTypeDashboard,
			assetID:   "id with spaces&more",
			wantPath:  "/goto/dashboards",
			wantQuery: url.Values{"dashboard_id": {"id with spaces&more"}},
		},
		{
			name:      "unknown asset type",
			baseURL:   testBaseURL,
			assetType: DeeplinkType("unknown"),
			assetID:   "abc",
			wantEmpty: true,
		},
		{
			name:      "empty base URL",
			baseURL:   "",
			assetType: DeeplinkTypeDashboard,
			assetID:   "abc",
			wantEmpty: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeeplinkURL(tt.baseURL, tt.assetType, tt.assetID)
			if tt.wantEmpty {
				if got != "" {
					t.Errorf("expected empty, got %q", got)
				}
				return
			}
			assertDeeplinkURL(t, got, testBaseURL, tt.wantPath, tt.wantQuery)
		})
	}
}

func TestDeeplinkURL_ViewTypes(t *testing.T) {
	tests := []struct {
		name         string
		deeplinkType DeeplinkType
		wantPath     string
	}{
		{"logs", DeeplinkTypeLogs, "/goto/logs"},
		{"spans", DeeplinkTypeSpans, "/goto/traces/explorer"},
		{"metrics", DeeplinkTypeMetrics, "/goto/metrics/explorer"},
		{"services", DeeplinkTypeServices, "/goto/services/map"},
		{"resources", DeeplinkTypeResources, "/goto/resources/table"},
		{"failed_checks", DeeplinkTypeFailedChecks, "/goto/alerting/failed-checks"},
		{"web_events", DeeplinkTypeWebEvents, "/goto/web-events/explorer"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeeplinkURL(testBaseURL, tt.deeplinkType, "v1")
			assertDeeplinkURL(t, got, testBaseURL, tt.wantPath, url.Values{"view_id": {"v1"}})
		})
	}
}


func TestFiltersToDeeplinkFilters(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		got := FiltersToDeeplinkFilters(nil)
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("single value operator", func(t *testing.T) {
		var val AttributeFilter_Value
		_ = val.FromAttributeFilterStringValue("my-service")
		filters := &FilterCriteria{
			{Key: "service.name", Operator: "is", Value: &val},
		}
		got := FiltersToDeeplinkFilters(filters)
		if len(got) != 1 {
			t.Fatalf("got %d filters, want 1", len(got))
		}
		if got[0].Key != "service.name" {
			t.Errorf("Key = %q, want %q", got[0].Key, "service.name")
		}
		if got[0].Operator != "is" {
			t.Errorf("Operator = %q, want %q", got[0].Operator, "is")
		}
		if got[0].Value != "my-service" {
			t.Errorf("Value = %q, want %q", got[0].Value, "my-service")
		}
	})

	t.Run("multi value operator", func(t *testing.T) {
		var item1, item2 AttributeFilter_Values_Item
		_ = item1.FromAttributeFilterStringValue("svc-a")
		_ = item2.FromAttributeFilterStringValue("svc-b")
		items := []AttributeFilter_Values_Item{item1, item2}
		filters := &FilterCriteria{
			{Key: "service.name", Operator: "is_one_of", Values: &items},
		}
		got := FiltersToDeeplinkFilters(filters)
		if len(got) != 1 {
			t.Fatalf("got %d filters, want 1", len(got))
		}
		if got[0].Value != "svc-a svc-b" {
			t.Errorf("Value = %q, want %q", got[0].Value, "svc-a svc-b")
		}
	})

	t.Run("no value operator", func(t *testing.T) {
		filters := &FilterCriteria{
			{Key: "error", Operator: "is_set"},
		}
		got := FiltersToDeeplinkFilters(filters)
		if len(got) != 1 {
			t.Fatalf("got %d filters, want 1", len(got))
		}
		if got[0].Value != "" {
			t.Errorf("Value = %q, want empty", got[0].Value)
		}
	})
}

func TestTracesExplorerURL(t *testing.T) {
	t.Run("with dataset", func(t *testing.T) {
		ds := "my-dataset"
		got := TracesExplorerURL(testBaseURL, "trace-abc", &ds)
		assertDeeplinkURL(t, got, testBaseURL, "/goto/traces/explorer", url.Values{
			"trace_id": {"trace-abc"},
			"dataset":  {"my-dataset"},
		})
	})

	t.Run("without dataset", func(t *testing.T) {
		got := TracesExplorerURL(testBaseURL, "trace-abc", nil)
		parsed := mustParseURL(t, got)
		if parsed.Query().Get("dataset") != "" {
			t.Error("expected no dataset parameter")
		}
		if parsed.Query().Get("trace_id") != "trace-abc" {
			t.Errorf("trace_id = %q, want %q", parsed.Query().Get("trace_id"), "trace-abc")
		}
	})

	t.Run("empty base URL", func(t *testing.T) {
		if got := TracesExplorerURL("", "trace-abc", nil); got != "" {
			t.Errorf("expected empty, got %q", got)
		}
	})
}

func TestLogsExplorerURL(t *testing.T) {
	t.Run("with filters", func(t *testing.T) {
		filters := []DeeplinkFilter{
			{Key: "service.name", Operator: "equals", Value: "my-svc"},
		}
		got := LogsExplorerURL(testBaseURL, filters, "now-1h", "now", nil)
		parsed := mustParseURL(t, got)
		assertURLBase(t, parsed, testBaseURL, "/goto/logs")
		assertQueryParam(t, parsed, "from", "now-1h")
		assertQueryParam(t, parsed, "to", "now")
		if parsed.Query().Get("filter") == "" {
			t.Error("expected filter parameter")
		}
	})

	t.Run("without filters", func(t *testing.T) {
		got := LogsExplorerURL(testBaseURL, nil, "now-1h", "now", nil)
		parsed := mustParseURL(t, got)
		assertQueryParam(t, parsed, "from", "now-1h")
		if parsed.Query().Get("filter") != "" {
			t.Error("expected no filter parameter")
		}
	})

	t.Run("with dataset", func(t *testing.T) {
		ds := "prod"
		got := LogsExplorerURL(testBaseURL, nil, "now-1h", "now", &ds)
		parsed := mustParseURL(t, got)
		assertQueryParam(t, parsed, "dataset", "prod")
	})
}

func TestSpansExplorerURL(t *testing.T) {
	got := SpansExplorerURL(testBaseURL, nil, "now-1h", "now", nil)
	assertDeeplinkURL(t, got, testBaseURL, "/goto/traces/explorer", url.Values{
		"from": {"now-1h"},
		"to":   {"now"},
	})
}

// --- URL assertion helpers ---

func mustParseURL(t *testing.T, rawURL string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("failed to parse URL %q: %v", rawURL, err)
	}
	return parsed
}

func assertURLBase(t *testing.T, parsed *url.URL, wantBase, wantPath string) {
	t.Helper()
	gotBase := parsed.Scheme + "://" + parsed.Host
	if gotBase != wantBase {
		t.Errorf("base = %q, want %q", gotBase, wantBase)
	}
	if parsed.Path != wantPath {
		t.Errorf("path = %q, want %q", parsed.Path, wantPath)
	}
}

func assertQueryParam(t *testing.T, parsed *url.URL, key, want string) {
	t.Helper()
	got := parsed.Query().Get(key)
	if got != want {
		t.Errorf("query param %q = %q, want %q", key, got, want)
	}
}

// assertDeeplinkURL parses the URL, verifies base+path, and checks that all
// expected query parameters are present with correct values.
func assertDeeplinkURL(t *testing.T, rawURL, wantBase, wantPath string, wantQuery url.Values) {
	t.Helper()
	parsed := mustParseURL(t, rawURL)
	assertURLBase(t, parsed, wantBase, wantPath)
	for key, wantValues := range wantQuery {
		got := parsed.Query().Get(key)
		if got != wantValues[0] {
			t.Errorf("query param %q = %q, want %q", key, got, wantValues[0])
		}
	}
}
