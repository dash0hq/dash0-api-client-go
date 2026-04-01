package dash0

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// Deeplink path patterns per asset type.
const (
	deeplinkPathDashboard        = "/goto/dashboards"
	deeplinkPathCheckRule        = "/goto/alerting/check-rules"
	deeplinkPathSyntheticCheck   = "/goto/alerting/synthetics"
	deeplinkPathTeam             = "/goto/settings/teams"
	deeplinkPathMember           = "/goto/settings/members"
	deeplinkPathViewLogs         = "/goto/logs"
	deeplinkPathViewTracing      = "/goto/traces/explorer"
	deeplinkPathViewMetrics      = "/goto/metrics/explorer"
	deeplinkPathViewServiceMap   = "/goto/services/map"
	deeplinkPathViewResources    = "/goto/resources/table"
	deeplinkPathViewFailedChecks = "/goto/alerting/failed-checks"
	deeplinkPathViewWebEvents    = "/goto/web-events/explorer"

	deeplinkQueryDashboard      = "dashboard_id"
	deeplinkQueryCheckRule      = "check_rule_id"
	deeplinkQuerySyntheticCheck = "check_id"
	deeplinkQueryView           = "view_id"
	deeplinkQueryTeam           = "team_id"
	deeplinkQueryMember         = "member_id"
)

// DeeplinkType identifies the type of Dash0 asset for deeplink URL construction.
type DeeplinkType string

// Supported deeplink types for DeeplinkURL.
const (
	DeeplinkTypeDashboard      DeeplinkType = "dashboard"
	DeeplinkTypeCheckRule      DeeplinkType = "checkrule"
	DeeplinkTypeSyntheticCheck DeeplinkType = "syntheticcheck"
	DeeplinkTypeTeam           DeeplinkType = "team"
	DeeplinkTypeMember         DeeplinkType = "member"
	DeeplinkTypeLogs           DeeplinkType = "logs"
	DeeplinkTypeSpans          DeeplinkType = "spans"
	DeeplinkTypeMetrics        DeeplinkType = "metrics"
	DeeplinkTypeServices       DeeplinkType = "services"
	DeeplinkTypeResources      DeeplinkType = "resources"
	DeeplinkTypeFailedChecks   DeeplinkType = "failed_checks"
	DeeplinkTypeWebEvents      DeeplinkType = "web_events"
)

// AppBaseURL derives the Dash0 app base URL from an API URL by extracting the
// domain suffix (e.g. "dash0.com" from "api.us-west-2.aws.dash0.com") and
// prepending "app.". Returns an empty string if the API URL is empty or cannot
// be parsed. Callers should call this once and pass the result to the deeplink
// functions.
func AppBaseURL(apiUrl string) string {
	if apiUrl == "" {
		return ""
	}

	parsed, err := url.Parse(apiUrl)
	if err != nil || parsed.Host == "" {
		return ""
	}

	parts := strings.Split(parsed.Hostname(), ".")
	if len(parts) < 2 {
		return ""
	}
	suffix := strings.Join(parts[len(parts)-2:], ".")

	return fmt.Sprintf("https://app.%s", suffix)
}

// DeeplinkURL constructs a deeplink URL for the given deeplink type and ID.
// The baseURL should be obtained from AppBaseURL.
func DeeplinkURL(baseURL string, deeplinkType DeeplinkType, id string) string {
	if baseURL == "" {
		return ""
	}

	path, queryParam := deeplinkPathAndQuery(deeplinkType)
	if path == "" {
		return ""
	}

	return fmt.Sprintf("%s%s?%s=%s", baseURL, path, queryParam, url.QueryEscape(id))
}

// DeeplinkFilter represents a single filter criterion for explorer deep links.
type DeeplinkFilter struct {
	Key      string `json:"key"`
	Operator string `json:"operator"`
	Value    string `json:"value,omitempty"`
}

// FiltersToDeeplinkFilters converts parsed API filter criteria to deep link
// filter objects suitable for URL serialization.
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

// LogsExplorerURL builds a deep link URL to the Dash0 logs explorer.
// The URL includes filter criteria, time range, and optional dataset as query parameters.
func LogsExplorerURL(baseURL string, filters []DeeplinkFilter, from, to string, dataset *string) string {
	return explorerURL(baseURL, deeplinkPathViewLogs, filters, from, to, dataset)
}

// SpansExplorerURL builds a deep link URL to the Dash0 traces explorer.
// The URL includes optional filter criteria, time range, and optional dataset as query parameters.
func SpansExplorerURL(baseURL string, filters []DeeplinkFilter, from, to string, dataset *string) string {
	return explorerURL(baseURL, deeplinkPathViewTracing, filters, from, to, dataset)
}

// TracesExplorerURL builds a deep link URL to the Dash0 traces explorer for a
// specific trace. The URL includes the trace ID and optional dataset as query
// parameters.
func TracesExplorerURL(baseURL, traceID string, dataset *string) string {
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

// explorerURL builds a deep link URL to a Dash0 explorer page.
func explorerURL(baseURL, path string, filters []DeeplinkFilter, from, to string, dataset *string) string {
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

// deeplinkPathAndQuery returns the URL path and query parameter name for a
// given deeplink type.
func deeplinkPathAndQuery(dt DeeplinkType) (string, string) {
	switch dt {
	case DeeplinkTypeDashboard:
		return deeplinkPathDashboard, deeplinkQueryDashboard
	case DeeplinkTypeCheckRule:
		return deeplinkPathCheckRule, deeplinkQueryCheckRule
	case DeeplinkTypeSyntheticCheck:
		return deeplinkPathSyntheticCheck, deeplinkQuerySyntheticCheck
	case DeeplinkTypeTeam:
		return deeplinkPathTeam, deeplinkQueryTeam
	case DeeplinkTypeMember:
		return deeplinkPathMember, deeplinkQueryMember
	case DeeplinkTypeLogs:
		return deeplinkPathViewLogs, deeplinkQueryView
	case DeeplinkTypeSpans:
		return deeplinkPathViewTracing, deeplinkQueryView
	case DeeplinkTypeMetrics:
		return deeplinkPathViewMetrics, deeplinkQueryView
	case DeeplinkTypeServices:
		return deeplinkPathViewServiceMap, deeplinkQueryView
	case DeeplinkTypeResources:
		return deeplinkPathViewResources, deeplinkQueryView
	case DeeplinkTypeFailedChecks:
		return deeplinkPathViewFailedChecks, deeplinkQueryView
	case DeeplinkTypeWebEvents:
		return deeplinkPathViewWebEvents, deeplinkQueryView
	default:
		return "", ""
	}
}
