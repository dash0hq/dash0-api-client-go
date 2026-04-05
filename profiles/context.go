package profiles

import "context"

type contextKey string

const configContextKey contextKey = "dash0-config"

// WithConfiguration returns a new context with the given configuration stored.
func WithConfiguration(ctx context.Context, cfg *Configuration) context.Context {
	return context.WithValue(ctx, configContextKey, cfg)
}

// FromContext retrieves the configuration from ctx, or nil if not present.
func FromContext(ctx context.Context) *Configuration {
	cfg, _ := ctx.Value(configContextKey).(*Configuration)
	return cfg
}
