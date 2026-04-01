# PR Review Criteria

This document describes the three review lenses applied to PR #7 and how each was evaluated.

---

## 1. Cleanliness of Abstractions

Evaluates whether each abstraction has a single, well-defined responsibility and whether the boundaries between abstractions are clear and intentional.

**What to look for:**
- Does each type, function, or module have one job? Mixed responsibilities (e.g., parsing + converting + validating in the same function) make code harder to test and reuse.
- Are similar operations handled by the same code path? Parallel implementations that diverge subtly create maintenance traps — one gets a bugfix, the other doesn't.
- Do types carry their weight? Type aliases (`=`) and `map[string]interface{}` provide naming without safety. Named types and structs encode invariants the compiler can check.
- Is the abstraction level consistent? Hand-rolling HTTP requests in a codebase built on a generated client breaks the layering. Bypassing the transport stack (retry, rate-limiting) is a correctness risk.

**Example findings from this review:**
- `ConvertPrometheusYAMLToCheckRule` duplicated conversion logic from `ConvertPrometheusRuleToCheckRule` instead of delegating.
- `RecordingRuleGroupDefinition = map[string]interface{}` is a type alias with no type safety.

---

## 2. Reusability of Functions

Evaluates whether functions can be composed and called from different contexts without surprises, and whether the API surface encourages correct usage.

**What to look for:**
- Do functions in the same family behave consistently? If `StripXServerFields` initializes nil containers for some types but not others, callers must know which type they're dealing with — the abstraction leaks.
- Are parameters typed narrowly enough? Free-form `string` parameters where a finite set of values is valid (`"dashboard"`, `"checkrule"`) invite typos that fail silently.
- Is duplicated logic factored into shared helpers? Two functions named `copyMap` and `cloneMap` doing the same thing for different map types suggest a missing utility.
- Can a function be called safely from a new context? Functions that silently return empty strings on invalid input (instead of erroring) make bugs invisible.

**Example findings from this review:**
- `Strip*ServerFields` had inconsistent nil-container initialization across entity types.
- `deeplinkPathAndQuery` accepts untyped strings with ad-hoc aliases.

---

## 3. Encapsulation

Evaluates whether implementation details are hidden behind clean boundaries, and whether modules expose only what they need to.

**What to look for:**
- Do functions do one thing? A function that parses YAML, converts types, extracts thresholds, extracts enabled flags, and cleans up annotations is doing five things. Each should be testable independently.
- Are mutations visible and intentional? Deleting keys from a map parameter is a mutation that callers may not expect, even if the map was copied internally.
- Are assumptions documented? A function that assumes two-label TLDs or a specific URL structure should say so in its doc comment.
- Is the dependency direction correct? The `yaml/` subpackage imports the root package for generated types — never the reverse. This keeps the YAML dependency from leaking to consumers who don't need it.
- Are internal helpers scoped correctly? Unexported functions that exist only to support a temporary implementation (e.g., `ensureMetadataLabels` for untyped recording rule groups) should be clearly grouped so the cleanup scope is visible when the generated types arrive.

**Example findings from this review:**
- `ConvertPrometheusYAMLToCheckRule` mixed five concerns in one function.
- `AppBaseURL` encodes an undocumented domain structure assumption.
