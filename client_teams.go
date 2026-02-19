package dash0

import (
	"context"
	"fmt"
	"net/http"
)

// ListTeams retrieves all teams.
func (c *client) ListTeams(ctx context.Context) ([]*TeamsListItem, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	resp, err := c.inner.GetApiTeamsWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("dash0: list teams failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return toPointerSlice(*resp.JSON200), nil
}

// CreateTeam creates a new team.
func (c *client) CreateTeam(ctx context.Context, team *TeamDefinition) (*TeamDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	resp, err := c.inner.PostApiTeamsWithResponse(ctx, *team)
	if err != nil {
		return nil, fmt.Errorf("dash0: create team failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200, nil
}

// GetTeam retrieves a team by origin or ID.
func (c *client) GetTeam(ctx context.Context, originOrID string) (*GetTeamResponse, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	resp, err := c.inner.GetApiTeamsOriginOrIdWithResponse(ctx, originOrID)
	if err != nil {
		return nil, fmt.Errorf("dash0: get team failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200, nil
}

// DeleteTeam deletes a team by origin or ID.
func (c *client) DeleteTeam(ctx context.Context, originOrID string) error {
	if err := c.requireAPI(); err != nil {
		return err
	}
	resp, err := c.inner.DeleteApiTeamsOriginOrIdWithResponse(ctx, originOrID)
	if err != nil {
		return fmt.Errorf("dash0: delete team failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// UpdateTeamDisplay updates the display settings of a team.
func (c *client) UpdateTeamDisplay(ctx context.Context, originOrID string, display *TeamDisplay) error {
	if err := c.requireAPI(); err != nil {
		return err
	}
	resp, err := c.inner.PutApiTeamsOriginOrIdDisplayWithResponse(ctx, originOrID, *display)
	if err != nil {
		return fmt.Errorf("dash0: update team display failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// AddTeamMembers adds members to a team.
func (c *client) AddTeamMembers(ctx context.Context, originOrID string, request *AddTeamMembersRequest) error {
	if err := c.requireAPI(); err != nil {
		return err
	}
	resp, err := c.inner.PostApiTeamsOriginOrIdMembersWithResponse(ctx, originOrID, *request)
	if err != nil {
		return fmt.Errorf("dash0: add team members failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// RemoveTeamMember removes a member from a team.
func (c *client) RemoveTeamMember(ctx context.Context, originOrID string, memberID string) error {
	if err := c.requireAPI(); err != nil {
		return err
	}
	resp, err := c.inner.DeleteApiTeamsOriginOrIdMembersMemberIDWithResponse(ctx, originOrID, memberID)
	if err != nil {
		return fmt.Errorf("dash0: remove team member failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// ListTeamsIter returns an iterator over all teams.
// This is a convenience wrapper around ListTeams for consistent iteration patterns.
func (c *client) ListTeamsIter(ctx context.Context) *Iter[TeamsListItem] {
	items, err := c.ListTeams(ctx)
	if err != nil {
		return newIterWithError[TeamsListItem](err)
	}
	return newIter(items, false, nil, nil)
}
