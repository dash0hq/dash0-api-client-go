// Package profiles manages named Dash0 configuration profiles.
//
// Profiles are stored in ~/.dash0/profiles.json as named sets of connection
// parameters (API URL, auth token, OTLP URL, dataset).
// One profile is marked as active in ~/.dash0/activeProfile and used as the
// default configuration.
// Environment variables (DASH0_API_URL, DASH0_AUTH_TOKEN, DASH0_OTLP_URL,
// DASH0_DATASET) override the active profile.
//
// This package imports the root dash0 package for [dash0.ClientOption] and
// [dash0.DatasetPtr] -- never the reverse.
package profiles

import (
	"time"

	dash0 "github.com/dash0hq/dash0-api-client-go"
)

// Configuration is the schema of one profile in profiles.json.
//
// It is what [Store.AddProfile] and [Store.UpdateProfile] read and write.
// To call the API, resolve a [ResolvedConfiguration] through a [Store] or one
// of the ResolveConfiguration functions instead.
type Configuration struct {
	ApiUrl string `json:"apiUrl"`
	// AuthToken is a static auth token.
	// For an OAuth profile it is a copy of [OAuthState.AccessToken], kept until
	// the release after v1.34.0 so code that reads or writes the access token
	// here keeps working.
	AuthToken string `json:"authToken"`
	OtlpUrl   string `json:"otlpUrl,omitempty"`
	Dataset   string `json:"dataset,omitempty"`

	// OAuth is set when the profile authorizes with an OAuth access token.
	// Refreshed per request by [ResolvedConfiguration.AuthTokenProvider], never on load.
	OAuth *OAuthState `json:"oauth,omitempty"`
}

// OAuthState carries the OAuth-specific state that must survive across CLI
// invocations: the dynamic client identifier issued at registration (RFC 7591),
// the current access token, the long-lived refresh token used to mint new
// access tokens, and the access token's expiry deadline.
type OAuthState struct {
	ClientID     string    `json:"clientId,omitempty"`
	AccessToken  string    `json:"accessToken,omitempty"`
	RefreshToken string    `json:"refreshToken,omitempty"`
	ExpiresAt    time.Time `json:"expiresAt,omitzero"`
}

// HasCredentials reports whether the profile holds a static auth token or an
// OAuth access token.
func (cfg *Configuration) HasCredentials() bool {
	if cfg == nil {
		return false
	}
	return cfg.AuthToken != "" || (cfg.OAuth != nil && cfg.OAuth.AccessToken != "")
}

// DatasetPtr returns the dataset as a *string suitable for Dash0 API calls.
// It returns nil for empty strings and "default", matching [dash0.DatasetPtr].
func (cfg *Configuration) DatasetPtr() *string {
	if cfg == nil {
		return nil
	}
	return dash0.DatasetPtr(cfg.Dataset)
}

// migrateAccessToken moves the OAuth access token from AuthToken, where
// releases before v1.34.0 stored it, into [OAuthState.AccessToken], and keeps
// AuthToken as a copy so older readers of the field and of profiles.json
// still find it.
//
// Remove the copy in the release after v1.34.0.
func migrateAccessToken(cfg *Configuration) {
	if cfg.OAuth == nil {
		return
	}
	if cfg.OAuth.AccessToken == "" {
		cfg.OAuth.AccessToken = cfg.AuthToken
	}
	cfg.AuthToken = cfg.OAuth.AccessToken
}

// syncAccessTokenCopy copies the access token from whichever of AuthToken and
// [OAuthState.AccessToken] an update changed into the other.
// Callers written before v1.34.0 change AuthToken, newer ones change
// AccessToken, and without this the field left alone would win on save, so a
// new token or a cleared one ("") would be lost.
func syncAccessTokenCopy(cfg *Configuration, authTokenBefore, accessTokenBefore string) {
	if cfg.OAuth == nil {
		return
	}
	authTokenChanged := cfg.AuthToken != authTokenBefore
	accessTokenChanged := cfg.OAuth.AccessToken != accessTokenBefore
	switch {
	case authTokenChanged && !accessTokenChanged:
		cfg.OAuth.AccessToken = cfg.AuthToken
	case accessTokenChanged && !authTokenChanged:
		cfg.AuthToken = cfg.OAuth.AccessToken
	}
}

// ResolvedConfiguration is a configuration resolved from a profile,
// environment variables, and caller arguments.
//
// Only a [Store] and the ResolveConfiguration functions build one, so a
// configuration with OAuth state always knows the profile its refreshed tokens
// are written back to.
// The zero value has no credentials.
// All methods are safe to call on a nil pointer.
type ResolvedConfiguration struct {
	values      Configuration
	profileName string
	configDir   string
	sources     [fieldCount]ConfigurationSource
}

// ConfigurationField names a field of a [ResolvedConfiguration] for [ResolvedConfiguration.Source].
type ConfigurationField int

