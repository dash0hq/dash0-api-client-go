# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands

```bash
make              # Clean, generate, tidy, fmt, lint, build, test (full cycle)
make test         # Run tests with race detection and coverage
make lint         # Run golangci-lint
make generate     # Regenerate generated.go from OpenAPI spec
make test-coverage # Generate HTML coverage report
```

Run a single test:
```bash
go test -v -run TestNewClient ./...
go test -v -run TestIter/fetches_next_page ./...
```

Live API tests (requires env vars):
```bash
DASH0_API_URL=https://api.eu-west-1.aws.dash0.com DASH0_AUTH_TOKEN=auth_xxx go test -v -run TestLiveAPI ./...
```

## Architecture

This is a Go client library for the Dash0 API, wrapping an OpenAPI-generated client with a high-level interface.

### Layer Stack

```
Client interface (client.go)          ← Public API users interact with
    ↓
client struct (client_*.go)           ← High-level method implementations
    ↓
ClientWithResponses (generated.go)    ← oapi-codegen generated client
    ↓
Transport stack:
  retryTransport (transport.go)       ← Exponential backoff retry
  rateLimitedTransport (transport.go) ← Semaphore-based concurrency limit
  http.DefaultTransport
```

### Key Design Patterns

**Interface-based client**: `Client` is an interface in `client.go`, with `client` struct as the implementation. This enables mocking via `dash0test.MockClient`.

**Idempotent POST marking**: POST endpoints that are read-only (GetSpans, GetLogRecords) use `withIdempotent(ctx)` to enable retry logic. See `context.go`.

**Generic iterator**: `Iter[T]` in `iterator.go` handles cursor-based pagination. Methods like `GetSpansIter` return iterators that auto-fetch pages.

**Code generation**: `generated.go` is produced by oapi-codegen from the Dash0 OpenAPI spec. The Makefile post-processes it to rename conflicting symbols (e.g., `ClientOption` → `generatedClientOption`).

### File Organization

| File | Purpose |
|------|---------|
| `client.go` | `Client` interface + `NewClient` factory |
| `client_*.go` | Domain-specific methods (dashboards, views, spans, etc.) |
| `transport.go` | HTTP middleware: rate limiting, retry with backoff |
| `iterator.go` | Generic pagination iterator |
| `errors.go` | `APIError` type and helpers (`IsNotFound`, `IsUnauthorized`, etc.) |
| `dash0test/mock.go` | `MockClient` for testing |

### Testing

Use `dash0test.MockClient` with function fields to mock specific methods:

```go
mock := &dash0test.MockClient{
    ListDashboardsFunc: func(ctx context.Context, dataset *string) ([]*dash0.DashboardApiListItem, error) {
        return []*dash0.DashboardApiListItem{{Id: dash0.Ptr("test")}}, nil
    },
}
```
