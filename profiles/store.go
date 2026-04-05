package profiles

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	// ConfigDirName is the name of the directory containing Dash0 configuration.
	ConfigDirName = ".dash0"
	// ProfilesFileName is the name of the file containing profile configurations.
	ProfilesFileName = "profiles.json"
	// ActiveProfileFileName is the name of the file containing the active profile name.
	ActiveProfileFileName = "activeProfile"

	// EnvApiUrl is the environment variable for the Dash0 API URL.
	EnvApiUrl = "DASH0_API_URL"
	// EnvAuthToken is the environment variable for the Dash0 auth token.
	EnvAuthToken = "DASH0_AUTH_TOKEN"
	// EnvOtlpUrl is the environment variable for the Dash0 OTLP URL.
	EnvOtlpUrl = "DASH0_OTLP_URL"
	// EnvDataset is the environment variable for the Dash0 dataset.
	EnvDataset = "DASH0_DATASET"
	// EnvConfigDir is the environment variable that overrides the default
	// configuration directory.
	EnvConfigDir = "DASH0_CONFIG_DIR"
)

var (
	// ErrNoActiveProfile is returned when there is no active profile.
	ErrNoActiveProfile = errors.New("no active profile configured")
	// ErrProfileNotFound is returned when a requested profile is not found.
	ErrProfileNotFound = errors.New("profile not found")
)

// Store handles profile storage and retrieval.
type Store struct {
	configDir string
}