const (
	// FieldApiUrl is the field returned by [ResolvedConfiguration.ApiUrl].
	FieldApiUrl ConfigurationField = iota
	// FieldAuthToken is the field returned by [ResolvedConfiguration.Token].
	FieldAuthToken
	// FieldOtlpUrl is the field returned by [ResolvedConfiguration.OtlpUrl].
	FieldOtlpUrl
	// FieldDataset is the field returned by [ResolvedConfiguration.Dataset].
	FieldDataset

	fieldCount
)

// ConfigurationSource is the layer that supplied the value of a [ConfigurationField].
type ConfigurationSource string

const (
	// SourceUnset means no layer supplied a value and the field is empty.
	SourceUnset ConfigurationSource = ""
	// SourceProfile means the value came from a profile in profiles.json.
	SourceProfile ConfigurationSource = "profile"
	// SourceEnvironment means the value came from an environment variable, such as DASH0_API_URL.
	SourceEnvironment ConfigurationSource = "environment"
	// SourceArgument means the value came from an argument the caller passed to a ResolveConfiguration function.
	SourceArgument ConfigurationSource = "argument"
)

// Source returns the layer that supplied the value of field.
// It returns [SourceUnset] when the field is empty or field is unknown.
func (cfg *ResolvedConfiguration) Source(field ConfigurationField) ConfigurationSource {
	if cfg == nil || field < 0 || field >= fieldCount {
		return SourceUnset
	}
	return cfg.sources[field]
}

// recordProfileSources marks every non-empty field as coming from the profile.
func (cfg *ResolvedConfiguration) recordProfileSources() {
	for field, value := range map[ConfigurationField]string{
		FieldApiUrl:    cfg.values.ApiUrl,
		FieldAuthToken: cfg.Token(),
		FieldOtlpUrl:   cfg.values.OtlpUrl,
		FieldDataset:   cfg.values.Dataset,
	} {
		if value != "" {
			cfg.sources[field] = SourceProfile
		}
	}
}

// ApiUrl returns the API URL.
func (cfg *ResolvedConfiguration) ApiUrl() string {
	if cfg == nil {
		return ""
	}
	return cfg.values.ApiUrl
}

// OtlpUrl returns the OTLP URL.
func (cfg *ResolvedConfiguration) OtlpUrl() string {
	if cfg == nil {
		return ""
	}
	return cfg.values.OtlpUrl
}

// Dataset returns the dataset.
func (cfg *ResolvedConfiguration) Dataset() string {
	if cfg == nil {
		return ""
	}
	return cfg.values.Dataset
}

// DatasetPtr returns the dataset as a *string suitable for Dash0 API calls.
// It returns nil for empty strings and "default", matching [dash0.DatasetPtr].
func (cfg *ResolvedConfiguration) DatasetPtr() *string {
	return dash0.DatasetPtr(cfg.Dataset())
}

// Token returns the credential as resolved: the OAuth access token for an
// OAuth profile, otherwise the static auth token.
// An OAuth access token may be expired.
// To authenticate requests, use [ResolvedConfiguration.AuthTokenProvider],
// which refreshes it.
func (cfg *ResolvedConfiguration) Token() string {
	if cfg == nil {
		return ""
	}
	if cfg.values.OAuth != nil {
		return cfg.values.OAuth.AccessToken
	}
	return cfg.values.AuthToken
}

// OAuth returns a copy of the OAuth state, or nil when the configuration uses a
// static auth token.
func (cfg *ResolvedConfiguration) OAuth() *OAuthState {
	if cfg == nil || cfg.values.OAuth == nil {
		return nil
	}
	oauthState := *cfg.values.OAuth
	return &oauthState
}

// ProfileName returns the name of the profile the configuration was resolved
// from, or "" when it came from environment variables or caller arguments
// alone.
func (cfg *ResolvedConfiguration) ProfileName() string {
	if cfg == nil {
		return ""
	}
	return cfg.profileName
}

// ConfigDir returns the directory of the profiles.json the configuration was
// resolved from, or "" when [ResolvedConfiguration.ProfileName] is "".
func (cfg *ResolvedConfiguration) ConfigDir() string {
	if cfg == nil {
		return ""
	}
	return cfg.configDir
}

// HasCredentials reports whether [ResolvedConfiguration.Token] is non-empty.
func (cfg *ResolvedConfiguration) HasCredentials() bool {
	return cfg.Token() != ""
}

func (cfg *ResolvedConfiguration) canRefresh() bool {
	return cfg != nil && cfg.values.OAuth != nil && cfg.values.OAuth.RefreshToken != ""
}

// Profile represents a named configuration profile.
type Profile struct {
	Name          string        `json:"name"`
	Configuration Configuration `json:"configuration"`
}

// ProfilesFile represents the file storing multiple profiles.
type ProfilesFile struct {
	Profiles []Profile `json:"profiles"`
}
