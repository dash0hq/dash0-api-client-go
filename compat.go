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
//
// # Migration guide (v1.28.x to <NEXT_RELEASE>)
//
// [DatasetSpec].Slug and [DatasetLabels].Dash0Comid are *string: use [String], [StringValue] or [GetDatasetID].
// [TimeReference] is a union: build it with [TimeReferenceFromString]. [PinnedRetentionClass] is its own enum.
// Bare enum names such as [Manual], [V1alpha2] and [N30d] stay as deprecated aliases. LinearLabelAdded* is gone.
// GetApiEdgeSettings takes a *[GetApiEdgeSettingsParams]; pass nil to keep the old request.

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

// DarkplanePullRequestEvaluation is a deprecated alias for [WebhookAgenticWorkflowTriggerKindDarkplanePullRequestEvaluation].
//
// Deprecated: since <NEXT_RELEASE>. Use [WebhookAgenticWorkflowTriggerKindDarkplanePullRequestEvaluation] instead.
const DarkplanePullRequestEvaluation = WebhookAgenticWorkflowTriggerKindDarkplanePullRequestEvaluation

// FailedCheckNew is a deprecated alias for [WebhookAgenticWorkflowTriggerKindFailedCheckNew].
//
// Deprecated: since <NEXT_RELEASE>. Use [WebhookAgenticWorkflowTriggerKindFailedCheckNew] instead.
const FailedCheckNew = WebhookAgenticWorkflowTriggerKindFailedCheckNew

// GithubCheckRun is a deprecated alias for [WebhookAgenticWorkflowTriggerKindGithubCheckRun].
//
// Deprecated: since <NEXT_RELEASE>. Use [WebhookAgenticWorkflowTriggerKindGithubCheckRun] instead.
const GithubCheckRun = WebhookAgenticWorkflowTriggerKindGithubCheckRun

// GithubCheckSuite is a deprecated alias for [WebhookAgenticWorkflowTriggerKindGithubCheckSuite].
//
// Deprecated: since <NEXT_RELEASE>. Use [WebhookAgenticWorkflowTriggerKindGithubCheckSuite] instead.
const GithubCheckSuite = WebhookAgenticWorkflowTriggerKindGithubCheckSuite

// GithubDeployment is a deprecated alias for [WebhookAgenticWorkflowTriggerKindGithubDeployment].
//
// Deprecated: since <NEXT_RELEASE>. Use [WebhookAgenticWorkflowTriggerKindGithubDeployment] instead.
const GithubDeployment = WebhookAgenticWorkflowTriggerKindGithubDeployment

// GithubDeploymentStatus is a deprecated alias for [WebhookAgenticWorkflowTriggerKindGithubDeploymentStatus].
//
// Deprecated: since <NEXT_RELEASE>. Use [WebhookAgenticWorkflowTriggerKindGithubDeploymentStatus] instead.
const GithubDeploymentStatus = WebhookAgenticWorkflowTriggerKindGithubDeploymentStatus

// GithubIssueComment is a deprecated alias for [WebhookAgenticWorkflowTriggerKindGithubIssueComment].
//
// Deprecated: since <NEXT_RELEASE>. Use [WebhookAgenticWorkflowTriggerKindGithubIssueComment] instead.
const GithubIssueComment = WebhookAgenticWorkflowTriggerKindGithubIssueComment

// GithubPullRequest is a deprecated alias for [WebhookAgenticWorkflowTriggerKindGithubPullRequest].
//
// Deprecated: since <NEXT_RELEASE>. Use [WebhookAgenticWorkflowTriggerKindGithubPullRequest] instead.
const GithubPullRequest = WebhookAgenticWorkflowTriggerKindGithubPullRequest

// GithubPullRequestReview is a deprecated alias for [WebhookAgenticWorkflowTriggerKindGithubPullRequestReview].
//
// Deprecated: since <NEXT_RELEASE>. Use [WebhookAgenticWorkflowTriggerKindGithubPullRequestReview] instead.
const GithubPullRequestReview = WebhookAgenticWorkflowTriggerKindGithubPullRequestReview

