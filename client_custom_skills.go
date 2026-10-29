package dash0

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ListCustomSkills retrieves the Agent0 custom skills the caller can read.
// Custom skills belong to the organization, not to a dataset.
// Pass [WithOriginPrefix] or [WithEnabled] to have the server filter the result.
func (c *client) ListCustomSkills(ctx context.Context, opts ...ListOption) ([]*SkillDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	options := NewListOptions(opts...)
	if options != (ListOptions{OriginPrefix: options.OriginPrefix, Enabled: options.Enabled}) {
		return nil, fmt.Errorf("dash0: list custom skills supports only the origin prefix and enabled filters")
	}
	params := &GetApiAgent0CustomSkillsParams{
		OriginPrefix: options.OriginPrefix,
		Enabled:      options.Enabled,
	}
	resp, err := c.inner.GetApiAgent0CustomSkillsWithResponse(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("dash0: list custom skills failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return toPointerSlice(resp.JSON200.Skills), nil
}

// GetCustomSkill retrieves an Agent0 custom skill by origin or ID.
func (c *client) GetCustomSkill(ctx context.Context, originOrID string) (*SkillDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	resp, err := c.inner.GetApiAgent0CustomSkillsOriginOrIdWithResponse(ctx, originOrID)
	if err != nil {
		return nil, fmt.Errorf("dash0: get custom skill failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return resp.JSON200, nil
}

// CreateCustomSkill creates an Agent0 custom skill.
// The server reads the skill's name from the SKILL.md frontmatter in spec.files.
// A skill created with a dash0.com/origin, or by an API token without a user, cannot carry spec.source.
func (c *client) CreateCustomSkill(ctx context.Context, skill *SkillDefinition) (*SkillDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	if skill == nil {
		return nil, fmt.Errorf("dash0: create custom skill failed: skill is nil")
	}
	resp, err := c.inner.PostApiAgent0CustomSkillsWithResponse(ctx, *skill)
	if err != nil {
		return nil, fmt.Errorf("dash0: create custom skill failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusCreated {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	var created SkillDefinition
	if err := json.Unmarshal(resp.Body, &created); err != nil {
		return nil, fmt.Errorf("dash0: failed to parse custom skill response: %w", err)
	}
	return &created, nil
}

// UpdateCustomSkill creates or replaces an Agent0 custom skill by origin or ID.
// An unknown origin creates the skill with that origin.
// When spec.permissions is absent, the stored grants stay unchanged.
func (c *client) UpdateCustomSkill(ctx context.Context, originOrID string, skill *SkillDefinition) (*SkillDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	if skill == nil {
		return nil, fmt.Errorf("dash0: update custom skill failed: skill is nil")
	}
	resp, err := c.inner.PutApiAgent0CustomSkillsOriginOrIdWithResponse(ctx, originOrID, *skill)
	if err != nil {
		return nil, fmt.Errorf("dash0: update custom skill failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return resp.JSON200, nil
}

// DeleteCustomSkill deletes an Agent0 custom skill by origin or ID.
func (c *client) DeleteCustomSkill(ctx context.Context, originOrID string) error {
	if err := c.requireAPI(); err != nil {
		return err
	}
	resp, err := c.inner.DeleteApiAgent0CustomSkillsOriginOrIdWithResponse(ctx, originOrID)
	if err != nil {
		return fmt.Errorf("dash0: delete custom skill failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// ListCustomSkillsIter returns an iterator over the Agent0 custom skills the caller can read.
// It accepts the same options as [Client.ListCustomSkills].
func (c *client) ListCustomSkillsIter(ctx context.Context, opts ...ListOption) *Iter[SkillDefinition] {
	items, err := c.ListCustomSkills(ctx, opts...)
	if err != nil {
		return newIterWithError[SkillDefinition](err)
	}
	return newIter(items, false, nil, nil)
}

// StripCustomSkillServerFields removes the fields the server sets or derives from a custom skill, so the result can be sent back on a write.
// It keeps dash0.com/id and dash0.com/origin, which address the skill on an update, and the user-defined spec.files and spec.permissions.
// It removes spec.source, because a skill managed with an origin cannot carry one, and dash0.com/enabled, because it is per-user state.
// It drops the admin grant the server always adds, unless that grant turns the skill off by default.
func StripCustomSkillServerFields(skill *SkillDefinition) {
	if skill == nil {
		return
	}
	skill.Metadata.Name = nil
	if skill.Metadata.Labels != nil {
		skill.Metadata.Labels.Dash0Comversion = nil
		skill.Metadata.Labels.Dash0Comsource = nil
	}
	skill.Metadata.Annotations = nil
	skill.Spec.PermittedActions = nil
	skill.Spec.Source = nil
	if skill.Spec.Permissions != nil {
		kept := make([]SkillPermission, 0, len(*skill.Spec.Permissions))
		for _, permission := range *skill.Spec.Permissions {
			if isDefaultAdminSkillPermission(permission) {
				continue
			}
			kept = append(kept, permission)
		}
		skill.Spec.Permissions = &kept
	}
}

// isDefaultAdminSkillPermission reports whether permission is the admin grant the server adds to every skill, with nothing a caller set on it.
// The admin role's actions are fixed by the server, so only a defaultEnabled of false carries caller intent.
func isDefaultAdminSkillPermission(permission SkillPermission) bool {
	if permission.Role == nil || *permission.Role != skillAdminRole {
		return false
	}
	return permission.DefaultEnabled == nil || *permission.DefaultEnabled
}

const skillAdminRole = "admin"

// ClearCustomSkillID removes the dash0.com/id label from a custom skill.
func ClearCustomSkillID(skill *SkillDefinition) {
	if skill == nil {
		return
	}
	if skill.Metadata.Labels != nil {
		skill.Metadata.Labels.Dash0Comid = nil
	}
}

// GetCustomSkillID extracts the dash0.com/id label from a custom skill.
func GetCustomSkillID(skill *SkillDefinition) string {
	if skill == nil || skill.Metadata.Labels == nil || skill.Metadata.Labels.Dash0Comid == nil {
		return ""
	}
	return *skill.Metadata.Labels.Dash0Comid
}

// GetCustomSkillName extracts the name of a custom skill.
// The server derives it from the SKILL.md frontmatter, so it is empty on a skill that was not read from the API.
func GetCustomSkillName(skill *SkillDefinition) string {
	if skill == nil || skill.Metadata.Name == nil {
		return ""
	}
	return *skill.Metadata.Name
}

// SetCustomSkillID sets the dash0.com/id label on a custom skill, initializing the labels struct if needed.
func SetCustomSkillID(skill *SkillDefinition, id string) {
	if skill == nil {
		return
	}
	if skill.Metadata.Labels == nil {
		skill.Metadata.Labels = &SkillLabels{}
	}
	skill.Metadata.Labels.Dash0Comid = &id
}

// SetCustomSkillIDIfAbsent sets the dash0.com/id label on a custom skill only if it is not already set, initializing the labels struct if needed.
func SetCustomSkillIDIfAbsent(skill *SkillDefinition, id string) {
	if skill == nil {
		return
	}
	if skill.Metadata.Labels == nil {
		skill.Metadata.Labels = &SkillLabels{}
	}
	if skill.Metadata.Labels.Dash0Comid == nil {
		skill.Metadata.Labels.Dash0Comid = &id
	}
}

// GetCustomSkillOrigin extracts the dash0.com/origin label from a custom skill.
func GetCustomSkillOrigin(skill *SkillDefinition) string {
	if skill == nil || skill.Metadata.Labels == nil || skill.Metadata.Labels.Dash0Comorigin == nil {
		return ""
	}
	return *skill.Metadata.Labels.Dash0Comorigin
}

// SetCustomSkillOrigin sets the dash0.com/origin label on a custom skill, initializing the labels struct if needed.
// The server stores the origin on create, and it cannot change afterwards.
func SetCustomSkillOrigin(skill *SkillDefinition, origin string) {
	if skill == nil {
		return
	}
	if skill.Metadata.Labels == nil {
		skill.Metadata.Labels = &SkillLabels{}
	}
	skill.Metadata.Labels.Dash0Comorigin = &origin
}
