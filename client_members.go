package dash0

import (
	"context"
	"fmt"
	"net/http"
)

// ListMembers retrieves all organization members.
func (c *client) ListMembers(ctx context.Context) ([]*MemberDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	resp, err := c.inner.GetApiMembersWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("dash0: list members failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return toPointerSlice(*resp.JSON200), nil
}

// InviteMember invites a new member to the organization.
func (c *client) InviteMember(ctx context.Context, request *InviteMemberRequest) error {
	if err := c.requireAPI(); err != nil {
		return err
	}
	resp, err := c.inner.PostApiMembersWithResponse(ctx, *request)
	if err != nil {
		return fmt.Errorf("dash0: invite member failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// DeleteMember removes a member from the organization.
func (c *client) DeleteMember(ctx context.Context, memberID string) error {
	if err := c.requireAPI(); err != nil {
		return err
	}
	resp, err := c.inner.DeleteApiMembersMemberIDWithResponse(ctx, memberID)
	if err != nil {
		return fmt.Errorf("dash0: delete member failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// ListMembersIter returns an iterator over all organization members.
// This is a convenience wrapper around ListMembers for consistent iteration patterns.
func (c *client) ListMembersIter(ctx context.Context) *Iter[MemberDefinition] {
	items, err := c.ListMembers(ctx)
	if err != nil {
		return newIterWithError[MemberDefinition](err)
	}
	return newIter(items, false, nil, nil)
}
