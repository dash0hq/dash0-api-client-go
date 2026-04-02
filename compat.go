package dash0

// This file provides backwards-compatible aliases and documents migration steps
// for breaking changes introduced by upstream OpenAPI spec updates.
// Remove the aliases after the next major version bump.
//
// # Migration guide (v1.5.x to v1.6.x)
//
// CheckThresholds.Degraded and CheckThresholds.Failed changed from *float32 to
// *float64. Update any float32 variables or casts to float64.
//
// PrometheusAlertRule.Annotations changed from *map[string]string to the typed
// *PrometheusAlertRule_Annotations struct. Use Annotations.Summary,
// Annotations.Description, and Annotations.AdditionalProperties instead of
// map access.
//
// PrometheusAlertRule.Description and PrometheusAlertRule.Summary were removed.
// Use PrometheusAlertRule.Annotations.Description and
// PrometheusAlertRule.Annotations.Summary instead.
//
// PrometheusAlertRule.KeepFiringFor changed from *string (camelCase JSON key
// keepFiringFor) to *Duration (snake_case JSON key keep_firing_for).
//
// DashboardSource constants were renamed from short names (Api, Operator,
// Terraform, Ui) to prefixed names (DashboardSourceApi, DashboardSourceOperator,
// DashboardSourceTerraform, DashboardSourceUi). The old names still work but
// are deprecated.

// Deprecated: since <NEXT_RELEASE>. Use [DashboardSourceApi] instead.
const Api = DashboardSourceApi

// Deprecated: since <NEXT_RELEASE>. Use [DashboardSourceOperator] instead.
const Operator = DashboardSourceOperator

// Deprecated: since <NEXT_RELEASE>. Use [DashboardSourceTerraform] instead.
const Terraform = DashboardSourceTerraform

// Deprecated: since <NEXT_RELEASE>. Use [DashboardSourceUi] instead.
const Ui = DashboardSourceUi