// GithubPullRequestReviewComment is a deprecated alias for [WebhookAgenticWorkflowTriggerKindGithubPullRequestReviewComment].
//
// Deprecated: since <NEXT_RELEASE>. Use [WebhookAgenticWorkflowTriggerKindGithubPullRequestReviewComment] instead.
const GithubPullRequestReviewComment = WebhookAgenticWorkflowTriggerKindGithubPullRequestReviewComment

// GithubRelease is a deprecated alias for [WebhookAgenticWorkflowTriggerKindGithubRelease].
//
// Deprecated: since <NEXT_RELEASE>. Use [WebhookAgenticWorkflowTriggerKindGithubRelease] instead.
const GithubRelease = WebhookAgenticWorkflowTriggerKindGithubRelease

// GithubWorkflowDispatch is a deprecated alias for [WebhookAgenticWorkflowTriggerKindGithubWorkflowDispatch].
//
// Deprecated: since <NEXT_RELEASE>. Use [WebhookAgenticWorkflowTriggerKindGithubWorkflowDispatch] instead.
const GithubWorkflowDispatch = WebhookAgenticWorkflowTriggerKindGithubWorkflowDispatch

// GithubWorkflowJob is a deprecated alias for [WebhookAgenticWorkflowTriggerKindGithubWorkflowJob].
//
// Deprecated: since <NEXT_RELEASE>. Use [WebhookAgenticWorkflowTriggerKindGithubWorkflowJob] instead.
const GithubWorkflowJob = WebhookAgenticWorkflowTriggerKindGithubWorkflowJob

// GithubWorkflowRun is a deprecated alias for [WebhookAgenticWorkflowTriggerKindGithubWorkflowRun].
//
// Deprecated: since <NEXT_RELEASE>. Use [WebhookAgenticWorkflowTriggerKindGithubWorkflowRun] instead.
const GithubWorkflowRun = WebhookAgenticWorkflowTriggerKindGithubWorkflowRun

// GitlabMergeRequest is a deprecated alias for [WebhookAgenticWorkflowTriggerKindGitlabMergeRequest].
//
// Deprecated: since <NEXT_RELEASE>. Use [WebhookAgenticWorkflowTriggerKindGitlabMergeRequest] instead.
const GitlabMergeRequest = WebhookAgenticWorkflowTriggerKindGitlabMergeRequest

// GitlabNote is a deprecated alias for [WebhookAgenticWorkflowTriggerKindGitlabNote].
//
// Deprecated: since <NEXT_RELEASE>. Use [WebhookAgenticWorkflowTriggerKindGitlabNote] instead.
const GitlabNote = WebhookAgenticWorkflowTriggerKindGitlabNote

// LinearComment is a deprecated alias for [WebhookAgenticWorkflowTriggerKindLinearComment].
//
// Deprecated: since <NEXT_RELEASE>. Use [WebhookAgenticWorkflowTriggerKindLinearComment] instead.
const LinearComment = WebhookAgenticWorkflowTriggerKindLinearComment

// Manual is a deprecated alias for [WebhookAgenticWorkflowTriggerKindManual].
//
// Deprecated: since <NEXT_RELEASE>. Use [WebhookAgenticWorkflowTriggerKindManual] instead.
const Manual = WebhookAgenticWorkflowTriggerKindManual

// Schedule is a deprecated alias for [WebhookAgenticWorkflowTriggerKindSchedule].
//
// Deprecated: since <NEXT_RELEASE>. Use [WebhookAgenticWorkflowTriggerKindSchedule] instead.
const Schedule = WebhookAgenticWorkflowTriggerKindSchedule

// SlackBotFailedCheckNew is a deprecated alias for [WebhookAgenticWorkflowTriggerKindSlackBotFailedCheckNew].
//
// Deprecated: since <NEXT_RELEASE>. Use [WebhookAgenticWorkflowTriggerKindSlackBotFailedCheckNew] instead.
const SlackBotFailedCheckNew = WebhookAgenticWorkflowTriggerKindSlackBotFailedCheckNew

// SlackBotMessage is a deprecated alias for [WebhookAgenticWorkflowTriggerKindSlackBotMessage].
//
// Deprecated: since <NEXT_RELEASE>. Use [WebhookAgenticWorkflowTriggerKindSlackBotMessage] instead.
const SlackBotMessage = WebhookAgenticWorkflowTriggerKindSlackBotMessage

