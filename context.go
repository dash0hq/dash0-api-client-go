package dash0

import "context"

type contextKey string

const (
	// idempotentKey is the context key for marking a request as idempotent.
	idempotentKey contextKey = "dash0_idempotent"
)

// withIdempotent returns a new context that marks the request as idempotent.
// This allows POST requests to be retried on transient failures.
// Use this for read-only POST endpoints like GetSpans and GetLogRecords.
//
// Example:
//
//	ctx := dash0.withIdempotent(context.Background())
//	spans, err := client.GetSpans(ctx, request)
func withIdempotent(ctx context.Context) context.Context {
	return context.WithValue(ctx, idempotentKey, true)
}

// isIdempotent returns true if the context has been marked as idempotent.
func isIdempotent(ctx context.Context) bool {
	v, ok := ctx.Value(idempotentKey).(bool)
	return ok && v
}
