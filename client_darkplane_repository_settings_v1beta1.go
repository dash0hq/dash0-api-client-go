package dash0

import (
	"context"
	"fmt"
	"net/http"
)

// GetDarkplaneRepositorySettingsV1Beta1 retrieves AutoMerge repository settings in the
// dash0.com/v1beta1 shape, addressed by the settings' dash0.com/id or their dash0.com/origin.
func (c *client) GetDarkplaneRepositorySettingsV1Beta1(ctx context.Context, originOrID string) (*RepositorySettingsV1beta1Definition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	if originOrID == "" {
		return nil, fmt.Errorf("dash0: repository settings origin or ID is required")
	}
	resp, err := c.inner.GetApiDarkplaneRepositorySettingsOriginOrIdWithResponse(ctx, originOrID)
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

// UpdateDarkplaneRepositorySettingsV1Beta1 replaces the whole spec of AutoMerge repository settings
// in the dash0.com/v1beta1 shape and returns the stored settings.
//
// originOrID is the settings' dash0.com/id or an origin. An origin no settings hold yet is claimed
// for the repository spec.repository names, so infrastructure-as-code tools can create and update
// settings by an origin of their choosing. Once claimed, an origin cannot change.
// When settings carries a dash0.com/version label, the server rejects the write if a newer version
// was stored since. Without the label the write replaces whatever is stored.
func (c *client) UpdateDarkplaneRepositorySettingsV1Beta1(ctx context.Context, originOrID string, settings *RepositorySettingsV1beta1Definition) (*RepositorySettingsV1beta1Definition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	if originOrID == "" {
		return nil, fmt.Errorf("dash0: repository settings origin or ID is required")
	}
	if settings == nil {
		return nil, fmt.Errorf("dash0: repository settings are required")
	}
	resp, err := c.inner.PutApiDarkplaneRepositorySettingsOriginOrIdWithResponse(ctx, originOrID, toRepositorySettingsV1Beta1Request(settings))
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

// toRepositorySettingsV1Beta1Request keeps only what the PUT reads: the spec and the
// dash0.com/origin and dash0.com/version labels. dash0.com/id and the annotations are server-set.
func toRepositorySettingsV1Beta1Request(settings *RepositorySettingsV1beta1Definition) RepositorySettingsV1beta1Request {
	request := RepositorySettingsV1beta1Request{
		ApiVersion: RepositorySettingsV1beta1ApiVersionDash0ComV1beta1,
		Kind:       RepositorySettingsV1beta1RequestKindDarkplaneRepositorySettings,
		Spec:       settings.Spec,
	}
	labels := settings.Metadata.Labels
	if labels.Dash0Comorigin != nil || labels.Dash0Comversion != nil {
		request.Metadata = &RepositorySettingsV1beta1RequestMetadata{Labels: &RepositorySettingsV1beta1Labels{
			Dash0Comorigin:  labels.Dash0Comorigin,
			Dash0Comversion: labels.Dash0Comversion,
		}}
	}
	return request
}

// GetDarkplaneRepositorySettingsV1Beta1ID returns the dash0.com/id label of v1beta1 repository settings, the settings' own ID, or "" if absent.
func GetDarkplaneRepositorySettingsV1Beta1ID(settings *RepositorySettingsV1beta1Definition) string {
	if settings == nil || settings.Metadata.Labels.Dash0Comid == nil {
		return ""
	}
	return *settings.Metadata.Labels.Dash0Comid
}

// GetDarkplaneRepositorySettingsV1Beta1Origin returns the dash0.com/origin label of v1beta1 repository settings, or "" if absent.
func GetDarkplaneRepositorySettingsV1Beta1Origin(settings *RepositorySettingsV1beta1Definition) string {
	if settings == nil || settings.Metadata.Labels.Dash0Comorigin == nil {
		return ""
	}
	return *settings.Metadata.Labels.Dash0Comorigin
}

// GetDarkplaneRepositorySettingsV1Beta1Version returns the dash0.com/version label of v1beta1 repository settings, or "" if absent.
func GetDarkplaneRepositorySettingsV1Beta1Version(settings *RepositorySettingsV1beta1Definition) string {
	if settings == nil || settings.Metadata.Labels.Dash0Comversion == nil {
		return ""
	}
	return *settings.Metadata.Labels.Dash0Comversion
}
