package profiles

import (
	"testing"
)

func TestClientOptions(t *testing.T) {
	t.Run("all fields set", func(t *testing.T) {
		cfg := &Configuration{
			ApiUrl:    "https://api.example.com",
			AuthToken: "auth_test-token",
			OtlpUrl:   "https://otlp.example.com",
			Dataset:   "my-dataset",
		}

		opts := cfg.ClientOptions()
		// ApiUrl, AuthToken, and OtlpUrl are mapped; Dataset is not.
		if len(opts) != 3 {
			t.Errorf("expected 3 options, got %d", len(opts))
		}
	})

	t.Run("only required fields", func(t *testing.T) {
		cfg := &Configuration{
			ApiUrl:    "https://api.example.com",
			AuthToken: "auth_test-token",
		}

		opts := cfg.ClientOptions()
		if len(opts) != 2 {
			t.Errorf("expected 2 options, got %d", len(opts))
		}
	})

	t.Run("empty configuration", func(t *testing.T) {
		cfg := &Configuration{}
		opts := cfg.ClientOptions()
		if len(opts) != 0 {
			t.Errorf("expected 0 options, got %d", len(opts))
		}
	})

	t.Run("OTLP only", func(t *testing.T) {
		cfg := &Configuration{
			AuthToken: "auth_test-token",
			OtlpUrl:   "https://otlp.example.com",
		}

		opts := cfg.ClientOptions()
		if len(opts) != 2 {
			t.Errorf("expected 2 options (AuthToken + OtlpUrl), got %d", len(opts))
		}
	})
}