// SlackBotReaction is a deprecated alias for [WebhookAgenticWorkflowTriggerKindSlackBotReaction].
//
// Deprecated: since <NEXT_RELEASE>. Use [WebhookAgenticWorkflowTriggerKindSlackBotReaction] instead.
const SlackBotReaction = WebhookAgenticWorkflowTriggerKindSlackBotReaction

// Webhook is a deprecated alias for [WebhookAgenticWorkflowTriggerKindWebhook].
//
// Deprecated: since <NEXT_RELEASE>. Use [WebhookAgenticWorkflowTriggerKindWebhook] instead.
const Webhook = WebhookAgenticWorkflowTriggerKindWebhook

// V1alpha2 is a deprecated alias for [SpamFilterApiVersionV1Alpha2V1alpha2].
//
// Deprecated: since <NEXT_RELEASE>. Use [SpamFilterApiVersionV1Alpha2V1alpha2] instead.
const V1alpha2 = SpamFilterApiVersionV1Alpha2V1alpha2

// Contains is a deprecated alias for [StringAssertionOperatorContains].
//
// Deprecated: since <NEXT_RELEASE>. Use [StringAssertionOperatorContains] instead.
const Contains = StringAssertionOperatorContains

// Csv is a deprecated alias for [WebEventExportFormatCsv].
//
// Deprecated: since <NEXT_RELEASE>. Use [WebEventExportFormatCsv] instead.
const Csv = WebEventExportFormatCsv

// DoesNotContain is a deprecated alias for [StringAssertionOperatorDoesNotContain].
//
// Deprecated: since <NEXT_RELEASE>. Use [StringAssertionOperatorDoesNotContain] instead.
const DoesNotContain = StringAssertionOperatorDoesNotContain

// DoesNotEndWith is a deprecated alias for [StringAssertionOperatorDoesNotEndWith].
//
// Deprecated: since <NEXT_RELEASE>. Use [StringAssertionOperatorDoesNotEndWith] instead.
const DoesNotEndWith = StringAssertionOperatorDoesNotEndWith

// DoesNotMatch is a deprecated alias for [StringAssertionOperatorDoesNotMatch].
//
// Deprecated: since <NEXT_RELEASE>. Use [StringAssertionOperatorDoesNotMatch] instead.
const DoesNotMatch = StringAssertionOperatorDoesNotMatch

// DoesNotStartWith is a deprecated alias for [StringAssertionOperatorDoesNotStartWith].
//
// Deprecated: since <NEXT_RELEASE>. Use [StringAssertionOperatorDoesNotStartWith] instead.
const DoesNotStartWith = StringAssertionOperatorDoesNotStartWith

// EndsWith is a deprecated alias for [StringAssertionOperatorEndsWith].
//
// Deprecated: since <NEXT_RELEASE>. Use [StringAssertionOperatorEndsWith] instead.
const EndsWith = StringAssertionOperatorEndsWith

// Github is a deprecated alias for [DarkplaneProviderGithub].
//
// Deprecated: since <NEXT_RELEASE>. Use [DarkplaneProviderGithub] instead.
const Github = DarkplaneProviderGithub

// GithubPush is a deprecated alias for [WebhookAgenticWorkflowTriggerKindGithubPush].
//
// Deprecated: since <NEXT_RELEASE>. Use [WebhookAgenticWorkflowTriggerKindGithubPush] instead.
const GithubPush = WebhookAgenticWorkflowTriggerKindGithubPush

// Gitlab is a deprecated alias for [DarkplaneProviderGitlab].
//
// Deprecated: since <NEXT_RELEASE>. Use [DarkplaneProviderGitlab] instead.
const Gitlab = DarkplaneProviderGitlab

// Gt is a deprecated alias for [SloComparisonOperatorGt].
//
// Deprecated: since <NEXT_RELEASE>. Use [SloComparisonOperatorGt] instead.
const Gt = SloComparisonOperatorGt

// Gte is a deprecated alias for [SloComparisonOperatorGte].
//
// Deprecated: since <NEXT_RELEASE>. Use [SloComparisonOperatorGte] instead.
const Gte = SloComparisonOperatorGte

// Is is a deprecated alias for [StringAssertionOperatorIs].
//
// Deprecated: since <NEXT_RELEASE>. Use [StringAssertionOperatorIs] instead.
const Is = StringAssertionOperatorIs

