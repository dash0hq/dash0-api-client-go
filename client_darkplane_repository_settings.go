package dash0

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// darkplaneRepositoriesPageLimit is the largest page the repositories endpoint accepts.
const darkplaneRepositoriesPageLimit = 200

// ListDarkplaneRepositories returns every repository AutoMerge observes whose account/name slug contains search, case-insensitively.
// An empty search returns every repository.
func (c *client) ListDarkplaneRepositories(ctx context.Context, search string) ([]RepositoryDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	limit := darkplaneRepositoriesPageLimit
	params := &GetApiDarkplaneRepositoriesParams{Limit: &limit}
	if search != "" {
		params.Search = &search
	}
	var repositories []RepositoryDefinition
	for {
		resp, err := c.inner.GetApiDarkplaneRepositoriesWithResponse(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("dash0: list Darkplane repositories failed: %w", err)
		}
		if resp.StatusCode() != http.StatusOK {
			return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
		}
		if resp.JSON200 == nil {
			return nil, fmt.Errorf("dash0: unexpected nil response")
		}
		page := resp.JSON200.Items
		repositories = append(repositories, page...)
		next := resp.JSON200.Cursors.After
		if next == nil || *next == "" {
			return repositories, nil
		}
		if len(page) == 0 {
			return nil, fmt.Errorf("dash0: list Darkplane repositories failed: server returned a cursor with an empty page after %d repositories", len(repositories))
		}
		if params.Cursor != nil && *params.Cursor == *next {
			return nil, fmt.Errorf("dash0: list Darkplane repositories failed: server returned the same cursor twice after %d repositories", len(repositories))
		}
		params.Cursor = next
	}
}

// FindDarkplaneRepository returns the repository whose account/name slug equals slug, case-insensitively.
// It returns an *APIError with status 404 when no repository has that slug.
func (c *client) FindDarkplaneRepository(ctx context.Context, slug string) (*RepositoryDefinition, error) {
	repositories, err := c.ListDarkplaneRepositories(ctx, slug)
	if err != nil {
		return nil, err
	}
	for i := range repositories {
		if strings.EqualFold(GetDarkplaneRepositorySlug(&repositories[i]), slug) {
			return &repositories[i], nil
		}
	}
	return nil, &APIError{StatusCode: http.StatusNotFound, Message: fmt.Sprintf("repository %q not found", slug)}
}

// GetDarkplaneRepositorySettings retrieves the AutoMerge settings of a repository.
// A repository that was never configured returns the defaults rather than an error.
func (c *client) GetDarkplaneRepositorySettings(ctx context.Context, repositoryID string) (*RepositorySettingsDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	if repositoryID == "" {
		return nil, fmt.Errorf("dash0: repository ID is required")
	}
	resp, err := c.inner.GetApiDarkplaneRepositoriesRepositoryIdSettingsWithResponse(ctx, repositoryID)
	if err != nil {
		return nil, fmt.Errorf("dash0: get Darkplane repository settings failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return resp.JSON200, nil
}

// UpdateDarkplaneRepositorySettings replaces the whole AutoMerge settings spec of a repository and returns the stored settings.
// When settings carries a dash0.com/version label, the server rejects the write if a newer version was stored since.
// Without the label the write replaces whatever is stored.
func (c *client) UpdateDarkplaneRepositorySettings(ctx context.Context, repositoryID string, settings *RepositorySettingsDefinition) (*RepositorySettingsDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	if repositoryID == "" {
		return nil, fmt.Errorf("dash0: repository ID is required")
	}
	if settings == nil {
		return nil, fmt.Errorf("dash0: repository settings are required")
	}
	resp, err := c.inner.PutApiDarkplaneRepositoriesRepositoryIdSettingsWithResponse(ctx, repositoryID, toRepositorySettingsRequest(settings))
	if err != nil {
		return nil, fmt.Errorf("dash0: update Darkplane repository settings failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return resp.JSON200, nil
}

// ListDarkplaneRepositorySettingsVersions returns the stored versions of a repository's AutoMerge settings, newest first.
func (c *client) ListDarkplaneRepositorySettingsVersions(ctx context.Context, repositoryID string) ([]RepositorySettingsDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	if repositoryID == "" {
		return nil, fmt.Errorf("dash0: repository ID is required")
	}
	resp, err := c.inner.GetApiDarkplaneRepositoriesRepositoryIdSettingsVersionsWithResponse(ctx, repositoryID)
	if err != nil {
		return nil, fmt.Errorf("dash0: list Darkplane repository settings versions failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return resp.JSON200.Versions, nil
}

// toRepositorySettingsRequest keeps only what the PUT reads: the apiVersion, the spec and the
// dash0.com/version label. The apiVersion is forwarded so the server rejects a document written for
// a version it does not accept, rather than reading it as the default.
// The spec types protectedPaths and softRiskCriteria as non-nullable arrays, so nil is sent as [].
func toRepositorySettingsRequest(settings *RepositorySettingsDefinition) RepositorySettingsRequest {
	spec := settings.Spec
	if spec.ProtectedPaths == nil {
		spec.ProtectedPaths = []string{}
	}
	if spec.SoftRiskCriteria == nil {
		spec.SoftRiskCriteria = []SoftRiskCriterion{}
	}
	request := RepositorySettingsRequest{Kind: RepositorySettingsRequestKindDarkplaneRepositorySettings, Spec: spec}
	if settings.ApiVersion != "" {
		apiVersion := settings.ApiVersion
		request.ApiVersion = &apiVersion
	}
	if version := GetDarkplaneRepositorySettingsVersion(settings); version != "" {
		request.Metadata = &RepositorySettingsRequestMetadata{Labels: &ResourceLabels{Dash0Comversion: &version}}
	}
	return request
}

// GetDarkplaneRepositorySlug returns the account/name slug of a repository, such as "acme/api".
func GetDarkplaneRepositorySlug(repository *RepositoryDefinition) string {
	if repository == nil {
		return ""
	}
	return repository.Spec.Account + "/" + repository.Spec.Name
}

// GetDarkplaneRepositoryID returns the dash0.com/id label of a repository, or "" if absent.
func GetDarkplaneRepositoryID(repository *RepositoryDefinition) string {
	if repository == nil || repository.Metadata.Labels.Dash0Comid == nil {
		return ""
	}
	return *repository.Metadata.Labels.Dash0Comid
}

// GetDarkplaneRepositorySettingsID returns the dash0.com/id label of repository settings, which is the repository's ID, or "" if absent.
func GetDarkplaneRepositorySettingsID(settings *RepositorySettingsDefinition) string {
	if settings == nil || settings.Metadata.Labels.Dash0Comid == nil {
		return ""
	}
	return *settings.Metadata.Labels.Dash0Comid
}

// GetDarkplaneRepositorySettingsVersion returns the dash0.com/version label of repository settings, or "" if absent.
func GetDarkplaneRepositorySettingsVersion(settings *RepositorySettingsDefinition) string {
	if settings == nil || settings.Metadata.Labels.Dash0Comversion == nil {
		return ""
	}
	return *settings.Metadata.Labels.Dash0Comversion
}
