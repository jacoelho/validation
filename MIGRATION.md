# Migration from v1

This is a breaking change from the published v1.0.x API. Change imports from `github.com/jacoelho/validation` to `github.com/jacoelho/validation/v2`, and upgrade to v2 when it is released. The v1.0.0 through v1.0.5 tags remain on the old module path.

Upgrade the consuming project to Go 1.27 or later before adopting this API.

When a child rule is already composed, adapt it with a generic method such as `child.Field("name", getter)`, `child.Project(getter)`, or `child.OptionalValue(getter)`. The function forms remain useful when declaring several child rules at one site.

Keep named input types through validation. Numeric, string, byte-slice, slice and map rules accept named types with the matching underlying type; typed getters should return the exact named field type. No conversion to an unnamed collection or `any` is needed.

| Before | Now |
|---|---|
| `Rule[T] func(T) *Error` and `Validate(...) Errors` | `Rule[T] func(T) error`; `Validate` returns literal `nil` or an error tree. Use `Issues(err)` to enumerate failures. |
| Mutable `Error.Field`, `Params`, `Fatal` | Immutable coded errors and typed details; `Field`, `Each`, and maps add structured location wrappers. |
| `StructValidator`, `StructField`, `SliceField`, `MapField` | `Struct` is a `Rule[T]`; nest ordinary `Field` and typed collection rules. |
| `SliceRule`, `MapRule`, `MapEntryRule` | `Rule[Slice]`, `Rule[Map]`, and `Rule[Entry[K,V]]`. |
| `NumbersMin`, `StringsRuneMinLength`, `SlicesForEach`, `MapsKey` | `Min`, `RuneMinLength`, `Each`, `MapRequiredKey` or `MapOptionalKey`. |
| `RuleStopOnError` and `Fatal` | Removed. All independent rules run. Use explicit presence or predicate guards when a child has no applicable input. |
| Error-based `Or` and `RuleNot` | Removed. Express one Boolean constraint with `Check`; its failure factory runs only when the full predicate fails. |
| Regex constructor | Removed from core. Write an application-owned typed rule if regex validation is needed. |
| Dot-joined field strings | `Issue.Path` contains Field, Index, and Key segments. `FormatPath` renders them unambiguously. |

`NotZero` checks the Go zero value; it does not mean a field was supplied. Use `RequiredPtr`, `RequiredValue`, or `MapRequiredKey` for known presence. A present pointer to zero is present and its child rules run.

Legacy pointer and error-slice callbacks need concrete nil checks before converting to `error`. For a legacy `*Error` result, copy its old declaration into your migration boundary or reference your pinned old package and adapt it explicitly:

```go
type LegacyError struct { Code string }
func (e *LegacyError) Error() string { return e.Code }

type LegacyErrors []*LegacyError
func (es LegacyErrors) Error() string { return "legacy failures" }

func adaptPointer[T any](old func(T) *LegacyError) func(T) error {
    return func(value T) error {
        e := old(value)
        if e == nil { return nil }
        return e
    }
}

func adaptSlice[T any](old func(T) LegacyErrors) func(T) error {
    return func(value T) error {
        es := old(value)
        if len(es) == 0 { return nil }
        children := make([]error, 0, len(es))
        for _, e := range es {
            if e != nil { children = append(children, e) }
        }
        if len(children) == 0 { return nil }
        return errors.Join(children...)
    }
}
```

Import `errors` for the slice example. An adapter must preserve all old non-nil occurrences and causes; do not stringify them or restore first-error/Fatal behavior. The example's concrete `LegacyError` and `LegacyErrors` mirror the old pointer and slice result shapes and are application migration code, not new library exports. Custom functions that already return `error` can be used as rules directly.
