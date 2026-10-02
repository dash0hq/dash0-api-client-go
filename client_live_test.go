package dash0

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
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

	t.Run("OriginPrefixFiltersLists", func(t *testing.T) {
		noMatch := WithOriginPrefix(fmt.Sprintf("iac-26-no-such-origin-%d_", time.Now().UnixNano()))
		checkRules, err := client.ListCheckRules(ctx, nil, noMatch)
		if err != nil {
			t.Fatalf("ListCheckRules failed: %v", err)
		}
		recordingRules, err := client.ListRecordingRules(ctx, nil, noMatch)
		if err != nil {
			t.Fatalf("ListRecordingRules failed: %v", err)
		}
		slos, err := client.ListSLOs(ctx, nil, noMatch)
		if err != nil {
			t.Fatalf("ListSLOs failed: %v", err)
		}
		if len(checkRules) != 0 || len(recordingRules) != 0 || len(slos) != 0 {
			t.Errorf("unmatched prefix returned %d check rules, %d recording rules, %d SLOs, want 0 each", len(checkRules), len(recordingRules), len(slos))
		}

		dataset := Ptr("default")
		prefix := fmt.Sprintf("iac-26-live-%d_", time.Now().UnixNano())
		checkOrigin, otherCheckOrigin := prefix+"check", fmt.Sprintf("iac-26-live-other-%d-check", time.Now().UnixNano())
		recordingOrigin, otherRecordingOrigin := prefix+"recording", fmt.Sprintf("iac-26-live-other-%d-recording", time.Now().UnixNano())
		t.Cleanup(func() {
			for _, origin := range []string{checkOrigin, otherCheckOrigin} {
				if err := client.DeleteCheckRule(context.Background(), origin, dataset); err != nil && !IsNotFound(err) {
					t.Logf("cleanup of check rule %s failed: %v", origin, err)
				}
			}
			for _, origin := range []string{recordingOrigin, otherRecordingOrigin} {
				if err := client.DeleteRecordingRule(context.Background(), origin, dataset); err != nil && !IsNotFound(err) {
					t.Logf("cleanup of recording rule %s failed: %v", origin, err)
				}
			}
		})

		for _, origin := range []string{checkOrigin, otherCheckOrigin} {
			rule := &PrometheusAlertRule{Name: "IAC26LiveTest", Expression: "vector(0) > 1"}
			if _, err := client.UpdateCheckRule(ctx, origin, rule, dataset); err != nil {
				t.Fatalf("UpdateCheckRule(%s) as upsert failed: %v", origin, err)
			}
		}
		for _, origin := range []string{recordingOrigin, otherRecordingOrigin} {
			rule := &RecordingRule{
				ApiVersion: MonitoringCoreosComv1,
				Kind:       PrometheusRuleKindPrometheusRule,
				Metadata:   PrometheusRuleMetadata{Name: "iac-26-live-test"},
				Spec: PrometheusRuleSpec{Groups: []PrometheusRuleGroup{{
					Name:  "iac-26-live-test",
					Rules: []PrometheusRuleDefinition{{Record: Ptr("iac_26_live_test"), Expr: "vector(0)"}},
				}}},
			}
			if _, err := client.UpdateRecordingRule(ctx, origin, rule, dataset); err != nil {
				t.Fatalf("UpdateRecordingRule(%s) as upsert failed: %v", origin, err)
			}
		}

		allCheckRules, err := client.ListCheckRules(ctx, dataset)
		if err != nil {
			t.Fatalf("ListCheckRules failed: %v", err)
		}
		matchingCheckRules, err := client.ListCheckRules(ctx, dataset, WithOriginPrefix(prefix))
		if err != nil {
			t.Fatalf("ListCheckRules with prefix failed: %v", err)
		}
		if len(matchingCheckRules) != 1 || StringValue(matchingCheckRules[0].Origin) != checkOrigin {
			t.Errorf("ListCheckRules with prefix returned %d rules, want exactly %s", len(matchingCheckRules), checkOrigin)
		}
		if len(allCheckRules) < 2 {
			t.Errorf("unfiltered ListCheckRules returned %d rules, want at least 2", len(allCheckRules))
		}
		iterated := 0
		for iter := client.ListCheckRulesIter(ctx, dataset, WithOriginPrefix(prefix)); iter.Next(); {
			iterated++
		}
		if iterated != 1 {
			t.Errorf("ListCheckRulesIter with prefix yielded %d rules, want 1", iterated)
		}

		allRecordingRules, err := client.ListRecordingRules(ctx, dataset)
		if err != nil {
			t.Fatalf("ListRecordingRules failed: %v", err)
		}
		matchingRecordingRules, err := client.ListRecordingRules(ctx, dataset, WithOriginPrefix(prefix))
		if err != nil {
			t.Fatalf("ListRecordingRules with prefix failed: %v", err)
		}
		recordingOriginLabel := func(rule *RecordingRule) string {
			if rule.Metadata.Labels == nil {
				return ""
			}
			return (*rule.Metadata.Labels)[LabelOrigin]
		}
		if len(matchingRecordingRules) != 1 || recordingOriginLabel(matchingRecordingRules[0]) != recordingOrigin {
			t.Errorf("ListRecordingRules with prefix returned %d rules, want exactly %s", len(matchingRecordingRules), recordingOrigin)
		}
		if len(allRecordingRules) < 2 {
			t.Errorf("unfiltered ListRecordingRules returned %d rules, want at least 2", len(allRecordingRules))
		}
		t.Logf("check rules: %d total, %d matching; recording rules: %d total, %d matching",
			len(allCheckRules), len(matchingCheckRules), len(allRecordingRules), len(matchingRecordingRules))
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

	t.Run("ListTeams", func(t *testing.T) {
		resp, err := client.ListTeams(ctx)
		if err != nil {
			t.Fatalf("ListTeams failed: %v", err)
		}

		if len(resp) == 0 {
			t.Error("expected at least one team in the response, got none")
		} else {
			t.Logf("Found teams: %v", len(resp))
			for _, team := range resp {
				t.Logf("Team name: %s (members: %v)", team.Name, team.TotalMemberCount)
			}
		}
	})
}
