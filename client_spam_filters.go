package dash0

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// SpamFilter is a convenience alias for SpamFilterDefinition (v1alpha1).
type SpamFilter = SpamFilterDefinition

// SpamFilterV1Alpha2 is a convenience alias for SpamFilterDefinitionV1Alpha2.
type SpamFilterV1Alpha2 = SpamFilterDefinitionV1Alpha2

// SpamFilterObject is the marker interface for spam filter values returned by
// [Client.GetSpamFilter]. The upstream API models the response as a union over
// v1alpha1 and v1alpha2 shapes; this SDK does not convert between versions, so
// the caller type-switches on the concrete type:
//
//	switch v := got.(type) {
//	case *SpamFilter:
//	    // v1alpha1: v.Spec.Contexts is a []TelemetryFilterContext
//	case *SpamFilterV1Alpha2:
//	    // v1alpha2: v.Spec.Context is a single TelemetryFilterContext
//	}
//
// This mirrors the Kubernetes runtime.Object pattern: decode into whichever
// concrete type the wire actually carried, expose it through a marker, and let
// the caller dispatch.
type SpamFilterObject interface {
	spamFilterObject()
}

func (*SpamFilter) spamFilterObject()         {}
func (*SpamFilterV1Alpha2) spamFilterObject() {}

