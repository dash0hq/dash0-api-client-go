package yaml

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	dash0 "github.com/dash0hq/dash0-api-client-go"
	sigsyaml "sigs.k8s.io/yaml"
)

// GetPrometheusRuleID extracts the ID from a PrometheusRules definition.
func GetPrometheusRuleID(rule *PrometheusRules) string {
	if rule.Metadata.Labels != nil {
		return rule.Metadata.Labels["dash0.com/id"]
	}
	return ""
}

// GetPrometheusRuleName extracts the name from a PrometheusRules definition.
func GetPrometheusRuleName(rule *PrometheusRules) string {
	return rule.Metadata.Name
}

// ClearPrometheusRuleID removes the dash0.com/id label from a PrometheusRules CRD.
func ClearPrometheusRuleID(rule *PrometheusRules) {
	if rule.Metadata.Labels != nil {
		delete(rule.Metadata.Labels, "dash0.com/id")
	}
}

// SetPrometheusRuleID sets the dash0.com/id label on a PrometheusRules CRD,
// initializing the labels map if needed.
func SetPrometheusRuleID(rule *PrometheusRules, id string) {
	if rule.Metadata.Labels == nil {
		rule.Metadata.Labels = map[string]string{}
	}
	rule.Metadata.Labels["dash0.com/id"] = id
}

// SetPrometheusRuleIDIfAbsent sets the dash0.com/id label on a PrometheusRules
// CRD only if it is not already set, initializing the labels map if needed.
func SetPrometheusRuleIDIfAbsent(rule *PrometheusRules, id string) {
	if rule.Metadata.Labels == nil {
		rule.Metadata.Labels = map[string]string{}
	}
	if _, ok := rule.Metadata.Labels["dash0.com/id"]; !ok {
		rule.Metadata.Labels["dash0.com/id"] = id
	}
}

// ConvertPrometheusRuleToPrometheusAlertRule converts a Prometheus alerting rule to a Dash0 CheckRule.
// It extracts Dash0-specific annotations (thresholds, enabled flag) from the
// rule annotations and maps them to dedicated fields on the returned rule.
func ConvertPrometheusRuleToPrometheusAlertRule(rule *PrometheusRule, groupInterval PromDuration, ruleID string) (*dash0.PrometheusAlertRule, error) {
	checkRule := &dash0.PrometheusAlertRule{
		Name:       rule.Alert,
		Expression: rule.Expr,
	}

	if ruleID != "" {
		checkRule.Id = &ruleID
	}

	if len(rule.Labels) > 0 {
		labels := copyMap(rule.Labels)
		checkRule.Labels = &labels
	}

	// Copy annotations before mutating (threshold/enabled extraction removes keys).
	annotations := copyMap(rule.Annotations)

	if forDur := time.Duration(rule.For); forDur != 0 {
		s := dash0.Duration(FormatDuration(forDur))
		checkRule.For = &s
	}

	if keepFiring := time.Duration(rule.KeepFiringFor); keepFiring != 0 {
		s := dash0.Duration(FormatDuration(keepFiring))
		checkRule.KeepFiringFor = &s
	}

	if interval := time.Duration(groupInterval); interval != 0 {
		s := dash0.Duration(FormatDuration(interval))
		checkRule.Interval = &s
	}

	// Extract Dash0-specific annotations into dedicated fields.
	thresholds, err := extractThresholdsFromAnnotations(annotations)
	if err != nil {
		return nil, err
	}
	checkRule.Thresholds = thresholds

	enabled, err := extractEnabledFromAnnotations(annotations)
	if err != nil {
		return nil, err
	}
	checkRule.Enabled = enabled

	// Build the typed Annotations struct from remaining annotation map.
	var ann dash0.PrometheusAlertRule_Annotations
	hasAnnotations := false
	if summary, ok := annotations["summary"]; ok {
		ann.Summary = &summary
		delete(annotations, "summary")
		hasAnnotations = true
	}
	if description, ok := annotations["description"]; ok {
		ann.Description = &description
		delete(annotations, "description")
		hasAnnotations = true
	}
	if len(annotations) > 0 {
		ann.AdditionalProperties = annotations
		hasAnnotations = true
	}
	if hasAnnotations {
		checkRule.Annotations = &ann
	}

	return checkRule, nil
}

