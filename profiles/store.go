package profiles

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
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

// configDirMode is the permission mode used when creating the configuration
// directory.
// The profile file persists OAuth refresh tokens, so the directory is created
// owner-only (0700) to mirror the protection already applied to
// oauth-clients.json.
const configDirMode = 0o700

// configFileMode is the permission mode used when writing the profile file
// and the active-profile pointer.
// Owner-only (0600) because both files can be load-bearing credentials.
const configFileMode = 0o600

var (
	// ErrNoActiveProfile is returned when there is no active profile.
	ErrNoActiveProfile = errors.New("no active profile configured")
	// ErrProfileNotFound is returned when a requested profile is not found.
	ErrProfileNotFound = errors.New("profile not found")
)

// Store handles profile storage and retrieval.
//
// Cross-process refresh: Store serializes OAuth token refreshes both within a
// process (via an in-memory mutex) and across processes that share the same
// config directory (via an OS-level advisory lock on .profile-lock, backed
// by [github.com/gofrs/flock]).
type Store struct {
	configDir string

	// refreshMu serializes OAuth token refreshes so that concurrent callers
	// of [Store.GetActiveConfigurationContext] do not race against each
	// other and invalidate a rotated refresh token.
	// Cross-process serialization is layered on top via [acquireProfileLock].
	refreshMu sync.Mutex
}

// NewStore creates a new profile store.
//
// The configuration directory is resolved in this order:
//  1. Explicit [WithConfigDir] option (if provided).
//  2. The DASH0_CONFIG_DIR environment variable (if set).
//  3. ~/.dash0/ (default).
func NewStore(opts ...StoreOption) (*Store, error) {
	configDir, err := resolveConfigDir(opts)
	if err != nil {
		return nil, err
	}
	return &Store{configDir: configDir}, nil
}

