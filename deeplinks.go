package dash0

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// DeeplinkAssetType identifies a Dash0 asset type for which a web app deep link
// can be built.
type DeeplinkAssetType string

// Supported deep link asset types.
const (
	DeeplinkAssetTypeDashboard           DeeplinkAssetType = "dashboard"
	DeeplinkAssetTypeCheckRule           DeeplinkAssetType = "check-rule"
	DeeplinkAssetTypeSyntheticCheck      DeeplinkAssetType = "synthetic-check"
	DeeplinkAssetTypeView                DeeplinkAssetType = "view"
	DeeplinkAssetTypeTeam                DeeplinkAssetType = "team"
	DeeplinkAssetTypeMember              DeeplinkAssetType = "member"
	DeeplinkAssetTypeNotificationChannel DeeplinkAssetType = "notification-channel"
)

// Deep link path patterns per asset type.
const (
	deeplinkPathDashboard           = "/goto/dashboards"
	deeplinkPathCheckRule           = "/goto/alerting/check-rules"
	deeplinkPathSyntheticCheck      = "/goto/alerting/synthetics"
	deeplinkPathView                = "/goto/logs"
	deeplinkPathTeam                = "/goto/settings/teams"
	deeplinkPathMember              = "/goto/settings/members"
	deeplinkPathNotificationChannel = "/goto/settings/notifications"

	// View-type-specific deep link paths.
	deeplinkPathViewLogs         = "/goto/logs"
	deeplinkPathViewTracing      = "/goto/traces/explorer"
	deeplinkPathViewMetrics      = "/goto/metrics/explorer"
	deeplinkPathViewServiceMap   = "/goto/services/map"
	deeplinkPathViewResources    = "/goto/resources/table"
	deeplinkPathViewFailedChecks = "/goto/alerting/failed-checks"
	deeplinkPathViewWebEvents    = "/goto/web-events/explorer"

	// Query parameter names per asset type.
	deeplinkQueryDashboard           = "dashboard_id"
	deeplinkQueryCheckRule           = "check_rule_id"
	deeplinkQuerySyntheticCheck      = "check_id"
	deeplinkQueryView                = "view_id"
	deeplinkQueryTeam                = "team_id"
	deeplinkQueryMember              = "member_id"
	deeplinkQueryNotificationChannel = "channel_id"
)

// AppBaseURL derives the Dash0 web app base URL from an API URL.
//
// It takes the registrable domain (the last two labels) of the API host and
// prefixes "app.". For example "https://api.us-west-2.aws.dash0.com" yields
// "https://app.dash0.com" and "https://api.dash0-dev.com" yields
// "https://app.dash0-dev.com". It returns an empty string if the API URL is
// empty or cannot be parsed into a host with at least two labels.
func AppBaseURL(apiURL string) string {
	if apiURL == "" {
		return ""
	}

	parsed, err := url.Parse(apiURL)
	if err != nil || parsed.Hostname() == "" {
		return ""
	}

	suffix := domainSuffix(parsed.Hostname())
	if suffix == "" {
		return ""
	}

	return fmt.Sprintf("https://app.%s", suffix)
}

// domainSuffix extracts the last two labels from a hostname.
// For example, "api.us-west-2.aws.dash0.com" returns "dash0.com".
func domainSuffix(hostname string) string {
	parts := strings.Split(hostname, ".")
	if len(parts) < 2 {
		return ""
	}
	return strings.Join(parts[len(parts)-2:], ".")
}

// DeeplinkURL constructs a Dash0 web app deep link for the given asset type and
// ID, derived from the API URL (see [AppBaseURL]).
//
// When dataset is non-nil and non-empty, a `dataset=<dataset>` query parameter
// is appended so the web app opens the page scoped to that dataset. Org-level
// assets (team, member, notification channel) do not live in a dataset; pass
// nil for those.
//
// It returns an empty string if the API URL cannot be parsed or the asset type
// is not supported. For views, prefer [ViewDeeplinkURL], which selects the
// correct page based on the view type.
func DeeplinkURL(apiURL string, assetType DeeplinkAssetType, assetID string, dataset *string) string {
	baseURL := AppBaseURL(apiURL)
	if baseURL == "" {
		return ""
	}

	path, queryParam := deeplinkPathAndQuery(assetType)
	if path == "" {
		return ""
	}

	params := url.Values{}
	params.Set(queryParam, assetID)
	if dataset != nil && *dataset != "" {
		params.Set("dataset", *dataset)
	}
	return fmt.Sprintf("%s%s?%s", baseURL, path, params.Encode())
}

// ViewDeeplinkURL constructs a Dash0 web app deep link for a view, using the
// view's type to select the correct page (for example "/goto/traces/explorer"
// for span views, "/goto/logs" for log views).
//
// When dataset is non-nil and non-empty, a `dataset=<dataset>` query parameter
// is appended so the web app opens the view scoped to that dataset.
//
// It returns an empty string if the API URL cannot be parsed or the view type
// has no associated page.
func ViewDeeplinkURL(apiURL string, viewType ViewType, viewID string, dataset *string) string {
	baseURL := AppBaseURL(apiURL)
	if baseURL == "" {
		return ""
	}

	path := viewTypePath(viewType)
	if path == "" {
		return ""
	}

	params := url.Values{}
	params.Set(deeplinkQueryView, viewID)
	if dataset != nil && *dataset != "" {
		params.Set("dataset", *dataset)
	}
	return fmt.Sprintf("%s%s?%s", baseURL, path, params.Encode())
}