// NewStore creates a new profile store.
//
// The configuration directory is resolved in this order:
//  1. Explicit [WithConfigDir] option (if provided).
//  2. The DASH0_CONFIG_DIR environment variable (if set).
//  3. ~/.dash0/ (default).
func NewStore(opts ...StoreOption) (*Store, error) {
	cfg := &storeConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.configDir != "" {
		return &Store{configDir: cfg.configDir}, nil
	}

	if configDir := os.Getenv(EnvConfigDir); configDir != "" {
		return &Store{configDir: configDir}, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	return &Store{configDir: filepath.Join(homeDir, ConfigDirName)}, nil
}

// GetActiveConfiguration returns the currently active configuration.
// Environment variables take precedence over the active profile.
func (s *Store) GetActiveConfiguration() (*Configuration, error) {
	envApiUrl := os.Getenv(EnvApiUrl)
	envAuthToken := os.Getenv(EnvAuthToken)
	envOtlpUrl := os.Getenv(EnvOtlpUrl)
	envDataset := os.Getenv(EnvDataset)

	// If auth token and at least one URL are set via env vars, use them
	// directly without requiring a profile.
	if envAuthToken != "" && (envApiUrl != "" || envOtlpUrl != "") {
		return &Configuration{
			ApiUrl:    envApiUrl,
			AuthToken: envAuthToken,
			OtlpUrl:   envOtlpUrl,
			Dataset:   envDataset,
		}, nil
	}

	// Otherwise, start with the active profile.
	activeProfile, err := s.GetActiveProfile()
	if err != nil {
		return nil, err
	}

	activeConfiguration := &activeProfile.Configuration

	// Override with env vars if set.
	if envApiUrl != "" {
		activeConfiguration.ApiUrl = envApiUrl
	}
	if envAuthToken != "" {
		activeConfiguration.AuthToken = envAuthToken
	}
	if envOtlpUrl != "" {
		activeConfiguration.OtlpUrl = envOtlpUrl
	}
	if envDataset != "" {
		activeConfiguration.Dataset = envDataset
	}

	return activeConfiguration, nil
}

// ResolveConfiguration loads the active profile, applies environment variable
// overrides, then applies the given parameter overrides on top.
// Non-empty parameters take highest precedence.
//
// This is a convenience function that creates a temporary [Store] internally.
// Pass [StoreOption] values (e.g. [WithConfigDir]) to control how the
// profile store is constructed.
func ResolveConfiguration(apiUrl, authToken string, opts ...StoreOption) (*Configuration, error) {
	return ResolveConfigurationWithOtlp(apiUrl, authToken, "", "", opts...)
}

// ResolveConfigurationWithOtlp is like [ResolveConfiguration] but also accepts
// OTLP URL and dataset overrides.
func ResolveConfigurationWithOtlp(apiUrl, authToken, otlpUrl, dataset string, opts ...StoreOption) (*Configuration, error) {
	result := &Configuration{}

	// Try to get the active configuration for defaults.
	var configErr error
	store, err := NewStore(opts...)
	if err == nil {
		cfg, err := store.GetActiveConfiguration()
		if err != nil {
			// Store the error but don't return yet -- we might have explicit overrides.
			configErr = err
		} else if cfg != nil {
			result.ApiUrl = cfg.ApiUrl
			result.AuthToken = cfg.AuthToken
			result.OtlpUrl = cfg.OtlpUrl
			result.Dataset = cfg.Dataset
		}
	}

	// Override with environment variables (in case GetActiveConfiguration failed
	// before it could apply them, e.g. no active profile but env vars are set).
	if envApiUrl := os.Getenv(EnvApiUrl); envApiUrl != "" && result.ApiUrl == "" {
		result.ApiUrl = envApiUrl
	}
	if envAuthToken := os.Getenv(EnvAuthToken); envAuthToken != "" && result.AuthToken == "" {
		result.AuthToken = envAuthToken
	}
	if envOtlpUrl := os.Getenv(EnvOtlpUrl); envOtlpUrl != "" && result.OtlpUrl == "" {
		result.OtlpUrl = envOtlpUrl
	}
	if envDataset := os.Getenv(EnvDataset); envDataset != "" && result.Dataset == "" {
		result.Dataset = envDataset
	}

	// Override with explicit parameters if provided.
	if apiUrl != "" {
		result.ApiUrl = apiUrl
	}
	if authToken != "" {
		result.AuthToken = authToken
	}
	if otlpUrl != "" {
		result.OtlpUrl = otlpUrl
	}
	if dataset != "" {
		result.Dataset = dataset
	}

	// If we had a config error and don't have complete configuration from
	// overrides, return the error.
	if configErr != nil && (result.AuthToken == "" || (result.ApiUrl == "" && result.OtlpUrl == "")) {
		return nil, configErr
	}

	// Final validation.
	if result.AuthToken == "" {
		return nil, fmt.Errorf("auth token is required; provide it as a parameter or configure a profile")
	}
	if result.ApiUrl == "" && result.OtlpUrl == "" {
		return nil, fmt.Errorf("at least one of API URL or OTLP URL is required; provide them as parameters or configure a profile")
	}

	return result, nil
}

// GetActiveProfile returns the currently active profile.
func (s *Store) GetActiveProfile() (*Profile, error) {
	activeProfileName, err := s.getActiveProfileName()
	if err != nil {
		return nil, err
	}

	profiles, err := s.GetProfiles()
	if err != nil {
		return nil, err
	}

	for _, profile := range profiles {
		if profile.Name == activeProfileName {
			return &profile, nil
		}
	}

	return nil, ErrProfileNotFound
}

// GetProfiles returns all available profiles.
func (s *Store) GetProfiles() ([]Profile, error) {
	profilesFilePath := filepath.Join(s.configDir, ProfilesFileName)
	if _, err := os.Stat(profilesFilePath); os.IsNotExist(err) {
		return []Profile{}, nil
	}

	data, err := os.ReadFile(profilesFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read profiles file: %w", err)
	}

	var profilesFile ProfilesFile
	if err := json.Unmarshal(data, &profilesFile); err != nil {
		return nil, fmt.Errorf(
			"failed to parse profiles file %s: %w\nHint: delete the file and recreate your profiles",
			profilesFilePath,
			err,
		)
	}

	return profilesFile.Profiles, nil
}

// AddProfile adds a new profile to the configuration.
// If a profile with the same name already exists, it is replaced.
// When adding the first profile, it is automatically set as active.
func (s *Store) AddProfile(profile Profile) error {
	profiles, err := s.GetProfiles()
	if err != nil {
		return err
	}

	// Check if profile already exists.
	for i, existing := range profiles {
		if existing.Name == profile.Name {
			profiles[i] = profile
			return s.saveProfiles(profiles)
		}
	}

	profiles = append(profiles, profile)

	err = s.saveProfiles(profiles)
	if err != nil {
		return err
	}

	// If this is the first profile, make it active.
	if len(profiles) == 1 {
		if err := s.SetActiveProfile(profile.Name); err != nil {
			return err
		}
	}

	return nil
}

// UpdateProfile finds a profile by name and applies updateFn to its
// configuration, then saves.
// Returns an error if no profile with the given name exists.
func (s *Store) UpdateProfile(name string, updateFn func(*Configuration)) error {
	profiles, err := s.GetProfiles()
	if err != nil {
		return err
	}

	found := false
	for i, profile := range profiles {
		if profile.Name == name {
			updateFn(&profiles[i].Configuration)
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("profile %q not found", name)
	}

	return s.saveProfiles(profiles)
}

// RemoveProfile removes a profile from the configuration.
// If the removed profile was the active profile, the first remaining profile
// becomes active.
func (s *Store) RemoveProfile(profileName string) error {
	profiles, err := s.GetProfiles()
	if err != nil {
		return err
	}

	var found bool
	var newProfiles []Profile
	for _, profile := range profiles {
		if profile.Name != profileName {
			newProfiles = append(newProfiles, profile)
		} else {
			found = true
		}
	}

	if !found {
		return ErrProfileNotFound
	}

	// Check if removing the active profile.
	activeProfileName, err := s.getActiveProfileName()
	if err == nil && activeProfileName == profileName {
		if len(newProfiles) > 0 {
			if err := s.SetActiveProfile(newProfiles[0].Name); err != nil {
				return err
			}
		} else {
			if err := os.Remove(filepath.Join(s.configDir, ActiveProfileFileName)); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove active profile file: %w", err)
			}
		}
	}

	return s.saveProfiles(newProfiles)
}

// SetActiveProfile sets the active profile by name.
// Returns [ErrProfileNotFound] if no profile with the given name exists.
func (s *Store) SetActiveProfile(profileName string) error {
	profiles, err := s.GetProfiles()
	if err != nil {
		return err
	}

	var found bool
	for _, profile := range profiles {
		if profile.Name == profileName {
			found = true
			break
		}
	}

	if !found {
		return ErrProfileNotFound
	}

	if err := os.MkdirAll(s.configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	activeProfilePath := filepath.Join(s.configDir, ActiveProfileFileName)
	if err := os.WriteFile(activeProfilePath, []byte(profileName), 0644); err != nil {
		return fmt.Errorf("failed to write active profile: %w", err)
	}

	return nil
}

func (s *Store) getActiveProfileName() (string, error) {
	activeProfilePath := filepath.Join(s.configDir, ActiveProfileFileName)
	if _, err := os.Stat(activeProfilePath); os.IsNotExist(err) {
		return "", ErrNoActiveProfile
	}

	data, err := os.ReadFile(activeProfilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read active profile: %w", err)
	}

	activeProfileName := string(data)
	if activeProfileName == "" {
		return "", ErrNoActiveProfile
	}

	return activeProfileName, nil
}

func (s *Store) saveProfiles(profiles []Profile) error {
	if err := os.MkdirAll(s.configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	profilesFile := ProfilesFile{Profiles: profiles}
	data, err := json.MarshalIndent(profilesFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal profiles: %w", err)
	}

	profilesFilePath := filepath.Join(s.configDir, ProfilesFileName)
	if err := os.WriteFile(profilesFilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write profiles file: %w", err)
	}

	return nil
}
