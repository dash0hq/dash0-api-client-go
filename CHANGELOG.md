# Changelog

## v1.28.0


### Bug Fixes
- keep the enabled annotation when stripping, and add v1alpha2 helpers (#64)

  `StripSpamFilterServerFields` no longer clears the `dash0.com/enabled` annotation, so a disabled filter stays disabled after a strip and write. It now also clears the `dash0.com/dataset` and `dash0.com/origin` labels.
  New v1alpha2 helpers `StripSpamFilterV1Alpha2ServerFields`, `GetSpamFilterV1Alpha2ID`, `GetSpamFilterV1Alpha2Name`, `SetSpamFilterV1Alpha2ID`, `SetSpamFilterV1Alpha2IDIfAbsent`, `ClearSpamFilterV1Alpha2ID`, `GetSpamFilterV1Alpha2Dataset`, and `SetSpamFilterV1Alpha2Dataset`, and a new `ListSpamFilterObjectsIter` method.
  `CreateSpamFilter`, `UpdateSpamFilter`, `CreateSpamFilterV1Alpha2`, and `UpdateSpamFilterV1Alpha2` return an error instead of panicking when the filter is nil.

  * test(spam-filters): check the strip round trip against the live API

  TestLiveAPI_SpamFilters creates a disabled v1alpha1 filter and a
  disabled v1alpha2 filter, then gets, strips, and updates each one. It
  checks that both stay disabled, that the v1alpha1 filter keeps the
  origin from the request path, and that ListSpamFilterObjectsIter yields
  the v1alpha2 filter with its spec.context.

  The test runs only when DASH0_API_URL and DASH0_AUTH_TOKEN are set, like
  TestLiveAPI. The filters match an attribute no telemetry carries and are
  disabled, and the test deletes them when it finishes.

  * refactor(spam-filters): drop ListSpamFilterObjectsIter, and fix the strip doc

  ListSpamFilterObjectsIter wrapped ListSpamFilterObjects in Iter, so
  Current returned a pointer to an interface and every caller had to
  dereference it before the type switch. The list loads every filter
  anyway, so callers range over ListSpamFilterObjects instead. The method
  was not released yet.

  The StripSpamFilterServerFields doc said that, unlike most other asset
  types, it clears dash0.com/id. About half the Strip helpers clear it, so
  the sentence is gone.

  * fix(spam-filters): keep the origin and dataset labels when stripping

  StripSpamFilterServerFields cleared dash0.com/origin and
  dash0.com/dataset along with the id and source labels. The server
  ignores the origin in an update body, but a create keeps it, so a
  filter recreated from a stripped copy got a new api-<uuid> origin, and
  the next upsert by the old origin added a second filter. Without the
  dataset label, a write that passes no dataset parameter went to the
  default dataset instead of the filter's own.

  Strip now clears only the id and source labels, as before this branch,
  and keeps the enabled annotation fix. The v1alpha2 helper shares the
  same code. A new live test recreates a filter from a stripped copy and
  checks that it keeps its origin.

  `StripSpamFilterServerFields` no longer clears the `dash0.com/enabled` annotation, so a disabled filter stays disabled after a strip and write. It still keeps the `dash0.com/origin` and `dash0.com/dataset` labels.
- pin the view type constant names and regenerate from the latest spec (#61)

  `SignalToMetricsOutput` gained the optional `Type`, `Value`, `Monotonic`, `MaxSize`, and `Count` fields.

### Features
- wrap the test and import endpoints and add list filters (#63)

  New `Client` methods `TestSignalToMetrics`, `ImportSignalToMetrics`, and `CountSignalToMetrics`, and new list options `WithName`, `WithSignal`, and `WithEnabled` for signal-to-metrics rules.
  `ListCheckRules`, `ListRecordingRules`, and `ListSLOs` return an error when given one of the new filters.

  * test(signal-to-metrics): check test, import, and count against the live API

  TestLiveAPI_SignalToMetricsTestImportAndCount runs TestSignalToMetrics
  on a valid rule and on one with a 1s interval, which the API rejects
  with a 400. It imports a spans rule and a logs rule twice, expecting 2
  created and then 2 updated. It then checks each list filter (origin
  prefix, signal, name, and enabled state) and CountSignalToMetrics
  against those 2 rules.

  The test runs only when DASH0_API_URL and DASH0_AUTH_TOKEN are set, like
  TestLiveAPI. The imported rules are disabled, and the test deletes them
  when it finishes.

  * fix(list-options): reject any filter other than origin prefix, and fix the option docs

  The ListOption doc said methods without a filter do not accept options,
  but ListCheckRules, ListRecordingRules, and ListSLOs accept them and
  return an error. The doc now says that.

  requireOriginPrefixOnly named each rejected filter, so a field added to
  ListOptions later would pass the check and be dropped. It now also
  rejects any ListOptions value with a field set other than OriginPrefix.
  A test sets each field in turn by reflection, so it covers new fields
  without a new case.

  WithName no longer describes the signal-to-metrics matching rule.
  ListSignalToMetrics documents it instead, so another endpoint can
  support WithName with its own matching.

  * docs(signal-to-metrics): name the fields the name filter searches

  The name filter matches spec.display.name or spec.output.name as a
  case-insensitive substring, or the rule ID in full, ignoring case. It
  never searches metadata.name. The ListSignalToMetrics doc now says so.

  * refactor(list-options): reject non-origin-prefix filters with one comparison

  requireOriginPrefixOnly named each rejected filter in a switch and then
  compared the options against a value that holds only OriginPrefix. The
  comparison already rejects every filter the switch named, so the switch
  is gone, and every rejection returns "supports only the origin prefix
  filter".

  The tests cover each filter alone and with an origin prefix, filters
  cleared by an empty value, and the operation name in the message. The
  reflection test now sets each field with and without an origin prefix.

  * refactor(list-options): rename WithName to WithSignalToMetricsName

  The signal-to-metrics name parameter is a search. It matches the display
  name or the output metric name as a case-insensitive substring, or the
  rule ID in full, and never metadata.name. Name means other fields on
  other resources, for example an exact variable name on agent0
  environment variables, so a generic WithName would match differently on
  each endpoint.

  The option and the ListOptions field carry the SignalToMetrics prefix,
  and the option documents the fields it searches. WithEnabled and
  WithSignal keep their names: enabled means the same on every resource,
  and the SignalToMetricsSignalType argument already ties WithSignal to
  signal-to-metrics.
- prefix the output type constants and test the new fields (#62)

  `SignalToMetricsOutput` has the optional fields `Type` (`SignalToMetricsOutputTypeSum`, `SignalToMetricsOutputTypeGauge` or `SignalToMetricsOutputTypeExponentialHistogram`), `Value`, `Monotonic`, `MaxSize`, and `Count`.
  A rule without `Type` keeps the default behavior: spans produce a duration histogram and logs a count.
  `SettingsPerOrganizationAndDatasetInfo` no longer has `ObservedPatterns` or `TelemetryTransformationRuleSettings`, and the types only they used are removed, because the edge settings response stopped sending them.

  * test(signal-to-metrics): check the output types against the live API

  The SignalToMetricsOutputTypes subtest of TestLiveAPI saves a disabled
  spans rule for each output type (sum, gauge, and exponential histogram)
  with a value expression, gets it back, and checks that the type and
  value survive the round trip.

  Saving a typed rule depends on a per-organization setting, so the
  subtest skips when the API rejects the rule with a "not enabled" error.
  It deletes the rules when it finishes.

  ---------
- add metadata helpers, nil guards, tests, and examples (#60)

  `CreateSamplingRule` and `UpdateSamplingRule` return an error instead of panicking when the rule is nil.

  * feat(sampling-rules): add metadata helpers

  Add StripSamplingRuleServerFields, GetSamplingRuleID,
  GetSamplingRuleName, SetSamplingRuleID, SetSamplingRuleIDIfAbsent, and
  ClearSamplingRuleID, the helper set every other asset type already has.
  All of them accept nil.

  Sampling rules carry no annotations, so the strip helper clears only the
  server-managed labels: version, source, dataset, and origin. It keeps the
  dash0.com/id label so a stripped rule still addresses the same rule on
  update, and it leaves spec.enabled and the custom labels alone.

  New helpers `StripSamplingRuleServerFields`, `GetSamplingRuleID`, `GetSamplingRuleName`, `SetSamplingRuleID`, `SetSamplingRuleIDIfAbsent`, and `ClearSamplingRuleID`.

  * test(sampling-rules): cover CRUD, error paths, and helpers

  Replace the single 201 create test with integration tests for every
  client method: request method, path, body, and the dataset query
  parameter, plus 201 and 204 success codes, 400, 403, and 404 errors, the
  nil-rule guards, and the unconfigured-API case. Each new helper gets a
  nil-safety test, and the strip test checks that spec.enabled survives.

  * docs(sampling-rules): add runnable examples

  Add examples for ListSamplingRules, CreateSamplingRule, and a
  get-strip-update round trip with UpdateSamplingRule, each against a local
  test server, plus one example per helper. All of them carry an Output
  comment, so go test verifies them.

  * feat(sampling-rules): add dataset helpers

  Add GetSamplingRuleDataset and SetSamplingRuleDataset. Every other asset
  type with a dash0.com/dataset label has this pair, so callers no longer
  need to read the label and write their own nil checks. Both accept nil,
  and each has tests and a runnable example.

  New helpers `GetSamplingRuleDataset` and `SetSamplingRuleDataset`.

  * test(sampling-rules): match the API error with errors.As

  A type assertion fails as soon as something wraps the *APIError, even
  though IsBadRequest still passes. Use errors.As, like the auth token
  tests.

  * test(sampling-rules): cover the dataset parameter in one table

  The five "sends the dataset query parameter" subtests differed only in
  the method they called, so a change to dataset handling needed the same
  edit in five places. One table-driven subtest now runs every method
  against the same assertion.

  * style(sampling-rules): one sentence per line in comments

  Reflow the test and example comments that wrapped mid-sentence, as the
  prose rules in CLAUDE.md ask.

  * test(sampling-rules): check the strip round trip against the live API

  TestLiveAPI_SamplingRules creates a disabled sampling rule, gets it,
  strips it, and updates it by the ID that Strip keeps. It checks that
  the update reaches the same rule, that GetSamplingRuleName follows the
  new display name, and that the rule stays disabled.

  The test runs only when DASH0_API_URL and DASH0_AUTH_TOKEN are set, like
  TestLiveAPI. The rule is disabled, and the test deletes it when it
  finishes.
- client manages the CloudWatch scrape config of an AWS integration (#65)

  `Client` gained `GetAwsMetricScrapeConfig`, `UpdateAwsMetricScrapeConfig`, and `DeleteAwsMetricScrapeConfig` for the CloudWatch metric scrape config of an AWS integration.

  * fix(aws-metric-scrape-configs): address review on the scrape config helpers

  - StripAwsMetricScrapeConfigServerFields also clears metadata.name, which the server sets and ignores on write.
  - UpdateAwsMetricScrapeConfig sends nil namespace and metric lists as [] instead of null, without modifying the caller's config.
  - Add GetAwsMetricScrapeConfigID and GetAwsMetricScrapeConfigName. There are no Set or Clear ID helpers, because the ID is the integration ID from the request path.
  - Pin the TopologyNodeKind constant names before they ship.
  - Add examples for the ID, name, and origin helpers.
  - Fail httptest handlers with t.Errorf instead of t.Fatalf.

  Refs SIG-835

## v1.27.0


### Bug Fixes
- clear deleted-at in StripCheckRuleServerFields (#57)

  `StripCheckRuleServerFields` now clears the `dash0.com/deleted-at` annotation, like the strip helpers for views, synthetic checks, and SLOs.
  When this leaves `Metadata.Annotations` or `Metadata` empty, it sets them to nil.

## v1.26.0


### Features
- typed AuthTokenAction enum and refreshed spec docs (#56)

  `AuthTokenSpec.Actions` and `AuthTokenResponseSpec.Actions` are now `*[]AuthTokenAction` instead of `*[]Action`.
  This breaks Go callers that build actions from plain strings: use the `AuthTokenAction` constants or convert with `AuthTokenAction(s)`.
- optional spec.slug and plain-array list responses (#55)

  `ListDatasets` and `ListAuthTokens` read the new list wire shape, a plain JSON array of full documents instead of `{datasets: [...]}` and `{authTokens: [...]}`.
  `DatasetListResponse` and `AuthTokenListResponse` are now slice types.
  `DatasetSpec.Slug` is a new optional field; when neither it nor the `dash0.com/id` label is set, create derives the slug from `metadata.name`.

  * fix(datasets): treat spec.slug as the slug in the dataset helpers

  GetDatasetID prefers spec.slug, SetDatasetID writes both spec.slug and
  the dash0.com/id label, SetDatasetIDIfAbsent checks both, and
  ClearDatasetID clears both. Before, a copy kept the source spec.slug, and
  IfAbsent could write a label that did not match spec.slug.

  `ListDatasets` and `ListAuthTokens` read the new list wire shape, a plain JSON array of full documents instead of `{datasets: [...]}` and `{authTokens: [...]}`.
  `DatasetListResponse` and `AuthTokenListResponse` are now slice types.
  `DatasetSpec.Slug` is a new optional field; when neither it nor the `dash0.com/id` label is set, create derives the slug from `metadata.name`.
  `GetDatasetID`, `SetDatasetID`, `SetDatasetIDIfAbsent`, and `ClearDatasetID` treat `spec.slug` and the `dash0.com/id` label as one slug.

## v1.25.0


### Features
- add auth token management to the client (#53)
- follow the new dataset document shape (#54)

## v1.24.0


### Bug Fixes
- ToJSON follows YAML 1.2 for keys and values (#51)

  `yaml.ToJSON` now follows YAML 1.2 for keys and values, so plain `y`, `n`, `yes`, `no`, `on`, and `off` stay strings everywhere.
  Before, a value such as `name: off` became `false`, and `[GB, NO]` became `["GB", false]`.
  Only `true` and `false` (in any case) are booleans.
  A mapping key that is not a string, such as `true: a`, is now an error.
  `yaml.Unmarshal` is not changed: it still reads these words as booleans when they are values, so `enabled: yes` stays `true`.

### Features
- add dataset management to the client (#52)

## v1.23.0


### Bug Fixes
- keep unquoted y/n/yes/no/on/off as strings (#50)
- preserve anomaly detector settings in check rules (#47)

### Features
- add signal-to-metrics wrappers and originPrefix filter on List* (#46)
- add optional member role support to client responses (#43)

## v1.22.0


### Features
- add CRUD, iterator, and metadata helpers (#42)

## v1.21.1


### Features
- add an option to retry 409 dataset version conflicts

### Refactoring
- drop the boolean from the conflict retry options

## v1.21.0


### Bug Fixes
- resolve the auth token per request via an AuthTokenProvider (#33)
- backfills empty client_id from DCR cache if not persisted in profile, best effort

### Features
- add semantic YAML/JSON comparison engine (#32)
- recover from a 401 by forcing a token refresh and replaying once (#35)
- add a per-name profile accessor (#34)

## v1.20.0


### Features
- export MergeAnnotations (#30)

## v1.19.1


### Bug Fixes
- merge top-level PrometheusRule annotations and validate thresholds (#29)

## v1.19.0


### Features
- add GetSLOOrigin helper (#28)

## v1.18.2


### Bug Fixes
- strip dash0.com/id in StripSLOServerFields

### CI
- drop Go proxy indexing step

## v1.18.1


### Bug Fixes
- merge env vars over the active profile per field (#26)
- strip read-only spec.routing.assets on write (#25)

## v1.18.1


### Bug Fixes
- merge env vars over the active profile per field (#26)
- strip read-only spec.routing.assets on write (#25)

## v1.18.0


### Features
- declarative CRD support for the Dash0 SLO API (#20)

## v1.17.1


### Bug Fixes
- index members by metadata.name in ResolveMemberIDsToEmails

## v1.17.0


### Features
- declarative CRD support for the Dash0 Teams API (#21)

## v1.16.0


### Features
- add APIs to retrieve failed checks

## v1.15.0


### Features
- add OAuthClient for unauthenticated OAuth 2.0 endpoints

## v1.14.0


### Features
- abort retries when Retry-After > max wait time

## v1.13.1


### Bug Fixes
- expose `dataset` attribute for deep link URLs

## Unreleased


### Bug Fixes
- scope deep link URLs to a dataset and add notification channel asset type

### Breaking
- `DeeplinkURL` and `ViewDeeplinkURL` gained a trailing `dataset *string` parameter; pass `nil` to preserve the previous behavior

## v1.13.0


### Bug Fixes
- rename generated PrometheusResultType String const

### Features
- add Dash0 web app deep link builders

## v1.12.3


### Bug Fixes
- compat updates
- normalize API groups for spam filters

## v1.12.2


### Bug Fixes
- cope with errors that are represented as strings

## v1.12.1


### Bug Fixes
- parse nested API error shape and preserve trace IDs

### Testing
- add integration tests for nested API error parsing

## v1.12.0


### Features
- spam filter support

## v1.11.1


### Bug Fixes
- use the dataset parameter for /api/recording-rules

## v1.11.0


## v1.9.1


### Bug Fixes
- accept both 200 and 201 for notification channel creation

## v1.9.0


### Features
- add support for notification channels

### Breaking
- remove the proprietary RecordingRuleGroup API

## v1.8.0


### Features
- expose transport stack for composable use with raw HTTP clients

## v1.7.0


### Features
- introduce profile management

## v1.6.0


### ENG-7866
- Add recording rules (#6)

### Features
- extract reusable capabilities from CLI and terraform provider, and introduce yaml subpackage (#7)

## v1.5.1


### Bug Fixes
- extract error message from JSON body when Message is not set

## v1.5.0
- updates to ViewApiListItem and SyntheticCheckAttempt

## v1.4.1
- fix wrong status-code expectations for AddTeamMembers

## v1.4.0
- add support for member and team management

## v1.1.0
- add sampling rules CRUD support

## v1.0.0
- Initial release