// viewTypePath maps a view type to the corresponding deep link path, or an
// empty string when the view type has no dedicated page.
func viewTypePath(viewType ViewType) string {
	switch viewType {
	case Logs:
		return deeplinkPathViewLogs
	case Spans:
		return deeplinkPathViewTracing
	case Metrics:
		return deeplinkPathViewMetrics
	case Services:
		return deeplinkPathViewServiceMap
	case Resources:
		return deeplinkPathViewResources
	case FailedChecks:
		return deeplinkPathViewFailedChecks
	case WebEvents:
		return deeplinkPathViewWebEvents
	default:
		return ""
	}
}

// deeplinkPathAndQuery returns the URL path and query parameter name for a
// given asset type.
func deeplinkPathAndQuery(assetType DeeplinkAssetType) (string, string) {
	switch assetType {
	case DeeplinkAssetTypeDashboard:
		return deeplinkPathDashboard, deeplinkQueryDashboard
	case DeeplinkAssetTypeCheckRule:
		return deeplinkPathCheckRule, deeplinkQueryCheckRule
	case DeeplinkAssetTypeSyntheticCheck:
		return deeplinkPathSyntheticCheck, deeplinkQuerySyntheticCheck
	case DeeplinkAssetTypeView:
		return deeplinkPathView, deeplinkQueryView
	case DeeplinkAssetTypeTeam:
		return deeplinkPathTeam, deeplinkQueryTeam
	case DeeplinkAssetTypeMember:
		return deeplinkPathMember, deeplinkQueryMember
	case DeeplinkAssetTypeNotificationChannel:
		return deeplinkPathNotificationChannel, deeplinkQueryNotificationChannel
	default:
		return "", ""
	}
}

// DeeplinkFilter represents a single filter criterion for explorer deep links.
type DeeplinkFilter struct {
	Key      string `json:"key"`
	Operator string `json:"operator"`
	Value    string `json:"value,omitempty"`
}

// FiltersToDeeplinkFilters converts parsed API filter criteria to deep link
// filter objects suitable for URL serialization. It returns nil when filters
// is nil.
func FiltersToDeeplinkFilters(filters *FilterCriteria) []DeeplinkFilter {
	if filters == nil {
		return nil
	}

	result := make([]DeeplinkFilter, 0, len(*filters))
	for _, f := range *filters {
		df := DeeplinkFilter{
			Key:      string(f.Key),
			Operator: string(f.Operator),
		}

		switch {
		case f.Values != nil:
			// Multi-value operators (is_one_of, is_not_one_of): join values with space.
			var parts []string
			for _, item := range *f.Values {
				if sv, err := item.AsAttributeFilterStringValue(); err == nil {
					parts = append(parts, sv)
				}
			}
			df.Value = strings.Join(parts, " ")
		case f.Value != nil:
			// Single-value operators.
			if sv, err := f.Value.AsAttributeFilterStringValue(); err == nil {
				df.Value = sv
			}
		}
		// No-value operators (is_set, is_not_set): Value stays empty.

		result = append(result, df)
	}
	return result
}

// LogsExplorerURL builds a deep link to the Dash0 logs explorer.
// The URL includes filter criteria, time range, and optional dataset as query parameters.
// It returns an empty string if the API URL is empty or cannot be parsed.
func LogsExplorerURL(apiURL string, filters []DeeplinkFilter, from, to string, dataset *string) string {
	return explorerURL(apiURL, deeplinkPathViewLogs, filters, from, to, dataset)
}

// SpansExplorerURL builds a deep link to the Dash0 traces explorer.
// The URL includes optional filter criteria, time range, and optional dataset as query parameters.
// It returns an empty string if the API URL is empty or cannot be parsed.
func SpansExplorerURL(apiURL string, filters []DeeplinkFilter, from, to string, dataset *string) string {
	return explorerURL(apiURL, deeplinkPathViewTracing, filters, from, to, dataset)
}

// FailedChecksExplorerURL builds a deep link to the Dash0 alerting failed checks view.
// The URL includes optional filter criteria, time range, and optional dataset as query parameters.
// It returns an empty string if the API URL is empty or cannot be parsed.
func FailedChecksExplorerURL(apiURL string, filters []DeeplinkFilter, from, to string, dataset *string) string {
	return explorerURL(apiURL, deeplinkPathViewFailedChecks, filters, from, to, dataset)
}

// TracesExplorerURL builds a deep link to the Dash0 traces explorer for a
// specific trace. The URL includes the trace ID and optional dataset as query
// parameters.
// It returns an empty string if the API URL is empty or cannot be parsed.
func TracesExplorerURL(apiURL, traceID string, dataset *string) string {
	baseURL := AppBaseURL(apiURL)
	if baseURL == "" {
		return ""
	}

	params := url.Values{}
	if dataset != nil && *dataset != "" {
		params.Set("dataset", *dataset)
	}
	params.Set("trace_id", traceID)

	return fmt.Sprintf("%s%s?%s", baseURL, deeplinkPathViewTracing, params.Encode())
}

// explorerURL builds a deep link to a Dash0 explorer page.
func explorerURL(apiURL, path string, filters []DeeplinkFilter, from, to string, dataset *string) string {
	baseURL := AppBaseURL(apiURL)
	if baseURL == "" {
		return ""
	}

	params := explorerParams(filters, from, to, dataset)
	return fmt.Sprintf("%s%s?%s", baseURL, path, params.Encode())
}

// explorerParams builds the common query parameters for explorer deep links.
func explorerParams(filters []DeeplinkFilter, from, to string, dataset *string) url.Values {
	params := url.Values{}
	if dataset != nil && *dataset != "" {
		params.Set("dataset", *dataset)
	}
	if len(filters) > 0 {
		filterJSON, err := json.Marshal(filters)
		if err == nil {
			params.Set("filter", string(filterJSON))
		}
	}
	params.Set("from", from)
	params.Set("to", to)
	return params
}
