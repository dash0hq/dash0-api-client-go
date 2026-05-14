package dash0

import (
	"encoding/json"
	"testing"
)

func TestErrorResponseUnmarshal_BareStringErrorField(t *testing.T) {
	var resp ErrorResponse
	if err := json.Unmarshal([]byte(`{"error": "auth required"}`), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp.Error.Message != "auth required" {
		t.Errorf("expected Message=%q, got %q", "auth required", resp.Error.Message)
	}
	if resp.Error.Code != 0 {
		t.Errorf("expected Code=0 for bare-string form, got %d", resp.Error.Code)
	}
	if resp.Error.TraceId != nil {
		t.Errorf("expected TraceId=nil for bare-string form, got %v", *resp.Error.TraceId)
	}
}

func TestErrorResponseUnmarshal_StructErrorField(t *testing.T) {
	var resp ErrorResponse
	body := []byte(`{"error": {"code": 401, "message": "unauthorized", "traceId": "abc123"}}`)
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp.Error.Code != 401 {
		t.Errorf("expected Code=401, got %d", resp.Error.Code)
	}
	if resp.Error.Message != "unauthorized" {
		t.Errorf("expected Message=%q, got %q", "unauthorized", resp.Error.Message)
	}
	if resp.Error.TraceId == nil || *resp.Error.TraceId != "abc123" {
		got := "<nil>"
		if resp.Error.TraceId != nil {
			got = *resp.Error.TraceId
		}
		t.Errorf("expected TraceId=%q, got %q", "abc123", got)
	}
}
