package profiles

import (
	"context"
	"testing"
)

func TestContextRoundTrip(t *testing.T) {
	cfg := &Configuration{
		ApiUrl:    "https://api.example.com",
		AuthToken: "auth_test-token",
		OtlpUrl:   "https://otlp.example.com",
		Dataset:   "my-dataset",
	}

	ctx := WithConfiguration(context.Background(), cfg)
	got := FromContext(ctx)

	if got == nil {
		t.Fatal("expected configuration from context, got nil")
	}
	if got.ApiUrl != cfg.ApiUrl {
		t.Errorf("expected ApiUrl %s, got %s", cfg.ApiUrl, got.ApiUrl)
	}
	if got.AuthToken != cfg.AuthToken {
		t.Errorf("expected AuthToken %s, got %s", cfg.AuthToken, got.AuthToken)
	}
	if got.OtlpUrl != cfg.OtlpUrl {
		t.Errorf("expected OtlpUrl %s, got %s", cfg.OtlpUrl, got.OtlpUrl)
	}
	if got.Dataset != cfg.Dataset {
		t.Errorf("expected Dataset %s, got %s", cfg.Dataset, got.Dataset)
	}
}

func TestFromContext_empty(t *testing.T) {
	got := FromContext(context.Background())
	if got != nil {
		t.Errorf("expected nil from empty context, got %v", got)
	}
}
