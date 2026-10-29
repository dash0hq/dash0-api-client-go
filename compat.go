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
// [SpamFilterApiVersionV1Alpha1] because the new [SpamFilterApiVersion]
// enum type would have collided with the short name.
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
//
// # Migration guide (v1.16.x to v1.17.0)
//
// The team CRD envelope is now exposed as [TeamDefinitionV1Alpha1] so the
// API-version suffix is visible in every call site. The unversioned
// [TeamDefinition] name survives as a deprecated type alias defined in
// client_teams.go; existing call sites keep compiling, but new code should
// reference [TeamDefinitionV1Alpha1] directly.
//
// [Client.GetTeam] now returns `*TeamDefinitionV1Alpha1` (the CRD
// envelope) instead of `*GetTeamResponse` (the UI-only enriched shape).
// The client extracts the envelope from the server's enriched response
// under the hood, so callers work with a single canonical type across
// create, get, and list flows. Use the accessor helpers ([GetTeamID],
// [GetTeamName], [GetTeamDisplayName], [GetTeamOrigin]) rather than
// reaching into the label struct.
//
// [Client.ResolveMemberIDsToEmails] is new: it translates the internal
// member IDs echoed by the server back to email addresses for display.
// Use it on the read path when rendering team membership to users.
//
// # Migration guide (v1.20.x to v1.21.0)
//
// [SamplingMode] constants were renamed from short forms ([Adaptive],
// [Disabled]) to prefixed names ([SamplingModeAdaptive],
// [SamplingModeDisabled]) because the upstream spec added a second
// `disabled`/`adaptive` enum, and oapi-codegen disambiguated both.
// The old names still work but are deprecated.
//
// # Migration guide (v1.22.x to v1.23.0)
//
// Several enum constants gained a type prefix because the upstream spec added
// enums with the same values: [Continue] and [Spawn] are now
// [AgenticWorkflowTriggerDeliveryContinue] and [AgenticWorkflowTriggerDeliverySpawn],
// [None] is now [OAuthTokenEndpointAuthMethodNone], [Alternative], [Both], and
// [Original] are now [SignalToMetricsTargetDatasetModeAlternative],
// [SignalToMetricsTargetDatasetModeBoth], and [SignalToMetricsTargetDatasetModeOriginal],
// and [Failure] and [Success] are now [SloRawTypeFailure] and [SloRawTypeSuccess].
// The old names still work but are deprecated.
//
// AgenticWorkflowGuardrails.NetworkLevel moved to [AgenticWorkflowSandbox.NetworkLevel].
//
// [HttpRequestSpec.Headers] and [HttpRequestSpec.QueryParameters] are now optional
// pointers ([HttpHeaders] and [HttpQueryParameters]); both alias []NameValuePair.
//
// GetApiSyntheticChecksLocationsResponse.JSON200 now lists [SyntheticCheckLocationEntry]
// values instead of bare location strings; use [SyntheticCheckLocationEntry.Id].
//
// [Client.ListCheckRules], [Client.ListRecordingRules], [Client.ListSLOs], and
// their Iter variants gained a trailing `opts ...ListOption` parameter for
// [WithOriginPrefix].
// Existing call sites compile unchanged.
// Custom [Client] implementations and wrappers must add the parameter, and
// should forward it to the wrapped client or ignore it.
// The dash0test.MockClient *Func fields keep their previous signatures.
//
// # Migration guide (v1.23.x to v1.24.0)
//
// The generated attribute pattern rule types and operations were removed
// because the public API spec no longer publishes them.
// This covers every AttributePatternRule* type, the matching
// *ApiAttributePatternRules* request and response types and client methods,
// and the Active, Inactive, UserDefined, and AutoDerived constants.
// There is no replacement.
//
// [Client] gained [Client.ListDatasets], [Client.GetDataset], [Client.CreateDataset],
// [Client.UpdateDataset], and [Client.ListDatasetsIter].
// Custom [Client] implementations and wrappers must add them.
// dash0test.MockClient implements them through new *Func fields.
//
// # Migration guide (v1.24.x to v1.25.0)
//
// The dataset document follows a new shape.
// [DatasetDefinition] gained a required ApiVersion field, which must be "dash0.com/v1alpha1" on create and update.
// The [DatasetSpec] fields that mirrored stored settings moved into three groups, and every field is optional:
//
//   - Tracing, Logging, Metrics, Profiling, and WebEvents became [DatasetRetention] fields on Spec.Retention.
//   - IpAddresses, GeoLocation, CardholderDataRedaction, and GenAiAttributeRedaction became [DatasetPrivacy] fields on Spec.Privacy.
//   - SemanticConventionUpgrades and Darkplane became [DatasetProcessing] fields on Spec.Processing, next to SpanEventsToLogs.
//
// Spec.TelemetryFilters is gone from the document, because spam filters own filters.
// An update never changes the stored filters, and [StripDatasetServerFields] no longer clears them.
//
// The same regeneration prefixed the generated constants of two enums that now share value names.
// The bare view type constants (Logs, Spans, Metrics, and so on) are now [ViewTypeLogs], [ViewTypeSpans], [ViewTypeMetrics], and so on.
// Restricted and Unrestricted are now [DatasetRestrictionRestricted] and [DatasetRestrictionUnrestricted].
// The bare names stay as deprecated aliases.
//
// [Client] gained [Client.ListAuthTokens], [Client.GetAuthToken], [Client.CreateAuthToken],
// [Client.UpdateAuthToken], [Client.DeleteAuthToken], and [Client.ListAuthTokensIter].
// Custom [Client] implementations and wrappers must add them.
// dash0test.MockClient implements them through new *Func fields.
//
// # Migration guide (v1.25.x to v1.26.0)
//
// [DatasetListResponse] and [AuthTokenListResponse] are now slices, because the list endpoints return a plain JSON array.
// [Client.ListDatasets] and [Client.ListAuthTokens] keep their signatures.
//
// The same regeneration dropped the type prefix from the [WebhookAgenticWorkflowTriggerKind] constants.
// WebhookAgenticWorkflowTriggerKindManual is now [Manual], WebhookAgenticWorkflowTriggerKindWebhook is now [Webhook], and so on.
// The prefixed names stay as deprecated aliases.
//
// [AuthTokenSpec] and [AuthTokenResponseSpec] Actions are now *[]AuthTokenAction instead of *[]Action.
// [AuthTokenAction] is a named enum.
// Use the constants ([IngestLogs], [AsteriskRead], and so on) or convert a string with AuthTokenAction(s).
// The API now rejects unknown actions with 400.
//
// # Migration guide (v1.27.x to v1.28.0)
//
// The public spec no longer exposes two fields of the edge collector settings, so [SettingsPerOrganizationAndDatasetInfo] lost TelemetryTransformationRuleSettings and ObservedPatterns.
// The types only these fields referenced are gone too: every TelemetryTransformationRule* type and constant, Dash0TelemetryTransformationRule, ObservedPatternEntry, and the OTTLConditions, OTTLStatement, and OTTLStatements aliases.
// [Client] never used them, and there is no replacement.
//
// The [ViewType] constants are unchanged: the prefixed names ([ViewTypeLogs], [ViewTypeSpans], and so on) stay canonical, and the bare names stay deprecated aliases.
// Code generation now pins the prefixed names, so they no longer depend on whether another enum in the spec shares their values.
//
// The [TeamApiVersion] constant is now [TeamApiVersionDash0Comv1alpha1], because [AwsMetricScrapeConfigApiVersion] shares its value.
// The bare Dash0Comv1alpha1 stays as a deprecated alias.

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

