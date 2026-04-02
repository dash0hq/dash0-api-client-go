# PR review criteria

This document describes the review lenses applied to code reviews for changes to this project.

---

## 1. Cleanliness of abstractions

Evaluates whether each abstraction has a single, well-defined responsibility and whether the boundaries between abstractions are clear and intentional.

**What to look for:**

- Does each type, function, or module have one job?
  Mixed responsibilities (e.g., parsing + converting + validating in the same function) make code harder to test and reuse.
- Are similar operations handled by the same code path?
  Parallel implementations that diverge subtly create maintenance traps -- one gets a bugfix, the other doesn't.
- Do types carry their weight?
  Type aliases (`=`) and `map[string]interface{}` provide naming without safety.
  Named types and structs encode invariants the compiler can check.
- Is the abstraction level consistent?
  Hand-rolling HTTP requests in a codebase built on a generated client breaks the layering.
  Bypassing the transport stack (retry, rate-limiting) is a correctness risk.

**Example findings:**

- `PrometheusRuleCRD` and `PrometheusRules` represented the same Kubernetes CRD with different metadata typing -- consolidated into a single `PrometheusRules` type.
- Helper functions for asset types were scattered across files organized by source format rather than by domain entity -- reorganized to follow the `client_*.go` convention.

---

## 2. Reusability of functions

Evaluates whether functions can be composed and called from different contexts without surprises, and whether the API surface encourages correct usage.

**What to look for:**

- Do functions in the same family behave consistently?
  Every asset type must have the full set of helpers (`Strip*`, `Get*`, `Set*`, `Clear*`) with consistent parameter names and nil-safety guarantees.
- Are parameters typed narrowly enough?
  Free-form `string` parameters where a finite set of values is valid invite typos that fail silently.
  Use typed constants (e.g., `DeeplinkType`) instead.
- Do Parse/Marshal functions use `[]byte` consistently?
  Mixing `string` and `[]byte` for the same kind of data forces unnecessary conversions at call sites.
- Are naming conventions followed?
  `ParseAs*` for smart parsing, `Unmarshal*`/`Marshal*` for direct serialization, `Convert*To*` for struct-to-struct conversion.
  Do not repeat the package name in function names (e.g., `yaml.MarshalPrometheusRule`, not `yaml.MarshalPrometheusRuleYAML`).

**Example findings:**

- `deeplinkPathAndQuery` accepted untyped strings with ad-hoc aliases (`"checkrule"`, `"check rule"`, `"prometheusrule"`) -- replaced with `DeeplinkType` constants.
- `UnmarshalPrometheusRule` took a `string` while all other parse functions took `[]byte` -- unified to `[]byte`.

---

## 3. Encapsulation

Evaluates whether implementation details are hidden behind clean boundaries, and whether modules expose only what they need to.

**What to look for:**

- Do functions do one thing?
  A function that parses YAML, converts types, extracts thresholds, extracts enabled flags, and cleans up annotations is doing five things.
  Each should be testable independently.
- Are mutations visible and intentional?
  Deleting keys from a map parameter is a mutation that callers may not expect, even if the map was copied internally.
- Are assumptions documented?
  A function that assumes two-label TLDs or a specific URL structure should say so in its doc comment.
- Is the dependency direction correct?
  The `yaml/` subpackage imports the root package for generated types -- never the reverse.
  This keeps the YAML dependency from leaking to consumers who don't need it.
- Is generated code post-processing minimal and auditable?
  The `tools/postprocess` tool should only fix what oapi-codegen cannot be configured to handle.
  Do not use `sed` for codegen post-processing.

**Example findings:**

- `DetectKind` fully unmarshaled documents into `map[string]any` just to check 3 top-level keys -- replaced with a minimal `kindProbe` struct.
- `AppBaseURL` encodes an undocumented two-label TLD assumption (breaks for `.co.uk`).

---

## 4. Backwards compatibility

Evaluates whether changes preserve the public API contract and whether breaking changes are handled deliberately.

**What to look for:**

- Are removed or renamed symbols aliased in `compat.go` with `// Deprecated: since <NEXT_RELEASE>` doc comments?
  Aliases let existing consumer code compile while surfacing migration guidance via Go tooling.
- Are unavoidable breaking changes (generated struct field type changes, interface expansions) listed in `api_compatibility_exceptions.txt` with a version comment and rationale?
- Does `make api-compat` pass?
  This runs `gorelease` and filters through the exceptions file.
  New unallowed incompatible changes must be either fixed or explicitly excepted.
- Is the migration guide in `compat.go` updated with instructions for each breaking change?
  This is visible via `go doc` and is how agents and developers discover migration steps.

**Example findings:**

- `DashboardSource` constants were renamed by codegen (`Api` -> `DashboardSourceApi`) without aliases -- added backwards-compatible constants in `compat.go`.
- `CheckThresholds` fields changed from `*float32` to `*float64` (upstream spec) -- cannot be aliased, documented in exceptions file.
