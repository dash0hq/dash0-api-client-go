package profiles

// This file keeps deprecated symbols and documents migration steps for
// breaking changes to the profiles package.
//
// # Migration guide (v1.33.x to v1.34.0)
//
// [Configuration] is now only the schema of one profile in profiles.json.
// [Store.GetActiveConfiguration], [Store.GetActiveConfigurationContext],
// [Store.GetConfigurationForProfile], [ResolveConfiguration],
// [ResolveConfigurationContext], [ResolveConfigurationWithOtlp],
// [ResolveConfigurationWithOtlpContext], [WithConfiguration], and
// [FromContext] use the new [ResolvedConfiguration] instead.
// Read its values through methods: cfg.ApiUrl becomes cfg.ApiUrl(), and the
// same for OtlpUrl, Dataset, ProfileName, and ConfigDir.
// Only a [Store] and the ResolveConfiguration functions build a
// [ResolvedConfiguration], so a configuration with OAuth state always knows
// the profile its refreshed tokens are written back to.
//
// Configuration.ProfileName and Configuration.ConfigDir were removed.
// Use [ResolvedConfiguration.ProfileName] and [ResolvedConfiguration.ConfigDir].
//
// Configuration.ClientOptions and Configuration.AuthTokenProvider moved to
// [ResolvedConfiguration.ClientOptions] and
// [ResolvedConfiguration.AuthTokenProvider].
// To build a client from a profile you hold as a [Configuration], resolve it
// with [Store.GetConfigurationForProfile] first.
//
// The OAuth access token moved from [Configuration.AuthToken] to
// [OAuthState.AccessToken], so [Configuration.AuthToken] only holds a static
// auth token.
// Read the credential with [ResolvedConfiguration.Token], which returns the
// access token for an OAuth profile and the static token otherwise.
// Code that writes an OAuth login with [Store.AddProfile] or
// [Store.UpdateProfile] sets [OAuthState.AccessToken].
// A [Store] migrates existing profiles.json files when it reads them.
// Until the release after v1.34.0, it also writes a copy of the access
// token to "authToken", so an older CLI or Terraform provider can still read a
// migrated file.
//
// [ResolvedConfiguration.HasCredentials] and [Configuration.HasCredentials]
// now report false for an OAuth profile with no access token, such as one
// that was created but never logged in.
//
// [ErrNoAssociatedProfile] is no longer returned, because a configuration
// with OAuth state can no longer be built without its profile.

import "errors"

// ErrNoAssociatedProfile was returned when an OAuth access token needed
// refreshing but the configuration was not associated with a stored profile.
//
// Deprecated: since v1.34.0. Never returned, because only a [Store]
// builds a [ResolvedConfiguration] with OAuth state.
var ErrNoAssociatedProfile = errors.New(
	"the credentials are not associated with a Dash0 profile, so the OAuth access token " +
		"cannot be refreshed",
)