// IsNot is a deprecated alias for [StringAssertionOperatorIsNot].
//
// Deprecated: since <NEXT_RELEASE>. Use [StringAssertionOperatorIsNot] instead.
const IsNot = StringAssertionOperatorIsNot

// IsNotOneOf is a deprecated alias for [StringAssertionOperatorIsNotOneOf].
//
// Deprecated: since <NEXT_RELEASE>. Use [StringAssertionOperatorIsNotOneOf] instead.
const IsNotOneOf = StringAssertionOperatorIsNotOneOf

// IsNotSet is a deprecated alias for [StringAssertionOperatorIsNotSet].
//
// Deprecated: since <NEXT_RELEASE>. Use [StringAssertionOperatorIsNotSet] instead.
const IsNotSet = StringAssertionOperatorIsNotSet

// IsOneOf is a deprecated alias for [StringAssertionOperatorIsOneOf].
//
// Deprecated: since <NEXT_RELEASE>. Use [StringAssertionOperatorIsOneOf] instead.
const IsOneOf = StringAssertionOperatorIsOneOf

// IsSet is a deprecated alias for [StringAssertionOperatorIsSet].
//
// Deprecated: since <NEXT_RELEASE>. Use [StringAssertionOperatorIsSet] instead.
const IsSet = StringAssertionOperatorIsSet

// Lt is a deprecated alias for [SloComparisonOperatorLt].
//
// Deprecated: since <NEXT_RELEASE>. Use [SloComparisonOperatorLt] instead.
const Lt = SloComparisonOperatorLt

// Lte is a deprecated alias for [SloComparisonOperatorLte].
//
// Deprecated: since <NEXT_RELEASE>. Use [SloComparisonOperatorLte] instead.
const Lte = SloComparisonOperatorLte

// Matches is a deprecated alias for [StringAssertionOperatorMatches].
//
// Deprecated: since <NEXT_RELEASE>. Use [StringAssertionOperatorMatches] instead.
const Matches = StringAssertionOperatorMatches

// N120d is a deprecated alias for [RetentionClassN120d].
//
// Deprecated: since <NEXT_RELEASE>. Use [RetentionClassN120d] instead.
const N120d = RetentionClassN120d

// N210d is a deprecated alias for [RetentionClassN210d].
//
// Deprecated: since <NEXT_RELEASE>. Use [RetentionClassN210d] instead.
const N210d = RetentionClassN210d

// N30d is a deprecated alias for [RetentionClassN30d].
//
// Deprecated: since <NEXT_RELEASE>. Use [RetentionClassN30d] instead.
const N30d = RetentionClassN30d

// N390d is a deprecated alias for [RetentionClassN390d].
//
// Deprecated: since <NEXT_RELEASE>. Use [RetentionClassN390d] instead.
const N390d = RetentionClassN390d

// N60d is a deprecated alias for [RetentionClassN60d].
//
// Deprecated: since <NEXT_RELEASE>. Use [RetentionClassN60d] instead.
const N60d = RetentionClassN60d

// N90d is a deprecated alias for [RetentionClassN90d].
//
// Deprecated: since <NEXT_RELEASE>. Use [RetentionClassN90d] instead.
const N90d = RetentionClassN90d

// Ndjson is a deprecated alias for [WebEventExportFormatNdjson].
//
// Deprecated: since <NEXT_RELEASE>. Use [WebEventExportFormatNdjson] instead.
const Ndjson = WebEventExportFormatNdjson

// StartsWith is a deprecated alias for [StringAssertionOperatorStartsWith].
//
// Deprecated: since <NEXT_RELEASE>. Use [StringAssertionOperatorStartsWith] instead.
const StartsWith = StringAssertionOperatorStartsWith

// Txt is a deprecated alias for [WebEventExportFormatTxt].
//
// Deprecated: since <NEXT_RELEASE>. Use [WebEventExportFormatTxt] instead.
const Txt = WebEventExportFormatTxt

// Yaml is a deprecated alias for [WebEventExportFormatYaml].
//
// Deprecated: since <NEXT_RELEASE>. Use [WebEventExportFormatYaml] instead.
const Yaml = WebEventExportFormatYaml
