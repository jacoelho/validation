# Architecture

The [refactor specification](docs/validation-refactor-spec.md) owns the target API, rule semantics, acceptance cases, and release gates. [README.md](README.md) is the usage entry point; [MIGRATION.md](MIGRATION.md) maps the previous public API to the target.

One `Rule[T] func(T) error` owns synchronous evaluation. Composite rules call children directly and return `nil`, the one original failure, or an immutable aggregate. A failure never controls sibling applicability; typed pointer, nullable, index, key, and predicate guards control it explicitly.

Generic `Rule` methods project an already-composed child rule to a parent type for named fields, unnamed projections, and explicit value presence. Variadic function forms build a child group once, then delegate to those methods. This keeps one execution policy for each adapter without builder state.

Each rule family owns its standard-type decision. The error module owns codes, typed diagnostics, immutable location wrappers, aggregate storage, `Issues`, and safe formatting. Rules attach locations only after a child fails. The internal push iterator traverses existing errors only for reporting; validation never uses it. Error causes remain ordinary Go causes for `errors.Is` and `errors.As` and are one issue when their coded parent is a failure.

Constructors freeze variadic configuration and prepare reusable lookup state. They reject invalid static inputs with `ConfigurationError` and never run value callbacks. Successful calls create no diagnostics or result storage. Callers own captured closures, shallow referenced configuration, and read-only inputs during concurrent use.

Map rules inspect entries in Go iteration order, retain only failing groups, then sort those groups using the caller's typed `KeyOrder`. The caller owns strict order and injective stable key text. `SliceUnique` uses a quadratic scan to avoid success-path scratch allocation. These choices trade CPU for bounded failure-only storage and zero successful heap allocation.

The earlier mutable `Error`, `Errors`, `StructValidator`, `SliceRule`, `MapRule`, fail-fast handling, and error-based `Or`/`RuleNot` were rejected because they split execution ownership, mutate diagnostics, or allocate for successful disjunction. The spec requires a single breaking migration; compatibility wrappers would retain those problems.
