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
//
// # Migration guide (v1.8.x to v1.9.0)
//
// [ViewType] constants were renamed from short names ([FailedChecks], [Logs],
// [Metrics], [Resources], [Services], [Spans], [Sql], [WebEvents]) to prefixed
// names ([ViewTypeFailedChecks], [ViewTypeLogs], [ViewTypeMetrics],
// [ViewTypeResources], [ViewTypeServices], [ViewTypeSpans], [ViewTypeSql],
// [ViewTypeWebEvents]).
// The old names still work but are deprecated.
//
// # Migration guide (v1.9.x to v1.12.0)
//
// [SignalToMetricsSource] was replaced by the shared [CrdSource] type.
// The old type and constants ([SignalToMetricsSourceApi],
// [SignalToMetricsSourceOperator], [SignalToMetricsSourceTerraform],
// [SignalToMetricsSourceUi]) still work as deprecated aliases.

// DashboardSource is a deprecated alias for [CrdSource].
//
// Deprecated: since v1.12.0. Use [CrdSource] instead.
type DashboardSource = CrdSource

// DashboardSourceApi is a deprecated alias for [Api].
//
// Deprecated: since v1.12.0. Use [Api] instead.
const DashboardSourceApi = Api

// DashboardSourceOperator is a deprecated alias for [Operator].
//
// Deprecated: since v1.12.0. Use [Operator] instead.
const DashboardSourceOperator = Operator

// DashboardSourceTerraform is a deprecated alias for [Terraform].
//
// Deprecated: since v1.12.0. Use [Terraform] instead.
const DashboardSourceTerraform = Terraform

// DashboardSourceUi is a deprecated alias for [Ui].
//
// Deprecated: since v1.12.0. Use [Ui] instead.
const DashboardSourceUi = Ui

// SignalToMetricsSource is a deprecated alias for [CrdSource].
//
// Deprecated: since v1.12.0. Use [CrdSource] instead.
type SignalToMetricsSource = CrdSource

// SignalToMetricsSourceApi is a deprecated alias for [Api].
//
// Deprecated: since v1.12.0. Use [Api] instead.
const SignalToMetricsSourceApi = Api

// SignalToMetricsSourceOperator is a deprecated alias for [Operator].
//
// Deprecated: since v1.12.0. Use [Operator] instead.
const SignalToMetricsSourceOperator = Operator

// SignalToMetricsSourceTerraform is a deprecated alias for [Terraform].
//
// Deprecated: since v1.12.0. Use [Terraform] instead.
const SignalToMetricsSourceTerraform = Terraform

// SignalToMetricsSourceUi is a deprecated alias for [Ui].
//
// Deprecated: since v1.12.0. Use [Ui] instead.
const SignalToMetricsSourceUi = Ui

// Deprecated: since v1.9.0. Use [ViewTypeFailedChecks] instead.
const FailedChecks = ViewTypeFailedChecks

// Deprecated: since v1.9.0. Use [ViewTypeLogs] instead.
const Logs = ViewTypeLogs

// Deprecated: since v1.9.0. Use [ViewTypeMetrics] instead.
const Metrics = ViewTypeMetrics

// Deprecated: since v1.9.0. Use [ViewTypeResources] instead.
const Resources = ViewTypeResources

// Deprecated: since v1.9.0. Use [ViewTypeServices] instead.
const Services = ViewTypeServices

// Deprecated: since v1.9.0. Use [ViewTypeSpans] instead.
const Spans = ViewTypeSpans

// Deprecated: since v1.9.0. Use [ViewTypeSql] instead.
const Sql = ViewTypeSql

// Deprecated: since v1.9.0. Use [ViewTypeWebEvents] instead.
const WebEvents = ViewTypeWebEvents
