package dash0

import (
	"context"
	"fmt"
	"net/http"
)

// GetSpans retrieves spans based on the provided request.
// This is a POST endpoint but is idempotent (read-only query).
func (c *client) GetSpans(ctx context.Context, request *GetSpansRequest) (*GetSpansResponse, error) {
	ctx = withIdempotent(ctx)
	resp, err := c.inner.PostApiSpansWithResponse(ctx, *request)
	if err != nil {
		return nil, fmt.Errorf("dash0: get spans failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200, nil
}

// GetSpansIter returns an iterator over spans matching the request.
// The iterator automatically fetches additional pages as needed.
//
// Example:
//
//	iter := client.GetSpansIter(ctx, &dash0.GetSpansRequest{
//	    TimeRange: dash0.TimeReferenceRange{From: "now-1h", To: "now"},
//	    Pagination: &dash0.CursorPagination{Limit: dash0.Int64(100)},
//	})
//	for iter.Next() {
//	    resourceSpan := iter.Current()
//	    // process resourceSpan
//	}
//	if err := iter.Err(); err != nil {
//	    // handle error
//	}
func (c *client) GetSpansIter(ctx context.Context, request *GetSpansRequest) *Iter[ResourceSpans] {
	// Make initial request
	resp, err := c.GetSpans(ctx, request)
	if err != nil {
		return newIterWithError[ResourceSpans](err)
	}

	items := toPointerSlice(resp.ResourceSpans)
	var cursor *string
	hasMore := false
	if resp.Cursors != nil && resp.Cursors.After != nil {
		cursor = (*string)(resp.Cursors.After)
		hasMore = true
	}

	return newIter(items, hasMore, cursor, func(cur *string) ([]*ResourceSpans, *string, error) {
		// Create a copy of the request with the cursor
		nextReq := *request
		if nextReq.Pagination == nil {
			nextReq.Pagination = &CursorPagination{}
		} else {
			paginationCopy := *nextReq.Pagination
			nextReq.Pagination = &paginationCopy
		}
		nextReq.Pagination.Cursor = (*Cursor)(cur)

		resp, err := c.GetSpans(ctx, &nextReq)
		if err != nil {
			return nil, nil, err
		}

		items := toPointerSlice(resp.ResourceSpans)
		var nextCursor *string
		if resp.Cursors != nil && resp.Cursors.After != nil {
			nextCursor = (*string)(resp.Cursors.After)
		}
		return items, nextCursor, nil
	})
}