// AwsLambda is a deprecated alias for [ViewTypeAwsLambda].
//
// Deprecated: since v1.25.0. Use [ViewTypeAwsLambda] instead.
const AwsLambda = ViewTypeAwsLambda

// FailedChecks is a deprecated alias for [ViewTypeFailedChecks].
//
// Deprecated: since v1.25.0. Use [ViewTypeFailedChecks] instead.
const FailedChecks = ViewTypeFailedChecks

// GcpCloudRunJobs is a deprecated alias for [ViewTypeGcpCloudRunJobs].
//
// Deprecated: since v1.25.0. Use [ViewTypeGcpCloudRunJobs] instead.
const GcpCloudRunJobs = ViewTypeGcpCloudRunJobs

// GcpCloudRunServices is a deprecated alias for [ViewTypeGcpCloudRunServices].
//
// Deprecated: since v1.25.0. Use [ViewTypeGcpCloudRunServices] instead.
const GcpCloudRunServices = ViewTypeGcpCloudRunServices

// GcpCloudSqlInstances is a deprecated alias for [ViewTypeGcpCloudSqlInstances].
//
// Deprecated: since v1.25.0. Use [ViewTypeGcpCloudSqlInstances] instead.
const GcpCloudSqlInstances = ViewTypeGcpCloudSqlInstances