// UnmarshalPrometheusRule converts a Prometheus rule YAML document to a
// PrometheusAlertRule (Dash0 API format). The YAML must contain exactly one
// group with one rule. The dataset is set on the returned rule. The check rule
// name is composed as "groupName - alertName".
func UnmarshalPrometheusRule(data []byte, dataset string) (*dash0.PrometheusAlertRule, error) {
	var promRules PrometheusRules
	if err := sigsyaml.Unmarshal(data, &promRules); err != nil {
		return nil, fmt.Errorf("error parsing Prometheus rule YAML: %w", err)
	}

	if len(promRules.Spec.Groups) != 1 {
		return nil, fmt.Errorf("currently only one group is supported")
	}
	group := promRules.Spec.Groups[0]

	if len(group.Rules) != 1 {
		return nil, fmt.Errorf("currently only one rule per group is supported")
	}

	alertRule, err := ConvertPrometheusRuleToPrometheusAlertRule(&group.Rules[0], group.Interval, "")
	if err != nil {
		return nil, err
	}

	// Override name to "groupName - alertName" format and set dataset.
	alertRule.Name = fmt.Sprintf("%s - %s", group.Name, group.Rules[0].Alert)
	alertRule.Dataset = &dataset

	return alertRule, nil
}

// MarshalPrometheusRule converts a PrometheusAlertRule (Dash0 API format)
// back to a Prometheus rule YAML document.
func MarshalPrometheusRule(rule *dash0.PrometheusAlertRule) ([]byte, error) {
	nameParts := strings.SplitN(rule.Name, " - ", 2)
	var groupName, alertName string
	if len(nameParts) == 2 {
		groupName = nameParts[0]
		alertName = nameParts[1]
	} else {
		groupName = rule.Name
		alertName = rule.Name
	}

	// Build annotations map from the typed Annotations struct.
	annotations := make(map[string]string)
	if rule.Annotations != nil {
		if rule.Annotations.Summary != nil && *rule.Annotations.Summary != "" {
			annotations["summary"] = *rule.Annotations.Summary
		}
		if rule.Annotations.Description != nil && *rule.Annotations.Description != "" {
			annotations["description"] = *rule.Annotations.Description
		}
		for k, v := range rule.Annotations.AdditionalProperties {
			annotations[k] = v
		}
	}

	// Only add enabled annotation if false (true is the default)
	if rule.Enabled != nil && !*rule.Enabled {
		annotations["dash0-enabled"] = strconv.FormatBool(false)
	}

	// Only include threshold annotations for non-zero values
	if rule.Thresholds != nil {
		if rule.Thresholds.Failed != nil && *rule.Thresholds.Failed != 0 {
			annotations["dash0-threshold-critical"] = strconv.FormatFloat(*rule.Thresholds.Failed, 'f', -1, 64)
		}
		if rule.Thresholds.Degraded != nil && *rule.Thresholds.Degraded != 0 {
			annotations["dash0-threshold-degraded"] = strconv.FormatFloat(*rule.Thresholds.Degraded, 'f', -1, 64)
		}
	}

	var labels map[string]string
	if rule.Labels != nil {
		labels = *rule.Labels
	}

	promRule := PrometheusRule{
		Alert:       alertName,
		Expr:        rule.Expression,
		Annotations: annotations,
		Labels:      labels,
	}

	// Convert *Duration (string) to PromDuration
	if rule.For != nil {
		d, err := time.ParseDuration(*rule.For)
		if err == nil {
			promRule.For = PromDuration(d)
		}
	}
	if rule.KeepFiringFor != nil {
		d, err := time.ParseDuration(*rule.KeepFiringFor)
		if err == nil {
			promRule.KeepFiringFor = PromDuration(d)
		}
	}

	var groupInterval PromDuration
	if rule.Interval != nil {
		d, err := time.ParseDuration(*rule.Interval)
		if err == nil {
			groupInterval = PromDuration(d)
		}
	}

	promRules := &PrometheusRules{
		APIVersion: "monitoring.coreos.com/v1",
		Kind:       "PrometheusRule",
		Metadata:   PrometheusRulesMetadata{},
		Spec: PrometheusRulesSpec{
			Groups: []PrometheusRulesGroup{
				{
					Name:     groupName,
					Interval: groupInterval,
					Rules:    []PrometheusRule{promRule},
				},
			},
		},
	}

	yamlBytes, err := sigsyaml.Marshal(promRules)
	if err != nil {
		return nil, fmt.Errorf("error marshaling Prometheus rules to YAML: %w", err)
	}

	return yamlBytes, nil
}

