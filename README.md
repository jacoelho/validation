# Validation

Validation is a Go library for building structural validation rules with minimal overhead. A `Rule[T]` is `func(T) error`: it returns `nil` or all applicable failures. Requires Go 1.27+.

## Install

```sh
go get github.com/jacoelho/validation
```

These examples describe the pending Go 1.27 refactor. Published v1.0.x tags still contain the previous API.

## Validate a struct

Combine field rules and call the result with a struct:

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
	for _, issue := range v.Issues(err) {
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
$.name: not_blank; $.age: min
not_blank at $.name
min at $.age
```

`Field` reads a struct field and labels its failures. `Struct` runs every field rule in order. Use `All` to combine rules for one value and `Each` to validate slice elements. `Project(getter)` adapts a rule without adding a path segment. The function forms of `Field` and `Project` accept multiple child rules.

For custom output, `Issues(err)` returns each failure's `Path`, `Code`, and terminal `Err` in order. `FormatPath` renders a path; `Format(err)` produces the default `path: code` text. Use `errors.Is` or `errors.As` to inspect the error tree; detail types include `LengthError`, `BoundsError[T]`, `DuplicateError`, and `IndexError`.

## Implement a validation interface

If your application uses a `Validate() error` interface, delegate the method to a reusable rule:

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
$.name: not_blank
```

The interface belongs to the application. Validation runs when you call the rule or the method.

## Model presence explicitly

An absent value and a present zero value can mean different things. The generic `OptionalValue` and `RequiredValue` methods use the getter's boolean to make that distinction:

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
$: min
$: required
```

For pointer fields, `OptionalPtr` skips a nil pointer and `RequiredPtr` reports `required` without calling its child. A failed guard does not prevent independent sibling rules from running. `AtIndex` and the map key rules similarly decide whether a child has an input. Ordinary zero, empty, and nil container values are passed to applicable rules; optionality comes from an explicit guard.

## Validate map values

Pass a key order to map rules so reported failures have a predictable order:

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
$["east"]: min; $["west"]: min
```

Map callbacks run in Go's unspecified map iteration order. The map rule sorts only failing key groups before returning them. `MapEach` validates typed key/value entries; `MapKeys` and `MapValues` validate one side. `When` and `Unless` apply a predicate guard to a rule group. For custom constraints, use `Check(predicate, failure)` or an ordinary `Rule[T]`; `NewViolation(code, cause)` adds a code to a custom cause.

Custom rules and failure factories must return literal `nil` on success. The library does not normalize typed-nil error interfaces or recover callback panics. Constructed rules can be shared across concurrent calls when their getters, predicates, comparators, failure factories, and captured state are safe for concurrent use; the library does not synchronize callbacks.

## More constraints

`Min` and `Max` include their bounds; `Between` includes both endpoints. `GreaterThan` and `LessThan` exclude their bounds. These rules accept integers and floats, including named types. String lengths are explicit: choose byte or rune rules.

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
$: trimmed; $: match
$: multiple_of
true
true
```

`NotBlank` rejects empty or whitespace-only strings; `Trimmed` rejects surrounding whitespace without modifying the value. `MultipleOf` uses exact integer division. `FloatMultipleOf` measures the absolute distance to the nearest multiple using a caller-supplied tolerance; NaN and infinities fail. `Match` uses a precompiled expression and matches substrings unless anchored. `Time(layout)` uses Go's `time.Parse`, so the layout can describe a date, time, or timestamp.

## Scope and cost

Rules use typed getters, constraints, comparators, and explicit projections. The library has no reflection, tag parser, format catalogue, network access, or context-bearing rule API.

The successful-path allocation target is zero heap objects and zero heap bytes for preconstructed simple rules with prepared inputs and non-allocating callbacks. `Match` and `Time` may allocate during standard-library matching and parsing. Failure reporting may allocate. `SliceUnique` leaves its input untouched and scans previous elements, making up to `n(n−1)/2` comparisons.

## Test

```sh
go test ./...
```
