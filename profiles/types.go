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
	// Potentially refreshed when close to the expiry. The actual OAuth access token
	// is located within the AuthToken field.
	OAuth *OAuthState `json:"oAuth,omitempty"`
}

type OAuthState struct {
	ClientID     string    `json:"clientId"`
	RefreshToken string    `json:"refreshToken"`
	ExpiresAt    time.Time `json:"expiresAt"`
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
