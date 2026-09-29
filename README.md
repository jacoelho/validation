# Validation

Validation builds typed rules for Go values. To validate a struct, project rules onto its fields with typed getters; the library uses no reflection or tags. A `Rule[T]` is `func(T) error`: `nil` means validation succeeded. `Struct` and `All` run every applicable child rule and return their failures. Requires Go 1.27 or later.

## Install from main

```sh
go get github.com/jacoelho/validation@main
```

## Validate a struct

Define the field rules once, then call the combined rule for each `User`:

```go
package main

import (
	"fmt"
	"strings"

	v "github.com/jacoelho/validation"
)

type User struct {
	Name string
	Age  int
}

var userRule = v.Struct(
	v.NotBlank[string]().Field("name", func(u User) string { return u.Name }),
	v.Min(18).Field("age", func(u User) int { return u.Age }),
)

func formatLines(err error) string {
	var lines []string
	for issue := range v.WalkIssues(err) {
		lines = append(lines, fmt.Sprintf("%s at %s", issue.Code, v.FormatPath(issue.Path)))
	}
	return strings.Join(lines, "\n")
}

func main() {
	err := userRule(User{Name: " ", Age: 16})
	fmt.Println(v.Format(err))
	fmt.Println(formatLines(err))
}
```

```text
$.name: must not be blank; $.age: must be at least 18
not_blank at $.name
min at $.age
```

`NotBlank` rejects the whitespace-only name; `Min(18)` rejects age 16. `Field` adds each field name to the error path. `Struct` runs both rules, so both failures appear.

`Project` adapts a rule to a parent type without adding a path segment.

`Format` and the returned error’s `Error` method print paths and readable failure reasons, including configured bounds, lengths, patterns, and multiples. Rejected values and cause messages are omitted; length failures include the actual length. Unknown custom codes and external failures retain their code as the message. Machine-readable codes remain available through `WalkIssues` and `Issues`. The `formatLines` function shows another representation: `WalkIssues` yields each failure in rule order. `Issues(err)` collects the same failures into an owned slice. Each issue has a `Path []Segment`, a `Code`, and an `Err` that may wrap a cause; `errors.Is` and `errors.As` inspect the returned error tree.

## Custom messages and localisation

Errors own failure codes and parameters; applications own localisation and presentation. `Parameterized` is the optional shared contract for built-in and custom errors:

```go
type Parameterized interface {
    error
    Code() Code
    Parameters() map[string]any
}
```

`Parameters()` returns nil for no parameters, otherwise a fresh map. Values retain their original types and precision. Referenced values must be immutable or defensively copied by the error owner. Typed accessors remain available; parameter maps are derived from the same fields.

Use `WalkIssues` to format failures with your own code or localisation library:

```go
failure := v.Min(2).Field("bar", func(n int) int { return n })(1)
for issue := range v.WalkIssues(failure) {
    message := v.DefaultMessage(issue)
    if detail, ok := issue.Err.(v.Parameterized); ok && issue.Code == v.CodeMin {
        message = fmt.Sprintf("deve ser pelo menos %v", detail.Parameters()["minimum"])
    }
    fmt.Printf("%s: %s\n", v.FormatPath(issue.Path), message)
}
// $.bar: deve ser pelo menos 2
```

`Format` remains the built-in English formatter. `DefaultMessage(issue)` returns its message without a path, for optional fallback. Paths belong to issues, so one error can appear at multiple locations. The library stores no locale and imposes no template syntax or formatter interface.

| Error type | Named parameters |
| --- | --- |
| `BoundsError[T]` | `minimum`, `minimum_inclusive` when a lower bound exists; `maximum`, `maximum_inclusive` when an upper bound exists |
| `LengthError` | `actual`, `minimum`, `unit`; `maximum` when present |
| `MultipleOfError[T]` | `base`; `tolerance` for float rules, including zero tolerance |
| `TextError` | `constraint`: substring, prefix, suffix, regexp, or time layout, identified by code |
| `DuplicateError` | `first_index` |
| `IndexError` | `index`, `length` |
| `Violation` | None |

Byte text constraints preserve their exact bytes in a string. Length units are stable `LengthUnit` values for the application to translate.

A custom error can supply any number of named parameters:

```go
type QuotaError struct { Limit int }

func (e QuotaError) Error() string { return "upload quota exceeded" }
func (e QuotaError) Code() v.Code { return "upload_quota" }
func (e QuotaError) Parameters() map[string]any {
    return map[string]any{"limit": e.Limit}
}
```

Return it from a `Rule[T]` or `Check` failure factory. `WalkIssues` preserves the code, parameters, and error identity; ordinary errors and coded errors without parameters still work. The default formatter prints unknown custom codes and omits cause messages.

## Implement a validation interface

If your application expects a `Validate() error` method, have it call the rule:

```go
package main

import (
	"fmt"

	v "github.com/jacoelho/validation"
)

type Validatable interface {
	Validate() error
}

type Account struct {
	Name string
}

var accountRule = v.NotBlank[string]().Field("name", func(a Account) string { return a.Name })

func (a Account) Validate() error { return accountRule(a) }

func main() {
	var value Validatable = Account{}
	fmt.Println(v.Format(value.Validate()))
}
```

```text
$.name: must not be blank
```

The application owns this interface. Calling `Validate` runs the same rule; the library does not invoke the method automatically.

## Model presence explicitly

When absence and a present zero value mean different things, return `(value, present)` from the getter:

```go
package main

import (
	"fmt"

	v "github.com/jacoelho/validation"
)

type Patch struct {
	Age    int
	HasAge bool
}

func main() {
	getAge := func(p Patch) (int, bool) { return p.Age, p.HasAge }
	minimumAge := v.Min(18)
	optionalAge := minimumAge.OptionalValue(getAge)
	requiredAge := minimumAge.RequiredValue(getAge)

	fmt.Println(optionalAge(Patch{}) == nil)
	fmt.Println(v.Format(optionalAge(Patch{HasAge: true})))
	fmt.Println(v.Format(requiredAge(Patch{})))
}
```

```text
true
$: must be at least 18
$: value is required
```

`OptionalValue` skips an absent value. `RequiredValue` reports `required` for absence; when the value is present, both methods run `Min`, even when age is zero. For pointers, `OptionalPtr` skips nil; `RequiredPtr` reports `required` for nil without calling its child. `AtIndex` reports an out-of-range index without calling its child. None of these guards stops independent sibling rules.

## Validate maps and slices

`MapValues` takes a key order for its returned failures. Rule callbacks still follow Go's unspecified map iteration order:

```go
package main

import (
	"fmt"

	v "github.com/jacoelho/validation"
)

func main() {
	minimum := v.MapValues[map[string]int](v.StringKeys[string](), v.Min(1))
	err := minimum(map[string]int{"west": 0, "east": -1, "ok": 1})
	fmt.Println(v.Format(err))
}
```

```text
$["east"]: must be at least 1; $["west"]: must be at least 1
```

Only failing key groups are sorted. `MapKeys` checks existing keys; `MapEach` checks typed key/value entries. To check one specified key, use `MapRequiredKey` to report absence or `MapOptionalKey` to validate it when present and skip it when absent. `Each` applies rules to slice elements, adding their indices to failure paths.

## More constraints

`Min`, `Max`, and `Between` include their bounds. `GreaterThan` and `LessThan` do not. They accept integers and floats, including named types. For string lengths, choose byte or rune rules explicitly.

```go
package main

import (
	"fmt"
	"regexp"
	"time"

	v "github.com/jacoelho/validation"
)

func main() {
	name := v.All(
		v.NotBlank[string](),
		v.Trimmed[string](),
		v.Match[string](regexp.MustCompile(`^[A-Z][a-z]+$`)),
	)
	fmt.Println(v.Format(name(" Ada")))
	fmt.Println(v.Format(v.All(v.Between(1, 20), v.MultipleOf(5))(12)))
	fmt.Println(v.FloatMultipleOf(0.1, 1e-12)(0.3) == nil)
	fmt.Println(v.Time[string](time.RFC3339)("2026-09-29T08:30:00Z") == nil)
}
```

```text
$: must not have leading or trailing whitespace; $: must match pattern "^[A-Z][a-z]+$"
$: must be a multiple of 5
true
true
```

`All` runs every name rule: the value is not blank, but it has leading whitespace and does not match the expression. `Trimmed` reports surrounding whitespace without changing the input. `Match` accepts a compiled regexp and searches for a substring unless you anchor it.

`MultipleOf` requires a zero integer remainder for a nonzero base. `FloatMultipleOf` checks distance to the nearest multiple against an absolute tolerance; NaN and infinities fail. `Time(layout)` uses `time.Parse`, so the layout can describe a date, time, or timestamp.

## Custom rules

`Check(predicate, failure)` makes a rule whose failure factory runs only when the predicate fails. `When` and `Unless` apply a rule group under a condition. For a custom failure, `NewViolation(code, cause)` attaches a machine-readable code to an error.

A custom `Rule[T]` must return a nil error interface on success; a typed nil stored in an `error` interface is non-nil. A `Check` failure factory must return a non-nil error when called. Callback panics propagate.

You can share a constructed rule across goroutines when its getters, predicates, comparators, failure factories, and captured state are safe for concurrent use; the library does not synchronize them. The library itself performs no network I/O, but custom callbacks can. `Rule[T]` has no context parameter.

## Cost

The allocation tests assert zero allocations per run for selected preconstructed rules with prepared inputs and non-allocating callbacks. `Match` and `Time` may allocate during standard-library matching and parsing; failure reporting may allocate too.

`SliceUnique` leaves its input unchanged and compares each element with earlier ones, making at most `n(n−1)/2` comparisons.

## Test

```sh
go test ./...
```
