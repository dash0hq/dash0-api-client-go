package dash0

import (
	"fmt"
	"strings"
	"time"
)

// PrometheusRules represents the Kubernetes Prometheus Operator PrometheusRule
// CRD document.
// This is the user-facing format used by the Terraform provider and Kubernetes
// operator.
type PrometheusRules struct {
	APIVersion string                  `json:"apiVersion"`
	Kind       string                  `json:"kind"`
	Metadata   PrometheusRulesMetadata `json:"metadata"`
	Spec       PrometheusRulesSpec     `json:"spec"`
}

// PrometheusRulesMetadata contains Kubernetes-style metadata for a
// PrometheusRules document.
type PrometheusRulesMetadata struct {
	Name        string            `json:"name,omitempty"`
	Namespace   string            `json:"namespace,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// PrometheusRulesSpec contains the groups of rules.
type PrometheusRulesSpec struct {
	Groups []PrometheusRulesGroup `json:"groups"`
}

// PrometheusRulesGroup represents a single rule group in Prometheus format.
type PrometheusRulesGroup struct {
	Name     string           `json:"name"`
	Interval time.Duration    `json:"interval,omitempty"`
	Rules    []PrometheusRule `json:"rules"`
}

// PrometheusRule represents a single alerting rule in Prometheus format.
type PrometheusRule struct {
	Alert         string            `json:"alert"`
	Expr          string            `json:"expr"`
	For           time.Duration     `json:"for,omitempty"`
	KeepFiringFor time.Duration     `json:"keep_firing_for,omitempty"`
	Annotations   map[string]string `json:"annotations"`
	Labels        map[string]string `json:"labels"`
}

// FormatDuration formats a time.Duration as a compact Prometheus-style
// duration string (e.g., "2m" instead of Go's "2m0s").
func FormatDuration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}

	var parts []string
	if h := int(d.Hours()); h > 0 {
		parts = append(parts, fmt.Sprintf("%dh", h))
		d -= time.Duration(h) * time.Hour
	}
	if m := int(d.Minutes()); m > 0 {
		parts = append(parts, fmt.Sprintf("%dm", m))
		d -= time.Duration(m) * time.Minute
	}
	if s := d.Seconds(); s > 0 {
		parts = append(parts, fmt.Sprintf("%gs", s))
	}
	return strings.Join(parts, "")
}
