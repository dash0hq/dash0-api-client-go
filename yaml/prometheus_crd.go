package yaml

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	dash0 "github.com/dash0hq/dash0-api-client-go"
	sigsyaml "sigs.k8s.io/yaml"
)

// Round-trip-conversion annotation keys written here and read back by
// semantic_equal.go's removeDefaultAnnotationValues, which treats their
// default values (dash0-enabled: true, thresholds: 0) as semantically
// absent since this conversion omits them.
const (
	enabledAnnotationKey           = "dash0-enabled"
	thresholdCriticalAnnotationKey = "dash0-threshold-critical"
	thresholdDegradedAnnotationKey = "dash0-threshold-degraded"
)

// MergeAnnotations merges a PrometheusRule document's top-level
// metadata.annotations into a rule's own annotations.
// Rule-level annotations win on key conflict, mirroring the Dash0 Operator's
// behavior (dash0-operator/internal/controller/prometheus_rules_controller.go,
// mergeAnnotations).
// Nil-safe: either or both inputs may be nil.
//
// UnmarshalPrometheusRule and ParseAsPrometheusAlertRules already apply this
// on the write path, so callers converting a document for the API do not need
// to. It is exported for downstream IaC tools that must model the same merge
// elsewhere: the Terraform provider, for example, compares a user's config
// against an API response that already reflects the merge, and so has to
// apply it to its comparison copy. Sharing the precedence rule keeps those
// consumers from drifting from the client.
func MergeAnnotations(metadataAnnotations, ruleAnnotations map[string]string) map[string]string {
	if len(metadataAnnotations) == 0 {
		return ruleAnnotations
	}

	merged := make(map[string]string, len(metadataAnnotations)+len(ruleAnnotations))
	for k, v := range metadataAnnotations {
		merged[k] = v
	}
	// Rule-level annotations are copied last, so they win on key conflict.
	for k, v := range ruleAnnotations {
		merged[k] = v
	}
	return merged
}

