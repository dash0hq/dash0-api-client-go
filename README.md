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
        dash0.WithAuthToken("auth_your-auth-token"),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close(context.Background())

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

## Sending Telemetry Data (OTLP)

The client can push OpenTelemetry traces, metrics, and logs to an OTLP endpoint with `otlp/json` encoding. You can create a client with only an OTLP endpoint, only a REST API URL, or both:

```go
// OTLP-only client (no REST API access)
client, err := dash0.NewClient(
    dash0.WithAuthToken("auth_your-auth-token"),
    dash0.WithOtlpEndpoint(dash0.OtlpEncodingJson, "https://otlp.eu-west-1.aws.dash0.com"),
)

// Combined client (REST API + OTLP)
client, err := dash0.NewClient(
    dash0.WithApiUrl("https://api.eu-west-1.aws.dash0.com"),
    dash0.WithAuthToken("auth_your-auth-token"),
    dash0.WithOtlpEndpoint(dash0.OtlpEncodingJson, "https://otlp.eu-west-1.aws.dash0.com"),
)
```

The `SendTraces`, `SendMetrics`, and `SendLogs` methods accept [pdata](https://pkg.go.dev/go.opentelemetry.io/collector/pdata) types (`ptrace.Traces`, `pmetric.Metrics`, `plog.Logs`). Signal-specific paths (`/v1/traces`, `/v1/metrics`, `/v1/logs`) are appended automatically to the base endpoint URL. Pass `nil` as the dataset to send data to the `default` dataset in Dash0.

```go
// Send to the default dataset
err = client.SendTraces(ctx, traces, nil)
err = client.SendMetrics(ctx, metrics, nil)
err = client.SendLogs(ctx, logs, nil)

// Send to a specific dataset
err = client.SendTraces(ctx, traces, dash0.String("my-dataset"))
err = client.SendMetrics(ctx, metrics, dash0.String("my-dataset"))
err = client.SendLogs(ctx, logs, dash0.String("my-dataset"))
```

OTLP requests use the same HTTP client, retry logic, and rate limiting as the REST API calls. Call `Close` when the underlying HTTP client is no longer needed.

## Configuration Options

| Option                         | Description                              | Default                     |
|--------------------------------|------------------------------------------|-----------------------------|
| `WithApiUrl(url)`              | Dash0 API URL (required for REST API)    | -                           |
| `WithAuthToken(token)`         | Auth token for authentication (required) | -                           |
| `WithOtlpEndpoint(enc, url)`   | OTLP/HTTP endpoint for telemetry push (required for OTLP)   | -                           |
| `WithMaxConcurrentRequests(n)` | Maximum concurrent API requests (1-10)   | 3                           |
| `WithTimeout(duration)`        | HTTP request timeout                     | 30s                         |
| `WithHTTPClient(client)`       | Custom HTTP client                       | -                           |
| `WithUserAgent(ua)`            | Custom User-Agent header                 | `dash0-api-client-go/1.0.0` |
| `WithMaxRetries(n)`            | Maximum retries for failed requests (0-5)| 1                           |
| `WithRetryWaitMin(duration)`   | Minimum wait between retries             | 500ms                       |
| `WithRetryWaitMax(duration)`   | Maximum wait between retries             | 30s                         |

`NewClient` requires `WithAuthToken` and at least one of `WithApiUrl` or `WithOtlpEndpoint`. REST API methods (dashboards, check rules, etc.) require `WithApiUrl`; OTLP methods (`SendTraces`, `SendMetrics`, `SendLogs`) require `WithOtlpEndpoint`. Calling a method whose endpoint was not configured returns an error.

## Automatic Retries

The client automatically retries failed requests with exponential backoff:

- **Retried errors**: 429 (rate limited) and 5xx (server errors)
- **Max retries**: 1 (configurable up to 5 via `WithMaxRetries`)
- **Backoff**: Exponential with jitter, starting at 500ms up to 30s
- **Retry-After**: Respected when present in response headers

Only idempotent requests (GET, PUT, DELETE, HEAD, OPTIONS) and OTLP sends are retried automatically.

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

### HTTP errors

Both REST API and OTLP methods return `*dash0.APIError` for non-2xx HTTP responses. The error includes the status code, message, and trace ID for support:

```go
err := client.SendTraces(ctx, traces, nil)
if err != nil {
    if apiErr, ok := err.(*dash0.APIError); ok {
        fmt.Printf("API error: %s (status: %d, trace_id: %s)\n",
            apiErr.Message, apiErr.StatusCode, apiErr.TraceID)
    }
}
```

The same helper functions work for both REST API and OTLP errors:

```go
if dash0.IsUnauthorized(err) {
    // Handle 401 - invalid or expired token
}
if dash0.IsRateLimited(err) {
    // Handle 429 - too many requests
}
if dash0.IsServerError(err) {
    // Handle 5xx - server errors
}
if dash0.IsNotFound(err) {
    // Handle 404
}
if dash0.IsForbidden(err) {
    // Handle 403 - insufficient permissions
}
if dash0.IsBadRequest(err) {
    // Handle 400 - invalid request
}
if dash0.IsConflict(err) {
    // Handle 409 - resource conflict
}
```

### Not-configured errors

Calling a method whose endpoint was not configured returns a sentinel error that can be checked with `errors.Is`:

```go
if errors.Is(err, dash0.ErrOTLPNotConfigured) {
    // SendTraces/SendMetrics/SendLogs called without WithOtlpEndpoint
}
if errors.Is(err, dash0.ErrAPINotConfigured) {
    // REST API method called without WithApiUrl
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
