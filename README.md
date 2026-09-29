# Validation

A typed, synchronous validation library for Go 1.27 and later. A `Rule[T]` is a `func(T) error`. Successful validation returns literal `nil`; every applicable rule runs and every failure remains in the returned error tree.

## Install

```sh
go get github.com/jacoelho/validation/v2
```

## Use

```go
package main

import (
    "fmt"

    v "github.com/jacoelho/validation/v2"
)

type User struct {
    Name string
    Age  *int
}

var userRule = v.All(
    v.All(v.NotEmpty[string](), v.RuneMinLength[string](2)).Field("name", func(u User) string { return u.Name }),
    v.OptionalPtr(v.Min(18)).Field("age", func(u User) *int { return u.Age }),
)

func main() {
    err := userRule.Validate(User{})
    for _, issue := range v.Issues(err) {
        fmt.Println(v.FormatPath(issue.Path), issue.Code)
    }
    // $.name not_empty
    // $.name rune_min_length
}
```

Construct rules once and reuse them. `All` and `Struct` both evaluate children in declaration order. Generic `Rule` methods such as `rule.Field(name, getter)`, `rule.Project(getter)`, `rule.OptionalValue(getter)` and `rule.RequiredValue(getter)` adapt a rule to a parent type. The matching functions accept several child rules. `Field` adds a typed location only when a child fails; `Project` changes the input type without adding a location. `Each` and map rules add element locations. `When` and `Unless` guard a group using a predicate on its input type. `Check` creates one custom constraint from a predicate and a failure factory.

A sibling's failure does not skip later rules. Explicit guards such as `OptionalPtr`, `RequiredPtr`, `OptionalValue`, `RequiredValue`, `AtIndex`, and map key rules decide whether children have an input. Zero, empty, and nil container values are validated under each rule's documented condition.

## Inspect failures

`Issues(err)` returns ordered, independent snapshots with a structured `Path`, stable `Code`, and original terminal `Err`. `Format(err)` renders `path: code` pairs without printing rejected values or cause messages. Use `errors.Is` and `errors.As` on the returned error for ordinary cause and typed detail inspection. Built-in details include `LengthError`, `BoundsError[T]`, `DuplicateError`, and `IndexError`. Custom rules can return any ordinary error; `NewViolation(code, cause)` supplies a coded failure.

A custom rule or failure factory must return literal `nil` on success. The library does not normalize typed-nil error interfaces or recover callback panics. Rules and captured callbacks must be safe for concurrent use if callers share a constructed rule.

## Scope and cost

Rules use typed getters, constraints, comparators, and explicit projections. There is no reflection, tag parser, format catalogue, network access, or context-bearing rule API. Built-in successful validation is designed to allocate no heap objects after construction with prepared inputs and non-allocating callbacks. Failure reporting may allocate. `SliceUnique` uses a non-mutating previous-elements scan and performs up to `n(n−1)/2` comparisons. Map traversal sorts only failing key groups; callback execution order follows Go map iteration and is unspecified.

See [ARCHITECTURE.md](ARCHITECTURE.md) for ownership and [MIGRATION.md](MIGRATION.md) for the breaking API change. The normative target is [validation-refactor-spec.md](docs/validation-refactor-spec.md); a specification document alone does not certify implementation acceptance.
