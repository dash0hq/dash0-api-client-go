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
//
// # Migration guide (v1.12.x to v1.12.3)
//
// The upstream OpenAPI spec regeneration reverted the v1.9.0 [ViewType]
// constant rename: the canonical names are once again the short forms
// ([FailedChecks], [Logs], [Metrics], [Resources], [Services], [Spans],
// [Sql], [WebEvents], plus the previously prefix-only [AwsLambda],
// [GcpCloudRunJobs], [GcpCloudRunServices], [GcpCloudStorage], [GcpPubsub],
// and [Profiles]).
// The prefixed names ([ViewTypeFailedChecks] etc.) remain as deprecated
// aliases.
//
// The spam-filter v1alpha1 constant was renamed from [V1alpha1] to
// [SpamFilterApiVersionV1Alpha1V1alpha1] because the new
// [SpamFilterApiVersion] union type would have collided with the short name.
// The old [V1alpha1] identifier is preserved as a deprecated alias.
//
// [DashboardAnnotations.Dash0Comsource] was moved by the upstream spec into
// the new [DashboardLabels] struct accessible via
// [DashboardMetadata.Labels.Dash0Comsource].
// Update read and write call sites accordingly; there is no in-package
// shim because struct fields cannot be aliased.
//
// # Migration guide (v1.15.x to v1.16.0)
//
// The upstream OpenAPI spec reverted the [HttpRequestBodyKind] constant
// rename: the canonical names are once again the short forms ([Form],
// [Graphql], [Json], [Raw]).
// The prefixed names ([HttpRequestBodyKindForm], [HttpRequestBodyKindGraphql],
// [HttpRequestBodyKindJson], [HttpRequestBodyKindRaw]) remain as deprecated
// aliases.
//
// The upstream spec also dropped the [ResponseFormat] type (and its
// [ResponseFormatJson] and [ResponseFormatYaml] constants) along with the
// `format` query parameter on the SLO GET endpoint
// ([GetApiSlosOriginOrIdParams.Format]).
// There is no in-package shim because the type, constants, and field were
// removed wholesale by the upstream spec.

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

// Deprecated: since v1.12.3. Use [AwsLambda] instead.
const ViewTypeAwsLambda = AwsLambda

// Deprecated: since v1.12.3. Use [FailedChecks] instead.
const ViewTypeFailedChecks = FailedChecks

// Deprecated: since v1.12.3. Use [GcpCloudRunJobs] instead.
const ViewTypeGcpCloudRunJobs = GcpCloudRunJobs

// Deprecated: since v1.12.3. Use [GcpCloudRunServices] instead.
const ViewTypeGcpCloudRunServices = GcpCloudRunServices

// Deprecated: since v1.12.3. Use [GcpCloudStorage] instead.
const ViewTypeGcpCloudStorage = GcpCloudStorage

// Deprecated: since v1.12.3. Use [GcpPubsub] instead.
const ViewTypeGcpPubsub = GcpPubsub

// Deprecated: since v1.12.3. Use [Logs] instead.
const ViewTypeLogs = Logs

// Deprecated: since v1.12.3. Use [Metrics] instead.
const ViewTypeMetrics = Metrics

// Deprecated: since v1.12.3. Use [Profiles] instead.
const ViewTypeProfiles = Profiles

// Deprecated: since v1.12.3. Use [Resources] instead.
const ViewTypeResources = Resources

// Deprecated: since v1.12.3. Use [Services] instead.
const ViewTypeServices = Services

// Deprecated: since v1.12.3. Use [Spans] instead.
const ViewTypeSpans = Spans

// Deprecated: since v1.12.3. Use [Sql] instead.
const ViewTypeSql = Sql

// Deprecated: since v1.12.3. Use [WebEvents] instead.
const ViewTypeWebEvents = WebEvents

// Deprecated: since v1.12.3. Use [SpamFilterApiVersionV1Alpha1V1alpha1] instead.
const V1alpha1 = SpamFilterApiVersionV1Alpha1V1alpha1

// Dash0SpamFilter is a deprecated alias for [SpamFilterDefinitionKindDash0SpamFilter].
//
// Deprecated: since v1.12.1. Use [SpamFilterDefinitionKindDash0SpamFilter] instead.
const Dash0SpamFilter = SpamFilterDefinitionKindDash0SpamFilter

// HttpRequestBodyKindForm is a deprecated alias for [Form].
//
// Deprecated: since v1.16.0. Use [Form] instead.
const HttpRequestBodyKindForm = Form

// HttpRequestBodyKindGraphql is a deprecated alias for [Graphql].
//
// Deprecated: since v1.16.0. Use [Graphql] instead.
const HttpRequestBodyKindGraphql = Graphql

// HttpRequestBodyKindJson is a deprecated alias for [Json].
//
// Deprecated: since v1.16.0. Use [Json] instead.
const HttpRequestBodyKindJson = Json

// HttpRequestBodyKindRaw is a deprecated alias for [Raw].
//
// Deprecated: since v1.16.0. Use [Raw] instead.
const HttpRequestBodyKindRaw = Raw