// ListSpamFilters retrieves all spam filters. The generated response type
// matches the v1alpha1 schema, so v1alpha2 entries are returned with their
// spec.contexts array empty — the spec.context scalar gets dropped during
// JSON unmarshalling. Callers that need to see v1alpha2 items in their
// native shape should use [client.ListSpamFilterObjects] instead.
func (c *client) ListSpamFilters(ctx context.Context, dataset *string) ([]*SpamFilter, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	params := &GetApiSpamFiltersParams{
		Dataset: dataset,
	}
	resp, err := c.inner.GetApiSpamFiltersWithResponse(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("dash0: list spam filters failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("dash0: unexpected nil response")
	}
	return toPointerSlice(resp.JSON200.SpamFilters), nil
}

// ListSpamFilterObjects retrieves all spam filters in their native apiVersion
// shape. Each item is returned through the [SpamFilterObject] marker so the
// caller can type-switch on *SpamFilter (v1alpha1) or *SpamFilterV1Alpha2.
// Unlike [client.ListSpamFilters], v1alpha2 entries are decoded with their
// spec.context scalar preserved.
func (c *client) ListSpamFilterObjects(ctx context.Context, dataset *string) ([]SpamFilterObject, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	params := &GetApiSpamFiltersParams{
		Dataset: dataset,
	}
	resp, err := c.inner.GetApiSpamFiltersWithResponse(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("dash0: list spam filters failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	var envelope struct {
		SpamFilters []json.RawMessage `json:"spamFilters"`
	}
	if err := json.Unmarshal(resp.Body, &envelope); err != nil {
		return nil, fmt.Errorf("dash0: failed to decode spam filter list: %w", err)
	}
	out := make([]SpamFilterObject, 0, len(envelope.SpamFilters))
	for i, raw := range envelope.SpamFilters {
		obj, err := decodeSpamFilterObject(raw)
		if err != nil {
			return nil, fmt.Errorf("dash0: failed to decode spam filter at index %d: %w", i, err)
		}
		out = append(out, obj)
	}
	return out, nil
}

// GetSpamFilter retrieves a spam filter by origin or ID. The version of the
// returned value is server-determined: it is *SpamFilter for a v1alpha1
// response and *SpamFilterV1Alpha2 for a v1alpha2 response. Use a type switch
// (see [SpamFilterObject]) to access version-specific fields.
func (c *client) GetSpamFilter(ctx context.Context, originOrID string, dataset *string) (SpamFilterObject, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	params := &GetApiSpamFiltersOriginOrIdParams{
		Dataset: dataset,
	}
	resp, err := c.inner.GetApiSpamFiltersOriginOrIdWithResponse(ctx, originOrID, params)
	if err != nil {
		return nil, fmt.Errorf("dash0: get spam filter failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return decodeSpamFilterObject(resp.Body)
}

// CreateSpamFilter creates a new spam filter.
func (c *client) CreateSpamFilter(ctx context.Context, filter *SpamFilter, dataset *string) (*SpamFilter, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	if filter == nil {
		return nil, fmt.Errorf("dash0: create spam filter requires a non-nil filter")
	}
	params := &PostApiSpamFiltersParams{
		Dataset: dataset,
	}
	var reqBody SpamFilterCreateRequest
	if err := reqBody.FromSpamFilterDefinition(*filter); err != nil {
		return nil, fmt.Errorf("dash0: failed to encode spam filter request: %w", err)
	}
	resp, err := c.inner.PostApiSpamFiltersWithResponse(ctx, params, reqBody)
	if err != nil {
		return nil, fmt.Errorf("dash0: create spam filter failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusCreated {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return unwrapSpamFilterResponse(resp.Body)
}

// UpdateSpamFilter updates an existing spam filter.
func (c *client) UpdateSpamFilter(ctx context.Context, originOrID string, filter *SpamFilter, dataset *string) (*SpamFilter, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	if filter == nil {
		return nil, fmt.Errorf("dash0: update spam filter requires a non-nil filter")
	}
	params := &PutApiSpamFiltersOriginOrIdParams{
		Dataset: dataset,
	}
	var reqBody SpamFilterResponse
	if err := reqBody.FromSpamFilterDefinition(*filter); err != nil {
		return nil, fmt.Errorf("dash0: failed to encode spam filter request: %w", err)
	}
	resp, err := c.inner.PutApiSpamFiltersOriginOrIdWithResponse(ctx, originOrID, params, reqBody)
	if err != nil {
		return nil, fmt.Errorf("dash0: update spam filter failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return unwrapSpamFilterResponse(resp.Body)
}

// decodeSpamFilterObject inspects the apiVersion on the response body and
// decodes the payload into the matching typed struct. The result is returned
// through the [SpamFilterObject] marker so callers can type-switch.
//
// Group-prefixed apiVersions emitted by the Dash0 Kubernetes operator (e.g.
// "operator.dash0.com/v1alpha1") are normalised via [NormalizeDash0ApiVersion]
// before the version is matched; foreign apiGroups are rejected.
func decodeSpamFilterObject(body []byte) (SpamFilterObject, error) {
	var disc struct {
		ApiVersion string `json:"apiVersion"`
	}
	if err := json.Unmarshal(body, &disc); err != nil {
		return nil, fmt.Errorf("dash0: failed to decode spam filter response: %w", err)
	}
	version, ok := NormalizeDash0ApiVersion(disc.ApiVersion)
	if !ok {
		return nil, fmt.Errorf("dash0: unsupported spam filter apiVersion %q", disc.ApiVersion)
	}
	switch version {
	case "", string(V1alpha1):
		var def SpamFilter
		if err := json.Unmarshal(body, &def); err != nil {
			return nil, fmt.Errorf("dash0: failed to decode v1alpha1 spam filter response: %w", err)
		}
		return &def, nil
	case string(V1alpha2):
		var def SpamFilterV1Alpha2
		if err := json.Unmarshal(body, &def); err != nil {
			return nil, fmt.Errorf("dash0: failed to decode v1alpha2 spam filter response: %w", err)
		}
		return &def, nil
	default:
		return nil, fmt.Errorf("dash0: unsupported spam filter apiVersion %q", disc.ApiVersion)
	}
}

// unwrapSpamFilterResponse decodes a spam filter response body as the v1alpha1
// SpamFilterDefinition. Used by Create/Update where the caller committed to
// v1alpha1 on the request; a response carrying a different version is reported
// as an error.
func unwrapSpamFilterResponse(body []byte) (*SpamFilter, error) {
	obj, err := decodeSpamFilterObject(body)
	if err != nil {
		return nil, err
	}
	v, ok := obj.(*SpamFilter)
	if !ok {
		return nil, fmt.Errorf("dash0: expected v1alpha1 spam filter response, got %T", obj)
	}
	return v, nil
}

// unwrapSpamFilterV1Alpha2Response decodes a spam filter response body as the
// v1alpha2 SpamFilterDefinitionV1Alpha2. Used by Create/Update where the caller
// committed to v1alpha2 on the request; a response carrying a different version
// is reported as an error.
func unwrapSpamFilterV1Alpha2Response(body []byte) (*SpamFilterV1Alpha2, error) {
	obj, err := decodeSpamFilterObject(body)
	if err != nil {
		return nil, err
	}
	v, ok := obj.(*SpamFilterV1Alpha2)
	if !ok {
		return nil, fmt.Errorf("dash0: expected v1alpha2 spam filter response, got %T", obj)
	}
	return v, nil
}

// CreateSpamFilterV1Alpha2 creates a new spam filter, sending the request body
// in the v1alpha2 shape (spec.context as a scalar). The response is decoded as
// v1alpha2; a response carrying a different apiVersion is reported as an error.
func (c *client) CreateSpamFilterV1Alpha2(ctx context.Context, filter *SpamFilterV1Alpha2, dataset *string) (*SpamFilterV1Alpha2, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	if filter == nil {
		return nil, fmt.Errorf("dash0: create spam filter requires a non-nil filter")
	}
	params := &PostApiSpamFiltersParams{
		Dataset: dataset,
	}
	var reqBody SpamFilterCreateRequest
	if err := reqBody.FromSpamFilterDefinitionV1Alpha2(*filter); err != nil {
		return nil, fmt.Errorf("dash0: failed to encode spam filter request: %w", err)
	}
	resp, err := c.inner.PostApiSpamFiltersWithResponse(ctx, params, reqBody)
	if err != nil {
		return nil, fmt.Errorf("dash0: create spam filter failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusCreated {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return unwrapSpamFilterV1Alpha2Response(resp.Body)
}

// UpdateSpamFilterV1Alpha2 updates an existing spam filter, sending the request
// body in the v1alpha2 shape (spec.context as a scalar). The response is decoded
// as v1alpha2; a response carrying a different apiVersion is reported as an
// error.
func (c *client) UpdateSpamFilterV1Alpha2(ctx context.Context, originOrID string, filter *SpamFilterV1Alpha2, dataset *string) (*SpamFilterV1Alpha2, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	if filter == nil {
		return nil, fmt.Errorf("dash0: update spam filter requires a non-nil filter")
	}
	params := &PutApiSpamFiltersOriginOrIdParams{
		Dataset: dataset,
	}
	var reqBody SpamFilterResponse
	if err := reqBody.FromSpamFilterDefinitionV1Alpha2(*filter); err != nil {
		return nil, fmt.Errorf("dash0: failed to encode spam filter request: %w", err)
	}
	resp, err := c.inner.PutApiSpamFiltersOriginOrIdWithResponse(ctx, originOrID, params, reqBody)
	if err != nil {
		return nil, fmt.Errorf("dash0: update spam filter failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return unwrapSpamFilterV1Alpha2Response(resp.Body)
}

// DeleteSpamFilter deletes a spam filter by origin or ID.
func (c *client) DeleteSpamFilter(ctx context.Context, originOrID string, dataset *string) error {
	if err := c.requireAPI(); err != nil {
		return err
	}
	params := &DeleteApiSpamFiltersOriginOrIdParams{
		Dataset: dataset,
	}
	resp, err := c.inner.DeleteApiSpamFiltersOriginOrIdWithResponse(ctx, originOrID, params)
	if err != nil {
		return fmt.Errorf("dash0: delete spam filter failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusNoContent {
		return newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return nil
}

// ListSpamFiltersIter returns an iterator over all spam filters.
// This is a convenience wrapper around ListSpamFilters for consistent iteration patterns.
func (c *client) ListSpamFiltersIter(ctx context.Context, dataset *string) *Iter[SpamFilter] {
	items, err := c.ListSpamFilters(ctx, dataset)
	if err != nil {
		return newIterWithError[SpamFilter](err)
	}
	return newIter(items, false, nil, nil)
}

// ListSpamFilterObjectsIter returns an iterator over all spam filters in their native apiVersion shape.
// It wraps [Client.ListSpamFilterObjects], so v1alpha2 filters keep their spec.context scalar, unlike with [Client.ListSpamFiltersIter].
// Current returns a pointer to the [SpamFilterObject]; dereference it and type-switch on the result.
func (c *client) ListSpamFilterObjectsIter(ctx context.Context, dataset *string) *Iter[SpamFilterObject] {
	items, err := c.ListSpamFilterObjects(ctx, dataset)
	if err != nil {
		return newIterWithError[SpamFilterObject](err)
	}
	return newIter(toPointerSlice(items), false, nil, nil)
}

// StripSpamFilterServerFields removes the server-managed fields from a v1alpha1 spam filter, so that it can be sent in a create or update request.
// It clears the dash0.com/id, dash0.com/source, dash0.com/dataset, and dash0.com/origin labels.
// It keeps the dash0.com/enabled annotation and the custom labels, because the user sets them.
//
// Unlike the Strip helpers of most other asset types, it also clears dash0.com/id.
// A caller that updates the filter by origin or ID must read them before stripping.
func StripSpamFilterServerFields(filter *SpamFilter) {
	if filter == nil {
		return
	}
	stripSpamFilterMetadataServerFields(&filter.Metadata)
}

// ClearSpamFilterID removes the ID from a v1alpha1 spam filter.
func ClearSpamFilterID(filter *SpamFilter) {
	if filter == nil {
		return
	}
	clearSpamFilterMetadataID(&filter.Metadata)
}

// GetSpamFilterID extracts the ID from a v1alpha1 spam filter.
func GetSpamFilterID(filter *SpamFilter) string {
	if filter == nil {
		return ""
	}
	return getSpamFilterMetadataID(&filter.Metadata)
}

// GetSpamFilterName extracts the name from a v1alpha1 spam filter.
func GetSpamFilterName(filter *SpamFilter) string {
	if filter == nil {
		return ""
	}
	return filter.Metadata.Name
}

// GetSpamFilterDataset extracts the dataset label from a v1alpha1 spam filter.
func GetSpamFilterDataset(filter *SpamFilter) string {
	if filter == nil {
		return ""
	}
	return getSpamFilterMetadataDataset(&filter.Metadata)
}

// SetSpamFilterDataset sets the dash0.com/dataset label on a v1alpha1 spam filter, initializing the labels struct if needed.
func SetSpamFilterDataset(filter *SpamFilter, dataset string) {
	if filter == nil {
		return
	}
	setSpamFilterMetadataDataset(&filter.Metadata, dataset)
}

// SetSpamFilterID sets the dash0.com/id label on a v1alpha1 spam filter, initializing the labels struct if needed.
func SetSpamFilterID(filter *SpamFilter, id string) {
	if filter == nil {
		return
	}
	setSpamFilterMetadataID(&filter.Metadata, id)
}

// SetSpamFilterIDIfAbsent sets the dash0.com/id label on a v1alpha1 spam filter only if it is not already set.
// It initializes the labels struct if needed.
func SetSpamFilterIDIfAbsent(filter *SpamFilter, id string) {
	if filter == nil {
		return
	}
	setSpamFilterMetadataIDIfAbsent(&filter.Metadata, id)
}

// StripSpamFilterV1Alpha2ServerFields removes the server-managed fields from a v1alpha2 spam filter, so that it can be sent in a create or update request.
// It clears the same fields as [StripSpamFilterServerFields] and keeps the dash0.com/enabled annotation and the custom labels.
func StripSpamFilterV1Alpha2ServerFields(filter *SpamFilterV1Alpha2) {
	if filter == nil {
		return
	}
	stripSpamFilterMetadataServerFields(&filter.Metadata)
}

// ClearSpamFilterV1Alpha2ID removes the ID from a v1alpha2 spam filter.
func ClearSpamFilterV1Alpha2ID(filter *SpamFilterV1Alpha2) {
	if filter == nil {
		return
	}
	clearSpamFilterMetadataID(&filter.Metadata)
}

// GetSpamFilterV1Alpha2ID extracts the ID from a v1alpha2 spam filter.
func GetSpamFilterV1Alpha2ID(filter *SpamFilterV1Alpha2) string {
	if filter == nil {
		return ""
	}
	return getSpamFilterMetadataID(&filter.Metadata)
}

// GetSpamFilterV1Alpha2Name extracts the name from a v1alpha2 spam filter.
func GetSpamFilterV1Alpha2Name(filter *SpamFilterV1Alpha2) string {
	if filter == nil {
		return ""
	}
	return filter.Metadata.Name
}

// GetSpamFilterV1Alpha2Dataset extracts the dataset label from a v1alpha2 spam filter.
func GetSpamFilterV1Alpha2Dataset(filter *SpamFilterV1Alpha2) string {
	if filter == nil {
		return ""
	}
	return getSpamFilterMetadataDataset(&filter.Metadata)
}

// SetSpamFilterV1Alpha2Dataset sets the dash0.com/dataset label on a v1alpha2 spam filter, initializing the labels struct if needed.
func SetSpamFilterV1Alpha2Dataset(filter *SpamFilterV1Alpha2, dataset string) {
	if filter == nil {
		return
	}
	setSpamFilterMetadataDataset(&filter.Metadata, dataset)
}

// SetSpamFilterV1Alpha2ID sets the dash0.com/id label on a v1alpha2 spam filter, initializing the labels struct if needed.
func SetSpamFilterV1Alpha2ID(filter *SpamFilterV1Alpha2, id string) {
	if filter == nil {
		return
	}
	setSpamFilterMetadataID(&filter.Metadata, id)
}

// SetSpamFilterV1Alpha2IDIfAbsent sets the dash0.com/id label on a v1alpha2 spam filter only if it is not already set.
// It initializes the labels struct if needed.
func SetSpamFilterV1Alpha2IDIfAbsent(filter *SpamFilterV1Alpha2, id string) {
	if filter == nil {
		return
	}
	setSpamFilterMetadataIDIfAbsent(&filter.Metadata, id)
}

// The v1alpha1 and v1alpha2 spam filters share SpamFilterMetadata, so both helper families delegate to the functions below.

func stripSpamFilterMetadataServerFields(metadata *SpamFilterMetadata) {
	if metadata.Labels != nil {
		metadata.Labels.Dash0Comid = nil
		metadata.Labels.Dash0Comsource = nil
		metadata.Labels.Dash0Comdataset = nil
		metadata.Labels.Dash0Comorigin = nil
	}
}

func clearSpamFilterMetadataID(metadata *SpamFilterMetadata) {
	if metadata.Labels != nil {
		metadata.Labels.Dash0Comid = nil
	}
}

func getSpamFilterMetadataID(metadata *SpamFilterMetadata) string {
	if metadata.Labels == nil || metadata.Labels.Dash0Comid == nil {
		return ""
	}
	return *metadata.Labels.Dash0Comid
}

func getSpamFilterMetadataDataset(metadata *SpamFilterMetadata) string {
	if metadata.Labels == nil || metadata.Labels.Dash0Comdataset == nil {
		return ""
	}
	return *metadata.Labels.Dash0Comdataset
}

func setSpamFilterMetadataDataset(metadata *SpamFilterMetadata, dataset string) {
	if metadata.Labels == nil {
		metadata.Labels = &SpamFilterLabels{}
	}
	metadata.Labels.Dash0Comdataset = &dataset
}

func setSpamFilterMetadataID(metadata *SpamFilterMetadata, id string) {
	if metadata.Labels == nil {
		metadata.Labels = &SpamFilterLabels{}
	}
	metadata.Labels.Dash0Comid = &id
}

func setSpamFilterMetadataIDIfAbsent(metadata *SpamFilterMetadata, id string) {
	if metadata.Labels == nil {
		metadata.Labels = &SpamFilterLabels{}
	}
	if metadata.Labels.Dash0Comid == nil {
		metadata.Labels.Dash0Comid = &id
	}
}