// GcpCloudStorage is a deprecated alias for [ViewTypeGcpCloudStorage].
//
// Deprecated: since v1.25.0. Use [ViewTypeGcpCloudStorage] instead.
const GcpCloudStorage = ViewTypeGcpCloudStorage

// GcpPubsub is a deprecated alias for [ViewTypeGcpPubsub].
//
// Deprecated: since v1.25.0. Use [ViewTypeGcpPubsub] instead.
const GcpPubsub = ViewTypeGcpPubsub

// Dash0Comv1alpha1 is a deprecated alias for [TeamApiVersionDash0Comv1alpha1].
//
// Deprecated: since v1.28.0. Use [TeamApiVersionDash0Comv1alpha1] instead.
const Dash0Comv1alpha1 = TeamApiVersionDash0Comv1alpha1

// Logs is a deprecated alias for [ViewTypeLogs].
//
// Deprecated: since v1.25.0. Use [ViewTypeLogs] instead.
const Logs = ViewTypeLogs

// Metrics is a deprecated alias for [ViewTypeMetrics].
//
// Deprecated: since v1.25.0. Use [ViewTypeMetrics] instead.
const Metrics = ViewTypeMetrics

// Profiles is a deprecated alias for [ViewTypeProfiles].
//
// Deprecated: since v1.25.0. Use [ViewTypeProfiles] instead.
const Profiles = ViewTypeProfiles

// Resources is a deprecated alias for [ViewTypeResources].
//
// Deprecated: since v1.25.0. Use [ViewTypeResources] instead.
const Resources = ViewTypeResources

// Services is a deprecated alias for [ViewTypeServices].
//
// Deprecated: since v1.25.0. Use [ViewTypeServices] instead.
const Services = ViewTypeServices

// Slos is a deprecated alias for [ViewTypeSlos].
//
// Deprecated: since v1.25.0. Use [ViewTypeSlos] instead.
const Slos = ViewTypeSlos

// Spans is a deprecated alias for [ViewTypeSpans].
//
// Deprecated: since v1.25.0. Use [ViewTypeSpans] instead.
const Spans = ViewTypeSpans

// Sql is a deprecated alias for [ViewTypeSql].
//
// Deprecated: since v1.25.0. Use [ViewTypeSql] instead.
const Sql = ViewTypeSql

// WebEvents is a deprecated alias for [ViewTypeWebEvents].
//
// Deprecated: since v1.25.0. Use [ViewTypeWebEvents] instead.
const WebEvents = ViewTypeWebEvents

// Restricted is a deprecated alias for [DatasetRestrictionRestricted].
//
// Deprecated: since v1.25.0. Use [DatasetRestrictionRestricted] instead.
const Restricted = DatasetRestrictionRestricted

// Unrestricted is a deprecated alias for [DatasetRestrictionUnrestricted].
//
// Deprecated: since v1.25.0. Use [DatasetRestrictionUnrestricted] instead.
const Unrestricted = DatasetRestrictionUnrestricted

// Deprecated: since v1.12.3. Use [SpamFilterApiVersionV1alpha1] instead
// (the enum value on the wider [SpamFilterApiVersion] type generated by the
// current upstream spec).
const V1alpha1 = SpamFilterApiVersionV1alpha1

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

// Adaptive is a deprecated alias for [SamplingModeAdaptive].
//
// Deprecated: since v1.21.0. Use [SamplingModeAdaptive] instead.
const Adaptive = SamplingModeAdaptive

// Disabled is a deprecated alias for [SamplingModeDisabled].
//
// Deprecated: since v1.21.0. Use [SamplingModeDisabled] instead.
const Disabled = SamplingModeDisabled

// Continue is a deprecated alias for [AgenticWorkflowTriggerDeliveryContinue].
//
// Deprecated: since v1.23.0. Use [AgenticWorkflowTriggerDeliveryContinue] instead.
const Continue = AgenticWorkflowTriggerDeliveryContinue

// Spawn is a deprecated alias for [AgenticWorkflowTriggerDeliverySpawn].
//
// Deprecated: since v1.23.0. Use [AgenticWorkflowTriggerDeliverySpawn] instead.
const Spawn = AgenticWorkflowTriggerDeliverySpawn

