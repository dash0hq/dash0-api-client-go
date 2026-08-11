package dash0

import (
	"fmt"
	"maps"
	"strconv"
	"strings"
	"time"
)

// Recognized threshold-annotation keys.
// These mirror the Dash0 Operator's constants
// (dash0-operator/internal/controller/prometheus_rules_controller.go:76-84):
// the "dash0-" prefixed names are current, the unprefixed names are legacy
// aliases kept for a grace period.
const (
	thresholdReference                = "$__threshold"
	thresholdDegradedAnnotation       = "dash0-threshold-degraded"
	thresholdDegradedAnnotationLegacy = "threshold-degraded"
	thresholdCriticalAnnotation       = "dash0-threshold-critical"
	thresholdCriticalAnnotationLegacy = "threshold-critical"
)

// GetPrometheusRuleDataset extracts the dataset from a PrometheusRules definition.
func GetPrometheusRuleDataset(rule *PrometheusRules) string {
	if rule == nil || rule.Metadata.Labels == nil {
		return ""
	}
	return rule.Metadata.Labels[LabelDataset]
}

// SetPrometheusRuleDataset sets the dash0.com/dataset label on a PrometheusRules
// CRD, initializing the labels map if needed.
func SetPrometheusRuleDataset(rule *PrometheusRules, dataset string) {
	if rule == nil {
		return
	}
	if rule.Metadata.Labels == nil {
		rule.Metadata.Labels = map[string]string{}
	}
	rule.Metadata.Labels[LabelDataset] = dataset
}

// GetPrometheusRuleID extracts the ID from a PrometheusRules definition.
func GetPrometheusRuleID(rule *PrometheusRules) string {
	if rule == nil || rule.Metadata.Labels == nil {
		return ""
	}
	return rule.Metadata.Labels[LabelID]
}

// GetPrometheusRuleName extracts the name from a PrometheusRules definition.
func GetPrometheusRuleName(rule *PrometheusRules) string {
	if rule == nil {
		return ""
	}
	return rule.Metadata.Name
}

// ClearPrometheusRuleID removes the dash0.com/id label from a PrometheusRules CRD.
func ClearPrometheusRuleID(rule *PrometheusRules) {
	if rule == nil {
		return
	}
	if rule.Metadata.Labels != nil {
		delete(rule.Metadata.Labels, LabelID)
	}
}

// SetPrometheusRuleID sets the dash0.com/id label on a PrometheusRules CRD,
// initializing the labels map if needed.
func SetPrometheusRuleID(rule *PrometheusRules, id string) {
	if rule == nil {
		return
	}
	if rule.Metadata.Labels == nil {
		rule.Metadata.Labels = map[string]string{}
	}
	rule.Metadata.Labels[LabelID] = id
}

// SetPrometheusRuleIDIfAbsent sets the dash0.com/id label on a PrometheusRules
// CRD only if it is not already set, initializing the labels map if needed.
func SetPrometheusRuleIDIfAbsent(rule *PrometheusRules, id string) {
	if rule == nil {
		return
	}
	if rule.Metadata.Labels == nil {
		rule.Metadata.Labels = map[string]string{}
	}
	if _, ok := rule.Metadata.Labels[LabelID]; !ok {
		rule.Metadata.Labels[LabelID] = id
	}
}

// ConvertPrometheusRuleToPrometheusAlertRule converts a Prometheus alerting rule
// to a Dash0 CheckRule.
// It extracts Dash0-specific annotations (thresholds, enabled flag) from the
// rule annotations and maps them to dedicated fields on the returned rule.
// It returns an error if a threshold annotation value is non-numeric, or if
// the rule's expression references the "$__threshold" token but declares
// neither a current nor legacy threshold annotation, mirroring the Dash0
// Operator's validation.
func ConvertPrometheusRuleToPrometheusAlertRule(rule *PrometheusRule, groupInterval time.Duration, ruleID string) (*PrometheusAlertRule, error) {
	checkRule := &PrometheusAlertRule{
		Name:       rule.Alert,
		Expression: rule.Expr,
	}

	if ruleID != "" {
		checkRule.Id = &ruleID
	}

	if len(rule.Labels) > 0 {
		labels := maps.Clone(rule.Labels)
		checkRule.Labels = &labels
	}

	// Copy annotations before mutating (threshold/enabled extraction removes keys).
	annotations := maps.Clone(rule.Annotations)

	if err := validateThresholdAnnotationPresence(rule.Expr, annotations); err != nil {
		return nil, err
	}

	if rule.For != 0 {
		s := Duration(FormatDuration(rule.For))
		checkRule.For = &s
	}

	if rule.KeepFiringFor != 0 {
		s := Duration(FormatDuration(rule.KeepFiringFor))
		checkRule.KeepFiringFor = &s
	}

	if groupInterval != 0 {
		s := Duration(FormatDuration(groupInterval))
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
	var ann PrometheusAlertRule_Annotations
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

// validateThresholdAnnotationPresence returns an error if expr references the
// "$__threshold" token but annotations declares neither a current
// ("dash0-threshold-degraded"/"dash0-threshold-critical") nor legacy
// ("threshold-degraded"/"threshold-critical") threshold annotation.
// A rule in this state would silently create an unresolved/broken check rule
// on the platform, so it is rejected here at parse/conversion time instead.
// This mirrors the Dash0 Operator's validateThreshold check
// (dash0-operator/internal/controller/prometheus_rules_controller.go:1089-1152).
func validateThresholdAnnotationPresence(expr string, annotations map[string]string) error {
	if !strings.Contains(expr, thresholdReference) {
		return nil
	}

	_, hasDegraded := annotations[thresholdDegradedAnnotation]
	if !hasDegraded {
		_, hasDegraded = annotations[thresholdDegradedAnnotationLegacy]
	}
	_, hasCritical := annotations[thresholdCriticalAnnotation]
	if !hasCritical {
		_, hasCritical = annotations[thresholdCriticalAnnotationLegacy]
	}

	if !hasDegraded && !hasCritical {
		return fmt.Errorf(
			"the rule uses the token %s in its expression, but has neither the %s nor the %s annotation",
			thresholdReference, thresholdDegradedAnnotation, thresholdCriticalAnnotation,
		)
	}
	return nil
}

// extractThresholdsFromAnnotations extracts dash0-threshold-critical and
// dash0-threshold-degraded from annotations, removing them from the map.
// Returns nil if no thresholds are present.
// Callers must invoke [validateThresholdAnnotationPresence] first to reject
// rules whose expression references "$__threshold" without any threshold
// annotation at all; this function only validates the numeric format of
// values that are present.
func extractThresholdsFromAnnotations(annotations map[string]string) (*CheckThresholds, error) {
	if annotations == nil {
		return nil, nil
	}
	var thresholds CheckThresholds
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
// annotations, removing it from the map.
// Defaults to true if not present.
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