// ParseAsPrometheusAlertRules detects whether data is a CheckRule or PrometheusRule
// CRD, unmarshals it, and returns one or more normalized check rules ready for
// the API. A plain CheckRule returns a slice of length 1. A PrometheusRule CRD
// returns one entry per alerting rule (recording rules are skipped).
func ParseAsPrometheusAlertRules(data []byte) ([]*dash0.PrometheusAlertRule, error) {
	detectedKind, err := DetectKind(data)
	if err != nil {
		return nil, err
	}
	kind := strings.ToLower(detectedKind)
	if kind == "prometheusrule" {
		var promRule PrometheusRules
		if err := sigsyaml.Unmarshal(data, &promRule); err != nil {
			return nil, fmt.Errorf("failed to parse PrometheusRule definition: %w", err)
		}

		ruleID := GetPrometheusRuleID(&promRule)
		var rules []*dash0.PrometheusAlertRule
		for _, group := range promRule.Spec.Groups {
			for i := range group.Rules {
				rule := &group.Rules[i]
				if rule.Alert == "" {
					continue
				}
				checkRule, err := ConvertPrometheusRuleToPrometheusAlertRule(rule, group.Interval, ruleID)
				if err != nil {
					return nil, fmt.Errorf("failed to convert rule %q: %w", rule.Alert, err)
				}
				rules = append(rules, checkRule)
			}
		}
		if len(rules) == 0 {
			return nil, fmt.Errorf("no alerting rules found in PrometheusRule (recording rules are not supported)")
		}
		return rules, nil
	}

	var rule dash0.PrometheusAlertRule
	if err := sigsyaml.Unmarshal(data, &rule); err != nil {
		return nil, fmt.Errorf("failed to parse check rule definition: %w", err)
	}
	return []*dash0.PrometheusAlertRule{&rule}, nil
}

// extractThresholdsFromAnnotations extracts dash0-threshold-critical and
// dash0-threshold-degraded from annotations, removing them from the map.
// Returns nil if no thresholds are present.
func extractThresholdsFromAnnotations(annotations map[string]string) (*dash0.CheckThresholds, error) {
	if annotations == nil {
		return nil, nil
	}
	var thresholds dash0.CheckThresholds
	hasThresholds := false
	if critStr, ok := annotations["dash0-threshold-critical"]; ok {
		critVal, err := strconv.ParseFloat(critStr, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid value for dash0-threshold-critical: %w", err)
		}
		thresholds.Failed = &critVal
		hasThresholds = true
		delete(annotations, "dash0-threshold-critical")
	}
	if degStr, ok := annotations["dash0-threshold-degraded"]; ok {
		degVal, err := strconv.ParseFloat(degStr, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid value for dash0-threshold-degraded: %w", err)
		}
		thresholds.Degraded = &degVal
		hasThresholds = true
		delete(annotations, "dash0-threshold-degraded")
	}
	if !hasThresholds {
		return nil, nil
	}
	return &thresholds, nil
}

// extractEnabledFromAnnotations extracts the dash0-enabled flag from
// annotations, removing it from the map. Defaults to true if not present.
func extractEnabledFromAnnotations(annotations map[string]string) (*bool, error) {
	if annotations != nil {
		if enabledStr, ok := annotations["dash0-enabled"]; ok {
			enabledBool, err := strconv.ParseBool(enabledStr)
			if err != nil {
				return nil, fmt.Errorf("invalid value for dash0-enabled: %w", err)
			}
			delete(annotations, "dash0-enabled")
			return &enabledBool, nil
		}
	}
	enabled := true
	return &enabled, nil
}

// copyMap returns a shallow copy of m. If m is nil, it returns nil.
func copyMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	cp := make(map[string]string, len(m))
	for k, v := range m {
		cp[k] = v
	}
	return cp
}