// UnmarshalPrometheusRule converts a Prometheus rule YAML document to a
// PrometheusAlertRule (Dash0 API format).
// The YAML must contain exactly one group with one rule.
// The check rule name is composed as "groupName - alertName".
// The document's top-level metadata.annotations are merged into the rule's
// own annotations before conversion, with rule-level annotations winning on
// key conflict.
func UnmarshalPrometheusRule(data []byte) (*dash0.PrometheusAlertRule, error) {
	var wire prometheusRulesWire
	if err := sigsyaml.Unmarshal(data, &wire); err != nil {
		return nil, fmt.Errorf("error parsing Prometheus rule YAML: %w", err)
	}
	promRules := wire.toPrometheusRules()

	if len(promRules.Spec.Groups) != 1 {
		return nil, fmt.Errorf("currently only one group is supported")
	}
	group := promRules.Spec.Groups[0]

	if len(group.Rules) != 1 {
		return nil, fmt.Errorf("currently only one rule per group is supported")
	}

	group.Rules[0].Annotations = MergeAnnotations(promRules.Metadata.Annotations, group.Rules[0].Annotations)

	ruleID := dash0.GetPrometheusRuleID(promRules)
	alertRule, err := dash0.ConvertPrometheusRuleToPrometheusAlertRule(&group.Rules[0], group.Interval, ruleID)
	if err != nil {
		return nil, err
	}

	// Override name to "groupName - alertName" format.
	alertRule.Name = fmt.Sprintf("%s - %s", group.Name, group.Rules[0].Alert)

	// Extract dataset from CRD metadata labels.
	if ds := promRules.Metadata.Labels[dash0.LabelDataset]; ds != "" {
		alertRule.Dataset = &ds
	}

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
		annotations[enabledAnnotationKey] = strconv.FormatBool(false)
	}

	// Only include threshold annotations for non-zero values
	if rule.Thresholds != nil {
		if rule.Thresholds.Failed != nil && *rule.Thresholds.Failed != 0 {
			annotations[thresholdCriticalAnnotationKey] = strconv.FormatFloat(*rule.Thresholds.Failed, 'f', -1, 64)
		}
		if rule.Thresholds.Degraded != nil && *rule.Thresholds.Degraded != 0 {
			annotations[thresholdDegradedAnnotationKey] = strconv.FormatFloat(*rule.Thresholds.Degraded, 'f', -1, 64)
		}
	}

	var labels map[string]string
	if rule.Labels != nil {
		labels = *rule.Labels
	}

	promRule := dash0.PrometheusRule{
		Alert:       alertName,
		Expr:        rule.Expression,
		Annotations: annotations,
		Labels:      labels,
	}

	// Convert *Duration (string) to time.Duration
	if rule.For != nil {
		d, err := time.ParseDuration(*rule.For)
		if err != nil {
			return nil, fmt.Errorf("invalid \"for\" duration %q: %w", *rule.For, err)
		}
		promRule.For = d
	}
	if rule.KeepFiringFor != nil {
		d, err := time.ParseDuration(*rule.KeepFiringFor)
		if err != nil {
			return nil, fmt.Errorf("invalid \"keep_firing_for\" duration %q: %w", *rule.KeepFiringFor, err)
		}
		promRule.KeepFiringFor = d
	}

	var groupInterval time.Duration
	if rule.Interval != nil {
		d, err := time.ParseDuration(*rule.Interval)
		if err != nil {
			return nil, fmt.Errorf("invalid \"interval\" duration %q: %w", *rule.Interval, err)
		}
		groupInterval = d
	}

	promRules := &dash0.PrometheusRules{
		APIVersion: "monitoring.coreos.com/v1",
		Kind:       "PrometheusRule",
		Metadata:   dash0.PrometheusRulesMetadata{},
		Spec: dash0.PrometheusRulesSpec{
			Groups: []dash0.PrometheusRulesGroup{
				{
					Name:     groupName,
					Interval: groupInterval,
					Rules:    []dash0.PrometheusRule{promRule},
				},
			},
		},
	}

	// Marshal via the wire type so durations serialize as strings.
	yamlBytes, err := sigsyaml.Marshal(fromPrometheusRules(promRules))
	if err != nil {
		return nil, fmt.Errorf("error marshaling Prometheus rules to YAML: %w", err)
	}

	return yamlBytes, nil
}

// ParseAsPrometheusAlertRules detects whether data is a CheckRule or
// PrometheusRule CRD, unmarshals it, and returns one or more normalized check
// rules ready for the API.
// A plain CheckRule returns a slice of length 1.
// A PrometheusRule CRD returns one entry per alerting rule (recording rules are
// skipped).
// For a PrometheusRule CRD, the document's top-level metadata.annotations are
// merged into each rule's own annotations before conversion, with rule-level
// annotations winning on key conflict.
func ParseAsPrometheusAlertRules(data []byte) ([]*dash0.PrometheusAlertRule, error) {
	detectedKind, err := DetectKind(data)
	if err != nil {
		return nil, err
	}
	kind := strings.ToLower(detectedKind)
	if kind == "prometheusrule" {
		var wire prometheusRulesWire
		if err := sigsyaml.Unmarshal(data, &wire); err != nil {
			return nil, fmt.Errorf("failed to parse PrometheusRule definition: %w", err)
		}
		promRules := wire.toPrometheusRules()

		ruleID := dash0.GetPrometheusRuleID(promRules)
		var dataset *string
		if ds := promRules.Metadata.Labels[dash0.LabelDataset]; ds != "" {
			dataset = &ds
		}

		var rules []*dash0.PrometheusAlertRule
		for _, group := range promRules.Spec.Groups {
			for i := range group.Rules {
				rule := &group.Rules[i]
				if rule.Alert == "" {
					continue
				}
				rule.Annotations = MergeAnnotations(promRules.Metadata.Annotations, rule.Annotations)
				checkRule, err := dash0.ConvertPrometheusRuleToPrometheusAlertRule(rule, group.Interval, ruleID)
				if err != nil {
					return nil, fmt.Errorf("failed to convert rule %q: %w", rule.Alert, err)
				}
				checkRule.Dataset = dataset
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
