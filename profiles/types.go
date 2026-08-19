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

// Configuration represents a Dash0 configuration with connection parameters.
type Configuration struct {
	ApiUrl    string `json:"apiUrl"`
	AuthToken string `json:"authToken"`
	OtlpUrl   string `json:"otlpUrl,omitempty"`
	Dataset   string `json:"dataset,omitempty"`

	// OAuth is set when the profile should authorize using an OAuth access token.
	// Potentially refreshed when close to the expiry.
	// The actual OAuth access token is located within the AuthToken field.
	OAuth *OAuthState `json:"oauth,omitempty"`

	// ProfileName and ConfigDir record where this configuration was resolved
	// from, and are both empty when it came from environment variables or
	// explicit parameters alone.
	//
	// Refreshing an OAuth access token needs both: ProfileName locates the
	// profile the rotated tokens are written back to, and ConfigDir says which
	// profiles.json holds it. The store sets them when it resolves a
	// configuration, so callers normally do not.
	// Set them together or not at all; a name without a directory refreshes
	// against whichever profiles.json the environment happens to point at. See
	// [Configuration.AuthTokenProvider].
	//
	// Neither is serialized. ProfileName would duplicate the key its [Profile]
	// is stored under, and ConfigDir is the directory the file itself lives in.
	ProfileName string `json:"-"`
	ConfigDir   string `json:"-"`
}

// OAuthState carries the OAuth-specific state that must survive across CLI
// invocations: the dynamic client identifier issued at registration (RFC 7591),
// the long-lived refresh token used to mint new access tokens, and the access
// token's expiry deadline.
// The current access token lives in [Configuration.AuthToken]; OAuthState only
// describes how to refresh it.
type OAuthState struct {
	ClientID     string    `json:"clientId,omitempty"`
	RefreshToken string    `json:"refreshToken,omitempty"`
	ExpiresAt    time.Time `json:"expiresAt,omitzero"`
}

// HasCredentials reports whether the configuration carries a credential of any
// kind, static or OAuth.
//
// It exists so callers do not have to know that an absent static token is an
// empty string while absent OAuth state is a nil pointer. Both representations
// are right for what they hold: encoding/json needs a pointer to tell an absent
// nested object from a zero one, and a string needs no such indirection. Asking
// one question keeps that difference in a single place instead of spreading a
// mixed `!= ""` and `!= nil` test across every caller.
func (cfg *Configuration) HasCredentials() bool {
	return cfg.AuthToken != "" || cfg.OAuth != nil
}

// DatasetPtr returns the dataset as a *string suitable for Dash0 API calls.
// It returns nil for empty strings and "default", matching [dash0.DatasetPtr].
func (cfg *Configuration) DatasetPtr() *string {
	return dash0.DatasetPtr(cfg.Dataset)
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
