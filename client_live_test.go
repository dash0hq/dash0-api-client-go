package dash0

import (
	"context"
	"os"
	"testing"
)

// TestLiveAPI runs integration tests against a real Dash0 API.
// These tests are skipped unless DASH0_API_URL and DASH0_AUTH_TOKEN are set.
func TestLiveAPI(t *testing.T) {
	apiUrl := os.Getenv("DASH0_API_URL")
	authToken := os.Getenv("DASH0_AUTH_TOKEN")

	if apiUrl == "" || authToken == "" {
		t.Skip("Skipping live API tests: DASH0_API_URL and DASH0_AUTH_TOKEN must be set")
	}

	client, err := NewClient(
		WithApiUrl(apiUrl),
		WithAuthToken(authToken),
	)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx := context.Background()

	t.Run("ListDashboards", func(t *testing.T) {
		dashboards, err := client.ListDashboards(ctx, nil)
		if err != nil {
			t.Fatalf("ListDashboards failed: %v", err)
		}

		if len(dashboards) == 0 {
			t.Error("expected at least one dashboard, got none")
		}

		t.Logf("Found %d dashboards", len(dashboards))
	})

	t.Run("ListCheckRules", func(t *testing.T) {
		checkRules, err := client.ListCheckRules(ctx, nil)
		if err != nil {
			t.Fatalf("ListCheckRules failed: %v", err)
		}

		if len(checkRules) == 0 {
			t.Error("expected at least one check rule, got none")
		}

		t.Logf("Found %d check rules", len(checkRules))
	})

	t.Run("GetSpans", func(t *testing.T) {
		request := GetSpansRequest{
			TimeRange: TimeReferenceRange{
				From: "now-5m",
				To:   "now",
			},
			Pagination: &CursorPagination{
				Limit: Int64(10),
			},
		}

		resp, err := client.GetSpans(ctx, &request)
		if err != nil {
			t.Fatalf("GetSpans failed: %v", err)
		}

		if len(resp.ResourceSpans) == 0 {
			t.Error("expected at least one span in the last 5 minutes, got none")
		} else {
			t.Logf("Found spans in the last 5 minutes")
		}
	})

	t.Run("GetLogRecords", func(t *testing.T) {
		request := GetLogRecordsRequest{
			TimeRange: TimeReferenceRange{
				From: "now-5m",
				To:   "now",
			},
			Pagination: &CursorPagination{
				Limit: Int64(10),
			},
		}

		resp, err := client.GetLogRecords(ctx, &request)
		if err != nil {
			t.Fatalf("GetLogRecords failed: %v", err)
		}

		if len(resp.ResourceLogs) == 0 {
			t.Error("expected at least one log record in the last 5 minutes, got none")
		} else {
			t.Logf("Found log records in the last 5 minutes")
		}
	})

	t.Run("GetSpansIter", func(t *testing.T) {
		request := &GetSpansRequest{
			TimeRange: TimeReferenceRange{
				From: "now-5m",
				To:   "now",
			},
			Pagination: &CursorPagination{
				Limit: Int64(10),
			},
		}

		iter := client.GetSpansIter(ctx, request)
		count := 0
		for iter.Next() {
			_ = iter.Current()
			count++
			if count >= 5 {
				break // Limit iterations since spans can be endless
			}
		}
		if err := iter.Err(); err != nil {
			t.Fatalf("GetSpansIter failed: %v", err)
		}
		t.Logf("Iterated over %d resource spans", count)
	})

	t.Run("GetLogRecordsIter", func(t *testing.T) {
		request := &GetLogRecordsRequest{
			TimeRange: TimeReferenceRange{
				From: "now-5m",
				To:   "now",
			},
			Pagination: &CursorPagination{
				Limit: Int64(10),
			},
		}

		iter := client.GetLogRecordsIter(ctx, request)
		count := 0
		for iter.Next() {
			_ = iter.Current()
			count++
			if count >= 5 {
				break // Limit iterations since logs can be endless
			}
		}
		if err := iter.Err(); err != nil {
			t.Fatalf("GetLogRecordsIter failed: %v", err)
		}
		t.Logf("Iterated over %d resource logs", count)
	})
}
