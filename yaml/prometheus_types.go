package yaml

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// PrometheusRules represents the Kubernetes Prometheus Operator PrometheusRule
// CRD document in YAML format. This is the user-facing format used by the
// Terraform provider and Kubernetes operator.
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
	Interval PromDuration     `json:"interval,omitempty"`
	Rules    []PrometheusRule `json:"rules"`
}

// PrometheusRule represents a single alerting rule in Prometheus format.
type PrometheusRule struct {
	Alert         string            `json:"alert"`
	Expr          string            `json:"expr"`
	For           PromDuration      `json:"for,omitempty"`
	KeepFiringFor PromDuration      `json:"keep_firing_for,omitempty"`
	Annotations   map[string]string `json:"annotations"`
	Labels        map[string]string `json:"labels"`
}

// PromDuration is a duration type that marshals/unmarshals to/from Go duration
// strings (e.g., "2m30s"). This is used for the Prometheus rule YAML format.
// It differs from the generated Duration type (which is a plain string alias).
type PromDuration time.Duration

// MarshalJSON outputs a quoted Go duration string.
func (d PromDuration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// UnmarshalJSON parses a Go duration string from JSON. When using
// sigs.k8s.io/yaml, YAML is converted to JSON first, so this method
// handles both YAML and JSON input.
func (d *PromDuration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case string:
		duration, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		*d = PromDuration(duration)
	case float64:
		*d = PromDuration(time.Duration(value))
	default:
		return fmt.Errorf("invalid duration type: %T", v)
	}
	return nil
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
