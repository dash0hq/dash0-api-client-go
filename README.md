# Dash0 Go API Client [![GoDoc](https://godoc.org/github.com/dash0hq/dash0-api-client-go?status.png)](https://godoc.org/github.com/dash0hq/dash0-api-client-go)

A Go client library for the [Dash0](https://www.dash0.com) API.

## Requirements

Go 1.25 or later.

## Installation

```bash
go get github.com/dash0hq/dash0-api-client-go
```

## Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/dash0hq/dash0-api-client-go"
)

func main() {
    // Create a new client
    client, err := dash0.NewClient(
        dash0.WithApiUrl("https://api.eu-west-1.aws.dash0.com"),
        dash0.WithAuthToken("your-auth-token"),
    )
    if err != nil {
        log.Fatal(err)
    }

    // List dashboards in the "default" dataset
    dashboards, err := client.ListDashboards(context.Background(), dash0.String("default"))
    if err != nil {
        if dash0.IsUnauthorized(err) {
            log.Fatal("Invalid API token")
        }
        log.Fatal(err)
    }

    for _, d := range dashboards {
        fmt.Printf("Dashboard: %s (ID: %s)\n", d.Name, d.Id)
    }
}
```

## Configuration Options

| Option                         | Description                              | Default                     |
|--------------------------------|------------------------------------------|-----------------------------|
| `WithApiUrl(url)`              | Dash0 API URL (required)                 | -                           |
| `WithAuthToken(token)`         | Auth token for authentication (required) | -                           |
| `WithMaxConcurrentRequests(n)` | Maximum concurrent API requests (1-10)   | 3                           |
| `WithTimeout(duration)`        | HTTP request timeout                     | 30s                         |
| `WithHTTPClient(client)`       | Custom HTTP client                       | -                           |
| `WithUserAgent(ua)`            | Custom User-Agent header                 | `dash0-api-client-go/1.0.0` |

## Automatic Retries

The client automatically retries failed requests with exponential backoff:

- **Retried errors**: 429 (rate limited) and 5xx (server errors)
- **Max retries**: 3 attempts
- **Backoff**: Exponential with jitter, starting at 500ms up to 30s
- **Retry-After**: Respected when present in response headers

Only idempotent requests (GET, PUT, DELETE, HEAD, OPTIONS) are retried automatically.

## Pagination with Iterators

For endpoints that return paginated results, use iterators to automatically fetch all pages:

```go
// Iterate over all spans in a time range
iter := client.GetSpansIter(ctx, &dash0.GetSpansRequest{
    TimeRange: dash0.TimeReferenceRange{
        From: "now-1h",
        To:   "now",
    },
})

for iter.Next() {
    resourceSpan := iter.Current()
    // process resourceSpan
}
if err := iter.Err(); err != nil {
    log.Fatal(err)
}
```

## Error Handling

All API errors are returned as `*dash0.APIError`, which includes the status code, message, and trace ID for support:

```go
dashboards, err := client.ListDashboards(ctx, nil)
if err != nil {
    if apiErr, ok := err.(*dash0.APIError); ok {
        fmt.Printf("API error: %s (status: %d, trace_id: %s)\n",
            apiErr.Message, apiErr.StatusCode, apiErr.TraceID)
    }
}
```

Helper functions for common error checks:

```go
if dash0.IsNotFound(err) {
    // Handle 404
}
if dash0.IsUnauthorized(err) {
    // Handle 401 - invalid or expired token
}
if dash0.IsForbidden(err) {
    // Handle 403 - insufficient permissions
}
if dash0.IsRateLimited(err) {
    // Handle 429 - too many requests
}
if dash0.IsServerError(err) {
    // Handle 5xx - server errors
}
if dash0.IsBadRequest(err) {
    // Handle 400 - invalid request
}
if dash0.IsConflict(err) {
    // Handle 409 - resource conflict
}
```

## Testing

The `dash0.Client` is an interface, making it easy to mock in tests. Use `dash0test.MockClient` for a ready-to-use mock implementation:

```go
package mypackage

import (
    "context"
    "testing"

    "github.com/dash0hq/dash0-api-client-go"
    "github.com/dash0hq/dash0-api-client-go/dash0test"
)

// MyService uses the Dash0 client
type MyService struct {
    client dash0.Client
}

func TestMyService(t *testing.T) {
    // Create a mock client with custom behavior
    mock := &dash0test.MockClient{
        ListDashboardsFunc: func(ctx context.Context, dataset *string) ([]*dash0.DashboardApiListItem, error) {
            return []*dash0.DashboardApiListItem{
                {Id: dash0.Ptr("dashboard-1"), Name: dash0.Ptr("My Dashboard")},
            }, nil
        },
    }

    // Inject the mock into your service
    svc := &MyService{client: mock}

    // Test your service...
    _ = svc
}
```

## License

See [LICENSE](LICENSE) for details.
