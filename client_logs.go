package dash0

import (
	"context"
	"fmt"
	"net/http"
)

// GetLogRecords retrieves log records based on the provided request.
// This is a POST endpoint but is idempotent (read-only query).
func (c *client) GetLogRecords(ctx context.Context, request *GetLogRecordsRequest) (*GetLogRecordsResponse, error) {
	ctx = withIdempotent(ctx)
	resp, err := c.inner.PostApiLogsWithResponse(ctx, *request)
	if err != nil {
		return nil, fmt.Errorf("dash0: get log records failed: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, newAPIErrorWithBody(resp.HTTPResponse, resp.Body)
	}
	return resp.JSON200, nil
}

// GetLogRecordsIter returns an iterator over log records matching the request.
// The iterator automatically fetches additional pages as needed.
//
// Example:
//
//	iter := client.GetLogRecordsIter(ctx, &dash0.GetLogRecordsRequest{
//	    TimeRange: dash0.TimeReferenceRange{From: "now-1h", To: "now"},
//	    Pagination: &dash0.CursorPagination{Limit: dash0.Int64(100)},
//	})
//	for iter.Next() {
//	    resourceLog := iter.Current()
//	    // process resourceLog
//	}
//	if err := iter.Err(); err != nil {
//	    // handle error
//	}
func (c *client) GetLogRecordsIter(ctx context.Context, request *GetLogRecordsRequest) *Iter[ResourceLogs] {
	// Make initial request
	resp, err := c.GetLogRecords(ctx, request)
	if err != nil {
		return newIterWithError[ResourceLogs](err)
	}

	items := toPointerSlice(resp.ResourceLogs)
	var cursor *string
	hasMore := false
	if resp.Cursors != nil && resp.Cursors.After != nil {
		cursor = (*string)(resp.Cursors.After)
		hasMore = true
	}

	return newIter(items, hasMore, cursor, func(cur *string) ([]*ResourceLogs, *string, error) {
		// Create a copy of the request with the cursor
		nextReq := *request
		if nextReq.Pagination == nil {
			nextReq.Pagination = &CursorPagination{}
		} else {
			paginationCopy := *nextReq.Pagination
			nextReq.Pagination = &paginationCopy
		}
		nextReq.Pagination.Cursor = (*Cursor)(cur)

		resp, err := c.GetLogRecords(ctx, &nextReq)
		if err != nil {
			return nil, nil, err
		}

		items := toPointerSlice(resp.ResourceLogs)
		var nextCursor *string
		if resp.Cursors != nil && resp.Cursors.After != nil {
			nextCursor = (*string)(resp.Cursors.After)
		}
		return items, nextCursor, nil
	})
}