// None is a deprecated alias for [OAuthTokenEndpointAuthMethodNone].
//
// Deprecated: since v1.23.0. Use [OAuthTokenEndpointAuthMethodNone] instead.
const None = OAuthTokenEndpointAuthMethodNone

// Alternative is a deprecated alias for [SignalToMetricsTargetDatasetModeAlternative].
//
// Deprecated: since v1.23.0. Use [SignalToMetricsTargetDatasetModeAlternative] instead.
const Alternative = SignalToMetricsTargetDatasetModeAlternative

// Both is a deprecated alias for [SignalToMetricsTargetDatasetModeBoth].
//
// Deprecated: since v1.23.0. Use [SignalToMetricsTargetDatasetModeBoth] instead.
const Both = SignalToMetricsTargetDatasetModeBoth

// Original is a deprecated alias for [SignalToMetricsTargetDatasetModeOriginal].
//
// Deprecated: since v1.23.0. Use [SignalToMetricsTargetDatasetModeOriginal] instead.
const Original = SignalToMetricsTargetDatasetModeOriginal

// Failure is a deprecated alias for [SloRawTypeFailure].
//
// Deprecated: since v1.23.0. Use [SloRawTypeFailure] instead.
const Failure = SloRawTypeFailure

// Success is a deprecated alias for [SloRawTypeSuccess].
//
// Deprecated: since v1.23.0. Use [SloRawTypeSuccess] instead.
const Success = SloRawTypeSuccess

// WebhookAgenticWorkflowTriggerKindDarkplanePullRequestEvaluation is a deprecated alias for [DarkplanePullRequestEvaluation].
//
// Deprecated: since v1.26.0. Use [DarkplanePullRequestEvaluation] instead.
const WebhookAgenticWorkflowTriggerKindDarkplanePullRequestEvaluation = DarkplanePullRequestEvaluation

// WebhookAgenticWorkflowTriggerKindFailedCheckNew is a deprecated alias for [FailedCheckNew].
//
// Deprecated: since v1.26.0. Use [FailedCheckNew] instead.
const WebhookAgenticWorkflowTriggerKindFailedCheckNew = FailedCheckNew

// WebhookAgenticWorkflowTriggerKindGithubCheckRun is a deprecated alias for [GithubCheckRun].
//
// Deprecated: since v1.26.0. Use [GithubCheckRun] instead.
const WebhookAgenticWorkflowTriggerKindGithubCheckRun = GithubCheckRun

// WebhookAgenticWorkflowTriggerKindGithubCheckSuite is a deprecated alias for [GithubCheckSuite].
//
// Deprecated: since v1.26.0. Use [GithubCheckSuite] instead.
const WebhookAgenticWorkflowTriggerKindGithubCheckSuite = GithubCheckSuite

// WebhookAgenticWorkflowTriggerKindGithubDeployment is a deprecated alias for [GithubDeployment].
//
// Deprecated: since v1.26.0. Use [GithubDeployment] instead.
const WebhookAgenticWorkflowTriggerKindGithubDeployment = GithubDeployment

// WebhookAgenticWorkflowTriggerKindGithubDeploymentStatus is a deprecated alias for [GithubDeploymentStatus].
//
// Deprecated: since v1.26.0. Use [GithubDeploymentStatus] instead.
const WebhookAgenticWorkflowTriggerKindGithubDeploymentStatus = GithubDeploymentStatus

// WebhookAgenticWorkflowTriggerKindGithubIssueComment is a deprecated alias for [GithubIssueComment].
//
// Deprecated: since v1.26.0. Use [GithubIssueComment] instead.
const WebhookAgenticWorkflowTriggerKindGithubIssueComment = GithubIssueComment

// WebhookAgenticWorkflowTriggerKindGithubPullRequest is a deprecated alias for [GithubPullRequest].
//
// Deprecated: since v1.26.0. Use [GithubPullRequest] instead.
const WebhookAgenticWorkflowTriggerKindGithubPullRequest = GithubPullRequest

// WebhookAgenticWorkflowTriggerKindGithubPullRequestReview is a deprecated alias for [GithubPullRequestReview].
//
// Deprecated: since v1.26.0. Use [GithubPullRequestReview] instead.
const WebhookAgenticWorkflowTriggerKindGithubPullRequestReview = GithubPullRequestReview

