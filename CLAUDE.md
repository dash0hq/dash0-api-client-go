# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build commands

```bash
make              # Clean, generate, tidy, fmt, lint, build, test (full cycle)
make test         # Run tests with race detection and coverage
make lint         # Run golangci-lint
make generate     # Regenerate generated.go from OpenAPI spec
make test-coverage # Generate HTML coverage report
make api-compat   # Check API compatibility against latest release tag
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

### Layer stack

```
Client interface (client.go)          <- Public API users interact with
    |
client struct (client_*.go)           <- High-level method implementations
    |
ClientWithResponses (generated.go)    <- oapi-codegen generated client
    |
Transport stack:
  retryTransport (transport.go)       <- Exponential backoff retry
  rateLimitedTransport (transport.go) <- Semaphore-based concurrency limit
  http.DefaultTransport
```

### Key design patterns

**Interface-based client**: `Client` is an interface in `client.go`, with `client` struct as the implementation.
This enables mocking via `dash0test.MockClient`.

**Idempotent POST marking**: POST endpoints that are read-only (GetSpans, GetLogRecords) use `withIdempotent(ctx)` to enable retry logic.
See `context.go`.

**Generic iterator**: `Iter[T]` in `iterator.go` handles cursor-based pagination.
Methods like `GetSpansIter` return iterators that auto-fetch pages.

**Code generation**: `generated.go` is produced by oapi-codegen from the Dash0 OpenAPI spec.
The `tools/postprocess` Go tool handles post-processing: renaming conflicting symbols and removing deprecated fields.
Do not use `sed` for codegen post-processing.
The generated code produces transport-specific type aliases (e.g., `PostApiImportCheckRuleJSONRequestBody = PrometheusAlertRule`).
The public `Client` interface must use domain types (`*PrometheusAlertRule`, `*DashboardDefinition`, etc.), never generated request/response body type names that leak HTTP methods, paths, or encodings.

**Domain-grouped helpers**: Each `client_*.go` file contains CRUD methods for a domain entity plus free helper functions that operate on that entity's generated types.
Keep helpers co-located with their domain's CRUD methods.

### File organization

| File | Purpose |
|------|---------|
| `client.go` | `Client` interface + `NewClient` factory |
| `client_*.go` | Domain-specific CRUD methods and helpers (dashboards, check rules, views, etc.) |
| `generated.go` | oapi-codegen generated client (do not edit manually) |
| `tools/postprocess/` | AST-based post-processing of `generated.go` |
| `transport.go` | HTTP middleware: rate limiting, retry with backoff |
| `iterator.go` | Generic pagination iterator |
| `context.go` | Context helpers (idempotent POST marking) |
| `errors.go` | `APIError` type and helpers (`IsNotFound`, `IsUnauthorized`, etc.) |
| `dataset.go` | `DatasetPtr` helper |
| `dependencies_test.go` | Guard test: verifies YAML deps don't leak into root package |
| `dash0test/mock.go` | `MockClient` for testing |
| `yaml/` | YAML/CRD parsing subpackage (see below) |

### YAML subpackage (`yaml/`)

All YAML-related types, parsing, and conversion logic lives in the `yaml/` subpackage.
This isolates the `sigs.k8s.io/yaml` dependency so that consumers who only use the API client (root package) do not pull it in.

**The root package must never import `sigs.k8s.io/yaml`, `gopkg.in/yaml.v3`, or any other YAML library.**
When adding new YAML/CRD functionality, always place it in `yaml/`.
The `TestNoYAMLDependency` guard test in `dependencies_test.go` enforces this at CI time.

The subpackage imports the root package for generated API types (e.g., `dash0.DashboardDefinition`, `dash0.PrometheusAlertRule`) -- never the reverse.
All struct tags in `yaml/` use `json:` only (no `yaml:` tags), because `sigs.k8s.io/yaml` unmarshals via JSON.

| File | Purpose |
|------|---------|
| `yaml/prometheus_types.go` | Prometheus CRD types (`PrometheusRules`, `PrometheusRule`, `PromDuration`) and `FormatDuration` |
| `yaml/perses_types.go` | Perses CRD types (`PersesDashboard`, `PersesDashboardMetadata`) |
| `yaml/prometheus_alert_rules.go` | Prometheus alert rule marshal/unmarshal, conversion, extraction, and parsing |
| `yaml/dashboards.go` | Dashboard conversion, extraction, and parsing (`ParseAsDashboard`, `ConvertPersesDashboardToDashboard`, etc.) |
| `yaml/kind.go` | `DetectKind` -- sniffs asset kind from raw YAML/JSON bytes |

### API compatibility

The CI workflow runs [`gorelease`](https://pkg.go.dev/golang.org/x/exp/cmd/gorelease) on pull requests to detect breaking API changes against the latest release tag.
It checks full type signatures (not just symbol names), covering both hand-written and generated code.
The build fails if unallowed incompatible changes are detected.

Intentional breaking changes (e.g., from upstream OpenAPI spec updates) must be listed in `api_compatibility_exceptions.txt`, one symbol per line.
Clear this file when cutting a new release tag.

When introducing a breaking change:

1. Add the symbol to `api_compatibility_exceptions.txt` with a version comment (e.g., `# <NEXT_RELEASE>: reason`).
2. If possible, add a backwards-compatible alias in `compat.go` with a `// Deprecated: since <NEXT_RELEASE>. Use [NewName] instead.` doc comment.
   Use `<NEXT_RELEASE>` as the version placeholder -- the prepare-release workflow replaces it with the actual version at release time.
   Go tooling (`go vet`, `staticcheck`, IDE hints) surfaces `// Deprecated:` annotations automatically, guiding both humans and AI agents to the replacement.
