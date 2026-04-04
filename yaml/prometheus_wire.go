package yaml

import (
	"encoding/json"
	"fmt"
	"time"

	dash0 "github.com/dash0hq/dash0-api-client-go"
)

// prometheusDuration is a wire type that marshals/unmarshals Go duration strings
// (e.g., "5m0s") for Prometheus rule YAML serialization via sigs.k8s.io/yaml.
// It is used internally by the marshal/unmarshal functions in this package.
type prometheusDuration time.Duration

// MarshalJSON outputs a quoted Go duration string.
func (d prometheusDuration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// UnmarshalJSON parses a Go duration string or numeric nanosecond value from
// JSON.
func (d *prometheusDuration) UnmarshalJSON(b []byte) error {
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
		*d = prometheusDuration(duration)
	case float64:
		*d = prometheusDuration(time.Duration(value))
	default:
		return fmt.Errorf("invalid duration type: %T", v)
	}
	return nil
}

// prometheusRulesWire mirrors [dash0.PrometheusRules] with
// [prometheusDuration] fields for YAML serialization.
type prometheusRulesWire struct {
	APIVersion string                      `json:"apiVersion"`
	Kind       string                      `json:"kind"`
	Metadata   dash0.PrometheusRulesMetadata `json:"metadata"`
	Spec       prometheusRulesSpecWire     `json:"spec"`
}

type prometheusRulesSpecWire struct {
	Groups []prometheusRulesGroupWire `json:"groups"`
}

type prometheusRulesGroupWire struct {
	Name     string               `json:"name"`
	Interval prometheusDuration   `json:"interval,omitempty"`
	Rules    []prometheusRuleWire `json:"rules"`
}

type prometheusRuleWire struct {
	Alert         string            `json:"alert"`
	Expr          string            `json:"expr"`
	For           prometheusDuration `json:"for,omitempty"`
	KeepFiringFor prometheusDuration `json:"keep_firing_for,omitempty"`
	Annotations   map[string]string `json:"annotations"`
	Labels        map[string]string `json:"labels"`
}

// toPrometheusRules converts the wire type to the domain type.
func (w *prometheusRulesWire) toPrometheusRules() *dash0.PrometheusRules {
	result := &dash0.PrometheusRules{
		APIVersion: w.APIVersion,
		Kind:       w.Kind,
		Metadata:   w.Metadata,
		Spec: dash0.PrometheusRulesSpec{
			Groups: make([]dash0.PrometheusRulesGroup, len(w.Spec.Groups)),
		},
	}
	for i, g := range w.Spec.Groups {
		group := dash0.PrometheusRulesGroup{
			Name:     g.Name,
			Interval: time.Duration(g.Interval),
			Rules:    make([]dash0.PrometheusRule, len(g.Rules)),
		}
		for j, r := range g.Rules {
			group.Rules[j] = dash0.PrometheusRule{
				Alert:         r.Alert,
				Expr:          r.Expr,
				For:           time.Duration(r.For),
				KeepFiringFor: time.Duration(r.KeepFiringFor),
				Annotations:   r.Annotations,
				Labels:        r.Labels,
			}
		}
		result.Spec.Groups[i] = group
	}
	return result
}

// fromPrometheusRules converts the domain type to the wire type.
func fromPrometheusRules(r *dash0.PrometheusRules) *prometheusRulesWire {
	wire := &prometheusRulesWire{
		APIVersion: r.APIVersion,
		Kind:       r.Kind,
		Metadata:   r.Metadata,
		Spec: prometheusRulesSpecWire{
			Groups: make([]prometheusRulesGroupWire, len(r.Spec.Groups)),
		},
	}
	for i, g := range r.Spec.Groups {
		group := prometheusRulesGroupWire{
			Name:     g.Name,
			Interval: prometheusDuration(g.Interval),
			Rules:    make([]prometheusRuleWire, len(g.Rules)),
		}
		for j, rule := range g.Rules {
			group.Rules[j] = prometheusRuleWire{
				Alert:         rule.Alert,
				Expr:          rule.Expr,
				For:           prometheusDuration(rule.For),
				KeepFiringFor: prometheusDuration(rule.KeepFiringFor),
				Annotations:   rule.Annotations,
				Labels:        rule.Labels,
			}
		}
		wire.Spec.Groups[i] = group
	}
	return wire
}
