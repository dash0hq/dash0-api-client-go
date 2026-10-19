package dash0

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"reflect"
	"slices"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

// redactedToken replaces the auth token secret in formatted and logged output.
const redactedToken = "[REDACTED]"

// ListAuthTokens retrieves all auth tokens in the organization.
// The responses carry no secret: the secret is returned only by the call that creates the token.
// Only fully unrestricted callers can manage auth tokens: an organization admin user, or an auth token with the * action and no dataset or signal restriction.
// Other callers get a 403 error.
func (c *client) ListAuthTokens(ctx context.Context) ([]*AuthTokenResponseDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	resp, err := c.inner.GetApiAuthTokensWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("dash0: list auth tokens failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return toPointerSlice(*resp.JSON200), nil
}

// GetAuthToken retrieves an auth token by origin or ID.
// The response carries no secret.
func (c *client) GetAuthToken(ctx context.Context, originOrID string) (*AuthTokenResponseDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	resp, err := c.inner.GetApiAuthTokensOriginOrIdWithResponse(ctx, originOrID)
	if err != nil {
		return nil, fmt.Errorf("dash0: get auth token failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return resp.JSON200, nil
}

// CreateAuthToken creates a new auth token.
// The server generates the dash0.com/id label and the secret.
// The returned response is the only one that carries the secret in spec.token, so store it right away.
// The request has no secret field.
func (c *client) CreateAuthToken(ctx context.Context, token *AuthTokenDefinition) (*AuthTokenResponseDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	if token == nil {
		return nil, fmt.Errorf("dash0: create auth token requires a non-nil auth token")
	}
	resp, err := c.inner.PostApiAuthTokensWithResponse(ctx, *token)
	if err != nil {
		return nil, fmt.Errorf("dash0: create auth token failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusCreated {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	var created AuthTokenResponseDefinition
	if err := json.Unmarshal(resp.Body, &created); err != nil {
		return nil, fmt.Errorf("dash0: failed to parse auth token response: %w", err)
	}
	return &created, nil
}

// UpdateAuthToken updates an auth token by origin or ID.
// The server creates the auth token when none matches, so a mistyped origin creates a new auth token instead of returning not found.
// Only a call that creates the auth token returns the secret.
// Fields omitted from spec keep their stored value.
func (c *client) UpdateAuthToken(ctx context.Context, originOrID string, token *AuthTokenDefinition) (*AuthTokenResponseDefinition, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	if token == nil {
		return nil, fmt.Errorf("dash0: update auth token requires a non-nil auth token")
	}
	resp, err := c.inner.PutApiAuthTokensOriginOrIdWithResponse(ctx, originOrID, *token)
	if err != nil {
		return nil, fmt.Errorf("dash0: update auth token failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusCreated {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 != nil {
		return resp.JSON200, nil
	}
	var updated AuthTokenResponseDefinition
	if err := json.Unmarshal(resp.Body, &updated); err != nil {
		return nil, fmt.Errorf("dash0: failed to parse auth token response: %w", err)
	}
	return &updated, nil
}

// DeleteAuthToken deletes an auth token by origin or ID.
// The server refuses to delete the last auth token of the organization.
// That refusal is returned as an [APIError].
func (c *client) DeleteAuthToken(ctx context.Context, originOrID string) error {
	if err := c.requireAPI(); err != nil {
		return err
	}
	resp, err := c.inner.DeleteApiAuthTokensOriginOrIdWithResponse(ctx, originOrID)
	if err != nil {
		return fmt.Errorf("dash0: delete auth token failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// ListAuthTokensIter returns an iterator over all auth tokens.
// This is a convenience wrapper around ListAuthTokens for consistent iteration patterns.
func (c *client) ListAuthTokensIter(ctx context.Context) *Iter[AuthTokenResponseDefinition] {
	items, err := c.ListAuthTokens(ctx)
	if err != nil {
		return newIterWithError[AuthTokenResponseDefinition](err)
	}
	return newIter(items, false, nil, nil)
}

// StripAuthTokenServerFields removes server-generated fields from an auth token definition.
// It clears the annotations and the dash0.com/id and dash0.com/source labels.
// It keeps dash0.com/origin, which is client-settable and identifies the auth token on update, so a copy created in the same organization needs a new origin.
func StripAuthTokenServerFields(token *AuthTokenDefinition) {
	if token == nil {
		return
	}
	token.Metadata.Annotations = nil
	if token.Metadata.Labels != nil {
		token.Metadata.Labels.Dash0Comid = nil
		token.Metadata.Labels.Dash0Comsource = nil
	}
}

// GetAuthTokenID extracts the dash0.com/id label from an auth token response.
// The label is the server-set UUID, not the client-settable dash0.com/origin.
// It returns an empty string when the label is not set.
func GetAuthTokenID(token *AuthTokenResponseDefinition) string {
	if token == nil || token.Metadata.Labels == nil || token.Metadata.Labels.Dash0Comid == nil {
		return ""
	}
	return token.Metadata.Labels.Dash0Comid.String()
}

// GetAuthTokenName extracts the name from an auth token response.
func GetAuthTokenName(token *AuthTokenResponseDefinition) string {
	if token == nil {
		return ""
	}
	return token.Metadata.Name
}

// SetAuthTokenID sets the dash0.com/id label (the server-set UUID) on an auth token definition.
// It initializes the labels struct if needed.
// It does nothing when id is not a valid UUID.
func SetAuthTokenID(token *AuthTokenDefinition, id string) {
	if token == nil {
		return
	}
	var uid openapi_types.UUID
	if err := uid.UnmarshalText([]byte(id)); err != nil {
		return
	}
	if token.Metadata.Labels == nil {
		token.Metadata.Labels = &AuthTokenLabels{}
	}
	token.Metadata.Labels.Dash0Comid = &uid
}

// SetAuthTokenIDIfAbsent sets the dash0.com/id label (the server-set UUID) on an auth token definition only if it is not already set.
// It initializes the labels struct if needed.
// It does nothing when id is not a valid UUID.
func SetAuthTokenIDIfAbsent(token *AuthTokenDefinition, id string) {
	if token == nil {
		return
	}
	if token.Metadata.Labels != nil && token.Metadata.Labels.Dash0Comid != nil {
		return
	}
	SetAuthTokenID(token, id)
}

// ClearAuthTokenID removes the dash0.com/id label (the server-set UUID) from an auth token definition.
// A create request must not carry this label.
func ClearAuthTokenID(token *AuthTokenDefinition) {
	if token == nil || token.Metadata.Labels == nil {
		return
	}
	token.Metadata.Labels.Dash0Comid = nil
}

// ConvertAuthTokenResponseToDefinition copies an auth token response into a request document.
// The copy carries no secret, so it is safe to send to [Client.UpdateAuthToken].
// It keeps the server-set metadata, so call [StripAuthTokenServerFields] to drop it before a create.
func ConvertAuthTokenResponseToDefinition(token *AuthTokenResponseDefinition) *AuthTokenDefinition {
	if token == nil {
		return nil
	}
	return &AuthTokenDefinition{
		ApiVersion: token.ApiVersion,
		Kind:       AuthTokenDefinitionKind(token.Kind),
		Metadata:   token.Metadata,
		Spec: AuthTokenSpec{
			Actions:   token.Spec.Actions,
			Ingestion: token.Spec.Ingestion,
			Scope:     token.Spec.Scope,
		},
	}
}

// Format implements [fmt.Formatter].
// It prints the spec with the secret in spec.token replaced by "[REDACTED]" for every verb.
// This covers a spec printed through a pointer or nested in an [AuthTokenResponseDefinition].
// JSON encoding is not affected and still carries the secret.
func (s AuthTokenResponseSpec) Format(f fmt.State, verb rune) {
	// A copy of the struct type with Token widened to any, so the redacted text prints as text and not as a pointer address.
	// The copy has no methods, so Format does not recurse.
	t := reflect.TypeOf(s)
	fields := make([]reflect.StructField, t.NumField())
	for i := range fields {
		fields[i] = t.Field(i)
		if fields[i].Name == "Token" {
			fields[i].Type = reflect.TypeFor[any]()
		}
	}
	src := reflect.ValueOf(s)
	dst := reflect.New(reflect.StructOf(fields)).Elem()
	for i := range fields {
		if fields[i].Name == "Token" {
			if s.Token != nil {
				dst.Field(i).Set(reflect.ValueOf(redactedToken))
			}
			continue
		}
		dst.Field(i).Set(src.Field(i))
	}
	_, _ = fmt.Fprintf(f, fmt.FormatString(f, verb), dst.Interface())
}

// LogValue implements [slog.LogValuer].
// It logs the spec as a group, keyed like the JSON form, with the secret in spec.token replaced by "[REDACTED]".
func (s AuthTokenResponseSpec) LogValue() slog.Value {
	b, err := json.Marshal(s)
	if err != nil {
		return slog.StringValue(redactedToken)
	}
	var fields map[string]any
	if err := json.Unmarshal(b, &fields); err != nil {
		return slog.StringValue(redactedToken)
	}
	if _, ok := fields["token"]; ok {
		fields["token"] = redactedToken
	}
	attrs := make([]slog.Attr, 0, len(fields))
	for _, k := range slices.Sorted(maps.Keys(fields)) {
		attrs = append(attrs, slog.Any(k, fields[k]))
	}
	return slog.GroupValue(attrs...)
}

// LogValue implements [slog.LogValuer].
// Without it, a JSON log handler marshals the definition directly and bypasses [AuthTokenResponseSpec.LogValue], which leaks the secret.
func (d AuthTokenResponseDefinition) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Any("kind", d.Kind),
		slog.Any("metadata", d.Metadata),
		slog.Any("spec", d.Spec),
	)
}