// WebhookAgenticWorkflowTriggerKindGithubPullRequestReviewComment is a deprecated alias for [GithubPullRequestReviewComment].
//
// Deprecated: since v1.26.0. Use [GithubPullRequestReviewComment] instead.
const WebhookAgenticWorkflowTriggerKindGithubPullRequestReviewComment = GithubPullRequestReviewComment

// WebhookAgenticWorkflowTriggerKindGithubRelease is a deprecated alias for [GithubRelease].
//
// Deprecated: since v1.26.0. Use [GithubRelease] instead.
const WebhookAgenticWorkflowTriggerKindGithubRelease = GithubRelease

// WebhookAgenticWorkflowTriggerKindGithubWorkflowDispatch is a deprecated alias for [GithubWorkflowDispatch].
//
// Deprecated: since v1.26.0. Use [GithubWorkflowDispatch] instead.
const WebhookAgenticWorkflowTriggerKindGithubWorkflowDispatch = GithubWorkflowDispatch

// WebhookAgenticWorkflowTriggerKindGithubWorkflowJob is a deprecated alias for [GithubWorkflowJob].
//
// Deprecated: since v1.26.0. Use [GithubWorkflowJob] instead.
const WebhookAgenticWorkflowTriggerKindGithubWorkflowJob = GithubWorkflowJob

// WebhookAgenticWorkflowTriggerKindGithubWorkflowRun is a deprecated alias for [GithubWorkflowRun].
//
// Deprecated: since v1.26.0. Use [GithubWorkflowRun] instead.
const WebhookAgenticWorkflowTriggerKindGithubWorkflowRun = GithubWorkflowRun

// WebhookAgenticWorkflowTriggerKindGitlabMergeRequest is a deprecated alias for [GitlabMergeRequest].
//
// Deprecated: since v1.26.0. Use [GitlabMergeRequest] instead.
const WebhookAgenticWorkflowTriggerKindGitlabMergeRequest = GitlabMergeRequest

// WebhookAgenticWorkflowTriggerKindGitlabNote is a deprecated alias for [GitlabNote].
//
// Deprecated: since v1.26.0. Use [GitlabNote] instead.
const WebhookAgenticWorkflowTriggerKindGitlabNote = GitlabNote

// WebhookAgenticWorkflowTriggerKindLinearComment is a deprecated alias for [LinearComment].
//
// Deprecated: since v1.26.0. Use [LinearComment] instead.
const WebhookAgenticWorkflowTriggerKindLinearComment = LinearComment

// WebhookAgenticWorkflowTriggerKindLinearLabelAdded is a deprecated alias for [LinearLabelAdded].
//
// Deprecated: since v1.26.0. Use [LinearLabelAdded] instead.
const WebhookAgenticWorkflowTriggerKindLinearLabelAdded = LinearLabelAdded

// WebhookAgenticWorkflowTriggerKindManual is a deprecated alias for [Manual].
//
// Deprecated: since v1.26.0. Use [Manual] instead.
const WebhookAgenticWorkflowTriggerKindManual = Manual

// WebhookAgenticWorkflowTriggerKindSchedule is a deprecated alias for [Schedule].
//
// Deprecated: since v1.26.0. Use [Schedule] instead.
const WebhookAgenticWorkflowTriggerKindSchedule = Schedule

// WebhookAgenticWorkflowTriggerKindSlackBotFailedCheckNew is a deprecated alias for [SlackBotFailedCheckNew].
//
// Deprecated: since v1.26.0. Use [SlackBotFailedCheckNew] instead.
const WebhookAgenticWorkflowTriggerKindSlackBotFailedCheckNew = SlackBotFailedCheckNew

// WebhookAgenticWorkflowTriggerKindSlackBotMessage is a deprecated alias for [SlackBotMessage].
//
// Deprecated: since v1.26.0. Use [SlackBotMessage] instead.
const WebhookAgenticWorkflowTriggerKindSlackBotMessage = SlackBotMessage

// WebhookAgenticWorkflowTriggerKindSlackBotReaction is a deprecated alias for [SlackBotReaction].
//
// Deprecated: since v1.26.0. Use [SlackBotReaction] instead.
const WebhookAgenticWorkflowTriggerKindSlackBotReaction = SlackBotReaction

// WebhookAgenticWorkflowTriggerKindWebhook is a deprecated alias for [Webhook].
//
// Deprecated: since v1.26.0. Use [Webhook] instead.
const WebhookAgenticWorkflowTriggerKindWebhook = Webhook
