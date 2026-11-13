package profiles

import (
	"context"
	"testing"
)

func TestContextRoundTrip(t *testing.T) {
	cfg := newResolved(Configuration{
		ApiUrl:    "https://api.example.com",
		AuthToken: "auth_test-token",
		OtlpUrl:   "https://otlp.example.com",
		Dataset:   "my-dataset",
	}, "", "")

	ctx := WithConfiguration(context.Background(), cfg)
	got := FromContext(ctx)

	if got != cfg {
		t.Errorf("expected the configuration stored in the context, got %+v", got)
	}
}

func TestFromContext_empty(t *testing.T) {
	got := FromContext(context.Background())
	if got != nil {
		t.Errorf("expected nil from empty context, got %v", got)
	}
}
