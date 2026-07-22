package dash0

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// TeamDefinition is a deprecated alias for [TeamDefinitionV1Alpha1].
//
// Deprecated: since v1.17.0. Use [TeamDefinitionV1Alpha1] instead. The
// alias is kept as a soft-rename shim so v1.x consumers can migrate without
// touching every call site in one go.
type TeamDefinition = TeamDefinitionV1Alpha1

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

// CreateTeam creates a new team via POST /api/teams. The server assigns
// dash0.com/id, dash0.com/origin (equal to the ID unless the caller supplied
// one), and dash0.com/source labels on the returned envelope.
func (c *client) CreateTeam(ctx context.Context, team *TeamDefinitionV1Alpha1) (*TeamDefinitionV1Alpha1, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	if team == nil {
		return nil, fmt.Errorf("dash0: create team requires a non-nil team")
	}
	resp, err := c.inner.PostApiTeamsWithResponse(ctx, *team)
	if err != nil {
		return nil, fmt.Errorf("dash0: create team failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusCreated {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	var created TeamDefinitionV1Alpha1
	if err := json.Unmarshal(resp.Body, &created); err != nil {
		return nil, fmt.Errorf("dash0: failed to parse team response: %w", err)
	}
	return &created, nil
}

// UpsertTeam create-or-replaces a team via PUT /api/teams/{originOrID}.
// If a team with the given origin or ID exists, it is updated in place;
// otherwise a new team is created with dash0.com/origin = originOrID.
// spec.display fully replaces the existing display values; spec.members is
// reconciled declaratively.
func (c *client) UpsertTeam(ctx context.Context, originOrID string, team *TeamDefinitionV1Alpha1) (*TeamDefinitionV1Alpha1, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	if team == nil {
		return nil, fmt.Errorf("dash0: upsert team requires a non-nil team")
	}
	resp, err := c.inner.PutApiTeamsOriginOrIdWithResponse(ctx, originOrID, *team)
	if err != nil {
		return nil, fmt.Errorf("dash0: upsert team failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	var upserted TeamDefinitionV1Alpha1
	if err := json.Unmarshal(resp.Body, &upserted); err != nil {
		return nil, fmt.Errorf("dash0: failed to parse team response: %w", err)
	}
	return &upserted, nil
}

// GetTeam retrieves a team by origin or ID and returns the CRD envelope.
// The server responds with an enriched shape that carries the envelope
// under `team` alongside accessible assets; only the envelope is returned
// so callers work with a single canonical type across create, get, and
// list flows. Callers that need the enriched members and accessible-asset
// arrays should call [GetTeamWithAssets] instead.
func (c *client) GetTeam(ctx context.Context, originOrID string) (*TeamDefinitionV1Alpha1, error) {
	enriched, err := c.GetTeamWithAssets(ctx, originOrID)
	if err != nil {
		return nil, err
	}
	team := enriched.Team
	return &team, nil
}

// GetTeamWithAssets retrieves a team by origin or ID and returns the enriched
// response: the CRD envelope under `.Team` plus arrays of accessible assets
// (Dashboards, CheckRules, SyntheticChecks, Views, Datasets) and the full
// [MemberDefinition] records for each member. Prefer [GetTeam] for the
// GET-diff-PUT round-trip; use this when rendering a team's detail view.
func (c *client) GetTeamWithAssets(ctx context.Context, originOrID string) (*GetTeamResponse, error) {
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
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
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
	if display == nil {
		return fmt.Errorf("dash0: update team display requires a non-nil display")
	}
	resp, err := c.inner.PutApiTeamsOriginOrIdDisplayWithResponse(ctx, originOrID, *display)
	if err != nil {
		return fmt.Errorf("dash0: update team display failed: %w", err)
	}
	// Server returns 204 No Content on success, per the OpenAPI spec.
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// AddTeamMembers adds members to a team.
func (c *client) AddTeamMembers(ctx context.Context, originOrID string, request *AddTeamMembersRequest) error {
	if err := c.requireAPI(); err != nil {
		return err
	}
	if request == nil {
		return fmt.Errorf("dash0: add team members requires a non-nil request")
	}
	resp, err := c.inner.PostApiTeamsOriginOrIdMembersWithResponse(ctx, originOrID, *request)
	if err != nil {
		return fmt.Errorf("dash0: add team members failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusCreated {
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

// ListTeamsIter returns an iterator over all teams. This is a convenience
// wrapper around [client.ListTeams] for consistent iteration patterns.
func (c *client) ListTeamsIter(ctx context.Context) *Iter[TeamsListItem] {
	items, err := c.ListTeams(ctx)
	if err != nil {
		return newIterWithError[TeamsListItem](err)
	}
	return newIter(items, false, nil, nil)
}

// ResolveMemberIDsToEmails translates a list of internal member IDs
// (`dash0.com/id`) back to the corresponding email addresses using
// [client.ListMembers]. IDs that do not resolve to an email are passed
// through as-is (defensive fallback for members without a display email or
// for orphaned membership rows). The output slice preserves the input
// order.
//
// This is the read-side companion to the server's write-side email
// resolution: server responses always echo internal IDs on
// [TeamDefinitionV1Alpha1.Spec.Members], and IaC tools that display members
// back to users call this to render them as emails.
func (c *client) ResolveMemberIDsToEmails(ctx context.Context, ids []string) ([]string, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []string{}, nil
	}
	members, err := c.ListMembers(ctx)
	if err != nil {
		return nil, fmt.Errorf("dash0: resolve member IDs to emails failed: %w", err)
	}
	// Members are indexed by both metadata.name (the canonical, required
	// identifier) and metadata.labels["dash0.com/id"] (an optional duplicate
	// label). The public /api/members endpoint currently populates
	// metadata.name for every member but leaves the dash0.com/id label unset,
	// so a resolver that keyed only on the label produced an empty map and
	// silently passed every ID through as an ID. Indexing on both fields
	// tolerates either shape.
	byID := make(map[string]string, len(members)*2)
	for _, m := range members {
		if m == nil || m.Spec.Display.Email == nil || *m.Spec.Display.Email == "" {
			continue
		}
		email := *m.Spec.Display.Email
		if m.Metadata.Name != "" {
			byID[m.Metadata.Name] = email
		}
		if m.Metadata.Labels != nil && m.Metadata.Labels.Dash0Comid != nil && *m.Metadata.Labels.Dash0Comid != "" {
			byID[*m.Metadata.Labels.Dash0Comid] = email
		}
	}
	out := make([]string, len(ids))
	for i, id := range ids {
		if email, ok := byID[id]; ok {
			out[i] = email
			continue
		}
		out[i] = id
	}
	return out, nil
}

// ResolveTeamMembersToEmails rewrites team.Spec.Members from internal IDs to
// email addresses using ResolveMemberIDsToEmails. It is a convenience
// wrapper around the same logic used by the CLI's export path and the
// Terraform provider's state-normalization path, so both IaC facilities
// display team membership as emails without duplicating the translation.
//
// A nil team, or a team with an empty Spec.Members, is a no-op. IDs that do
// not resolve to an email are left in place (via the pass-through behavior of
// ResolveMemberIDsToEmails). Errors from the underlying members lookup are
// returned so callers can decide whether the mismatch matters — typical
// callers log-and-ignore, keeping the raw IDs.
func ResolveTeamMembersToEmails(ctx context.Context, apiClient Client, team *TeamDefinitionV1Alpha1) error {
	if team == nil || len(team.Spec.Members) == 0 {
		return nil
	}
	emails, err := apiClient.ResolveMemberIDsToEmails(ctx, team.Spec.Members)
	if err != nil {
		return err
	}
	team.Spec.Members = emails
	return nil
}

// GetTeamID extracts the dash0.com/id label from a team definition. Returns
// the empty string when the team is nil, has no labels, or the label is
// unset.
func GetTeamID(team *TeamDefinitionV1Alpha1) string {
	if team == nil || team.Metadata.Labels == nil || team.Metadata.Labels.Dash0Comid == nil {
		return ""
	}
	return *team.Metadata.Labels.Dash0Comid
}

// GetTeamName extracts the metadata.name (technical name) from a team
// definition. Returns the empty string when the team is nil.
func GetTeamName(team *TeamDefinitionV1Alpha1) string {
	if team == nil {
		return ""
	}
	return team.Metadata.Name
}

// GetTeamDisplayName extracts the spec.display.name (human-facing name) from
// a team definition. Returns the empty string when the team is nil.
func GetTeamDisplayName(team *TeamDefinitionV1Alpha1) string {
	if team == nil {
		return ""
	}
	return team.Spec.Display.Name
}

// GetTeamOrigin extracts the dash0.com/origin label from a team definition.
// Returns the empty string when the team is nil, has no labels, or the label
// is unset.
func GetTeamOrigin(team *TeamDefinitionV1Alpha1) string {
	if team == nil || team.Metadata.Labels == nil || team.Metadata.Labels.Dash0Comorigin == nil {
		return ""
	}
	return *team.Metadata.Labels.Dash0Comorigin
}

// SetTeamID sets the dash0.com/id label on a team definition, initializing
// the labels struct if needed. No-op when the team is nil.
func SetTeamID(team *TeamDefinitionV1Alpha1, id string) {
	if team == nil {
		return
	}
	if team.Metadata.Labels == nil {
		team.Metadata.Labels = &TeamLabels{}
	}
	team.Metadata.Labels.Dash0Comid = &id
}

// SetTeamIDIfAbsent sets the dash0.com/id label on a team definition only
// when it is not already set. No-op when the team is nil.
func SetTeamIDIfAbsent(team *TeamDefinitionV1Alpha1, id string) {
	if team == nil {
		return
	}
	if team.Metadata.Labels == nil {
		team.Metadata.Labels = &TeamLabels{}
	}
	if team.Metadata.Labels.Dash0Comid == nil {
		team.Metadata.Labels.Dash0Comid = &id
	}
}

// ClearTeamID removes the dash0.com/id label from a team definition. No-op
// when the team is nil or has no labels.
func ClearTeamID(team *TeamDefinitionV1Alpha1) {
	if team == nil || team.Metadata.Labels == nil {
		return
	}
	team.Metadata.Labels.Dash0Comid = nil
}

// StripTeamServerFields clears the server-managed fields from a team
// definition so callers can round-trip a team through a write endpoint
// without accidentally shipping a stale server-set ID, origin, or source.
// Cleared fields are:
//
//   - metadata.labels["dash0.com/id"], [".../origin"], [".../source"].
//
// Arbitrary user-supplied labels are preserved. No-op when the team is nil.
func StripTeamServerFields(team *TeamDefinitionV1Alpha1) {
	if team == nil {
		return
	}
	if team.Metadata.Labels != nil {
		team.Metadata.Labels.Dash0Comid = nil
		team.Metadata.Labels.Dash0Comsource = nil
		// dash0.com/origin is intentionally preserved — it is client-settable
		// on create and immutable thereafter, so IaC callers rely on it to
		// drive PUT-by-origin idempotency.
		// The server accepts arbitrary user-supplied label keys on write but
		// silently drops any key outside the reserved dash0.com/* set. Clear
		// AdditionalProperties here so IaC state normalization does not
		// perpetually diff on custom labels the server refuses to persist.
		team.Metadata.Labels.AdditionalProperties = nil
	}
	if team.Metadata.Annotations != nil {
		team.Metadata.Annotations.Dash0ComcreatedAt = nil
		team.Metadata.Annotations.Dash0ComupdatedAt = nil
		// Same treatment as labels — user-supplied annotation keys outside
		// dash0.com/* are dropped by the server; strip them so read-side
		// normalization matches.
		team.Metadata.Annotations.AdditionalProperties = nil
	}
}
