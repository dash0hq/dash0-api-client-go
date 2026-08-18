package dash0

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// IssueInstanceStatus is the lifecycle status of a failed-check instance.
type IssueInstanceStatus string

const (
	// IssueInstanceStatusCritical indicates the issue is currently breaching its alerting threshold.
	IssueInstanceStatusCritical IssueInstanceStatus = "critical"

	// IssueInstanceStatusDegraded indicates the issue is currently breaching a degraded threshold.
	IssueInstanceStatusDegraded IssueInstanceStatus = "degraded"

	// IssueInstanceStatusHealthy indicates the issue has recovered and is no longer breaching any threshold.
	IssueInstanceStatusHealthy IssueInstanceStatus = "healthy"

	// IssueInstanceStatusInactive indicates the issue has been resolved.
	IssueInstanceStatusInactive IssueInstanceStatus = "inactive"

	// IssueInstanceStatusPending indicates the issue has been raised but not yet evaluated against its `for` duration.
	IssueInstanceStatusPending IssueInstanceStatus = "pending"
)

// IssueCheckRule is the subset of check-rule fields returned alongside each failed-check instance.
type IssueCheckRule struct {
	Id                  string   `json:"id"`
	Name                string   `json:"name"`
	Version             int      `json:"version"`
	Modes               []string `json:"modes,omitempty"`
	SummaryTemplate     string   `json:"summaryTemplate,omitempty"`
	DescriptionTemplate string   `json:"descriptionTemplate,omitempty"`
}

// Issue represents an active or historical failed-check instance raised by a check rule.
type Issue struct {
	// Id is the server-assigned identifier of this issue instance.
	Id string `json:"id"`

	// CheckRule is the check rule that raised this issue.
	CheckRule IssueCheckRule `json:"checkRule"`

	// InstanceStatus is the current lifecycle status of the issue.
	InstanceStatus IssueInstanceStatus `json:"instanceStatus"`

	// Summary is the rendered summary for this issue.
	Summary string `json:"summary"`

	// Description is the rendered description for this issue.
	Description string `json:"description"`

	// Start is the time the issue was first raised, as an RFC 3339 timestamp.
	Start string `json:"start"`

	// End is the time the issue was resolved, as an RFC 3339 timestamp.
	// The zero value `1970-01-01T00:00:00Z` indicates that the issue is still active.
	End string `json:"end"`

	// Labels are the rendered label set attached to the issue.
	Labels []KeyValue `json:"labels"`

	// EarliestEvaluatedTime is the earliest time the check rule was evaluated against this issue.
	EarliestEvaluatedTime string `json:"earliestEvaluatedTime,omitempty"`

	// IssueIdentifier is the deduplication identifier of the issue.
	IssueIdentifier string `json:"issueIdentifier,omitempty"`

	// AffectedResourceSummaries describes resources affected by the issue.
	AffectedResourceSummaries []map[string]any `json:"affectedResourceSummaries,omitempty"`

	// Annotations are the rendered annotations attached to the issue.
	Annotations []map[string]any `json:"annotations,omitempty"`
}

// GetFailedChecksRequest is the request body for [Client.GetFailedChecks].
type GetFailedChecksRequest struct {
	// Dataset selects the dataset to query.
	// When nil, the organization's default dataset is used.
	Dataset *Dataset `json:"dataset,omitempty"`

	// Filter narrows the results to issues matching the criteria.
	Filter *FilterCriteria `json:"filter,omitempty"`

	// Ordering controls the result order.
	// Defaults to descending start time when nil.
	Ordering *OrderingCriteria `json:"ordering,omitempty"`

	// Pagination controls cursor-based paging.
	Pagination *CursorPagination `json:"pagination,omitempty"`

	// TimeRange is the time range to query.
	TimeRange TimeReferenceRange `json:"timeRange"`
}