// resolveConfigDir computes the configuration directory using the shared
// precedence rule: explicit [WithConfigDir] > DASH0_CONFIG_DIR env var > ~/.dash0/.
// Used by both [NewStore] and [NewOAuthClientStore] so the resolution stays in
// one place.
func resolveConfigDir(opts []StoreOption) (string, error) {
	cfg := &storeConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.configDir != "" {
		return cfg.configDir, nil
	}
	if d := os.Getenv(EnvConfigDir); d != "" {
		return d, nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(homeDir, ConfigDirName), nil
}

// GetActiveConfiguration returns the currently active configuration.
// Environment variables take precedence over the active profile.
// If the active profile uses OAuth, the access token is refreshed when close
// to expiry, using [context.Background] for the refresh request.
// Use [Store.GetActiveConfigurationContext] to plumb a cancellable context
// through the refresh.
func (s *Store) GetActiveConfiguration() (*Configuration, error) {
	return s.getActiveConfigurationContext(context.Background(), true)
}

// GetActiveConfigurationContext is the context-aware variant of
// [Store.GetActiveConfiguration].
// The ctx is propagated through the OAuth refresh round-trip so a hung
// authorization server does not pin the caller for the full HTTP timeout.
func (s *Store) GetActiveConfigurationContext(ctx context.Context) (*Configuration, error) {
	return s.getActiveConfigurationContext(ctx, true)
}

// getActiveConfigurationContext is the shared implementation behind
// [Store.GetActiveConfiguration] and [Store.GetActiveConfigurationContext].
// The refreshOAuth flag controls whether an OAuth-backed access token close
// to expiry is refreshed; callers that intend to override the auth token
// anyway can pass false to skip the refresh network call.
func (s *Store) getActiveConfigurationContext(ctx context.Context, refreshOAuth bool) (*Configuration, error) {
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

	// If the profile uses OAuth and no env var overrides the auth token,
	// refresh the access token when it is close to expiry.
	// Callers that will override the token anyway pass refreshOAuth=false to
	// skip this.
	if refreshOAuth && activeConfiguration.OAuth != nil && envAuthToken == "" {
		if err := refreshOAuthToken(ctx, s, activeProfile.Name, activeConfiguration); err != nil {
			return nil, err
		}
	}

	// Override with env vars if set.
	if envApiUrl != "" {
		activeConfiguration.ApiUrl = envApiUrl
	}
	if envAuthToken != "" {
		activeConfiguration.AuthToken = envAuthToken
		activeConfiguration.OAuth = nil
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
//
// OAuth refresh, if needed, uses [context.Background]; use
// [ResolveConfigurationContext] to plumb a cancellable context.
func ResolveConfiguration(apiUrl, authToken string, opts ...StoreOption) (*Configuration, error) {
	return ResolveConfigurationWithOtlpContext(context.Background(), apiUrl, authToken, "", "", opts...)
}

// ResolveConfigurationContext is the context-aware variant of
// [ResolveConfiguration].
func ResolveConfigurationContext(ctx context.Context, apiUrl, authToken string, opts ...StoreOption) (*Configuration, error) {
	return ResolveConfigurationWithOtlpContext(ctx, apiUrl, authToken, "", "", opts...)
}

// ResolveConfigurationWithOtlp is like [ResolveConfiguration] but also accepts
// OTLP URL and dataset overrides.
//
// OAuth refresh, if needed, uses [context.Background]; use
// [ResolveConfigurationWithOtlpContext] to plumb a cancellable context.
func ResolveConfigurationWithOtlp(apiUrl, authToken, otlpUrl, dataset string, opts ...StoreOption) (*Configuration, error) {
	return ResolveConfigurationWithOtlpContext(context.Background(), apiUrl, authToken, otlpUrl, dataset, opts...)
}

// ResolveConfigurationWithOtlpContext is the context-aware variant of
// [ResolveConfigurationWithOtlp].
func ResolveConfigurationWithOtlpContext(ctx context.Context, apiUrl, authToken, otlpUrl, dataset string, opts ...StoreOption) (*Configuration, error) {
	result := &Configuration{}

	// Try to get the active configuration for defaults.
	// Skip the OAuth refresh when the caller is about to override the auth
	// token -- the refreshed token would be discarded and the network call
	// wasted (and could rotate the refresh token for no good reason).
	refreshOAuth := authToken == ""
	var configErr error
	store, err := NewStore(opts...)
	if err == nil {
		cfg, err := store.getActiveConfigurationContext(ctx, refreshOAuth)
		if err != nil {
			// Store the error but don't return yet -- we might have explicit overrides.
			configErr = err
		} else if cfg != nil {
			result.ApiUrl = cfg.ApiUrl
			result.AuthToken = cfg.AuthToken
			result.OtlpUrl = cfg.OtlpUrl
			result.Dataset = cfg.Dataset
			result.OAuth = cfg.OAuth
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
		result.OAuth = nil
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
//
// The read-modify-write sequence is serialized cross-process via the
// .profile-lock sentinel; see [Store] for the locking model.
// Lock acquisition uses [context.Background]; callers that need to bound the
// wait should use [Store.AddProfileContext].
func (s *Store) AddProfile(profile Profile) error {
	return s.AddProfileContext(context.Background(), profile)
}

// AddProfileContext is the context-aware variant of [Store.AddProfile].
// The ctx bounds the wait to acquire the cross-process [.profile-lock].
func (s *Store) AddProfileContext(ctx context.Context, profile Profile) error {
	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()
	release, err := acquireProfileLock(ctx, s.configDir)
	if err != nil {
		return err
	}
	defer release()

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

	if err := s.saveProfiles(profiles); err != nil {
		return err
	}

	// If this is the first profile, make it active. Use the unlocked
	// helper because we already hold both locks.
	if len(profiles) == 1 {
		if err := s.setActiveProfileLocked(profile.Name); err != nil {
			return err
		}
	}

	return nil
}

// UpdateProfile finds a profile by name and applies updateFn to its
// configuration, then saves.
// Returns an error if no profile with the given name exists.
//
// The read-modify-write sequence is serialized cross-process via the
// .profile-lock sentinel; see [Store] for the locking model.
// Lock acquisition uses [context.Background]; callers that need to bound the
// wait should use [Store.UpdateProfileContext].
func (s *Store) UpdateProfile(name string, updateFn func(*Configuration)) error {
	return s.UpdateProfileContext(context.Background(), name, updateFn)
}

// UpdateProfileContext is the context-aware variant of [Store.UpdateProfile].
// The ctx bounds the wait to acquire the cross-process [.profile-lock].
func (s *Store) UpdateProfileContext(ctx context.Context, name string, updateFn func(*Configuration)) error {
	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()
	release, err := acquireProfileLock(ctx, s.configDir)
	if err != nil {
		return err
	}
	defer release()
	return s.updateProfileLocked(name, updateFn)
}

// updateProfileLocked is the unlocked body of [Store.UpdateProfile] for
// callers that already hold refreshMu and the .profile-lock file lock (notably
// [refreshOAuthToken]).
func (s *Store) updateProfileLocked(name string, updateFn func(*Configuration)) error {
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
// If the profile has OAuth state, the refresh token is revoked before removal,
// using [context.Background] for the revoke request.
// Use [Store.RemoveProfileContext] to plumb a cancellable context.
//
// Revocation is best-effort: if it fails, the local profile is still removed
// and the returned error wraps [ErrRevocationFailed].
// Callers can detect this with [errors.Is] and surface a "revoke manually"
// hint -- the refresh token may still be live on the authorization server.
//
// If the removed profile was the active profile, the first remaining profile
// becomes active.
func (s *Store) RemoveProfile(profileName string) error {
	return s.RemoveProfileContext(context.Background(), profileName)
}

// RemoveProfileContext is the context-aware variant of [Store.RemoveProfile].
func (s *Store) RemoveProfileContext(ctx context.Context, profileName string) error {
	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()
	release, err := acquireProfileLock(ctx, s.configDir)
	if err != nil {
		return err
	}
	defer release()

	profiles, err := s.GetProfiles()
	if err != nil {
		return err
	}

	var found bool
	var removedConfig Configuration
	var newProfiles []Profile
	for _, profile := range profiles {
		if profile.Name != profileName {
			newProfiles = append(newProfiles, profile)
		} else {
			found = true
			removedConfig = profile.Configuration
		}
	}

	if !found {
		return ErrProfileNotFound
	}

	// Best-effort: revoke OAuth tokens before removing the profile.
	// Capture the failure so it can be surfaced to the caller alongside the
	// (still-successful) local removal.
	revokeErr := revokeOAuthTokens(ctx, &removedConfig)

	// Persist the profiles list first so that, if the active-profile pointer
	// update fails below, on-disk state is at least consistent with the caller's
	// request (the profile is gone). Recovering a dangling active-profile
	// pointer is a softer failure mode than leaving a "removed" profile on disk.
	if err := s.saveProfiles(newProfiles); err != nil {
		return err
	}

	// If we just removed the active profile, point activeProfile at the first
	// remaining profile, or remove the pointer when no profiles remain.
	activeProfileName, err := s.getActiveProfileName()
	if err == nil && activeProfileName == profileName {
		if len(newProfiles) > 0 {
			if err := s.setActiveProfileLocked(newProfiles[0].Name); err != nil {
				return err
			}
		} else {
			if err := os.Remove(filepath.Join(s.configDir, ActiveProfileFileName)); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove active profile file: %w", err)
			}
		}
	}

	if revokeErr != nil {
		return fmt.Errorf("%w: %v", ErrRevocationFailed, revokeErr)
	}
	return nil
}

// SetActiveProfile sets the active profile by name.
// Returns [ErrProfileNotFound] if no profile with the given name exists.
//
// The change is serialized cross-process via the .profile-lock sentinel; see
// [Store] for the locking model.
// Lock acquisition uses [context.Background]; callers that need to bound the
// wait should use [Store.SetActiveProfileContext].
func (s *Store) SetActiveProfile(profileName string) error {
	return s.SetActiveProfileContext(context.Background(), profileName)
}

// SetActiveProfileContext is the context-aware variant of [Store.SetActiveProfile].
// The ctx bounds the wait to acquire the cross-process [.profile-lock].
func (s *Store) SetActiveProfileContext(ctx context.Context, profileName string) error {
	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()
	release, err := acquireProfileLock(ctx, s.configDir)
	if err != nil {
		return err
	}
	defer release()
	return s.setActiveProfileLocked(profileName)
}

// setActiveProfileLocked is the unlocked body of [Store.SetActiveProfile] for
// callers that already hold both locks.
func (s *Store) setActiveProfileLocked(profileName string) error {
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

	activeProfilePath := filepath.Join(s.configDir, ActiveProfileFileName)
	return writeFileAtomic(s.configDir, activeProfilePath, []byte(profileName))
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
	profilesFile := ProfilesFile{Profiles: profiles}
	data, err := json.MarshalIndent(profilesFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal profiles: %w", err)
	}

	profilesFilePath := filepath.Join(s.configDir, ProfilesFileName)
	return writeFileAtomic(s.configDir, profilesFilePath, data)
}

// writeFileAtomic writes data to path via a sibling temp file followed by
// os.Rename, so a crash mid-write leaves either the previous contents intact
// or the new contents fully written.
// The parent directory is created with [configDirMode] (0700) and the file
// with [configFileMode] (0600) — these files persist OAuth refresh tokens,
// so they are owner-only.
func writeFileAtomic(dir, path string, data []byte) error {
	if err := os.MkdirAll(dir, configDirMode); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file in %s: %w", dir, err)
	}
	tmpPath := tmp.Name()
	// Best-effort cleanup if anything below fails before the rename.
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	if err := tmp.Chmod(configFileMode); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to chmod temp file: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to rename %s to %s: %w", tmpPath, path, err)
	}
	cleanup = false
	return nil
}
