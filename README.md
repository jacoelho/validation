# Validation

Validation is a Go library for validating values with typed rules. Struct fields are read through getter functions, without reflection or struct tags.

A `Rule[T]` is a `func(T) error`. Call it with a value; it returns `nil` on success.

## Install from main

Requires Go 1.27 or later.

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

`Field` adds the field name to the error path. Use `Project` instead when you need a getter without a path segment.

`Struct` and `All` run every child rule and collect their failures. Neither stops at the first failure.

## Read validation errors

`Format(err)` prints each failure with its path and a default message. The library's error types use the same formatting in their `Error` methods. Built-in messages include configured bounds, lengths, patterns, and multiples. They omit rejected values and cause messages, though length failures include the actual length. Custom coded errors can supply their own messages through `MessageProvider`. Other unknown codes and external failures use their code as the message.

For individual failures, use `WalkIssues(err)`, as in `formatLines` above. It yields issues in rule order. `Issues(err)` collects them into a slice owned by the caller. Each issue has three fields:

- `Path []Segment`: the location of the failure.
- `Code`: the machine-readable failure code.
- `Err`: the error, which may wrap a cause.

Use `errors.Is` and `errors.As` to inspect the returned error tree. Paths are stored on issues, so the same error can appear at more than one location.

## Custom messages and localisation

For a custom default message, implement `MessageProvider`:

```go
type MessageProvider interface {
    Coded
    Message() string
}
```

`Message()` returns text intended for display, without a path prefix. A nonempty message takes precedence over the built-in message for the code; an empty string uses the normal code-based fallback. The formatter reads only the current coded failure, without searching its causes. Ordinary errors with only a `Message()` method remain `external`.

`Message()` must not call `Format` or `DefaultMessage` for the same error, because that would recurse. Implement the custom error's own `Error()` method too, as in the example below.

To translate a message, read the failure code and its parameters. Built-in errors provide these through `Parameterized`; custom errors can implement it too:

```go
type Parameterized interface {
    error
    Code() Code
    Parameters() map[string]any
}
```

`Parameters()` returns a fresh map, or nil when there are no parameters. Values keep their original types and precision. If you implement this method, keep referenced data immutable or copy it before returning it. Built-in errors also have typed accessors that read the same fields.

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

`DefaultMessage(issue)` supplies the custom default message or built-in English fallback without a path. Use it as a fallback for codes you do not translate. The library stores no locale and requires no particular template syntax or formatter interface.

| Error type | Named parameters |
| --- | --- |
| `BoundsError[T]` | `minimum`, `minimum_inclusive` when a lower bound exists; `maximum`, `maximum_inclusive` when an upper bound exists |
| `LengthError` | `actual`, `minimum`, `unit`; `maximum` when present |
| `MultipleOfError[T]` | `base`; `tolerance` for float rules, including zero tolerance |
| `TextError` | `constraint`: substring, prefix, suffix, regexp, or time layout, identified by code |
| `DuplicateError` | `first_index` |
| `IndexError` | `index`, `length` |
| `Violation` | None |

Byte text constraints preserve their exact bytes in a string. Length units are the stable `LengthUnit` values `bytes`, `runes`, `elements`, and `entries`. Translate these labels in your application.

A custom error can supply a default message and named parameters:

```go
type QuotaError struct { Limit int }

func (e QuotaError) Error() string { return e.Message() }
func (e QuotaError) Code() v.Code { return "upload_quota" }
func (e QuotaError) Message() string {
    return fmt.Sprintf("upload quota exceeded (limit %d)", e.Limit)
}
func (e QuotaError) Parameters() map[string]any {
    return map[string]any{"limit": e.Limit}
}
```

Return it from a `Rule[T]` or `Check` failure factory. `WalkIssues` keeps the original error in `issue.Err`, so its code and parameters remain available. Errors do not have to implement `Parameterized`: an error with only `Code()` keeps its code, and an ordinary error is reported with code `external`.

```go
rule := v.Check(
    func(count int) bool { return count <= 3 },
    func(int) error { return QuotaError{Limit: 3} },
).Field("uploads", func(count int) int { return count })
fmt.Println(v.Format(rule(4)))
// $.uploads: upload quota exceeded (limit 3)
```

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

Define this interface in your application. The library does not call `Validate` automatically.

## Optional and required values

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

Both methods validate a present value, including zero. For an absent value, `OptionalValue` skips validation and `RequiredValue` reports `required`.

For pointers, use `OptionalPtr` to skip nil or `RequiredPtr` to report `required`. Neither calls its child rule for nil. These guards do not stop independent sibling rules.

## Validate maps and slices

`MapValues` returns failures in the key order you provide. It calls rules in Go's unspecified map iteration order:

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

Only failing key groups are sorted. Use `MapKeys` to check existing keys, or `MapEach` to check typed key/value entries.

For one specified key, `MapRequiredKey` reports absence. `MapOptionalKey` validates the value when present and skips it when absent.

`Each` validates slice elements and adds their indices to failure paths. `AtIndex` checks one index. If the index is out of range, it reports a failure without calling its child rule; independent sibling rules still run.

`SliceUnique` leaves its input unchanged and compares each element with earlier ones, making at most `n(n−1)/2` comparisons.

## Combine rules

Use `All` to apply several constraints to the same value:

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

`When` runs a group of rules when its predicate is true. `Unless` runs the group when its predicate is false.

### Numbers

`Min`, `Max`, and `Between` include their bounds. `GreaterThan` and `LessThan` exclude them. These rules accept integers and floats, including named types.

`MultipleOf` requires a zero integer remainder for a nonzero base. `FloatMultipleOf` checks distance to the nearest multiple against an absolute tolerance. NaN and infinities fail.

### Text and time

`Trimmed` reports surrounding whitespace without changing the input. `Match` accepts a compiled regexp and searches for a substring unless you anchor it. For string lengths, choose byte or rune rules explicitly.

`Time(layout)` uses `time.Parse`, so the layout can describe a date, time, or timestamp.

## Custom rules

`Check(predicate, failure)` calls the failure factory only when the predicate fails. To attach a machine-readable code to an error, use `NewViolation(code, cause)`.

A custom `Rule[T]` must return a nil error interface on success. A typed nil stored in an `error` interface is non-nil. A `Check` failure factory must return a non-nil error when called. Callback panics propagate.

## Share rules across goroutines

A constructed rule can be shared across goroutines if its getters, predicates, comparators, failure factories, and captured state are safe for concurrent use. The library does not synchronize them.

The library performs no network I/O. If your callbacks do I/O, handle cancellation there: `Rule[T]` has no context parameter.

## Allocations

The [allocation tests](allocation_test.go) check zero allocations per run for selected preconstructed rules with prepared inputs and non-allocating callbacks. This is not a guarantee for every rule or input. Standard-library matching and parsing in `Match` and `Time` may allocate, as may failure reporting.

## Test

```sh
go test ./...
```