// GetFailedChecksResponse is the response of [Client.GetFailedChecks].
type GetFailedChecksResponse struct {
	// Cursors carries the pagination cursors for fetching adjacent result pages.
	Cursors *NextCursors `json:"cursors,omitempty"`

	// ExecutionTime is the server-side execution timestamp of the request.
	ExecutionTime *FixedTime `json:"executionTime,omitempty"`

	// Issues holds the failed-check instances matching the request.
	Issues []Issue `json:"issues"`

	// TimeRange is the resolved time range for the response.
	TimeRange *TimeRange `json:"timeRange,omitempty"`
}

// failedChecksAPIPath is the API path for the failed-checks endpoint.
const failedChecksAPIPath = "/api/alerting/failed-checks"

// GetFailedChecks retrieves failed-check instances matching the request.
// This is a POST endpoint that is idempotent (read-only query) and is therefore retried on transient failures.
func (c *client) GetFailedChecks(ctx context.Context, request *GetFailedChecksRequest) (*GetFailedChecksResponse, error) {
	if err := c.requireAPI(); err != nil {
		return nil, err
	}
	if request == nil {
		return nil, fmt.Errorf("dash0: get failed checks failed: request is required")
	}

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("dash0: get failed checks failed to marshal request: %w", err)
	}

	url := strings.TrimRight(c.config.apiUrl, "/") + failedChecksAPIPath
	req, err := http.NewRequestWithContext(withIdempotent(ctx), http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("dash0: get failed checks failed to build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	// The Authorization header is set by authTransport, which wraps
	// c.httpClient's transport stack.
	req.Header.Set("User-Agent", c.config.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("dash0: get failed checks request failed: %w", err)
	}
	respBody, readErr := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if readErr != nil {
		return nil, fmt.Errorf("dash0: get failed checks failed to read response: %w", readErr)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, newAPIErrorWithBody(resp, respBody)
	}

	var result GetFailedChecksResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("dash0: get failed checks failed to parse response: %w", err)
	}
	return &result, nil
}

// GetFailedChecksIter returns an iterator over failed-check instances matching the request.
// The iterator automatically fetches additional pages by following the `after` cursor returned by the server.
//
// Example:
//
//	iter := client.GetFailedChecksIter(ctx, &dash0.GetFailedChecksRequest{
//	    TimeRange: dash0.TimeReferenceRange{From: "now-1h", To: "now"},
//	    Pagination: &dash0.CursorPagination{Limit: dash0.Int64(100)},
//	})
//	for iter.Next() {
//	    issue := iter.Current()
//	    // process issue
//	}
//	if err := iter.Err(); err != nil {
//	    // handle error
//	}
func (c *client) GetFailedChecksIter(ctx context.Context, request *GetFailedChecksRequest) *Iter[Issue] {
	resp, err := c.GetFailedChecks(ctx, request)
	if err != nil {
		return newIterWithError[Issue](err)
	}

	items := toPointerSlice(resp.Issues)
	var cursor *string
	hasMore := false
	if resp.Cursors != nil && resp.Cursors.After != nil {
		cursor = (*string)(resp.Cursors.After)
		hasMore = true
	}

	return newIter(items, hasMore, cursor, func(cur *string) ([]*Issue, *string, error) {
		nextReq := *request
		if nextReq.Pagination == nil {
			nextReq.Pagination = &CursorPagination{}
		} else {
			paginationCopy := *nextReq.Pagination
			nextReq.Pagination = &paginationCopy
		}
		nextReq.Pagination.Cursor = (*Cursor)(cur)

		resp, err := c.GetFailedChecks(ctx, &nextReq)
		if err != nil {
			return nil, nil, err
		}

		items := toPointerSlice(resp.Issues)
		var nextCursor *string
		if resp.Cursors != nil && resp.Cursors.After != nil {
			nextCursor = (*string)(resp.Cursors.After)
		}
		return items, nextCursor, nil
	})
}