3. Document the migration in the `compat.go` file-level doc comment under the "Migration guide" section.
   This comment is visible via `go doc dash0`, which is how agents and developers discover migration steps when upgrading.
   Use `go doc` [linking syntax](https://tip.golang.org/doc/comment#links) (`[NewSymbol]`) to cross-reference replacement symbols.

To run locally:

```bash
make api-compat
```

### Code review

See [REVIEW-CRITERIA.md](REVIEW-CRITERIA.md) for the review lenses used on this codebase: abstraction cleanliness, reusability, and encapsulation.

## Prose rules

Follow these rules when writing or editing prose in this project (doc comments, README, CLAUDE.md, etc.).

### Line and paragraph structure

- **One sentence per line** (semantic line breaks).
  Each sentence starts on its own line; do not wrap mid-sentence.
- Separate paragraphs with a single blank line.
- Keep paragraphs between 2 and 5 sentences.

### Section headers

Write section headers in sentence case, e.g., "This is an example".

### Links

- Use inline Markdown links: `[visible text](url)`.
- Link the most specific relevant term, not generic phrases like "click here" or "this page."

### Code blocks

- Fence with triple backticks and a language identifier (e.g., ` ```yaml `).
- Use code blocks to provide illustrative examples.

### Punctuation and typography

- End sentences with full stops.
- Use the **Oxford comma** (e.g., "error status, latency thresholds, rate limits, and so on").
- Write numbers as digits and spell out "percent" (e.g., "10 percent", not "10%" or "ten percent").

## Code style

### Error handling

- Wrap errors with descriptive messages using `fmt.Errorf("... %w", err)`.
- Never use lazy pluralization like `error(s)` or `file(s)` -- use proper singular/plural forms based on the actual count (e.g., "1 error", "3 errors").

### Dependencies

- Never add dependencies with licenses incompatible with Apache 2.0.
  Acceptable licenses: Apache 2.0, MIT, BSD-2-Clause, BSD-3-Clause, ISC, and MPL-2.0.
  Reject GPL, LGPL, AGPL, SSPL, and other copyleft licenses.
- Always check the license before adding a dependency.

## Style guide

### Naming conventions

**Helper functions** for each asset type follow a consistent pattern.
Every asset type must have the full set:

```
Strip<Asset>ServerFields(<param> *<Type>)
Get<Asset>ID(<param> *<Type>) string
Get<Asset>Name(<param> *<Type>) string
Set<Asset>ID(<param> *<Type>, id string)
Set<Asset>IDIfAbsent(<param> *<Type>, id string)
Clear<Asset>ID(<param> *<Type>)
```

- `Set*ID` unconditionally overwrites the ID.
  `Set*IDIfAbsent` only sets the ID when it is not already present.
- Parameter names must be consistent within a file (e.g., all functions in `client_check_rules.go` use `rule`, not `r`).
- All helper functions that take a pointer parameter must be nil-safe (no panics on nil pointers or zero-value structs).

**Parse/Marshal functions** in the `yaml/` subpackage:

- `ParseAs<Type>(data []byte)` -- smart parsing that detects kind and may convert between formats (e.g., Perses -> Dashboard).
- `Unmarshal<Type>(data []byte, ...)` -- direct deserialization from YAML bytes.
- `Marshal<Type>(obj *<Type>) ([]byte, error)` -- direct serialization to YAML bytes.
- `Convert<Source>To<Target>(src *<Source>) *<Target>` -- struct-to-struct conversion with no serialization.
- All byte-oriented functions use `[]byte`, not `string`.
  Callers convert at the boundary.
- Do not repeat the package name in function names (e.g., `yaml.MarshalPrometheusRule`, not `yaml.MarshalPrometheusRuleYAML`).

**Typed constants** over free-form strings:

- Use typed constants for any value passed by callers where typos would fail silently.

### Dependency isolation

- The root package must have zero YAML dependencies.
  All YAML logic goes in `yaml/`.
- Use a single YAML library (`sigs.k8s.io/yaml`), never mix with `gopkg.in/yaml.v3`.
- Struct tags in `yaml/` use `json:` only (no `yaml:` tags).
- The `yaml/` subpackage imports the root package; never the reverse.

### Testing

**Test file naming**: Each `foo.go` file should have a corresponding `foo_test.go`.
Do not combine tests for multiple source files into a single test file.
Shared test helpers go in `helpers_test.go`.

**URL assertions**: When testing URL output, parse with `net/url` and assert on scheme, host, path, and query parameters individually.
Do not use substring matching on URL strings.

**Nil-safety tests**: Every helper function that takes a pointer parameter (`Get*`, `Set*`, `Clear*`, `Strip*`) must have a test that passes `nil` to verify it does not panic.

Use `dash0test.MockClient` with function fields to mock specific methods:

```go
mock := &dash0test.MockClient{
    ListDashboardsFunc: func(ctx context.Context, dataset *string) ([]*dash0.DashboardApiListItem, error) {
        return []*dash0.DashboardApiListItem{{Id: dash0.Ptr("test")}}, nil
    },
}
```

**Example tests**: Each package has a single `example_test.go` file containing `Example*` functions.
When adding a new public function or method, add a corresponding `Example` function in `example_test.go`.
Examples must include an `// Output:` comment so `go test` verifies them.
Do not split examples across multiple files.
