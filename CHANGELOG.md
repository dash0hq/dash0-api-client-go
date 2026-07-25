# Changelog

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
