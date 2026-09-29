# Validation library refactor
## Requirements and Gherkin acceptance specification

**Specification:** VRS-1.3\
**Date:** 29 September 2026\
**Repository:** `github.com/jacoelho/validation/v2`\
**Planning baseline:** `6f5d23b1ad620728e121bd45be03fd81340c64d4`\
**Input:** the agreed refactor constraints and `validation-refactor-plan.md` supplied in this conversation.\
**Deliverable:** a normative target specification, not a repository implementation or a certificate that the target behaviour already passes.

## 1. Purpose and document authority

Refactor the library into one synchronous typed validation engine that returns every applicable failure, interoperates with ordinary Go errors, and performs successful built-in validation without heap allocation. Retain generic, explicit field access. Do not replace the library with a reflective validator or a format catalogue.

“Shall” identifies a mandatory requirement. Requirement IDs remain stable when wording is clarified; a changed contract requires a new specification revision. Section 7 is the requirement catalogue. Sections 3–6 and 8 define the terms, API, semantics and verification procedures those requirements reference. Appendix A provides the complete Gherkin acceptance suite. The feature files in the ZIP are exact copies of those blocks, not a different suite.

The specification supersedes conflicting suggestions in the earlier review, particularly context-bearing rules, default fail-fast field chains, a new `is` library and arbitrary error-based `Or`/`Not`. It does not assert that earlier experimental allocation results apply to Go 1.27. No runtime result is certified in this document.

An acceptance scenario is a contractual example. A `.feature` file is not an executed test until connected to an implementation by step definitions or an explicitly mapped Go test. The delivery checker validates specification structure and traceability only. Gherkin terminology and syntax follow the Cucumber reference [S2].

### Navigation

1. Purpose and document authority
2. Scope and fixed decisions
3. Terms and observable semantics
4. Public API and standard-type rule catalogue
5. Error tree, paths, reporting and iterator semantics
6. Construction and evaluation behaviour
7. Numbered requirements and acceptance links
8. Verification procedures and allocation gate
9. Acceptance fixture and binding conventions
10. Implementation sequence and migration
11. Risks, evidence boundaries and references
12. Appendix A: complete Gherkin suite

## 2. Scope and fixed decisions

| Decision | Required outcome |
|---|---|
| Execution contract | `type Rule[T any] func(T) error`; no mandatory reporter or iterator argument. |
| Error collection | All applicable configured rules run; several failures at one path are retained. |
| Value access | Typed getters, generic constraints and explicit projection. |
| Success performance | Preconstructed built-in rule + prepared input + compliant callbacks → literal nil, zero heap objects and zero heap bytes. |
| Error handling | Immutable location wrappers and standard single/multiple unwrapping. |
| Extensibility | Ordinary functions, `Check`, `Project`, typed comparators and explicit presence adapters. |
| Scope | Standard-type primitives only; no `is`/format catalogue, network checks, tag engine or runtime schema. |
| Context | No context parameter, import, wrapper, fallback dispatcher or request capture in this refactor. |
| Iterators | Internal push iteration over existing failures; direct loops for value validation. |
| Uniqueness | Initial non-mutating quadratic scan; no success-path scratch allocation. |
| Maps | Sort only failing entry groups; result order is deterministic, execution order is not. |
| Compatibility | One breaking API migration, not two permanent engines. |
| Go | Require Go 1.27. Use generic methods on `Rule[T]` for typed parent projection, named fields and explicit value presence. Keep `Rule[T] func(T) error` as the execution contract; do not add builder state. |

### Excluded work

No regular-expression constructor or regex compilation, email/URL/UUID catalogue, phone/country/currency registry, automatic `sql.Null` or `driver.Valuer` discovery, implicit `Validate()` discovery, reflection, unsafe reinterpretation, heterogeneous `any` validation, input normalisation, code generation, networking or context adapter is included. External libraries may be used inside application-supplied typed functions; their correctness, allocation and effect contracts remain application-owned.

No failure-path allocation target or absolute nanosecond SLA is promised. Failure-path work must remain finite and proportional to the data inspected and failures reported, subject to the explicitly quadratic duplicate scan and sorting of failed map groups. No global optimisation percentage is invented without target-compiler measurements.

## 3. Terms and observable semantics

**Rule occurrence:** one position in a constructed rule graph. Two positions containing the same function are two occurrences.

**Applicable rule:** a configured occurrence whose explicit structural/predicate guards permit its input to be evaluated. Guards are pointer presence, nullable presence, key presence, index validity and `When`/`Unless`; a sibling rule failure is not a guard.

**Failure occurrence:** one independent coded failure or uncoded terminal error in the reported error tree. A coded error's underlying cause is evidence for that one failure, not a second validation issue. A custom uncoded join can contribute several independent occurrences.

**Valid/successful call:** all applicable rules return literal nil. A Go error interface holding a typed-nil pointer is not success and violates the extension contract.

**Prepared input:** caller-owned values and backing storage have been created outside the measured validation call. Passing a value does not authorise the library to mutate it or retain the entire input in a diagnostic.

**Constructed rule:** configuration is copied and reusable lookup data is prepared. Captured callbacks and objects reachable through shallow-copied references remain caller-owned and must be stable while the rule is in use.

**Compliant callback:** a typed function that returns normally, follows nil/error conventions, does not mutate shared state unsafely, and meets any purity, ordering or no-allocation obligations relevant to its use.

**Validation occurrence order:** outer rule declaration order; then each child subtree's own order; slices use ascending index; a map-traversing child sorts its failing key groups. Separate map rules do not globally regroup their errors by key.

### Meaning of “return all errors”

The following example has five failures, not one or three:

```text
All(
  SliceMaxLength(1),
  Each(NotEmpty, RuneMinLength(2)),
)
input: ["", ""]

$: max_length
$[0]: not_empty
$[0]: rune_min_length
$[1]: not_empty
$[1]: rune_min_length
```

This contract is deliberately different from “one error per field” and “stop after a required check”. It does not deduplicate equal codes, paths or error identities. Consumers may choose a presentation policy, but the returned result remains complete.

`RequiredPtr(nil)` produces its absence failure without invoking its children. It does not stop the next independent field. The same distinction applies to missing map keys and invalid indices. An input containing a nil pointer passed to an unguarded application getter may panic; the library does not recover it or pretend the rest of the call completed.

`All()` is valid and returns nil. `Field(name, getter)` and `Project(getter)` still invoke the getter once. Empty child groups do not neutralise presence guards: `RequiredPtr()` still rejects nil and `MapRequiredKey()` still rejects a missing key. A non-negative `AtIndex(i)` still checks index presence even with no value rules.

## 4. Public API and standard-type rule catalogue

The declarations below define the target API, not existing source code. Semantics and names are normative for this refactor. Function bodies, private representation, file placement and performance-preserving inlining remain implementation choices. No function declaration with an omitted body below is intended to be compiled as a delivered implementation.

### 4.1 Core and typed adapters

```go
type Rule[T any] func(T) error

func (r Rule[T]) Validate(value T) error

func All[T any](rules ...Rule[T]) Rule[T]
func Struct[T any](rules ...Rule[T]) Rule[T] // Thin All alias.

func (r Rule[F]) Field[P any](name string, get func(P) F) Rule[P]
func (r Rule[V]) Project[P any](get func(P) V) Rule[P]
func Field[P, F any](name string, get func(P) F, rules ...Rule[F]) Rule[P]
func Project[P, V any](get func(P) V, rules ...Rule[V]) Rule[P]

func When[T any](predicate func(T) bool, rules ...Rule[T]) Rule[T]
func Unless[T any](predicate func(T) bool, rules ...Rule[T]) Rule[T]
func Check[T any](predicate func(T) bool, failure func(T) error) Rule[T]

func OptionalPtr[T any](rules ...Rule[T]) Rule[*T]
func RequiredPtr[T any](rules ...Rule[T]) Rule[*T]
func (r Rule[V]) OptionalValue[P any](get func(P) (V, bool)) Rule[P]
func (r Rule[V]) RequiredValue[P any](get func(P) (V, bool)) Rule[P]
func OptionalValue[P, V any](get func(P) (V, bool), rules ...Rule[V]) Rule[P]
func RequiredValue[P, V any](get func(P) (V, bool), rules ...Rule[V]) Rule[P]

func Each[S ~[]E, E any](rules ...Rule[E]) Rule[S]
func AtIndex[S ~[]E, E any](index int, rules ...Rule[E]) Rule[S]
```

`Rule` methods adapt one preconstructed child rule to a parent type. The function forms accept a variadic child group and delegate to the same method implementation. `Field` adds one field segment only on failure. `Project` changes only the value presented to its children. All adapters construct their combined child group once, not on each validation call. The library must not call user value callbacks while constructing that group.

No universal `Required(any)` is provided. Explicit required-pointer, nullable and key adapters are presence checks; `NotZero` and `NotEmpty` are content checks. A caller models omitted, explicit null and present-zero as separate states when its domain requires that distinction.

### 4.2 Type constraints

```go
type Signed interface {
    ~int | ~int8 | ~int16 | ~int32 | ~int64
}
type Unsigned interface {
    ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}
type Integer interface { Signed | Unsigned }
type Float interface { ~float32 | ~float64 }
type Number interface { Integer | Float }
```

Named values retain their type. `Number` excludes strings and complex numbers. It does not coerce integer values through float64. Safely comparable concrete values may use Go equality, but the `comparable` constraint must not be misrepresented as a guarantee against every runtime panic for arbitrary interface-bearing inputs [S5]. Such inputs are not part of the supported comparable domain.

### 4.3 Comparable and numeric rules

| Constructor | Input constraint | Successful condition | Failure code |
|---|---|---|---|
| `Equal[T](want T)` | `T comparable` | `value == want` | `equal` |
| `NotEqual[T](other T)` | `T comparable` | `value != other` | `not_equal` |
| `Zero[T]()` | `T comparable` | Equal to `var zero T` | `zero` |
| `NotZero[T]()` | `T comparable` | Not equal to `var zero T` | `not_zero` |
| `OneOf[T](allowed ...T)` | `T comparable` | Equal to at least one configured value | `one_of` |
| `NotOneOf[T](forbidden ...T)` | `T comparable` | Equal to no configured value | `not_one_of` |
| `Min[T](min T)` | `T Number` | `value >= min` | `min` |
| `Max[T](max T)` | `T Number` | `value <= max` | `max` |
| `Between[T](min, max T)` | `T Number` | `value >= min && value <= max` | `between` |
| `GreaterThan[T](min T)` | `T Number` | `value > min` | `greater_than` |
| `LessThan[T](max T)` | `T Number` | `value < max` | `less_than` |
| `Positive[T]()` | `T Number` | `value > zero` | `positive` |
| `NonNegative[T]()` | `T Number` | `value >= zero` | `non_negative` |
| `Negative[T]()` | `T Number` | `value < zero` | `negative` |
| `NonPositive[T]()` | `T Number` | `value <= zero` | `non_positive` |
| `NotNaN[T]()` | `T Float` | Value is not NaN | `not_nan` |
| `Finite[T]()` | `T Float` | Value is neither NaN nor an infinity | `finite` |

Every constructor returns `Rule[T]`. Empty `OneOf` always fails; empty `NotOneOf` always passes. NaN equality is not rewritten: `OneOf(NaN)` does not match NaN, whereas numeric comparison constructors reject NaN bounds. Both infinities are valid bounds when their order is coherent. A separate `Finite` rule excludes infinities; normal comparisons use their declared relation [S5, S8].

```go
func MinBy[T any](min T, compare func(T, T) int) Rule[T]
func MaxBy[T any](max T, compare func(T, T) int) Rule[T]
func BetweenBy[T any](min, max T, compare func(T, T) int) Rule[T]
```

`MinBy` and `MaxBy` call `compare(value, bound)` once. `BetweenBy` compares configured bounds once at construction, then performs each of its two value-to-bound comparisons once per value even when the first bound fails. One configured range constraint emits at most one `between` issue. Negative, zero and positive results are interpreted by sign, not by exact values -1 and 1. The comparator must define a consistent order and cannot use errors to report an operational failure.

### 4.4 Strings and byte slices

For all string constructors, `T ~string` and the return type is `Rule[T]`. Byte constructors use `B ~[]byte` and return `Rule[B]`. Byte configuration arguments use the same `B` type; constructors copy that slice before capture.

| String constructor | Byte-slice counterpart | Failure code |
|---|---|---|
| `NotEmpty[T]()` | `BytesNotEmpty[B]()` | `not_empty` |
| `ByteLength[T](n int)` | `BytesLength[B](n int)` | `byte_length` |
| `ByteMinLength[T](min int)` | `BytesMinLength[B](min int)` | `byte_min_length` |
| `ByteMaxLength[T](max int)` | `BytesMaxLength[B](max int)` | `byte_max_length` |
| `ByteLengthBetween[T](min, max int)` | `BytesLengthBetween[B](min, max int)` | `byte_length_between` |
| `RuneLength[T](n int)` | Not included | `rune_length` |
| `RuneMinLength[T](min int)` | Not included | `rune_min_length` |
| `RuneMaxLength[T](max int)` | Not included | `rune_max_length` |
| `RuneLengthBetween[T](min, max int)` | Not included | `rune_length_between` |
| `Contains[T](part T)` | `BytesContains[B](part B)` | `contains` |
| `HasPrefix[T](prefix T)` | `BytesHasPrefix[B](prefix B)` | `prefix` |
| `HasSuffix[T](suffix T)` | `BytesHasSuffix[B](suffix B)` | `suffix` |
| `UTF8[T]()` | `BytesUTF8[B]()` | `utf8` |
| Use `Equal[T]` | `BytesEqual[B](want B)` | `equal` |

All length bounds are inclusive except that an exact-length rule requires equality. No trimming, case-folding or Unicode normalisation occurs. UTF-8 validity and rune count are separate rules: rune counting follows Go decoding, not grapheme segmentation [S7]. An empty substring/prefix/suffix matches an empty input. Nil byte slices are empty byte sequences; nil and empty slices compare equal under `BytesEqual`.

### 4.5 Slices

These constructors use `S ~[]E`, preserve `S`, and return `Rule[S]`. `E` is `any` for length and traversal rules and `comparable` for equality-based rules.

| Constructor | Behaviour | Failure code / location |
|---|---|---|
| `SliceLength[S, E](n int)` | Exact number of elements | `length` at container |
| `SliceMinLength[S, E](min int)` | At least min elements | `min_length` at container |
| `SliceMaxLength[S, E](max int)` | At most max elements | `max_length` at container |
| `SliceLengthBetween[S, E](min, max int)` | Inclusive length interval | `length_between` at container |
| `Each[S, E](rules ...Rule[E])` | All rules on every element | Child failures at Index(i) |
| `AtIndex[S, E](i int, rules ...Rule[E])` | Address one element | `index_out_of_range` at Index(i), or child failures |
| `SliceContains[S, E](want E)` | At least one equal element | One `contains` at container |
| `SliceOneOf[S, E](allowed ...E)` | All elements belong to set | `one_of` at every offending index |
| `SliceNotOneOf[S, E](forbidden ...E)` | No element belongs to set | `not_one_of` at every offending index |
| `SliceUnique[S, E]()` | No repeated equal element | `unique` at every later duplicate, with earliest index |

NaN values are not duplicate matches under `==`; +0 and -0 are. Membership can prepare an immutable lookup at construction; uniqueness depends on the input and therefore uses the stated previous-elements scan. It must not silently allocate a “seen” map on successful calls.

There is no universal generic array-length parameter or unsafe array reinterpretation. Use a typed getter on an original `*Parent` to expose its array as a slice. A by-value getter that returns a slice of a copied array may require heap escape and cannot be presented as an allocation-free example [S5].

### 4.6 Maps

```go
type Entry[K comparable, V any] struct {
    Key K
    Value V
}
type KeyOrder[K comparable] struct {
    Less func(K, K) bool
    Text func(K) string
}

func StringKeys[K ~string]() KeyOrder[K]

func MapEach[M ~map[K]V, K comparable, V any](
    order KeyOrder[K], rules ...Rule[Entry[K, V]],
) Rule[M]
func MapKeys[M ~map[K]V, K comparable, V any](
    order KeyOrder[K], rules ...Rule[K],
) Rule[M]
func MapValues[M ~map[K]V, K comparable, V any](
    order KeyOrder[K], rules ...Rule[V],
) Rule[M]

func MapRequiredKey[M ~map[K]V, K comparable, V any](
    key K, order KeyOrder[K], rules ...Rule[V],
) Rule[M]
func MapOptionalKey[M ~map[K]V, K comparable, V any](
    key K, order KeyOrder[K], rules ...Rule[V],
) Rule[M]
func MapKeysOneOf[M ~map[K]V, K comparable, V any](
    order KeyOrder[K], allowed ...K,
) Rule[M]
func MapKeysNotOneOf[M ~map[K]V, K comparable, V any](
    order KeyOrder[K], forbidden ...K,
) Rule[M]
```

`MapLength`, `MapMinLength`, `MapMaxLength` and `MapLengthBetween` use the same `M, K, V` type parameters and length arguments as their slice counterparts. They return the corresponding `length`, `min_length`, `max_length` or `length_between` code at the container. Nil maps have length zero.

`MapEach` supplies a typed `Entry` by value; `Project` can expose its key and value to ordinary rules without inventing path fields named `Key` or `Value`. `MapKeys` and `MapValues` attach the entry's Key segment. Multiple key and value failures at that same segment remain independent occurrences. Value membership is `MapValues(order, OneOf(...))`; no duplicate “map value membership” engine is needed.

An absent required key returns `key_required` at that Key segment. Absence of an optional key succeeds. Neither invokes child rules without a value. Both retain the specified `KeyOrder` to share the location policy, even though a single addressed key does not require sorting.

The supported key domain must have reflexive equality, a strict total order and injective stable text. `StringKeys` orders strings lexicographically and uses their exact text, including punctuation and empty strings. An integer-keyed map can use typed less-than and `strconv.Itoa`. No key type switch, input conversion to `any`, or `%v`-based universal identity scheme is supplied. NaN-bearing keys are deliberately outside this contract. Callback correctness is a caller obligation; the engine does not perform a full-input probe of order or encoding on successful calls.

### 4.7 Time

Each time rule accepts and returns `Rule[time.Time]`.

| Constructor | Relation | Failure code |
|---|---|---|
| `TimeBefore(other)` | Strictly before | `before` |
| `TimeBeforeOrEqual(other)` | Before or same instant | `before_or_equal` |
| `TimeAfter(other)` | Strictly after | `after` |
| `TimeAfterOrEqual(other)` | After or same instant | `after_or_equal` |
| `TimeBetween(min, max)` | Inclusive instant interval | `between` |
| `TimeNotZero()` | `!value.IsZero()` | `not_zero` |

Use `time.Time` comparisons, including their monotonic-clock semantics, not struct equality or string comparison [S9]. An inverted time interval is a construction error. Fixed values supplied in tests remove wall-clock dependence. “Now” is application data; no built-in calls `time.Now`.

### 4.8 Example: nested typed rule graph

This example targets the specified API. It must become a compiled repository example as part of implementation acceptance.

```go
package example

import v "github.com/jacoelho/validation/v2"

type Address struct { City string }
type User struct {
    Name string
    Age *int
    Addresses []Address
}

var addressRule = v.All(
    v.NotEmpty[string]().Field("city", func(a Address) string { return a.City }),
)

var userRule = v.All(
    v.All(v.NotEmpty[string](), v.RuneMinLength[string](2)).Field("name", func(u User) string { return u.Name }),
    v.OptionalPtr(v.Min(0)).Field("age", func(u User) *int { return u.Age }),
    v.Each[[]Address](addressRule).Field("addresses", func(u User) []Address { return u.Addresses }),
)

func ValidateUser(u User) error {
    return userRule.Validate(u)
}
```

Construction is outside calls to `ValidateUser`. A function returning `*LegacyError` is not directly assignment-compatible with `func(T) error`; migration needs an explicit nil-normalising wrapper. A method expression already returning `error` can be used as a typed `Rule` directly.

## 5. Error tree, paths, reporting and iterator semantics

### 5.1 Public observation contract

```go
type Code string
const CodeExternal Code = "external"

type Coded interface {
    error
    Code() Code
}

// Private storage; constructors return a non-nil error value.
type Violation struct { /* private */ }
func NewViolation(code Code, cause error) *Violation
func (e *Violation) Code() Code
func (e *Violation) Error() string
func (e *Violation) Unwrap() error

type SegmentKind uint8
const (
    FieldSegment SegmentKind = iota + 1
    IndexSegment
    KeySegment
)

type Segment struct {
    Kind SegmentKind
    Name string // Field name or canonical key text; empty for Index.
    Index int   // Index value; zero/ignored for Field and Key.
}
type Issue struct {
    Path []Segment // Independent, caller-owned snapshot.
    Code Code
    Err error      // The actual coded terminal or uncoded leaf.
}

func Issues(err error) []Issue
func FormatPath(path []Segment) string
func Format(err error) string
```

Codes in the catalogue are exported as constants named from the code, such as `CodeMin`, `CodeRuneMinLength`, `CodeUnique` and `CodeKeyRequired`. Their literal strings are the compatibility surface. A custom `Coded` error must return a non-empty stable code. `NewViolation(code, nil)` is a failure with no cause; it is not an aggregate and does not return nil.

Aggregate and location-wrapper concrete types remain private. Their observable contracts are ordinary unwrapping, stable ordering, safe formatting and immutable owned storage. Multi-Unwrap returns a fresh slice so the caller cannot overwrite aggregate storage. This allocation occurs only during failure inspection. Wrapped application error values are not deep-copied or mutated; callers remain responsible for their own mutable error objects.

`errors.Is` uses error identity and any normal custom Is methods; the library does not redefine code equality as sentinel identity. Use `Coded` or `Issues` to inspect codes. Go single- and multi-error unwrapping support normal discovery through wrappers and joins [S1].

### 5.2 Typed diagnostic details

No mandatory `map[string]any`, raw rejected-input field, or generic formatting of the input is required. The following built-in diagnostic observation types are part of the target contract; their storage is private and they implement `Coded` and `error`:

| Diagnostic type | Required observations |
|---|---|
| `LengthError` | `Actual() int`, `Minimum() int`, `Maximum() (int, bool)`, `Unit() LengthUnit`. Exact length uses equal minimum/maximum; a missing upper bound returns false. Unit is bytes, runes, elements or entries. |
| `BoundsError[T any]` | `Lower() (T, bool, bool)` and `Upper() (T, bool, bool)`, returning value, inclusive, present. Used for numeric/time/comparator bound failures. It does not store the rejected input. |
| `DuplicateError` | `FirstIndex() int`, identifying the earliest equal occurrence; the duplicate's own index is its location segment. |
| `IndexError` | `Index() int` and `Length() int` for an absent addressed element. |
| `Violation` | Code and optional ordinary cause for failures without additional standard detail. |

`LengthUnit` is a string-like type with values `bytes`, `runes`, `elements` and `entries`. Empty/non-empty and membership failures need only their code. Sign failures need no rejected value; their code defines their relation to zero. Bounds and configuration retained in diagnostics must not be confused with the original rejected value. Captured custom bound objects follow the shallow-copy ownership contract.

These types allow `errors.As` to inspect exact typed data without asking the validation engine to erase or rediscover the value type. They do not imply every built-in failure must unwrap to `*Violation`; the common contract is `Coded`.

### 5.3 Canonical path grammar

| Segment | Rendering |
|---|---|
| Empty path | `$` |
| Field matching `[A-Za-z_][A-Za-z0-9_]*` | `.` followed by the exact name |
| Other non-empty Field | `.[` + Go-quoted exact name + `]` |
| Index | `[` + base-10 non-negative integer + `]` |
| Key | `[` + Go-quoted exact canonical key text + `]` |

Examples: `$.name`, `$.["a.b"]`, `$.a.b`, `$["a.b"]`, `$[2]` and `$["2"]` are distinct. The unusual `.[quoted]` form deliberately marks a literal Field rather than a map Key. This is the library's path notation, not a claim of JSONPath compatibility. Quotes, backslashes, control characters and invalid UTF-8 bytes use normal Go quoted-string escapes. No parser is required in the production library; tests can use an independent decoder to check round-tripping of valid segments.

An empty Field name is invalid configuration. An empty map key is valid. Passing malformed caller-created `Segment` values to `FormatPath`—an unknown kind, negative Index, or empty Field—panics with `ConfigurationError` rather than emitting an ambiguous path.

### 5.4 Issue traversal versus cause traversal

`Issues(nil)` returns a nil slice. For a well-formed non-nil tree, the internal iterator follows this order at each current node:

1. A library location wrapper appends its one segment and recurses into its child.
2. A directly implemented `Coded` error yields one issue with the accumulated path and stops traversing that node's cause. It checks the current node, not `errors.As` on the entire subtree.
3. A multi-Unwrap node traverses each child in returned order, carrying the same accumulated path.
4. An ordinary single-Unwrap node traverses its non-nil child. A single-Unwrap node with no child is an external terminal.
5. Any other terminal yields one `CodeExternal` issue retaining that terminal error.

Standard `errors.Is`/`errors.As` still traverse a coded error's cause independently. `Issues` intentionally does not: a cause is not necessarily another requirement failure. Repeated nodes are visited once per occurrence, not once per pointer identity. Cyclic external error graphs and malformed multi-Unwrap nil children violate the extension contract; there is no global deduplication or reflection-based cycle repair.

A library-created aggregate always contains genuine child failures. A pathological external aggregate with no children is outside the failure-tree contract: custom successful functions must return literal nil rather than an empty non-nil aggregate.

### 5.5 Iterator lifetime and presentation

The internal shape is `walkIssues(error) iter.Seq[Issue]`. The push protocol requires propagation of the false return from `yield`; after it occurs, no later sibling is inspected. A fresh iteration starts at the root. Saved paths remain valid independently of later iterations. This follows the standard iterator shape, while its placement on the failure path is a design decision [S3].

`Issues` may collect this sequence into a slice; that is reporting work. `Format` may stream it directly. Ordinary validation must not build or consume this iterator.

`Format(err)` is the ordered list of `<canonical path>: <code>`, joined by `; `. `Format(nil)` is empty. It does not call external `Error` methods or print causes/details. Library-owned Error methods use the same semantics. When a single root application error is returned unchanged, that application's own `Error()` remains application-controlled; callers requiring the safe rendering contract must call `Format`.

Map keys and field identifiers appear in paths and can themselves be sensitive. This formatter avoids accidental rejected-value and cause-message disclosure; it is not a substitute for application logging/redaction policy.

## 6. Construction and evaluation behaviour

### 6.1 Constructor policy

Invalid programmer-supplied configuration panics with `ConfigurationError`, including a nil child callback, negative length/index, reversed range, NaN numeric bound, invalid path segment or empty coded-violation code. `ConfigurationError` is an error type with a stable `Constructor() string` observation identifying the rejecting operation; exact prose is not asserted. Constructors do not interpret malformed configuration as a normal input violation.

No fallible regex or dynamic-schema constructors are in scope. An application accepting dynamic configuration validates it before calling these constructors. Valid zero-size and empty membership configurations are supported, not configuration errors.

Constructors copy variadic slices and mutable byte configuration. They do not deep-clone arbitrary values, captured pointers or closures. For `[]Rule[T]`, replacing the caller's list entry cannot change the constructed rule; changing state inside a callback closure is the callback owner's responsibility. Fixed membership lookups can be built once and used read-only.

A bare nil `Rule` invoked directly has ordinary nil-function panic behaviour. Supplying it to a library constructor is caught as `ConfigurationError`. A `Check` factory that returns literal nil when its predicate fails causes a `ConfigurationError` at evaluation; the engine must not turn the failed predicate into successful validation. Arbitrary typed-nil callback errors are not normalised reflectively.

### 6.2 Collection of failures

Each composite starts with no allocated result collection. For every non-nil child result, append the existing result, applying an immutable location wrapper where required. Do not preallocate to the number of configured rules or input elements. On return, normalise zero/one/many results as required; do not repeatedly rebuild a joined aggregate inside the loop.

A rule evaluates a constraint, not an arbitrary number of subconstraints. `Between` reports one range failure when the range constraint fails. `All(Min(...), Max(...))` consists of two independently configured rules and retains both failures. A predicate `hasEmail || hasPhone` is one contact-presence constraint; it emits one factory result only when the entire predicate is false.

The engine must not rerun rules, cache an input-dependent result in a constructor, or mutate a previously returned error's field path. Built-ins do not retain input slices/maps in their errors merely to format them later.

### 6.3 Failure-only map processing

Validate entries in ordinary map iteration order. For an entry with no failures, retain no entry record and do not call key-order or rendering callbacks. For a failing entry, retain only the information necessary for that failing group. After entry validation, sort those groups by `Less`; render each group's key once and wrap its errors. Preserve child-rule order within each group.

A map with no failures has no failure group allocation, no Less/Text calls and no sorting. Sorting three failing keys in a map of ten thousand entries processes those three keys, not a list of every key. The engine must not promise deterministic callback order because it does not pre-sort all entries. It also must not rerun callbacks after sorting.

### 6.4 Uniqueness and disjunction

For each index i, the initial uniqueness rule searches earlier indices for the first equal value. On finding a match, report one issue for i and move to the next index. With unique values it uses n(n−1)/2 comparisons and no input-size workspace. A future scratch-based alternative is a different explicit API and is not part of this release.

Error-based `Or`/`Not` are removed because a failing alternative can allocate diagnostics even when the whole expression succeeds. Use short-circuit Boolean predicates within `Check`; the failure factory runs only for an overall failed constraint. No compulsory two-pass deferred-diagnostics framework is introduced.

## 7. Numbered requirements and acceptance links

All requirements in this section are mandatory. Verification categories identify how evidence is obtained; they do not mark implementation completion. Multiple linked scenarios provide different aspects or boundary cases of the same contract.

### Architecture and scope

#### REQ-ARCH-001 — One typed execution contract

The library shall expose Rule[T] as func(T) error, with Validate(T) error invoking that function. All, Struct, Field, Project, pointer adapters and collection adapters shall compose through this contract without an additional value-erasing execution engine.

Verification: compile, source, behaviour.\
Acceptance: `AT-CORE-001`, `AT-CORE-006`, `AT-SLICES-009`, `AT-EXTENSIONS-001`, `AT-TYPING_ARCHITECTURE-001`.

#### REQ-ARCH-002 — No reflection or unsafe input inspection

Library-owned production code shall not import reflect or unsafe, invoke reflection helpers, use unsafe directives, or inspect struct tags to validate input. This restriction applies to the library and its owned runtime dependencies, not to the implementation of the Go standard library.

Verification: source.\
Acceptance: `AT-CONSTRUCTION-006`, `AT-TYPING_ARCHITECTURE-003`.

#### REQ-ARCH-003 — Typed values and bounded conversions

Value validation shall use typed parameters, constraints and callbacks, not conversion to any, value type switches, or value type assertions. The only permitted input conversions are explicitly reviewed representation-preserving named-type conversions to their underlying type at standard-library boundaries. Numeric comparisons shall retain their original type; byte slices shall not be converted to strings for validation. Error-interface conversion and failure-side assertions on error nodes are permitted.

Verification: source, compile.\
Acceptance: `AT-COMPARABLE-003`, `AT-NUMBERS-002`, `AT-STRINGS_BYTES-006`, `AT-TYPING_ARCHITECTURE-004`.

#### REQ-ARCH-004 — No context or implicit I/O

The runtime API and engine shall have no context parameter, context import, context-aware dispatcher, captured request state, network access, filesystem access, environment lookup, goroutine launch, or implicit database operation. Custom callbacks remain application-controlled; the core shall not add or manage these operations.

Verification: source.\
Acceptance: `AT-TYPING_ARCHITECTURE-005`.

#### REQ-ARCH-005 — Standard-type scope only

The refactor shall provide the standard-type primitives in the rule catalogue and ordinary extension functions. It shall not add an is/format catalogue, regex parser, email/URL/UUID rule set, tag engine, registry lookup, SQL driver discovery, schema interpreter, code generator, fluent builder, or context wrapper.

Verification: source, compile.\
Acceptance: `AT-TYPING_ARCHITECTURE-005`, `AT-QUALITY_RELEASE-007`.

#### REQ-ARCH-006 — No runtime dependencies beyond the standard library

Library-owned runtime packages shall depend only on the Go standard library. Acceptance-test tools may be development-only dependencies and shall not enter the library module runtime dependency graph.

Verification: source.\
Acceptance: `AT-TYPING_ARCHITECTURE-005`.

#### REQ-ARCH-007 — Direct synchronous evaluation

Validation shall invoke typed rules through direct loops and normal synchronous calls. Iterator/reporting callbacks, channels, iter.Pull, reflection and pooled reporters shall not be the mandatory validation execution protocol.

Verification: source, behaviour.\
Acceptance: `AT-ITERATION-005`, `AT-ALLOCATION-007`.

### Core evaluation

#### REQ-CORE-001 — Literal nil success

Every built-in rule and supported composition shall return a literal nil error interface when validation succeeds, including when the result crosses another function returning error.

Verification: behaviour.\
Acceptance: `AT-CORE-001`, `AT-CORE-005`, `AT-QUALITY_RELEASE-008`.

#### REQ-CORE-002 — Exhaustive independent validation

All shall evaluate every applicable configured rule and preserve every non-nil result. A failure, including an unclassified application error, shall not skip later independent rules.

Verification: behaviour.\
Acceptance: `AT-CORE-002`, `AT-CORE-003`, `AT-CORE-008`, `AT-ERRORS-008`, `AT-PRESENCE-002`, `AT-STRINGS_BYTES-002`, `AT-SLICES-001`, `AT-SLICES-003`, `AT-MAPS-004`.

#### REQ-CORE-003 — Exactly-once evaluation

Each applicable rule occurrence shall execute once per validation of its input. Rules shall not be replayed to reconstruct errors. A configured occurrence of the same function in two positions shall execute twice; Field and Project shall invoke their getter once per call, and When/Unless shall invoke their predicate once per call.

Verification: behaviour.\
Acceptance: `AT-CORE-002`, `AT-CORE-004`, `AT-SLICES-004`, `AT-EXTENSIONS-002`.

#### REQ-CORE-004 — Defined ordering

Errors shall follow rule declaration order, depth-first within a child rule, then ascending slice index within Each. A map-traversing rule shall order its failing entry groups according to its configured key order; independent map rules retain their outer declaration order.

Verification: behaviour.\
Acceptance: `AT-CORE-002`, `AT-CORE-003`, `AT-PATHS-001`, `AT-MAPS-003`.

#### REQ-CORE-005 — Valid empty compositions

All() and Struct() shall succeed. Field and Project with no child rules shall still call their getter once and succeed. Empty Each, MapEach and conditional child groups shall produce no failures; guard behaviour for pointer, index and required-key adapters remains as specified.

Verification: behaviour.\
Acceptance: `AT-CORE-005`, `AT-PRESENCE-006`.

#### REQ-CORE-006 — Parent-aware conditional validation

When and Unless shall evaluate their predicate against the exact input type of their child group, execute all children when enabled, and skip only that group when disabled. A condition on a parent object shall be able to guard a Field rule without a value-type cast.

Verification: behaviour.\
Acceptance: `AT-CORE-006`.

#### REQ-CORE-007 — No hidden error cap or deduplication

The library shall not truncate, coalesce, or deduplicate independent error occurrences by path, code, pointer identity or count. The same sentinel at two rule positions shall remain two reported occurrences.

Verification: behaviour.\
Acceptance: `AT-CORE-002`, `AT-CORE-004`, `AT-CORE-008`.

#### REQ-CORE-008 — Panic propagation

The library shall not recover a panic raised by a user rule, getter, comparator, predicate, failure factory or error-inspection method. Exhaustive evaluation is guaranteed only while callbacks return normally; no claim of further evaluation applies after a panic.

Verification: behaviour.\
Acceptance: `AT-CORE-007`, `AT-CONSTRUCTION-005`, `AT-TYPING_ARCHITECTURE-006`.

#### REQ-CORE-009 — No implicit optionality

Ordinary scalar and container rules shall validate zero, empty and nil-container values using their documented semantics rather than silently treating them as optional. Optionality shall be introduced only by explicit adapters or conditions.

Verification: behaviour.\
Acceptance: `AT-CORE-009`, `AT-PRESENCE-001`, `AT-STRINGS_BYTES-004`, `AT-TIME-002`, `AT-SLICES-002`, `AT-MAPS-001`.

### Construction and reusable state

#### REQ-CONF-001 — Construction errors are explicit

A constructor receiving invalid static configuration shall panic with a library ConfigurationError before returning a rule. Invalid configuration includes nil required callbacks or child rules, negative lengths or indices, reversed ranges, NaN numeric bounds, an empty Field name, an empty violation code, or an incomplete KeyOrder. Exact panic message text is not part of the contract.

Verification: behaviour.\
Acceptance: `AT-CONSTRUCTION-001`, `AT-PATHS-006`, `AT-TYPING_ARCHITECTURE-006`.

#### REQ-CONF-002 — Freeze supplied configuration

Constructors shall copy supplied variadic rule/value slices and byte-slice configuration before storing them, and shall build fixed lookup data at construction. Later mutation of the original slice storage shall not change the rule. Copies are shallow for reference-containing elements; callers shall not mutate objects reachable through captured references or callback closures.

Verification: behaviour, source.\
Acceptance: `AT-CONSTRUCTION-002`, `AT-CONSTRUCTION-003`.

#### REQ-CONF-003 — Reusable rules own no mutable per-call state

Constructed built-in rules shall be safely reusable concurrently with read-only inputs. They shall not retain input references after a call except in explicitly supplied custom errors, cache per-input results, or share mutable scratch/error storage between calls.

Verification: race, source, behaviour.\
Acceptance: `AT-ERRORS-005`, `AT-MAPS-010`, `AT-ALLOCATION-007`, `AT-QUALITY_RELEASE-001`.

#### REQ-CONF-004 — Lazy diagnostics

Successful validation shall not invoke diagnostic factories, format keys or paths, construct result wrappers or error parameter objects, allocate aggregate storage, or sort map entries. Diagnostic generation shall begin only after the relevant failure.

Verification: behaviour, allocation, source.\
Acceptance: `AT-MAPS-005`, `AT-MAPS-011`, `AT-EXTENSIONS-003`, `AT-ALLOCATION-001`.

#### REQ-CONF-005 — Failure factories must return a failure

Check shall require a non-nil predicate and failure factory; it shall call the predicate once and invoke the factory once only when the predicate is false. A factory returning literal nil after predicate failure shall cause a ConfigurationError panic, not successful validation.

Verification: behaviour.\
Acceptance: `AT-CONSTRUCTION-005`, `AT-EXTENSIONS-003`, `AT-EXTENSIONS-004`.

#### REQ-CONF-006 — Construction and evaluation do not run user rules early

Constructors shall not invoke value getters, predicates, validators, key renderers or failure factories. BetweenBy may compare its two configured bounds once to validate their order; later per-value comparison counts are defined by that rule.

Verification: behaviour.\
Acceptance: `AT-CONSTRUCTION-004`.

### Errors and interoperability

#### REQ-ERR-001 — Ordinary error interoperability

Library wrappers shall implement Unwrap() error and aggregates shall implement Unwrap() []error so errors.Is and errors.As can reach children and causes through library wrappers, fmt.Errorf with %w and errors.Join. Library aggregates shall not publish nil children.

Verification: behaviour.\
Acceptance: `AT-ERRORS-002`, `AT-ERRORS-003`.

#### REQ-ERR-002 — Aggregate normalisation

A composite shall return literal nil for zero failures, the original error for one failure when no location or required grouping must be added, and an aggregate for multiple failures. A zero-length typed error slice shall never be returned as success through error.

Verification: behaviour.\
Acceptance: `AT-CORE-001`, `AT-ERRORS-001`.

#### REQ-ERR-003 — Error ownership

The library shall never change an error returned by a rule, including its code, field, cause or custom payload. Library-owned errors and aggregate storage shall remain unchanged after publication, and accessors exposing mutable storage shall return independent copies. Arbitrary application-owned errors must themselves be safe for concurrent inspection.

Verification: behaviour, race.\
Acceptance: `AT-ERRORS-005`, `AT-ERRORS-006`, `AT-PATHS-004`, `AT-QUALITY_RELEASE-001`.

#### REQ-ERR-004 — Stable machine-readable codes

Built-in failures shall expose the stable codes and typed details defined in the rule catalogue. Custom errors may implement Coded with Code() Code. Built-in code meanings shall not depend on Error() wording; error details shall not require map[string]any or boxing the validated value.

Verification: behaviour.\
Acceptance: `AT-ERRORS-003`, `AT-ERRORS-010`, `AT-ERRORS-011`.

#### REQ-ERR-005 — Causes are not extra violations

Issues shall emit one occurrence when it reaches a directly coded error, retaining that coded error and its cause chain. It shall not emit the cause as another issue. Standard errors.Is/As traversal shall still reach the cause.

Verification: behaviour.\
Acceptance: `AT-ERRORS-003`, `AT-ITERATION-001`.

#### REQ-ERR-006 — Unclassified external failures are preserved

A non-coded error shall remain in the returned error tree unchanged. Issues shall traverse generic wrapping and joins to produce one external issue per uncoded terminal error, with CodeExternal, without treating it as successful validation or Boolean false.

Verification: behaviour.\
Acceptance: `AT-ERRORS-001`, `AT-ERRORS-004`, `AT-ERRORS-008`, `AT-EXTENSIONS-006`, `AT-ITERATION-001`.

#### REQ-ERR-007 — Custom typed-nil boundary

Custom Rule functions and failure factories shall return literal nil on success and non-nil, usable error values on failure. A non-nil error interface containing a typed-nil pointer is a callback contract violation; the library shall not use reflection to normalise it. Explicit legacy adapters shall check concrete pointer/slice nils before interface conversion.

Verification: behaviour, source.\
Acceptance: `AT-CONSTRUCTION-006`, `AT-QUALITY_RELEASE-008`.

#### REQ-ERR-008 — Predictable safe presentation

Format(nil) shall return an empty string. Format(err) and library-owned Error methods shall present issue paths and codes in occurrence order, joined by "; ", without automatically printing rejected values, diagnostic details, cause messages or calling external Error methods. Paths may contain user-provided field/key identifiers; they are not a confidentiality boundary.

Verification: behaviour.\
Acceptance: `AT-ERRORS-007`.

#### REQ-ERR-009 — Finite well-formed error-tree contract

Library-generated error trees shall be finite and acyclic. Extension errors shall obey the same constraint, shall not return nil children from multi-Unwrap, and shall remain inspectable without mutation. Repeated error identity in an acyclic tree is supported and shall not trigger global deduplication. Cyclic or malformed external trees are outside the inspection contract.

Verification: behaviour, source.\
Acceptance: `AT-CORE-004`, `AT-ERRORS-004`, `AT-ERRORS-009`, `AT-ITERATION-003`.

### Paths and location ownership

#### REQ-PATH-001 — Typed path segments

Locations shall distinguish Field, Index and Key segments. Field names and rendered key strings shall be opaque text, never parsed as embedded paths; indices shall remain integers until presentation.

Verification: behaviour.\
Acceptance: `AT-PATHS-001`, `AT-PATHS-002`, `AT-MAPS-009`.

#### REQ-PATH-002 — Compositional location

Field, Each, indexed and map-entry adapters shall prepend their location to child locations without replacing them. Project, All, Struct, conditions and nullable/pointer adapters shall not invent path segments.

Verification: behaviour.\
Acceptance: `AT-PATHS-001`, `AT-PATHS-005`.

#### REQ-PATH-003 — Path snapshots

Each Issue returned by Issues shall own an independent path snapshot. Mutating one returned path or issue list shall not alter another issue, the original error, or a later traversal.

Verification: behaviour.\
Acceptance: `AT-PATHS-004`, `AT-ITERATION-004`.

#### REQ-PATH-004 — Canonical rendering

FormatPath shall use the canonical grammar in section 5: "$" for root, ".identifier" for identifier fields, ".[quoted]" for other field names, "[integer]" for indices and "[quoted]" for map keys. Quoting shall escape special/control/invalid UTF-8 bytes without changing segment identity.

Verification: behaviour.\
Acceptance: `AT-ERRORS-007`, `AT-PATHS-002`, `AT-PATHS-003`, `AT-PATHS-006`.

#### REQ-PATH-005 — Failure-only key identity

Map-key renderers shall run only for failing or missing addressed keys, at most once per failing key group within one map rule call. The renderer shall provide stable, injective canonical text for its supported key domain; distinct string keys shall not collide. Arbitrary callbacks with ambiguous encodings are outside that contract.

Verification: behaviour, allocation.\
Acceptance: `AT-MAPS-005`, `AT-MAPS-006`, `AT-MAPS-009`, `AT-MAPS-011`.

### Pointers and nullable presence

#### REQ-PTR-001 — Explicit pointer presence

OptionalPtr shall succeed for nil and invoke no value rules; RequiredPtr shall emit required for nil and invoke no value rules. For a non-nil pointer, both shall invoke all child rules on the pointed-to value without treating zero as missing.

Verification: behaviour.\
Acceptance: `AT-PRESENCE-001`, `AT-PRESENCE-002`, `AT-PRESENCE-006`.

#### REQ-PTR-002 — Recursive pointer composition

Pointer adapters shall support nested pointer layers and pointers to structs and shall preserve child paths without dereferencing a nil layer. Missing one pointer shall not stop other independent rules.

Verification: behaviour.\
Acceptance: `AT-PATHS-005`, `AT-PRESENCE-002`, `AT-PRESENCE-003`.

#### REQ-PTR-003 — Explicit nullable projection

OptionalValue and RequiredValue shall accept a typed getter returning (value, present). They shall call it once. When absent they shall skip value rules and respectively succeed or return required; when present they shall validate the returned value even if zero.

Verification: behaviour.\
Acceptance: `AT-PRESENCE-004`, `AT-PRESENCE-006`.

#### REQ-PTR-004 — No invented presence information

The library shall not infer omitted-versus-explicit-null state, missing-versus-zero fields, or sql/driver semantics from values. Callers shall supply any required presence discriminator through typed input or a getter.

Verification: behaviour, source.\
Acceptance: `AT-PRESENCE-004`, `AT-PRESENCE-005`, `AT-EXTENSIONS-006`.

### Standard values and typed comparisons

#### REQ-TYPE-001 — Comparable primitives

Equal, NotEqual, Zero, NotZero, OneOf and NotOneOf shall implement the equality/membership semantics in the catalogue for concrete safely comparable values. Empty OneOf shall fail every value; empty NotOneOf shall pass every value. Duplicate configured members shall not create additional failures.

Verification: behaviour.\
Acceptance: `AT-COMPARABLE-001`, `AT-COMPARABLE-002`.

#### REQ-TYPE-002 — Exact numeric bounds

Numeric rules shall accept built-in and named integer/unsigned/float types, exclude strings and complex types at compile time, preserve exact integer comparisons, and implement the documented inclusive/exclusive bounds without overflow-prone subtraction.

Verification: behaviour, compile.\
Acceptance: `AT-ERRORS-011`, `AT-NUMBERS-001`, `AT-NUMBERS-002`, `AT-NUMBERS-005`.

#### REQ-TYPE-003 — NaN and infinity policies

All numeric bound and sign rules shall reject a NaN input. NotNaN shall reject only NaN; Finite shall reject NaN and both infinities. Infinite bounds shall be allowed except reversed bounds; other numeric rules shall accept infinity when the declared relation is true.

Verification: behaviour.\
Acceptance: `AT-NUMBERS-003`, `AT-NUMBERS-004`.

#### REQ-TYPE-004 — Zero and sign semantics

Positive, NonNegative, Negative and NonPositive shall use the mathematical sign relations in the catalogue. Both signed floating-point zeros shall be treated as zero. False, zero numbers and empty strings shall not be considered missing except under the explicitly selected zero/non-empty rule.

Verification: behaviour.\
Acceptance: `AT-COMPARABLE-001`, `AT-NUMBERS-004`.

#### REQ-TYPE-005 — String byte and rune semantics

String length rules shall distinguish bytes from Unicode code points and shall not perform Unicode normalisation or grapheme counting. Rune counts shall follow Go UTF-8 decoding, including one replacement rune per invalid byte. UTF8 shall separately reject malformed encoding.

Verification: behaviour.\
Acceptance: `AT-ERRORS-010`, `AT-STRINGS_BYTES-001`, `AT-STRINGS_BYTES-002`, `AT-STRINGS_BYTES-003`.

#### REQ-TYPE-006 — String predicate semantics

NotEmpty, Contains, HasPrefix and HasSuffix shall use exact, case-sensitive string semantics, including empty substring/prefix/suffix matches. Whitespace shall not be trimmed, and no optional-empty skip shall occur unless explicitly configured.

Verification: behaviour.\
Acceptance: `AT-CORE-009`, `AT-STRINGS_BYTES-004`.

#### REQ-TYPE-007 — Dedicated byte rules

Byte-slice rules shall operate on original bytes and preserve named byte-slice types without conversion to strings. Nil and empty byte slices shall be equal under byte-content comparison and have length zero; explicit required-pointer/presence checks are separate.

Verification: behaviour, compile, source.\
Acceptance: `AT-CONSTRUCTION-003`, `AT-STRINGS_BYTES-005`, `AT-STRINGS_BYTES-006`.

#### REQ-TYPE-008 — Time relations

TimeBefore, TimeBeforeOrEqual, TimeAfter, TimeAfterOrEqual and TimeBetween shall use time.Time chronological comparisons, not == or formatted strings. Equal instants shall receive the documented inclusive/exclusive results. TimeNotZero shall use IsZero; no rule shall read the system clock.

Verification: behaviour, source.\
Acceptance: `AT-TIME-001`, `AT-TIME-002`, `AT-TIME-003`.

#### REQ-TYPE-009 — Named values and typed arrays

Named numbers, strings, slices and maps shall work through typed getters without conversion to any or an unnamed collection. Fixed arrays shall be supported by explicit typed slice projection; pointer-parent examples shall avoid taking a slice of a copied array.

Verification: compile, allocation.\
Acceptance: `AT-SLICES-009`, `AT-MAPS-008`, `AT-TYPING_ARCHITECTURE-001`, `AT-TYPING_ARCHITECTURE-002`.

#### REQ-TYPE-010 — Typed comparison extensions

MinBy, MaxBy and BetweenBy shall compare values using a supplied func(T,T) int without numeric conversion or runtime type discovery. The comparator shall define a consistent order; returned signs shall be interpreted rather than assuming only -1, 0 and 1.

Verification: behaviour.\
Acceptance: `AT-NUMBERS-006`.

#### REQ-TYPE-011 — Explicit comparison-domain limits

The contract for comparable rules shall use Go == semantics: NaN is not equal to itself and signed zeros are equal. Interface-bearing values whose equality can panic are not supported comparison inputs; the library shall not claim the compiler rejects every such instantiation or inspect them reflectively.

Verification: behaviour, source.\
Acceptance: `AT-COMPARABLE-002`, `AT-COMPARABLE-003`, `AT-SLICES-008`.

### Slices and collections

#### REQ-SLICE-001 — Slice shape constraints

Slice length rules shall validate len, treat nil as length zero, and return their own container issue without skipping independent Each or other collection rules.

Verification: behaviour.\
Acceptance: `AT-ERRORS-010`, `AT-SLICES-001`, `AT-SLICES-002`.

#### REQ-SLICE-002 — Exhaustive element validation

Each shall validate all elements in ascending index order, run all child rules for each element, support nested Rule[E] values and named slice types, and preserve every located failure.

Verification: behaviour, compile.\
Acceptance: `AT-SLICES-001`, `AT-SLICES-009`.

#### REQ-SLICE-003 — Indexed applicability

AtIndex shall require a non-negative configured index; an index outside the input length shall return index_out_of_range without invoking element rules, while other independent rules continue. A valid index shall run every child rule even if some fail.

Verification: behaviour.\
Acceptance: `AT-PRESENCE-006`, `AT-SLICES-003`, `AT-SLICES-004`.

#### REQ-SLICE-004 — Collection membership and exclusion

SliceOneOf and SliceNotOneOf shall report one issue for every offending element occurrence. SliceContains shall emit one container contains issue when the sought value does not occur, rather than one issue for every non-matching element.

Verification: behaviour.\
Acceptance: `AT-SLICES-005`, `AT-SLICES-006`.

#### REQ-SLICE-005 — All duplicate occurrences

SliceUnique shall report one unique issue at each duplicate occurrence after the first equal occurrence, with first_index identifying the earliest equal input index. It shall neither stop at the first duplicate nor report every matching pair.

Verification: behaviour.\
Acceptance: `AT-SLICES-007`, `AT-SLICES-008`.

#### REQ-SLICE-006 — Non-mutating zero-scratch uniqueness

The initial SliceUnique implementation shall use a non-mutating previous-elements scan with O(n^2) worst-case comparisons and no input-size-dependent successful-path workspace. It shall not sort input, allocate a per-call seen map, or hide mutable scratch/pools in a reusable rule.

Verification: behaviour, source, allocation.\
Acceptance: `AT-SLICES-007`, `AT-ALLOCATION-006`.

#### REQ-SLICE-007 — Read-only collection validation

Built-in slice and byte rules shall not change input length, capacity, element values or backing arrays. Returned errors shall not require retaining entire slice inputs.

Verification: behaviour, race.\
Acceptance: `AT-STRINGS_BYTES-005`, `AT-SLICES-010`, `AT-QUALITY_RELEASE-003`.

### Maps and key policies

#### REQ-MAP-001 — Map shape and absence

Map size rules shall use len, with nil length zero. MapRequiredKey shall use lookup presence, report key_required when absent and skip child rules; MapOptionalKey shall succeed when absent. Present zero values shall be validated.

Verification: behaviour.\
Acceptance: `AT-ERRORS-010`, `AT-PRESENCE-006`, `AT-MAPS-001`, `AT-MAPS-002`, `AT-MAPS-011`.

#### REQ-MAP-002 — Exhaustive typed map rules

MapEach, MapKeys and MapValues shall support named maps and typed Entry[K,V] data, invoke all children for every entry and preserve every key/value failure. MapKeysOneOf and MapKeysNotOneOf shall report every forbidden key; map-value membership shall be expressible by composing MapValues with ordinary membership rules.

Verification: behaviour.\
Acceptance: `AT-MAPS-003`, `AT-MAPS-004`.

#### REQ-MAP-003 — Failure-only deterministic result ordering

Map-traversing rules shall collect only failed key groups, sort those groups by KeyOrder.Less after validation and retain declared child-rule order within each group. A successful call shall neither invoke Less nor allocate a complete-key list.

Verification: behaviour, allocation, source.\
Acceptance: `AT-MAPS-005`, `AT-MAPS-006`, `AT-MAPS-007`.

#### REQ-MAP-004 — Ordering callbacks are explicit

StringKeys shall provide exact lexicographic ordering and identity text for string-like keys. Other key domains shall require a typed KeyOrder with non-nil Less and Text callbacks. Supported map keys shall be reflexive under == and the order shall strictly distinguish different supported keys. NaN-bearing and runtime-incomparable keys are outside the map-key contract.

Verification: behaviour, compile.\
Acceptance: `AT-MAPS-008`, `AT-MAPS-009`.

#### REQ-MAP-005 — No deterministic execution claim

Map callbacks shall be allowed to execute in native map iteration order. Result ordering shall not require callback re-execution, a pre-sort of all keys, or a second validation pass. Independent callbacks shall not depend on entry execution order.

Verification: behaviour, source.\
Acceptance: `AT-MAPS-006`, `AT-MAPS-007`, `AT-QUALITY_RELEASE-003`.

#### REQ-MAP-006 — Map ownership

Map validation shall not add, delete, replace or reorder caller-visible entries, retain the whole map in a built-in error, or write per-call results into constructor-owned state. Simultaneous validation of read-only maps shall be safe.

Verification: behaviour, race.\
Acceptance: `AT-MAPS-010`, `AT-QUALITY_RELEASE-001`.

### Extension contracts

#### REQ-EXT-001 — Ordinary function extensions

A caller shall be able to supply func(T) error directly as a Rule[T], including a typed method expression, without registration, inheritance, reflection, or conversion of the input to any.

Verification: compile, behaviour.\
Acceptance: `AT-NUMBERS-006`, `AT-EXTENSIONS-001`, `AT-ALLOCATION-005`.

#### REQ-EXT-002 — Typed projection

Project shall invoke a typed getter once and apply all child rules to the projected value without adding location. A getter can expose a custom nullable discriminator, amount, array view or value object without a core type catalogue.

Verification: compile, behaviour.\
Acceptance: `AT-PATHS-005`, `AT-PRESENCE-005`, `AT-EXTENSIONS-002`, `AT-TYPING_ARCHITECTURE-002`.

#### REQ-EXT-003 — Predicate alternatives and negation

The core shall not expose arbitrary error-based Or or RuleNot. Applications shall compose Boolean predicates inside Check; failed predicate alternatives that lead to overall success shall not construct errors, invoke failure factories, or replay a predicate.

Verification: behaviour, source, allocation.\
Acceptance: `AT-EXTENSIONS-003`, `AT-EXTENSIONS-004`, `AT-EXTENSIONS-005`.

#### REQ-EXT-005 — Generic method adapters

On Go 1.27, a preconstructed `Rule[V]` shall expose generic `Field`, `Project`, `OptionalValue` and `RequiredValue` methods that infer the parent type from typed getters. Their results shall be ordinary `Rule[P]` values with the same paths, presence guards, evaluation order and configuration errors as the variadic function forms. The function forms shall delegate to the method implementations so each adapter has one execution policy.

Verification: compile, behaviour, allocation.\
Acceptance: `AT-TYPING_ARCHITECTURE-007`, `AT-TYPING_ARCHITECTURE-008`.

#### REQ-EXT-004 — Operational failures remain independent errors

Ordinary application errors shall participate in exhaustive collection, not be automatically classified as user-input violations, negated, swallowed or translated to absence. Explicitly wrapping a cause in a coded violation is a caller decision.

Verification: behaviour.\
Acceptance: `AT-ERRORS-008`, `AT-EXTENSIONS-006`.

### Performance and allocation

#### REQ-PERF-001 — Zero-allocation valid evaluation

For preconstructed built-in rules, prepared supported inputs and non-allocating callbacks, every successful supported composition shall allocate zero heap objects and zero heap bytes attributable to validation on the compiler/platform matrix in section 8. The target includes the first invocation after construction; library lazy warm-up is not an exception.

Verification: allocation.\
Acceptance: `AT-MAPS-005`, `AT-EXTENSIONS-003`, `AT-ALLOCATION-001`, `AT-ALLOCATION-003`, `AT-QUALITY_RELEASE-002`.

#### REQ-PERF-002 — Separated cost boundaries

Allocation tests shall exclude construction, fixture preparation, reporting, test-runner and runtime bootstrap costs while including every operation performed by Validate. Custom callback allocations shall be reported separately and shall not weaken the built-in guarantee.

Verification: allocation, source.\
Acceptance: `AT-ALLOCATION-001`, `AT-ALLOCATION-004`, `AT-ALLOCATION-005`.

#### REQ-PERF-003 — No masking by pools or caches

Successful-path tests and implementations shall not rely on sync.Pool, per-rule mutable caches, prewarmed validator calls, silently supplied scratch buffers or disabling runtime allocation reporting to meet the contract.

Verification: source, allocation.\
Acceptance: `AT-ALLOCATION-003`, `AT-ALLOCATION-007`.

#### REQ-PERF-004 — Representative allocation matrix

Allocation gates shall exercise stored rules from an external Go test package, all built-in families, flat/deep composition, pointer/nullable/named types, fixed arrays, maps, membership and unique scans, using the sizes and callbacks in section 8.

Verification: allocation.\
Acceptance: `AT-ALLOCATION-001`, `AT-TYPING_ARCHITECTURE-002`.

#### REQ-PERF-005 — Both success and failure benchmarks

Benchmarks shall separately record construction, successful validation, failing validation, error traversal and formatting, including ns/op, B/op, allocs/op, compiler, architecture and input size. CPU timing is evidence, not a fabricated absolute SLA; report quadratic uniqueness scaling explicitly.

Verification: benchmark.\
Acceptance: `AT-ALLOCATION-004`, `AT-ALLOCATION-006`.

#### REQ-PERF-006 — No rounded-average proof of zero

Zero-allocation release evidence shall include total allocation/byte deltas for isolated first-call and batch probes in addition to AllocsPerRun and -benchmem. A positive library-attributable delta shall fail the gate even when averages round down to zero. Noisy probes shall be investigated, not silently subtracted or accepted.

Verification: allocation.\
Acceptance: `AT-ALLOCATION-002`, `AT-ALLOCATION-003`.

### Internal iterators

#### REQ-ITER-001 — Failure-side push iterator

An internal walkIssues(error) iter.Seq[Issue] shall traverse the existing result for Issues/Format. Validation shall not invoke it, build a sequence of input errors or eagerly materialise an iterator just to validate values.

Verification: source, behaviour.\
Acceptance: `AT-ITERATION-001`, `AT-ITERATION-005`.

#### REQ-ITER-002 — Iterator semantics

The issue iterator shall support nil, generic single wrappers, multiple wrappers, coded terminals, location wrappers and repeated occurrences, using the traversal algorithm in section 5. It shall not call errors.As repeatedly as an enumeration mechanism.

Verification: behaviour.\
Acceptance: `AT-ITERATION-001`.

#### REQ-ITER-003 — Early termination and replay

Once yield returns false, the iterator shall stop without yielding or unwrapping later siblings. Starting another iteration shall start from the original root and produce the complete sequence independently. Iterator traversal shall not mutate errors or shared path buffers.

Verification: behaviour.\
Acceptance: `AT-ITERATION-002`, `AT-ITERATION-003`.

#### REQ-ITER-004 — Independent yielded paths

Paths yielded by the internal iterator shall be independently retainable snapshots, not a reused scratch slice valid only during yield. Failure-side traversal may allocate and has no zero-allocation promise.

Verification: behaviour.\
Acceptance: `AT-PATHS-004`, `AT-ITERATION-004`.

### Verification and release evidence

#### REQ-QUAL-001 — Native supported toolchains

Release checks shall run on the declared Go 1.27 minimum using a pinned patch version. Native correctness and zero-allocation evidence shall cover linux/amd64, linux/arm64 and darwin/arm64; cross-compilation alone shall not certify allocations.

Verification: release.\
Acceptance: `AT-QUALITY_RELEASE-002`.

#### REQ-QUAL-002 — Race-safe reuse

Race tests shall reuse one constructed validator and shared immutable errors across at least 32 goroutines and 100 validations per goroutine, covering successful and failing inputs, paths and read-only maps.

Verification: race.\
Acceptance: `AT-QUALITY_RELEASE-001`.

#### REQ-QUAL-003 — Property and fuzz verification

The suite shall include bounded property/fuzz tests for path rendering, numeric boundaries, exhaustive collection counts/order, input preservation and map insertion-order invariance. Oracles shall be independent of the implementation under test.

Verification: fuzz.\
Acceptance: `AT-QUALITY_RELEASE-003`.

#### REQ-QUAL-004 — Compiled examples and negative typing tests

All normative API examples and named-type fixtures shall compile. Unsupported concrete types shall be tested as compile failures without comparing compiler error wording. Examples shall not depend on a live service, wall-clock time or a package unavailable from the stated dependency set.

Verification: compile.\
Acceptance: `AT-NUMBERS-005`, `AT-TYPING_ARCHITECTURE-001`, `AT-QUALITY_RELEASE-009`.

#### REQ-QUAL-005 — Traceable acceptance evidence

Each requirement shall have at least one acceptance scenario or outline linked by ID; each scenario shall have a unique ID and a declared verification category. An outline example row is a distinct case. Missing step bindings, skipped required cases or stale evidence shall not count as passing acceptance.

Verification: traceability.\
Acceptance: `AT-QUALITY_RELEASE-004`, `AT-QUALITY_RELEASE-005`, `AT-QUALITY_RELEASE-009`.

#### REQ-QUAL-006 — Current-revision evidence

Recorded acceptance results shall identify the implementation commit, spec revision/content hash, feature-file hashes, toolchain and platform. Results from another code/spec revision shall not satisfy the release gate. Parser/traceability checks alone shall not be labelled implementation acceptance.

Verification: release, traceability.\
Acceptance: `AT-ALLOCATION-002`, `AT-QUALITY_RELEASE-002`, `AT-QUALITY_RELEASE-006`, `AT-QUALITY_RELEASE-009`.

#### REQ-QUAL-007 — Read-only scope auditing

Source gates shall use parsed/type-aware checks over library-owned runtime packages, including generated and build-tagged files. They shall distinguish any as a type constraint from interface boxing, and shall allow standard error traversal only in failure/reporting code. A simple grep of dependencies is insufficient evidence.

Verification: source.\
Acceptance: `AT-TYPING_ARCHITECTURE-003`, `AT-TYPING_ARCHITECTURE-004`.

### Migration

#### REQ-MIG-001 — One engine after migration

Legacy scalar, struct, slice and map signatures shall be migrated together to Rule[T] func(T) error. Struct may remain a thin All alias; StructField/SliceField/MapField shall be removed in the breaking API, not retained as separate engines.

Verification: compile, source.\
Acceptance: `AT-TYPING_ARCHITECTURE-001`, `AT-QUALITY_RELEASE-007`.

#### REQ-MIG-002 — Document intentional behaviour breaks

Migration guidance shall enumerate all-error semantics, removed Fatal/RuleStopOnError and error-based Or/RuleNot, pointer presence rules, coded path errors, old-to-new rule naming, regex removal and nil-normalising legacy adapters. Legacy examples shall not be represented as working unchanged.

Verification: release.\
Acceptance: `AT-QUALITY_RELEASE-007`, `AT-QUALITY_RELEASE-008`.

#### REQ-MIG-003 — No weakening to preserve compatibility

The release shall treat the Rule return-type and related API changes as breaking, use the `/v2` module path after the published v1.0.x tags, and shall not reintroduce reflection, context, first-error return, runtime type dispatch or allocating successful disjunction to preserve old call sites.

Verification: release, source.\
Acceptance: `AT-QUALITY_RELEASE-007`.

#### REQ-MIG-004 — Complete implementation gate

The refactor shall be accepted only when all required behavioural, compile, source, race, fuzz, allocation and release cases pass on the required matrix and the requirements/features/docs describe the same revision. A specification-only bundle shall not be described as the implemented refactor.

Verification: release.\
Acceptance: `AT-QUALITY_RELEASE-005`, `AT-QUALITY_RELEASE-006`, `AT-QUALITY_RELEASE-009`.

## 8. Verification procedures and allocation gate

### 8.1 Required matrix

Require `go 1.27` as the minimum language/module requirement and test a pinned Go 1.27 patch version on each native platform. Go 1.27 permits generic methods on concrete types; the `Rule` adapters above use them for typed composition without adding builder state [S5, S6].

| Native platform | Go 1.27 | Required evidence |
|---|---|---|
| linux/amd64 | Required | Compile, correctness, allocation |
| linux/arm64 | Required | Compile, correctness, allocation |
| darwin/arm64 | Required | Compile, correctness, allocation |

Run source checks, full race tests and bounded fuzz/property gates on at least Go 1.27 linux/amd64; compile the complete suite on the other required pairs. Additional supported platforms can be added only with recorded claims distinguishing compile support from measured allocation support. An unavailable native runner is a missing release result, not a presumed pass.

Record the resolved exact version and platform rather than treating `1.27.x` as a reproducible measurement. The specification alone does not certify any run.

### 8.2 Test categories and execution routing

| Tag | Required binding and evidence |
|---|---|
| `@behaviour` | Deterministic Go unit/integration test or Gherkin step binding with exact assertions. |
| `@compile` | Positive/negative external-package compile fixture, plus mapped behaviour tests when the scenario also asserts calls/results. |
| `@source` | Parsed/type-aware owned-source check or a recorded source audit where retention/call-graph properties need review. |
| `@allocation` | Dedicated uninstrumented allocation test/probe on the native matrix. |
| `@benchmark` | Reproducible benchmark output and review against the stated algorithm/cost contract. |
| `@race` | Concurrent test under `go test -race`, with outcome assertions. |
| `@fuzz` | Deterministic property seeds plus bounded fuzz runs and retained regression corpus. |
| `@traceability` | Requirement/scenario/example/tag checks and document-copy synchronisation. |
| `@release` | Current-revision manifest, documentation/API migration checks and complete matrix results. |

A source or release Gherkin scenario is not pretended to be an end-user behaviour test. Its observable result is the build/audit/gate result. A mapped Go test may satisfy several scenarios only when every assertion and every outline row is actually executed; it must report those IDs explicitly.

### 8.3 Allocation measurement protocol

Build normal optimised binaries without race, coverage or debug instrumentation for performance gates. Stabilise test-runner/runtime bootstrap before measurement but never call the validator to warm an undocumented cache. Construct rules, allocate input storage, set up counters and prepare non-allocating callbacks before entering the measured region.

Use a stored `Rule` from a separate `validation_test` package rather than only a directly inlined literal. Observe the result so the compiler cannot erase validation; do not format errors, build fixtures or update an allocating/global concurrent sink inside the measured call. Assert the literal nil result after measurement.

The per-family matrix includes every catalogue constructor with a valid fixture, plus the compositions below:

| Family | Fixtures |
|---|---|
| Scalar | Signed/unsigned extremes, floats including permitted infinities, named types, strings, bytes, time. |
| Fields | 1 and 16 fields; nested depth 1, 4 and 16; pointer/value getters. |
| Presence | Absent optional pointers/nullable values, present required values including zero, nested pointers. |
| Slices | Lengths 0, 1, 8, 64, 1024; named slices; slices of structs; nested slices. |
| Maps | Sizes 0, 1, 8, 64, 1024 and 10000; string and typed integer keys; named maps; all values valid. |
| Membership | Set sizes 0, 1, 4, 32 and 1024; include only successful inputs in the success gate. Empty `OneOf` has no valid input and is checked on the failure side. |
| Uniqueness | Unique input lengths 0, 1, 8, 64 and 1024; original input unchanged. |
| Extensions | Non-allocating direct rule/method, typed comparator, Project, array slice from pointer-parent, last-alternative predicate success. |

For each required native compiler/platform pair, collect `AllocsPerRun` regression measurements, benchmarks with `-benchmem`, and isolated first-invocation and repeated-batch total allocation/byte evidence. `AllocsPerRun` includes an initial warm-up and reports an integral average, so it cannot by itself disprove first-call or rare allocations [S4].

For total probes, use a fresh isolated process, no parallel test activity, a preinitialised measurement harness and storage, and snapshots immediately around either one first invocation or a batch. Record both total allocated objects and total bytes, not only divided metrics. Use a blank control to identify harness/runtime noise; do not silently subtract an unexplained positive count. When a count is positive, use escape diagnostics/allocation traces or profiles to attribute it. Any library-attributable object/byte allocation fails the gate. An inconclusive noisy probe is investigated rather than labelled passing.

Include deliberately allocating controls: one allocates only on first use, another every hundredth call. The gate must reject both. This verifies the measurement system rather than only the validator.

Measure custom allocating callbacks separately; they do not meet the whole-composition guarantee. Report their cost honestly while checking the same core composition with non-allocating callbacks. This is not permission to exclude a built-in's own work by relabelling it as a callback.

### 8.4 Race, source, property and fuzz procedures

The race fixture uses one constructed validator, read-only prepared inputs and shared immutable error values across 32 goroutines × 100 calls. Save representative first-call results, validate more inputs, and confirm saved paths and causes did not change. Application callbacks used in this fixture must themselves be race-safe.

Source checks cover library-owned production packages and generated/build-tagged variants, not just the default build. They reject forbidden imports and unsafe directives, input boxing/type switches, hot-path iterator/reporting machinery, implicit contexts/I/O, mutable per-rule scratch and lazy membership construction. They must distinguish `T any` from `any(value)` and distinguish standard error-node assertions in reporting from value dispatch. Tools may use parsing, type information and reflection in tests; the runtime library may not. Review the explicit allowlist of named-type conversions at standard-library boundaries, and link each such call site to an allocation fixture.

Fuzz/property targets use bounded sizes (for example, at most 64 collection elements and 16 path segments per generated case). Their oracle computes expected occurrences independently, does not call the implementation's own flattening/helper routines, and preserves the seed/corpus for regression. Test path escaping and segment distinction, integer/float boundaries, duplicate earliest-index semantics, exhaustive counts, nested ordering, read-only inputs and map insertion permutations. Run fixed seed cases on every test invocation and at least 60 seconds per fuzz target in the release job. Resource exhaustion and adversarial cyclic application error graphs are not normal fuzz inputs for the well-formed-tree contract.

### 8.5 Suggested implementation verification commands

These commands apply to the implementation repository after the tests are bound. They are not evidence that the implementation has been run in this delivery.

```sh
go test ./...
go test -race ./...
go test -run '^TestAlloc' ./...
go test -run '^$' -bench . -benchmem -count=10 ./...
go test -gcflags='all=-m=2' ./...
go test -run '^$' -fuzz '^FuzzPaths$' -fuzztime=60s ./...
```

Run each fuzz target in its containing package; a concrete job replaces the final package argument with that package if more than one package contains a target. Keep allocation tests in an explicitly uninstrumented test build (for example, `!race` allocation-test files) so the race run does not falsely certify normal-build performance. They remain mandatory in the separate normal-build job. Compile-negative fixtures must assert failure, not match exact compiler wording.

### 8.6 Evidence identity

Each implementation acceptance report records: specification ID, hash of the standalone spec, hashes of all feature files, implementation commit SHA, exact Go version, GOOS/GOARCH, build flags, fixture/case IDs, executed test names, timestamps and result. A green result from another code or spec revision cannot close this gate. A skipped, unbound or missing case remains not executed; no “expected future pass” is substituted.

The bundle's `evidence/spec-check.json` concerns document structure only. The `implementation_evidence` field is explicitly `not_run`. The specification checker can optionally use an installed official Gherkin parser, but the supplied zero-dependency structural checker does not claim to validate step bindings or run the library.

## 9. Acceptance fixture and binding conventions

### 9.1 Feature-file conventions

Every scenario carries one unique `@AT-...` tag, one or more `@REQ-...` tags and one verification category. Feature IDs are stable file stems. Each outline row is a separate case identified by scenario ID plus Examples block and row ordinal; descriptions alone are not identifiers.

The examples use unambiguous named fixtures rather than a runtime type-erasing validator. Test step code selects predeclared typed Go fixtures. It must not interpret rule expressions from arbitrary text in the production library. Omitting a type argument or StringKeys in a Gherkin sentence is fixture shorthand; the binding supplies the exact typed form defined in section 4.

`none` in a code table means zero issues, not a code literally named “none”. `[]`, nil slice, empty map, zero `time.Time`, pointer to zero, NaN and infinities are distinct fixtures. NaN is constructed deliberately in tests; signed zero is preserved when specified. Unicode examples describe exact code points, not visual similarity. Go-quoted data-table cells use Gherkin escaping for backslashes and vertical bars [S2].

When a table says the ordered issues are “exactly” a sequence, assert count, order, segment kinds, codes and stated detail fields, with no additional issue allowed. String path assertions are supported by segment-kind assertions, not used instead of them. For identities A, B, E or S, use distinct immutable sentinel instances and compare with normal `errors.Is` or direct identity when the scenario requires it.

Counter fixtures use preallocated/stack counters and do not append call logs inside zero-allocation measurements. Source/compile scenarios may use richer harnesses outside the measured region. `panicRule` is a deliberate guard test; no production recovery is added.

### 9.2 How the suite becomes executable

The initial binding target is ordinary Go tests to preserve direct control over generic instantiations, allocation probes, compile-negative fixtures and race tests. Teams may additionally bind the `.feature` files through Cucumber/Godog; this is a test-only choice, not a runtime dependency. A binding manifest maps each scenario row to its concrete Go test/subtest or step-definition execution result.

Binding order is core/exhaustive semantics, errors/paths, configuration/presence, collection completeness, type families, extension logic, allocation, iteration and release gates. This is execution planning, not an alternate priority that makes some requirements optional.

The supplied traceability data links requirements to scenarios, not to unimplemented step functions. Do not generate placeholder tests that return success: an unbound scenario must remain visibly not executed until its assertions are implemented.

## 10. Implementation sequence and migration

### 10.1 Cohesive implementation increments

| Increment | Primary work | Gate before proceeding |
|---|---|---|
| P0: contracts and feasibility | Bind core/nil/all-error cases, source policy, representative allocation probes on Go 1.27. | Direct typed composition has passing feasibility evidence; no design substituted without measurement. |
| P1: immutable native errors | Standard unwrapping, failure-only paths, diagnostic codes/details, reporting and ownership tests. | Identities/causes/occurrences preserved and saved results immutable. |
| P2: one typed engine | Change scalar signatures and Field/Struct together; add conditions, Project, Check and presence guards. | Branch compiles without parallel engines; all applicable rules run. |
| P3: exhaustive collections | Named slices/maps, every offender, deterministic failed-map groups, scan uniqueness. | Completeness, ordering, non-mutation and success allocation gates pass. |
| P4: catalogue correctness | Numbers, Unicode/bytes, time, custom comparators, typed examples. | Entire declared catalogue compiles and passes boundary/allocation tests. |
| P5: reporting iteration | Internal walkIssues with safe snapshots and early stop; no hot-path coupling. | Traversal/format cases pass with no validation regression. |
| P6: release evidence | Native matrix, race/fuzz, docs, migration and current-revision manifest. | Every required acceptance case is bound, run and passing. |

P1 may use a simple temporary direct reporting traversal internally until P5, but the final API must meet the iterator requirements. Intermediate commits are not released as the completed refactor merely because a subset passes.

### 10.2 Migration map

| Reviewed API / pattern | Target |
|---|---|
| `Rule[T] func(T) *Error` | `Rule[T] func(T) error`; concrete nil normalisation for legacy adapters. |
| `Validate(...) Errors` | `Validate(...) error`; literal nil on success; `Issues(err)` for complete located occurrences. |
| Mutable Error fields / `Fatal` | Coded error + immutable location wrapper; no control flow inside returned errors. |
| `RuleStopOnError` | Removed. Explicit pointer/key/index/predicate applicability guards only. |
| Error-based `Or` / `RuleNot` | Predicate composition through `Check`; ordinary application errors never negated. |
| StructValidator / fieldValidator hierarchy | `Rule[Parent]` produced by `Field`; `Struct` is an All alias. |
| `StructField`, `SliceField`, `MapField` | Normal `Field` containing typed nested/collection rules. |
| `SliceRule`, `MapRule`, `MapEntryRule` | `Rule[S]`, `Rule[M]`, `Rule[Entry[K,V]]`. |
| `NumbersMin/Max/Between/...` | `Min/Max/Between/...` with numeric-only constraints. |
| `StringsRune...` | `Rune...`; explicit byte-length alternatives. |
| `StringsMatchesRegex` | Removed from core; application-supplied typed rule with its own costs. |
| `Slices...` names | `Slice...`; `SlicesForEach` becomes `Each`, `SlicesAtIndex` becomes `AtIndex`. |
| `Maps...` names | `Map...`; explicit KeyOrder for traversing/addressing keys. |
| `MapsValuesOneOf` / `MapsValuesNotOneOf` | `MapValues(order, OneOf(...))` / `MapValues(order, NotOneOf(...))`. |
| Automatic/non-explicit IsZero or Validate invocation | Explicit typed method/rule or `Check`; `TimeNotZero` for time.Time. |
| Dot-joined field strings | Structured segments and unambiguous canonical presentation. |

### 10.3 Nil-safe legacy examples

These are explicit migration patterns for application code, not a second runtime engine:

```go
func adaptLegacyPointer[T any](legacy func(T) *LegacyError) Rule[T] {
    return func(value T) error {
        err := legacy(value)
        if err == nil {
            return nil
        }
        return err
    }
}
```

For a legacy slice, check `len(errs) == 0` before converting it to `error`. For non-empty slices, adapt each concrete child without mutating it and preserve all occurrences and causes. Do not stringify the old errors or implicitly restore their fatal/first-error behaviour. The migration guide must supply a concrete compiled example for the actual legacy types present in the repository; the illustrative `LegacyError` name above is not a new target export.

The repository has published v1.0.0 through v1.0.5 tags. Release this incompatible API as v2 under `github.com/jacoelho/validation/v2`. The release notes must make the incompatible signature and semantic changes explicit.

## 11. Risks, evidence boundaries and references

### 11.1 Deliberate trade-offs

The zero-allocation requirement does not imply minimum CPU time. Scan uniqueness is quadratic. Exhaustive error reporting can allocate memory proportional to invalid input. Applications handling untrusted large data should apply an upstream input-size policy; a failing `SliceMaxLength` inside All does not authorise silently skipping Each.

Deterministic map diagnostics are not deterministic map execution. Mutable or effectful callbacks cannot use result ordering as an execution guarantee. Key encoders must not collide or rely on unstable external state.

Go compiler escape behaviour can change. The no-allocation promise is a tested implementation requirement on an explicit matrix, not a language theorem about all callback programs or future compiler versions. A failed performance gate must be investigated before release rather than reclassified as an acceptable small allocation.

Plain external error methods, malformed/cyclic error trees, callback panics, unsafe concurrent input mutation and typed-nil custom errors lie outside the supported callback contract. The library does not add reflection or broad panic recovery to simulate safety for those cases.

### 11.2 Delivery evidence

This repository contains the refactored library, specification prose, numbered requirements, feature files, traceability data and checkers. The earlier Go 1.23.2 feasibility spike is historical planning evidence only and is not Go 1.27 acceptance evidence. A scenario is accepted only when its binding executes and passes on the required matrix.

The local check reports state which specification and acceptance checks ran. Full official parser availability is reported separately. Native release evidence must be recorded before certification.

### 11.3 Sources

The user's agreed constraints and supplied refactor plan are the authority for design requirements. The external references below establish language, library and Gherkin protocol details, not that the proposed implementation exists. References were consulted on 29 September 2026.

| Ref | Primary reference | Use |
|---|---|---|
| S1 | Go `errors` documentation: `https://pkg.go.dev/errors` | Single/multi-Unwrap and normal error discovery. |
| S2 | Cucumber Gherkin reference: `https://cucumber.io/docs/gherkin/reference/` | Feature/scenario/outline/tag/table syntax. |
| S3 | Go `iter` documentation: `https://pkg.go.dev/iter` | Push iterator shape and early termination. |
| S4 | Go `testing.AllocsPerRun`: `https://pkg.go.dev/testing#AllocsPerRun` | Warm-up and integral averaging caveat. |
| S5 | Go language specification: `https://go.dev/ref/spec` | Generic constraints, equality, conversion and array semantics. |
| S6 | Go toolchains: `https://go.dev/doc/toolchain` | Minimum module version versus selected toolchain. |
| S7 | Go `unicode/utf8`: `https://pkg.go.dev/unicode/utf8` | Byte decoding, rune counting and encoding validity. |
| S8 | Go `math`: `https://pkg.go.dev/math` | Float special-value conventions used in tests. |
| S9 | Go `time`: `https://pkg.go.dev/time` | Chronological comparisons, monotonic semantics and IsZero. |
| P1 | Supplied `validation-refactor-plan.md` | Agreed design baseline; no fresh repository mutation. |

## Appendix A. Complete Gherkin acceptance suite

Each block below is reproduced exactly in its named `.feature` file in the bundle. The blocks are normative test specifications. They require implementation bindings before they are executable acceptance evidence.

### 01_core.feature

```gherkin
# language: en
@validation_refactor
Feature: Typed exhaustive validation
  Every independently configured rule occurrence is evaluated once; successful results are ordinary nil errors.

  @AT-CORE-001 @REQ-ARCH-001 @REQ-CORE-001 @REQ-ERR-002 @behaviour
  Scenario: A valid result remains nil across an error-returning boundary
    Given a preconstructed All rule whose 3 child rules all succeed
    When its result is returned through a function declared to return error
    Then the returned error interface is nil
    And Issues returns 0 occurrences
    And Format returns an empty string

  @AT-CORE-002 @REQ-CORE-002 @REQ-CORE-003 @REQ-CORE-004 @REQ-CORE-007 @behaviour
  Scenario: All failures at one field are retained
    Given the field "name" has NotEmpty and RuneMinLength(2) in that order
    And the input name is the empty string
    When the parent is validated once
    Then the getter is called once and each child rule is called once
    And the ordered issues are exactly:
      | path   | code            |
      | $.name | not_empty       |
      | $.name | rune_min_length |

  @AT-CORE-003 @REQ-CORE-002 @REQ-CORE-004 @behaviour
  Scenario: Independent fields continue after earlier failures
    Given fields "name", "age" and "city" are configured in that order
    And their child rules return codes "not_empty", "min" and "not_empty"
    When the parent is validated
    Then all 3 field getters are called
    And the ordered issues are exactly:
      | path   | code      |
      | $.name | not_empty |
      | $.age  | min       |
      | $.city | not_empty |

  @AT-CORE-004 @REQ-CORE-003 @REQ-CORE-007 @REQ-ERR-009 @behaviour
  Scenario: The same configured function is two occurrences
    Given the same rule returning sentinel S is configured twice in All
    When the value is validated once
    Then that function is called twice
    And Issues returns 2 occurrences referring to sentinel S
    And errors.Is of the result and S is true

  @AT-CORE-005 @REQ-CORE-005 @REQ-CORE-001 @behaviour
  Scenario Outline: Empty compositions have defined behaviour
    Given the empty composition <composition> and a prepared valid container
    When it is validated
    Then the result is nil
    And the getter call count is <getters>

    Examples:
      | composition                          | getters |
      | All()                                | 0       |
      | Struct()                             | 0       |
      | Field("name", getter)                | 1       |
      | Project(getter)                      | 1       |
      | Each() over 3 values                 | 0       |
      | MapEach(StringKeys()) over 3 entries | 0       |

  @AT-CORE-006 @REQ-CORE-006 @REQ-ARCH-001 @behaviour
  Scenario Outline: Parent conditions guard a field without type erasure
    Given a parent-aware <adapter> predicate returns <predicate>
    And its children are two failing Field rules
    And another failing Field is outside the conditional group
    When the parent is validated once
    Then the predicate is called once
    And the conditional child rule call count is <calls>
    And Issues returns <issues> occurrences

    Examples:
      | adapter | predicate | calls | issues |
      | When    | true      | 2     | 3      |
      | When    | false     | 0     | 1      |
      | Unless  | true      | 0     | 1      |
      | Unless  | false     | 2     | 3      |

  @AT-CORE-007 @REQ-CORE-008 @behaviour
  Scenario: Panics propagate without becoming validation failures
    Given the first rule panics with sentinel panic value P
    And a later rule records whether it is called
    When the input is validated under a test-only recover
    Then the recovered panic value is P
    And the later rule is not called
    And no aggregate result is returned

  @AT-CORE-008 @REQ-CORE-007 @REQ-CORE-002 @behaviour
  Scenario: No hidden failure cap
    Given Each has 2 failing child rules
    And the input contains 257 elements
    When the slice is validated
    Then Issues returns exactly 514 occurrences
    And each index from 0 through 256 appears twice in ascending order
    And both child rules were called 257 times

  @AT-CORE-009 @REQ-CORE-009 @REQ-TYPE-006 @behaviour
  Scenario: An empty value is not implicitly optional
    Given the scalar rule RuneMinLength(2)
    When the empty string is validated
    Then there is exactly 1 root issue with code "rune_min_length"
```

### 02_construction.feature

```gherkin
# language: en
@validation_refactor
Feature: Construction, ownership and malformed configuration
  Configuration is immutable after construction; programming errors are distinguished from invalid values.

  @AT-CONSTRUCTION-001 @REQ-CONF-001 @behaviour
  Scenario Outline: Reject malformed static configuration before returning a rule
    Given the invalid construction request <request>
    When the constructor is called under a test-only recover
    Then it panics with a ConfigurationError
    And no reusable rule is returned

    Examples:
      | request                                            |
      | All with a nil child Rule                          |
      | Field with an empty name                           |
      | Field with a nil getter                            |
      | Project with a nil getter                          |
      | When with a nil predicate                          |
      | RuneMinLength(-1)                                  |
      | SliceLengthBetween(3, 2)                           |
      | Between(5, 1)                                      |
      | Min(NaN)                                           |
      | Between(0, NaN)                                    |
      | AtIndex(-1)                                        |
      | MapEach with nil Less                              |
      | MapEach with nil Text                              |
      | Check with a nil factory                           |
      | MinBy with a nil comparator                        |
      | NewViolation with an empty code                    |
      | Max(NaN)                                           |
      | GreaterThan(NaN)                                   |
      | LessThan(NaN)                                      |
      | TimeBetween with its chronological bounds reversed |

  @AT-CONSTRUCTION-002 @REQ-CONF-002 @behaviour
  Scenario: Freeze rule-list storage
    Given caller slice R contains a rule returning code "original"
    And All is constructed from R
    When R[0] is replaced with a rule returning code "replacement"
    And the constructed All rule is evaluated
    Then its only issue has code "original"

  @AT-CONSTRUCTION-003 @REQ-CONF-002 @REQ-TYPE-007 @behaviour
  Scenario Outline: Freeze membership and byte configuration
    Given a rule constructed from <configuration>
    When the caller changes the original configuration storage to <replacement>
    Then validating <value> still uses the originally configured data

    Examples:
      | configuration                    | replacement  | value              |
      | OneOf from ["a", "b"]            | ["x", "y"]   | "a"                |
      | BytesHasPrefix from bytes [0x61] | bytes [0x7a] | bytes [0x61, 0x62] |

  @AT-CONSTRUCTION-004 @REQ-CONF-006 @behaviour
  Scenario: Do not execute value callbacks during construction
    Given counters on getters, child rules, predicates, factories and key renderers
    When all composition constructors are called with valid configuration
    Then every value callback counter remains zero
    And constructing BetweenBy invokes only its bounds comparator once

  @AT-CONSTRUCTION-005 @REQ-CONF-005 @REQ-CORE-008 @behaviour
  Scenario: Factory returning nil cannot turn a failed check into success
    Given Check has a predicate returning false and a factory returning literal nil
    When the value is validated under a test-only recover
    Then the predicate and factory were each called once
    And it panics with a ConfigurationError

  @AT-CONSTRUCTION-006 @REQ-ERR-007 @REQ-ARCH-002 @behaviour
  Scenario: A nil-error callback is not reflectively repaired
    Given a custom rule returns a non-nil error interface containing a typed-nil pointer
    When All invokes that rule without inspecting the returned error
    Then the result is not nil
    And the callback is documented as violating the custom-rule contract
    And the core does not perform reflection to normalise the error
```

### 03_errors.feature

```gherkin
# language: en
@validation_refactor
Feature: Go errors and exhaustive occurrence reporting
  Identity, causes and independent failures survive normal Go wrapping.

  @AT-ERRORS-001 @REQ-ERR-002 @REQ-ERR-006 @behaviour
  Scenario: A single root failure preserves identity
    Given All contains one failing rule returning sentinel S and two successful rules
    When the value is validated
    Then the returned error is the exact sentinel S
    And Issues returns one external root occurrence referring to S

  @AT-ERRORS-002 @REQ-ERR-001 @behaviour
  Scenario Outline: Standard traversal reaches children through multiple wrapper forms
    Given field "a" returns sentinel A and field "b" returns custom typed error B
    And their aggregate is surrounded by <wrapper>
    When standard error discovery is performed
    Then errors.Is finds A
    And errors.As finds the exact custom error B
    And no library multi-Unwrap contains a nil child

    Examples:
      | wrapper                                 |
      | no additional wrapper                   |
      | fmt.Errorf with %w                      |
      | errors.Join with independent sentinel C |
      | fmt.Errorf around errors.Join           |

  @AT-ERRORS-003 @REQ-ERR-005 @REQ-ERR-001 @REQ-ERR-004 @behaviour
  Scenario: Coded causes are discoverable but not counted twice
    Given field "amount" returns a coded violation "min" wrapping sentinel S
    When its parent is validated
    Then errors.Is of the result and S is true
    And Issues returns exactly one occurrence at "$.amount" with code "min"
    And that occurrence retains the coded error and its cause chain

  @AT-ERRORS-004 @REQ-ERR-006 @REQ-ERR-009 @behaviour
  Scenario: Generic joins produce one occurrence per independent terminal
    Given a field returns fmt.Errorf wrapping errors.Join(A, B)
    And A and B are uncoded sentinel errors
    When Issues traverses the validation result
    Then it returns 2 external occurrences at that field path in join order
    And A and B remain discoverable with errors.Is

  @AT-ERRORS-005 @REQ-ERR-003 @REQ-CONF-003 @behaviour
  Scenario: Shared errors cannot acquire mutable field state
    Given one reusable error object E is returned from fields "left" and "right"
    When the parent is validated twice
    Then E has exactly the state it had before both validations
    And each result contains separate "$.left" and "$.right" occurrences
    And inspecting the second result does not change the first

  @AT-ERRORS-006 @REQ-ERR-003 @behaviour
  Scenario: Modifying an exposed aggregate slice does not rewrite the error
    Given an aggregate with independent sentinels A and B
    When a caller obtains its multi-Unwrap slice and replaces the first element
    Then a fresh multi-Unwrap still contains A followed by B
    And errors.Is still finds A and B in the original aggregate

  @AT-ERRORS-007 @REQ-ERR-008 @REQ-PATH-004 @behaviour
  Scenario: Safe formatting does not reveal rejected content or call external formatters
    Given field "password" returns an external error whose Error method panics
    And another rule returns a coded violation "not_empty" with a secret cause message
    When Format is called on the combined result
    Then it returns "$.password: external; $: not_empty"
    And no external Error method is called
    And neither the secret rejected value nor the cause message occurs in the text

  @AT-ERRORS-008 @REQ-ERR-006 @REQ-EXT-004 @REQ-CORE-002 @behaviour
  Scenario: Unclassified operational errors do not suppress later rules
    Given the first rule returns an ordinary application error E
    And the second and third rules return coded violations "min" and "max"
    When the value is validated
    Then all 3 rules are called once
    And the issue codes are "external", "min", "max" in that order
    And errors.Is finds E

  @AT-ERRORS-009 @REQ-ERR-009 @source
  Scenario: Malformed external error trees are outside the inspection guarantee
    Given the extension contract documentation and the library-owned error tests
    When error-tree conformance is reviewed
    Then all library-generated graphs are finite and acyclic with non-nil children
    And cyclic or nil-child external graphs are explicitly unsupported
    And repeated sentinel identity in an acyclic graph remains supported

  @AT-ERRORS-010 @REQ-ERR-004 @REQ-TYPE-005 @REQ-SLICE-001 @REQ-MAP-001 @behaviour
  Scenario Outline: Length failures expose typed units and limits
    Given the failing rule <rule> validates <input>
    When errors.As finds its LengthError
    Then Actual is <actual> and Minimum is <minimum>
    And Maximum is <maximum> with the documented presence flag
    And Unit is <unit>

    Examples:
      | rule                   | input                 | actual | minimum | maximum   | unit     |
      | RuneMinLength(2)       | the empty string      | 0      | 2       | absent    | runes    |
      | ByteMaxLength(1)       | precomposed U+00E9    | 2      | 0       | 1 present | bytes    |
      | SliceLength(2)         | a slice of 3 elements | 3      | 2       | 2 present | elements |
      | MapLengthBetween(1, 2) | a nil map             | 0      | 1       | 2 present | entries  |

  @AT-ERRORS-011 @REQ-ERR-004 @REQ-TYPE-002 @behaviour
  Scenario: Bounds diagnostics retain the exact configured type
    Given Max[uint64](9007199254740992) validates uint64 value 9007199254740993
    When errors.As finds BoundsError[uint64]
    Then its upper bound is exactly 9007199254740992 and is inclusive and present
    And its lower bound is absent
    And no rejected input is required in the diagnostic payload
```

### 04_paths.feature

```gherkin
# language: en
@validation_refactor
Feature: Structured paths and safe reporting
  Paths identify field, key and index boundaries without mutating child errors.

  @AT-PATHS-001 @REQ-PATH-001 @REQ-PATH-002 @REQ-CORE-004 @behaviour
  Scenario: Compose nested field and slice locations
    Given field "addresses" contains Each of an Address rule
    And only element 2 fails NotEmpty on its field "city"
    When the parent is validated
    Then the issue segments are Field("addresses"), Index(2), Field("city")
    And FormatPath returns "$.addresses[2].city"

  @AT-PATHS-002 @REQ-PATH-001 @REQ-PATH-004 @behaviour
  Scenario Outline: Distinguish literal fields, keys and indices
    Given the structured path <segments>
    When FormatPath renders it
    Then the output is <rendering>

    Examples:
      | segments               | rendering |
      | empty path             | $         |
      | Field("name")          | $.name    |
      | Field("a.b")           | $.["a.b"] |
      | Field("a"), Field("b") | $.a.b     |
      | Key("a.b")             | $["a.b"]  |
      | Index(2)               | $[2]      |
      | Key("2")               | $["2"]    |
      | Field("2")             | $.["2"]   |
      | Key("")                | $[""]     |

  @AT-PATHS-003 @REQ-PATH-004 @behaviour
  Scenario: Control characters are escaped without losing path kind
    Given a key contains quote, backslash, newline and the invalid byte 0xff
    When FormatPath renders a Key segment for that key
    Then the quoted form uses Go string-literal escapes for those bytes
    And the output contains no raw newline
    And its segment remains a Key rather than Field or Index

  @AT-PATHS-004 @REQ-PATH-003 @REQ-ERR-003 @REQ-ITER-004 @behaviour
  Scenario: Issue paths are independent snapshots
    Given Issues returns occurrences at "$.items[0]" and "$.items[1]"
    When the caller changes the first returned path and truncates the returned list
    Then the second saved issue path remains "$.items[1]"
    And a fresh Issues call returns both original paths unchanged

  @AT-PATHS-005 @REQ-PATH-002 @REQ-EXT-002 @REQ-PTR-002 @behaviour
  Scenario: Projection and presence wrappers add no synthetic location
    Given field "address" applies RequiredPtr to Project(identity) of a city Field rule
    And the non-nil address has an empty city
    When the parent is validated
    Then the only issue path is "$.address.city"
    And no pointer, projection or synthetic root segment is inserted

  @AT-PATHS-006 @REQ-PATH-004 @REQ-CONF-001 @behaviour
  Scenario Outline: Malformed caller-created segments are not rendered ambiguously
    Given a caller-created path containing <segment>
    When FormatPath is called under a test-only recover
    Then it panics with a ConfigurationError

    Examples:
      | segment                            |
      | an unknown SegmentKind             |
      | an Index segment with value -1     |
      | a Field segment with an empty name |
```

### 05_presence.feature

```gherkin
# language: en
@validation_refactor
Feature: Pointers, nullable values and guarded applicability
  Presence is separate from zero values; only unsafe child operations are skipped.

  @AT-PRESENCE-001 @REQ-PTR-001 @REQ-CORE-009 @behaviour
  Scenario Outline: Presence and zero are independent
    Given <adapter> contains Min(0)
    When the pointer value is <input>
    Then the result has <count> issues with codes <codes>
    And the child rule call count is <calls>

    Examples:
      | adapter     | input         | count | codes    | calls |
      | OptionalPtr | nil           | 0     | none     | 0     |
      | RequiredPtr | nil           | 1     | required | 0     |
      | OptionalPtr | pointer to 0  | 0     | none     | 1     |
      | RequiredPtr | pointer to 0  | 0     | none     | 1     |
      | RequiredPtr | pointer to -1 | 1     | min      | 1     |

  @AT-PRESENCE-002 @REQ-PTR-001 @REQ-PTR-002 @REQ-CORE-002 @behaviour
  Scenario: A missing pointer skips only its own children
    Given field "address" is a nil pointer guarded by RequiredPtr
    And its child would panic if evaluated
    And field "age" independently fails Min(0)
    When the parent is validated
    Then no panic occurs
    And the ordered issues are exactly:
      | path      | code     |
      | $.address | required |
      | $.age     | min      |

  @AT-PRESENCE-003 @REQ-PTR-002 @behaviour
  Scenario: Nested pointer layers remain guarded
    Given RequiredPtr contains RequiredPtr containing Min(0)
    When the outer pointer is non-nil and the inner pointer is nil
    Then one root required issue is returned
    And Min(0) is never called

  @AT-PRESENCE-004 @REQ-PTR-003 @REQ-PTR-004 @behaviour
  Scenario Outline: Nullable projection supplies presence explicitly
    Given <adapter> has a typed getter returning value <value> and present <present>
    And its child rule is Min(0)
    When the nullable object is validated
    Then the getter is called once
    And there are <count> issues with codes <codes>
    And the child rule call count is <calls>

    Examples:
      | adapter       | value | present | count | codes    | calls |
      | OptionalValue | -1    | false   | 0     | none     | 0     |
      | RequiredValue | 0     | false   | 1     | required | 0     |
      | RequiredValue | 0     | true    | 0     | none     | 1     |
      | OptionalValue | -1    | true    | 1     | min      | 1     |

  @AT-PRESENCE-005 @REQ-PTR-004 @REQ-EXT-002 @compile
  Scenario: Absent, null and value states belong to the input model
    Given an application type explicitly distinguishes omitted, null and value states
    And typed conditions guard a RequiredValue rule using those states
    When each state is validated
    Then behaviour follows the supplied discriminator and configured conditions
    And the core does not inspect tags or discover driver.Valuer

  @AT-PRESENCE-006 @REQ-CORE-005 @REQ-PTR-001 @REQ-PTR-003 @REQ-MAP-001 @REQ-SLICE-003 @behaviour
  Scenario Outline: Empty child groups do not disable structural guards
    Given the guarded rule <rule> has no value rules
    When it validates <input>
    Then the issue codes are <codes>

    Examples:
      | rule                  | input                            | codes              |
      | RequiredPtr()         | nil                              | required           |
      | OptionalPtr()         | nil                              | none               |
      | RequiredValue(getter) | a discriminator reporting absent | required           |
      | MapRequiredKey("x")   | an empty map                     | key_required       |
      | MapOptionalKey("x")   | an empty map                     | none               |
      | AtIndex(2)            | a slice of length 1              | index_out_of_range |
```

### 06_comparable.feature

```gherkin
# language: en
@validation_refactor
Feature: Comparable values and sign-independent presence
  Equality uses the original value type without boxing or universal emptiness inference.

  @AT-COMPARABLE-001 @REQ-TYPE-001 @REQ-TYPE-004 @behaviour
  Scenario Outline: Comparable primitives have exact semantics
    Given the comparable rule <rule>
    When it validates <value>
    Then the issue codes are <codes>

    Examples:
      | rule                 | value | codes      |
      | Equal(3)             | 3     | none       |
      | Equal(3)             | 4     | equal      |
      | NotEqual(3)          | 3     | not_equal  |
      | Zero[int]()          | 0     | none       |
      | Zero[int]()          | 1     | zero       |
      | NotZero[int]()       | 0     | not_zero   |
      | Equal(false)         | false | none       |
      | NotZero[bool]()      | false | not_zero   |
      | OneOf("a", "b", "b") | "b"   | none       |
      | OneOf[string]()      | "a"   | one_of     |
      | NotOneOf[string]()   | "a"   | none       |
      | NotOneOf("a", "b")   | "b"   | not_one_of |

  @AT-COMPARABLE-002 @REQ-TYPE-011 @REQ-TYPE-001 @behaviour
  Scenario Outline: IEEE equality remains explicit in comparable rules
    Given the comparable rule <rule>
    When it validates <value>
    Then the issue codes are <codes>

    Examples:
      | rule          | value | codes  |
      | Equal(NaN)    | NaN   | equal  |
      | OneOf(NaN)    | NaN   | one_of |
      | NotOneOf(NaN) | NaN   | none   |
      | Equal(+0.0)   | -0.0  | none   |

  @AT-COMPARABLE-003 @REQ-TYPE-011 @REQ-ARCH-003 @source
  Scenario: Interface comparability is not falsely advertised as a static guarantee
    Given the documented domain for comparable built-ins
    When the compile and source contracts are reviewed
    Then concrete safely comparable named values are supported
    And potentially panicking interface-bearing equality is outside the guarantee
    And no reflect-based safety probe is added
    And documentation does not claim Equal[any] is universally rejected by the compiler
```

### 07_numbers.feature

```gherkin
# language: en
@validation_refactor
Feature: Exact numeric and custom ordered constraints
  Bounds preserve integer precision and use explicit NaN, infinity and signed-zero policies.

  @AT-NUMBERS-001 @REQ-TYPE-002 @behaviour
  Scenario Outline: Bounds include or exclude their boundary deliberately
    Given the numeric rule <rule>
    When it validates <value>
    Then the issue codes are <codes>

    Examples:
      | rule           | value | codes        |
      | Min(5)         | 5     | none         |
      | Min(5)         | 4     | min          |
      | Max(5)         | 5     | none         |
      | Max(5)         | 6     | max          |
      | GreaterThan(5) | 5     | greater_than |
      | GreaterThan(5) | 6     | none         |
      | LessThan(5)    | 5     | less_than    |
      | LessThan(5)    | 4     | none         |
      | Between(2, 5)  | 2     | none         |
      | Between(2, 5)  | 5     | none         |
      | Between(2, 5)  | 6     | between      |
      | Between(5, 5)  | 5     | none         |

  @AT-NUMBERS-002 @REQ-TYPE-002 @REQ-ARCH-003 @behaviour
  Scenario Outline: Integer extremes do not pass through floating point
    Given the rule <rule> constructed with typed integer bounds
    When it validates <value>
    Then the issue codes are <codes>

    Examples:
      | rule                                                      | value                | codes |
      | Min[uint64](9007199254740993)                             | 9007199254740992     | min   |
      | Min[uint64](18446744073709551615)                         | 18446744073709551615 | none  |
      | Max[int64](-9223372036854775808)                          | -9223372036854775808 | none  |
      | Between[int64](-9223372036854775808, 9223372036854775807) | 9223372036854775807  | none  |

  @AT-NUMBERS-003 @REQ-TYPE-003 @behaviour
  Scenario Outline: All bound and sign rules reject NaN input
    Given the float rule <rule>
    When it validates NaN
    Then exactly one issue with code <code> is returned

    Examples:
      | rule                   | code         |
      | Min(0.0)               | min          |
      | Max(1.0)               | max          |
      | Between(0.0, 1.0)      | between      |
      | GreaterThan(0.0)       | greater_than |
      | LessThan(0.0)          | less_than    |
      | Positive[float64]()    | positive     |
      | NonNegative[float64]() | non_negative |
      | Negative[float64]()    | negative     |
      | NonPositive[float64]() | non_positive |

  @AT-NUMBERS-004 @REQ-TYPE-003 @REQ-TYPE-004 @behaviour
  Scenario Outline: Float-specific policies and signed zeros
    Given the float rule <rule>
    When it validates <value>
    Then the issue codes are <codes>

    Examples:
      | rule                | value | codes    |
      | NotNaN()            | NaN   | not_nan  |
      | NotNaN()            | +Inf  | none     |
      | Finite()            | NaN   | finite   |
      | Finite()            | +Inf  | finite   |
      | Finite()            | -Inf  | finite   |
      | Finite()            | -0.0  | none     |
      | Positive()          | -0.0  | positive |
      | Negative()          | +0.0  | negative |
      | NonNegative()       | -0.0  | none     |
      | NonPositive()       | +0.0  | none     |
      | Min(0.0)            | +Inf  | none     |
      | Max(0.0)            | -Inf  | none     |
      | Between(-Inf, +Inf) | +Inf  | none     |
      | Between(-Inf, +Inf) | NaN   | between  |

  @AT-NUMBERS-005 @REQ-TYPE-002 @REQ-QUAL-004 @compile
  Scenario Outline: Numeric constraints reject non-numeric types at compile time
    Given an isolated compile fixture instantiating <expression>
    When the fixture is compiled
    Then compilation fails because the concrete type is outside the numeric constraint
    And the assertion does not depend on compiler diagnostic wording

    Examples:
      | expression                |
      | Min[string]("a")          |
      | Positive[string]()        |
      | Between[complex128](0, 1) |

  @AT-NUMBERS-006 @REQ-TYPE-010 @REQ-EXT-001 @behaviour
  Scenario: A custom amount comparator does not require casts
    Given Amount is a custom struct with exact integer minor units
    And its comparator returns -7, 0 or 9 according to order
    And BetweenBy is constructed with Amount bounds 100 and 200
    When Amount values 99, 100, 200 and 201 are validated
    Then their issue codes are "between", none, none and "between"
    And no input value is converted to float64 or any
    And each value is compared with both bounds exactly once after one construction-time bounds comparison
```

### 08_strings_bytes.feature

```gherkin
# language: en
@validation_refactor
Feature: Strings, Unicode and bytes
  Byte counts, rune counts and UTF-8 validity are distinct constraints.

  @AT-STRINGS_BYTES-001 @REQ-TYPE-005 @behaviour
  Scenario Outline: Code points are not bytes or grapheme clusters
    Given the input string <text> has the literal Unicode sequence specified
    When byte and rune length rules evaluate the input
    Then its byte length is <bytes> and rune count is <runes>

    Examples:
      | text                      | bytes | runes |
      | ""                        | 0     | 0     |
      | "A"                       | 1     | 1     |
      | precomposed U+00E9        | 2     | 1     |
      | U+0065 followed by U+0301 | 3     | 2     |
      | U+1F642                   | 4     | 1     |

  @AT-STRINGS_BYTES-002 @REQ-TYPE-005 @REQ-CORE-002 @behaviour
  Scenario: Invalid UTF-8 is checked independently of rune length
    Given a string containing exactly bytes 0xff and 0xfe
    And All contains UTF8 and RuneLength(2)
    When the string is validated
    Then RuneLength(2) succeeds
    And the only issue code is "utf8"
    And both rules execute

  @AT-STRINGS_BYTES-003 @REQ-TYPE-005 @behaviour
  Scenario Outline: String lengths have explicit boundary outcomes
    Given the string rule <rule>
    When it validates <value>
    Then the issue codes are <codes>

    Examples:
      | rule                    | value                     | codes           |
      | ByteLength(2)           | precomposed U+00E9        | none            |
      | ByteMinLength(3)        | precomposed U+00E9        | byte_min_length |
      | ByteMaxLength(1)        | precomposed U+00E9        | byte_max_length |
      | ByteLengthBetween(2, 2) | precomposed U+00E9        | none            |
      | RuneLength(2)           | precomposed U+00E9        | rune_length     |
      | RuneMinLength(1)        | precomposed U+00E9        | none            |
      | RuneMaxLength(1)        | U+0065 followed by U+0301 | rune_max_length |
      | RuneLengthBetween(2, 2) | U+0065 followed by U+0301 | none            |

  @AT-STRINGS_BYTES-004 @REQ-TYPE-006 @REQ-CORE-009 @behaviour
  Scenario Outline: No trimming, case folding or implicit empty-string skip
    Given the string rule <rule>
    When it validates <value>
    Then the issue codes are <codes>

    Examples:
      | rule            | value           | codes     |
      | NotEmpty()      | one ASCII space | none      |
      | Contains("A")   | "abc"           | contains  |
      | Contains("")    | ""              | none      |
      | HasPrefix("")   | ""              | none      |
      | HasSuffix("")   | ""              | none      |
      | HasPrefix("ab") | "abc"           | none      |
      | HasSuffix("bc") | "abc"           | none      |
      | NotEmpty()      | ""              | not_empty |

  @AT-STRINGS_BYTES-005 @REQ-TYPE-007 @REQ-SLICE-007 @behaviour
  Scenario Outline: Dedicated byte rules preserve byte semantics
    Given the byte-slice rule <rule>
    When it validates <value>
    Then the issue codes are <codes>
    And the input backing array is unchanged

    Examples:
      | rule                   | value        | codes           |
      | BytesEqual(empty)      | nil          | none            |
      | BytesNotEmpty()        | nil          | not_empty       |
      | BytesLength(2)         | [0xff, 0xfe] | none            |
      | BytesUTF8()            | [0xff, 0xfe] | utf8            |
      | BytesContains([0xff])  | [0x00, 0xff] | none            |
      | BytesHasPrefix([0x00]) | [0x00, 0xff] | none            |
      | BytesHasSuffix([0xff]) | [0x00, 0xff] | none            |
      | BytesMaxLength(1)      | [0x00, 0xff] | byte_max_length |

  @AT-STRINGS_BYTES-006 @REQ-TYPE-007 @REQ-ARCH-003 @source
  Scenario: Dedicated byte implementations do not copy through strings
    Given all owned runtime byte validation functions
    When the type-aware source gate inspects their call paths
    Then there is no byte-slice-to-string conversion for value validation
    And named byte slices compile against the dedicated byte constructors
```

### 09_time.feature

```gherkin
# language: en
@validation_refactor
Feature: Time constraints without implicit clocks
  Chronological equality and zero-time checks are explicit and reproducible.

  @AT-TIME-001 @REQ-TYPE-008 @behaviour
  Scenario Outline: Equal instants distinguish inclusive and exclusive bounds
    Given boundary B is 2026-09-29T08:00:00Z
    And value V is 2026-09-29T09:00:00+01:00
    When <rule> validates V against B
    Then the issue codes are <codes>

    Examples:
      | rule              | codes  |
      | TimeBefore        | before |
      | TimeBeforeOrEqual | none   |
      | TimeAfter         | after  |
      | TimeAfterOrEqual  | none   |

  @AT-TIME-002 @REQ-TYPE-008 @REQ-CORE-009 @behaviour
  Scenario Outline: Time range is inclusive and zero is not automatically absent
    Given TimeBetween has boundaries 2026-09-01T00:00:00Z and 2026-10-01T00:00:00Z
    When it validates <value>
    Then the issue codes are <codes>

    Examples:
      | value                    | codes   |
      | 2026-09-01T00:00:00Z     | none    |
      | 2026-10-01T00:00:00Z     | none    |
      | 2026-10-01T00:00:01Z     | between |
      | the zero time.Time value | between |

  @AT-TIME-003 @REQ-TYPE-008 @behaviour
  Scenario: Explicit zero-time rule
    Given TimeNotZero
    When it validates the zero time.Time value
    Then one root issue with code "not_zero" is returned
    And the rule does not query the current clock
```

### 10_slices.feature

```gherkin
# language: en
@validation_refactor
Feature: Exhaustive, non-mutating slice validation
  Collection-level errors coexist with all applicable element-level errors.

  @AT-SLICES-001 @REQ-SLICE-001 @REQ-SLICE-002 @REQ-CORE-002 @behaviour
  Scenario: Container length failure does not suppress element failures
    Given All contains SliceMaxLength(1) followed by Each(NotEmpty, RuneMinLength(2))
    And the input is ["", ""]
    When the slice is validated
    Then the ordered issues are exactly:
      | path | code            |
      | $    | max_length      |
      | $[0] | not_empty       |
      | $[0] | rune_min_length |
      | $[1] | not_empty       |
      | $[1] | rune_min_length |

  @AT-SLICES-002 @REQ-SLICE-001 @REQ-CORE-009 @behaviour
  Scenario Outline: Nil and empty collections still satisfy explicit length semantics
    Given the slice rule <rule>
    When the input is <input>
    Then the issue codes are <codes>

    Examples:
      | rule                     | input | codes          |
      | SliceLength(0)           | nil   | none           |
      | SliceMinLength(1)        | nil   | min_length     |
      | SliceMaxLength(0)        | []    | none           |
      | SliceLengthBetween(1, 2) | []    | length_between |
      | Each(NotEmpty)           | nil   | none           |

  @AT-SLICES-003 @REQ-SLICE-003 @REQ-CORE-002 @behaviour
  Scenario: Out-of-range index guards children without ending independent validation
    Given All contains AtIndex(3, panicRule) and SliceMinLength(2)
    And the input has 1 element
    When the slice is validated
    Then panicRule is never called
    And the ordered issues are exactly:
      | path | code               |
      | $[3] | index_out_of_range |
      | $    | min_length         |

  @AT-SLICES-004 @REQ-SLICE-003 @REQ-CORE-003 @behaviour
  Scenario: A valid addressed element runs every rule
    Given AtIndex(1) contains NotEmpty and RuneMinLength(2)
    When ["valid", ""] is validated
    Then 2 issues are returned at "$[1]" in declaration order
    And both child rules are called once

  @AT-SLICES-005 @REQ-SLICE-004 @behaviour
  Scenario Outline: Membership reports every offending occurrence
    Given <rule>
    When it validates <values>
    Then the failing indices are exactly <indices>
    And each issue has code <code>

    Examples:
      | rule                 | values               | indices | code       |
      | SliceOneOf("a", "b") | ["x", "a", "y", "x"] | 0, 2, 3 | one_of     |
      | SliceNotOneOf("x")   | ["x", "a", "x"]      | 0, 2    | not_one_of |
      | SliceOneOf[string]() | ["a", "b"]           | 0, 1    | one_of     |

  @AT-SLICES-006 @REQ-SLICE-004 @behaviour
  Scenario: Containment is one container constraint
    Given SliceContains("x")
    When ["a", "b", "c"] is validated
    Then there is one root issue with code "contains"
    And there are no element-level issues

  @AT-SLICES-007 @REQ-SLICE-005 @REQ-SLICE-006 @behaviour
  Scenario: Duplicates report each later occurrence and its earliest match
    Given SliceUnique[string]()
    When ["a", "b", "a", "a", "b"] is validated
    Then the ordered duplicate issues are exactly:
      | path | code   | first_index |
      | $[2] | unique | 0           |
      | $[3] | unique | 0           |
      | $[4] | unique | 1           |
    And the input sequence is unchanged

  @AT-SLICES-008 @REQ-SLICE-005 @REQ-TYPE-011 @behaviour
  Scenario: Uniqueness follows declared equality rather than a NaN identity convention
    Given SliceUnique[float64]()
    When [NaN, NaN, +0.0, -0.0] is validated
    Then exactly one unique issue is returned at "$[3]"
    And its first_index is 2

  @AT-SLICES-009 @REQ-SLICE-002 @REQ-TYPE-009 @REQ-ARCH-001 @compile
  Scenario: Nested named slices preserve types and locations
    Given Rows is a named slice of Cells and Cells is a named slice of string
    And a typed getter returns Rows without converting it to an unnamed slice
    And each cell is validated by NotEmpty
    When the only empty cell is at row 1 column 2
    Then the only issue path ends with "[1][2]"
    And the fixture compiles without value-type assertions

  @AT-SLICES-010 @REQ-SLICE-007 @source
  Scenario: Slice and byte storage remains read-only
    Given a prepared slice with spare capacity and sentinel bytes beyond its length
    When all applicable built-in slice and byte rules are evaluated
    Then its length, capacity, visible contents and spare-capacity sentinel bytes are unchanged
    And a built-in error does not retain the entire input backing array
```

### 11_maps.feature

```gherkin
# language: en
@validation_refactor
Feature: Typed map validation with deterministic diagnostics
  Validation order may vary; complete result ordering does not depend on it.

  @AT-MAPS-001 @REQ-MAP-001 @REQ-CORE-009 @behaviour
  Scenario Outline: Map absence differs from a present zero value
    Given <rule> with child Min(1)
    When the map is <input>
    Then the issue codes are <codes>
    And the child rule call count is <calls>

    Examples:
      | rule                | input    | codes        | calls |
      | MapRequiredKey("x") | nil      | key_required | 0     |
      | MapRequiredKey("x") | {"x": 0} | min          | 1     |
      | MapOptionalKey("x") | {}       | none         | 0     |
      | MapOptionalKey("x") | {"x": 0} | min          | 1     |
      | MapRequiredKey("x") | {"x": 1} | none         | 1     |

  @AT-MAPS-002 @REQ-MAP-001 @behaviour
  Scenario: Nil map participates in size validation
    Given All contains MapLength(0) and MapMinLength(1)
    When it validates a nil map
    Then the only issue code is "min_length"
    And both size rules were evaluated

  @AT-MAPS-003 @REQ-MAP-002 @REQ-CORE-004 @behaviour
  Scenario: Report every key and value failure
    Given MapEach uses StringKeys and two rules: key NotEmpty then value Min(1)
    When the map contains entries {"b": 0, "": 0, "a": 1}
    Then the ordered issues are exactly:
      | path  | code      |
      | $[""] | not_empty |
      | $[""] | min       |
      | $["b"]| min       |
    And both rules were called once for each entry

  @AT-MAPS-004 @REQ-MAP-002 @REQ-CORE-002 @behaviour
  Scenario: Forbidden keys are all reported without suppressing value checks
    Given All contains MapKeysOneOf("a") followed by MapValues(Min(1))
    When it validates {"z": 0, "y": 0, "a": 1}
    Then the ordered issues are exactly:
      | path   | code   |
      | $["y"] | one_of |
      | $["z"] | one_of |
      | $["y"] | min    |
      | $["z"] | min    |

  @AT-MAPS-005 @REQ-MAP-003 @REQ-PATH-005 @REQ-CONF-004 @REQ-PERF-001 @allocation
  Scenario: Valid maps allocate no key list or rendering work
    Given MapValues has a KeyOrder with counting Less and Text callbacks
    And all 10000 prepared map values pass a non-allocating rule
    When the map is validated
    Then the result is nil
    And Less and Text are each called zero times
    And successful validation allocates 0 objects and 0 bytes

  @AT-MAPS-006 @REQ-MAP-003 @REQ-PATH-005 @REQ-MAP-005 @behaviour
  Scenario: Sort only failed entry groups and render each group once
    Given 100 map entries and only keys "z", "a" and "m" fail two child rules
    And KeyOrder callbacks record their arguments
    When the map is validated
    Then Less receives only keys from {"a", "m", "z"}
    And Text is called once for each of those 3 keys
    And 6 issues appear in key order "a", "m", "z" with child declaration order
    And no value rule is replayed

  @AT-MAPS-007 @REQ-MAP-003 @REQ-MAP-005 @behaviour
  Scenario: Insertion order does not affect issue order
    Given all 6 insertion permutations of 3 failing string-keyed entries
    When MapValues validates each resulting map 20 times
    Then every Issues result has the same paths and codes in lexical key order
    And the test makes no assertion about callback execution order

  @AT-MAPS-008 @REQ-MAP-004 @REQ-TYPE-009 @compile
  Scenario: Typed non-string keys use an explicit ordering and rendering policy
    Given a named map with int keys and a KeyOrder using numeric less-than and strconv.Itoa
    When keys 10 and 2 both fail
    Then the issue paths are "$["2"]" followed by "$["10"]"
    And no fmt-based or any-based key dispatch is needed

  @AT-MAPS-009 @REQ-MAP-004 @REQ-PATH-001 @REQ-PATH-005 @behaviour
  Scenario: Literal key punctuation is preserved
    Given StringKeys and failing keys "a.b", "a[0]" and ""
    When the map is validated
    Then the 3 keys remain distinct Key segments
    And no key text becomes a nested Field or Index path

  @AT-MAPS-010 @REQ-MAP-006 @REQ-CONF-003 @source
  Scenario: Map validation is read-only
    Given a read-only input map and a reusable validator
    When successful and failing validations are repeated
    Then map membership and every entry value remain unchanged
    And the validator retains no per-input error or map state

  @AT-MAPS-011 @REQ-MAP-001 @REQ-PATH-005 @REQ-CONF-004 @behaviour
  Scenario: Missing required keys render only their own failure
    Given All contains MapRequiredKey("x") and MapOptionalKey("y") with counting key renderers
    And both keys are absent
    When the empty map is validated
    Then exactly one key_required issue has path "$["x"]"
    And the renderer for "x" is called once and the renderer for "y" is never called
    And no child value rule is invoked
```

### 12_extensions.feature

```gherkin
# language: en
@validation_refactor
Feature: Typed extension without a format framework
  Custom checks remain ordinary functions with explicit effects and costs.

  @AT-EXTENSIONS-001 @REQ-EXT-001 @REQ-ARCH-001 @compile
  Scenario: Use a typed validation method directly
    Given Money has a Validate() error method on *Money
    When its method expression is supplied as Rule[*Money] to Field
    Then the fixture compiles without a registry or adapter interface
    And the method is called once per applicable field validation
    And its returned cause remains discoverable

  @AT-EXTENSIONS-002 @REQ-EXT-002 @REQ-CORE-003 @behaviour
  Scenario: Project exposes a custom value once
    Given Project extracts integer minor units from a custom amount type
    And two child numeric rules both fail
    When the amount is validated
    Then the getter is called once
    And both failures appear at the original path without an extra segment

  @AT-EXTENSIONS-003 @REQ-EXT-003 @REQ-CONF-005 @REQ-CONF-004 @REQ-PERF-001 @allocation
  Scenario: A successful alternative does not allocate discarded diagnostics
    Given Check has a predicate evaluating A(value) OR B(value) OR C(value)
    And A and B return false and C returns true without allocating
    And the failure factory would allocate if called
    When the prepared value is validated
    Then A, B and C are each called once
    And the failure factory is not called
    And the result is nil with 0 objects and 0 bytes allocated

  @AT-EXTENSIONS-004 @REQ-EXT-003 @REQ-CONF-005 @behaviour
  Scenario: A failed compound predicate has its own diagnostic
    Given Check has a predicate "has email OR has phone"
    And neither property is present
    And its failure factory returns code "contact_required"
    When the input is validated
    Then the factory is called once
    And exactly one root issue has code "contact_required"
    And no branch predicate is replayed to construct the diagnostic

  @AT-EXTENSIONS-005 @REQ-EXT-003 @behaviour
  Scenario: Predicate negation does not require a failing error-returning rule
    Given Check uses NOT isForbidden(value)
    And isForbidden returns false
    When the prepared input is validated
    Then the result is nil
    And no failure object or error-returning alternative is constructed

  @AT-EXTENSIONS-006 @REQ-EXT-004 @REQ-ERR-006 @REQ-PTR-004 @behaviour
  Scenario: Core does not swallow external failures as absence
    Given an application rule returns sentinel extraction error E
    And an independent required-field rule fails
    When All validates the input
    Then both errors are retained
    And E is not translated into nil, optional absence or a failed Boolean predicate
```

### 13_iteration.feature

```gherkin
# language: en
@validation_refactor
Feature: Failure-side issue iteration
  Reporting traversal is separate from ordinary Go cause discovery and value execution.

  @AT-ITERATION-001 @REQ-ITER-001 @REQ-ITER-002 @REQ-ERR-005 @REQ-ERR-006 @behaviour
  Scenario: Nil and mixed error trees yield the specified occurrence sequence
    Given an error tree with location wrappers, a coded error with a cause, and a joined pair of uncoded leaves
    When walkIssues is fully consumed
    Then it yields one coded occurrence followed by two external occurrences
    And it does not yield the coded error cause as another issue
    And walking nil yields no occurrences

  @AT-ITERATION-002 @REQ-ITER-003 @behaviour
  Scenario: Early stop does not inspect later siblings
    Given an aggregate whose first child is a coded error
    And a later child has an Unwrap method that panics
    When the first yielded issue causes yield to return false
    Then exactly one issue has been yielded
    And the later child Unwrap method is never called
    And no panic occurs

  @AT-ITERATION-003 @REQ-ITER-003 @REQ-ERR-009 @behaviour
  Scenario: A stopped iterator can be started again from the root
    Given an issue iterator over three independent occurrences of sentinel S
    When one traversal stops after its first occurrence
    And a new traversal consumes that iterator to completion
    Then the new traversal yields all 3 occurrences in original order
    And error identity is not used to suppress repeated occurrences

  @AT-ITERATION-004 @REQ-ITER-004 @REQ-PATH-003 @behaviour
  Scenario: Retaining a yielded path is safe
    Given issues at fields "left" and "right"
    When the consumer saves both yielded Issue values beyond their yield calls
    And modifies the first saved path
    Then the second saved path remains "$.right"
    And another iteration yields both original paths

  @AT-ITERATION-005 @REQ-ITER-001 @REQ-ARCH-007 @source
  Scenario: Reporting does not become part of successful validation
    Given the validation and reporting call graphs
    When the source gate inspects them
    Then only reporting calls walkIssues
    And validation does not depend on iter.Pull, channels or callback collectors
    And failure-side traversal may allocate without weakening the success budget
```

### 14_allocation.feature

```gherkin
# language: en
@validation_refactor
Feature: Zero-allocation success and honest cost measurement
  Construction and reporting are outside the budget; hidden first-call work is not.

  @AT-ALLOCATION-001 @REQ-PERF-001 @REQ-PERF-002 @REQ-PERF-004 @REQ-CONF-004 @allocation
  Scenario Outline: All supported valid composition families are allocation-free
    Given a stored preconstructed rule for fixture <fixture>
    And prepared valid inputs and non-allocating callbacks
    When first-call and steady-state allocation probes run on every required native toolchain and platform
    Then every result is nil
    And validation-attributable allocated objects and bytes are both zero

    Examples:
      | fixture                                                          |
      | numeric and comparable scalar catalogue                          |
      | string and byte catalogue                                        |
      | time catalogue                                                   |
      | flat parent with 16 fields                                       |
      | nested parent depth 16                                           |
      | OptionalPtr and RequiredPtr present                              |
      | OptionalPtr absent                                               |
      | OptionalValue absent and RequiredValue present                   |
      | Each at lengths 0, 1, 8, 64, 1024                                |
      | nested named slices                                              |
      | named maps at sizes 0, 1, 8, 64, 1024, 10000                     |
      | membership cardinalities 0, 1, 4, 32, 1024 with valid cases only |
      | scan uniqueness lengths 0, 1, 8, 64, 1024                        |
      | pointer-parent projection of a fixed array                       |
      | Check whose last Boolean alternative succeeds                    |
      | MinBy and BetweenBy with non-allocating comparators              |

  @AT-ALLOCATION-002 @REQ-PERF-006 @REQ-QUAL-006 @allocation
  Scenario: AllocsPerRun rounding cannot conceal rare allocations
    Given a control fixture deliberately allocates once every 100 validation calls
    When the allocation gate inspects total batch allocation and byte deltas
    Then it rejects the fixture
    And an averaged display of 0 allocs/op is not accepted as proof

  @AT-ALLOCATION-003 @REQ-PERF-001 @REQ-PERF-003 @REQ-PERF-006 @allocation
  Scenario: First use cannot hide a lazy successful-path allocation
    Given a control fixture deliberately allocates only on its first validation call
    And the rule has been constructed but never invoked
    When isolated first-call probes run with a stabilised runtime harness
    Then the gate rejects the fixture
    And prior rule execution or pool warm-up is not used to make it pass

  @AT-ALLOCATION-004 @REQ-PERF-002 @REQ-PERF-005 @benchmark
  Scenario: Construction and reporting costs are measured separately
    Given constructors that snapshot rule lists and prepare membership data
    And reporting that allocates paths for invalid results
    When benchmark suites are executed
    Then construction, valid evaluation, invalid evaluation, traversal and formatting have separate result rows
    And each row records compiler, platform, input size, ns/op, B/op and allocs/op
    And valid evaluation includes none of the fixture setup or reporting work

  @AT-ALLOCATION-005 @REQ-PERF-002 @REQ-EXT-001 @allocation
  Scenario: A custom allocating callback does not gain a false guarantee
    Given a custom successful rule calls make and retains a new buffer each time
    When it is composed with allocation-free built-in rules
    Then whole-composition allocation evidence is non-zero
    And documentation attributes the custom cost instead of claiming the composition is allocation-free
    And the equivalent composition with a non-allocating callback still meets the zero gate

  @AT-ALLOCATION-006 @REQ-SLICE-006 @REQ-PERF-005 @benchmark
  Scenario: Uniqueness has an explicit CPU-versus-memory trade-off
    Given SliceUnique validates unique inputs of length 8, 64 and 1024
    When its source and benchmarks are reviewed
    Then the implementation uses a previous-elements scan without per-call maps or input mutation
    And valid runs allocate zero objects and bytes
    And the documented worst-case comparison count is n times (n minus 1) divided by 2
    And no absolute latency SLA is invented from the allocation target

  @AT-ALLOCATION-007 @REQ-PERF-003 @REQ-CONF-003 @REQ-ARCH-007 @source
  Scenario: Source excludes hidden state used to mask allocation
    Given every library-owned runtime source file and applicable build tag
    When the architecture gate inspects the implementation
    Then there is no sync.Pool, mutable per-rule scratch, input cache or reporter-based hot-path ABI
    And fixed membership lookup state is created only during construction and thereafter read-only
```

### 15_typing_architecture.feature

```gherkin
# language: en
@validation_refactor
Feature: Compile-time and source constraints
  The type-safe engine cannot quietly become a runtime value interpreter.

  @AT-TYPING_ARCHITECTURE-001 @REQ-ARCH-001 @REQ-TYPE-009 @REQ-QUAL-004 @REQ-MIG-001 @compile
  Scenario: Core signatures and named-value examples compile
    Given fixtures for named numeric, string, byte-slice, slice and map types
    And typed Field getters returning those exact types
    When the new API examples are compiled from an external test package
    Then no value conversion to any or an unnamed collection is required
    And nested struct and collection validators are ordinary Rule values
    And Struct delegates to All rather than a separate engine

  @AT-TYPING_ARCHITECTURE-002 @REQ-TYPE-009 @REQ-EXT-002 @REQ-PERF-004 @allocation
  Scenario: Slice an array from its original pointer-parent storage
    Given Parent contains a fixed array and validation receives *Parent
    And its typed getter returns a slice of that array without copying Parent
    When the valid array is checked with Each
    Then the fixture compiles
    And the original array is not changed
    And successful validation allocates zero objects and bytes

  @AT-TYPING_ARCHITECTURE-003 @REQ-ARCH-002 @REQ-QUAL-007 @source
  Scenario: Reflection and unsafe are absent from owned runtime code
    Given all owned runtime packages including generated and build-tagged files
    When parsed source and type-aware call checks run
    Then there is no reflect import, unsafe import, reflective helper or unsafe directive
    And standard-library internals are not incorrectly treated as owned source violations

  @AT-TYPING_ARCHITECTURE-004 @REQ-ARCH-003 @REQ-QUAL-007 @source
  Scenario: Input type erasure is rejected while error traversal is allowed
    Given a source-control fixture contains any(input), a value type switch and an error-node Unwrap assertion
    When the source policy checks the fixture
    Then it rejects the value boxing and value type switch
    And it permits the error assertion only in failure-side inspection code
    And any used solely as a generic constraint is not reported as boxing

  @AT-TYPING_ARCHITECTURE-005 @REQ-ARCH-004 @REQ-ARCH-005 @REQ-ARCH-006 @source
  Scenario: Scope and dependency boundaries are enforceable
    Given the runtime exports, owned source imports and dependency graph
    When the scope gate runs
    Then no context API, context import, format catalogue, regex constructor, tag engine or implicit I/O is present
    And runtime dependencies are limited to the Go standard library
    And development-only Gherkin tooling does not enter runtime dependencies

  @AT-TYPING_ARCHITECTURE-006 @REQ-CONF-001 @REQ-CORE-008 @behaviour
  Scenario: Invalid zero-value rules are programming errors rather than hidden no-ops
    Given a zero-valued Rule[int] function
    When it is supplied to All
    Then construction panics with ConfigurationError
    And invoking a nil Rule directly follows ordinary nil-function panic behaviour

  @AT-TYPING_ARCHITECTURE-007 @REQ-EXT-005 @REQ-EXT-002 @behaviour @compile
  Scenario: Generic methods project an already-composed rule to a parent type
    Given a preconstructed Rule[string] with NotEmpty and RuneMinLength children
    When its Field and Project methods adapt typed getters from a parent struct
    Then each result is assignable to Rule[parent]
    And the failures retain both child codes in order
    And Field adds the named path while Project leaves the root path

  @AT-TYPING_ARCHITECTURE-008 @REQ-EXT-005 @REQ-PTR-003 @behaviour
  Scenario: Generic presence methods preserve explicit absence and present-zero semantics
    Given a preconstructed Min(1) rule and a typed getter returning value and presence
    When OptionalValue and RequiredValue methods adapt the parent
    Then an absent optional value succeeds and an absent required value reports required
    And a present zero value runs Min and reports min under either method
    And the getter is called once per validation
```

### 16_quality_release.feature

```gherkin
# language: en
@validation_refactor
Feature: Verification, traceability and migration
  Only current-revision implementation evidence can close the release gate.

  @AT-QUALITY_RELEASE-001 @REQ-CONF-003 @REQ-ERR-003 @REQ-MAP-006 @REQ-QUAL-002 @race
  Scenario: Concurrent reuse does not mutate rules, inputs or errors
    Given one preconstructed validator with shared immutable sentinel errors
    And prepared read-only structs, slices and maps containing valid and invalid cases
    When 32 goroutines each validate and inspect 100 cases under the race detector
    Then no race is reported
    And each case has the expected complete issue sequence
    And no prior result or input changes

  @AT-QUALITY_RELEASE-002 @REQ-QUAL-001 @REQ-QUAL-006 @REQ-PERF-001 @release
  Scenario: Native toolchain matrix is recorded rather than assumed
    Given the release declares Go 1.27 minimum
    When release evidence is collected
    Then each required native compiler and platform pair has correctness and allocation results
    And the exact compiler patch, OS and architecture are recorded
    And cross-compilation without native execution is not counted as allocation evidence

  @AT-QUALITY_RELEASE-003 @REQ-QUAL-003 @REQ-SLICE-007 @REQ-MAP-005 @fuzz
  Scenario: Bounded properties use independent oracles
    Given bounded generated inputs for paths, numeric bounds, slices and maps
    When property and fuzz tests compare the implementation with independent oracles
    Then issue count, order and codes match the oracle
    And inputs remain unchanged
    And map insertion permutations preserve diagnostic order
    And failures retain their reproducible seed or corpus entry

  @AT-QUALITY_RELEASE-004 @REQ-QUAL-005 @traceability
  Scenario: Every requirement and example row has acceptance routing
    Given this specification and its tagged feature files
    When the specification checker runs
    Then every requirement has one or more matching scenario IDs
    And scenario IDs are unique and reference only existing requirements
    And each Scenario Outline has complete non-empty Examples rows
    And each scenario records a verification category

  @AT-QUALITY_RELEASE-005 @REQ-QUAL-005 @REQ-MIG-004 @release
  Scenario: A scenario with no implementation binding is not a pass
    Given a required scenario has no Go test or step-definition binding
    When implementation acceptance is summarised
    Then that case is reported as not executed rather than passed
    And release acceptance is blocked
    And successful specification parsing does not override the missing result

  @AT-QUALITY_RELEASE-006 @REQ-QUAL-006 @REQ-MIG-004 @release
  Scenario: Reject evidence from another code or specification revision
    Given a passing result records an older code commit or different feature-file hash
    When the current release checks its evidence manifest
    Then that result is excluded from the current acceptance gate
    And implementation acceptance remains incomplete until matching evidence is supplied

  @AT-QUALITY_RELEASE-007 @REQ-MIG-001 @REQ-MIG-002 @REQ-MIG-003 @REQ-ARCH-005 @release
  Scenario: Migration does not retain the old engines or fail-fast semantics
    Given the migration guide and the refactored public API
    When old-to-new examples and exported symbols are reviewed
    Then Rule returns error and fields, structs and collections use the one typed engine
    And Fatal, RuleStopOnError, arbitrary error-based Or/RuleNot and regex constructors are removed
    And guidance covers all-error results, pointer presence, paths, naming and concrete nil adaptation
    And compatibility does not reintroduce context, reflection or allocating successful alternatives

  @AT-QUALITY_RELEASE-008 @REQ-MIG-002 @REQ-ERR-007 @REQ-CORE-001 @compile
  Scenario: Legacy nil values are normalised before error conversion
    Given explicit migration examples adapt a legacy pointer result and a legacy error-slice result
    When those legacy functions return a nil pointer or a nil or empty slice
    Then each adapter returns a literal nil error interface
    And non-empty legacy failures remain discoverable through ordinary wrapping
    And the examples use concrete nil checks rather than reflection

  @AT-QUALITY_RELEASE-009 @REQ-MIG-004 @REQ-QUAL-004 @REQ-QUAL-005 @REQ-QUAL-006 @release
  Scenario: The full gate requires all verification categories
    Given a release candidate for this exact specification revision
    When completion is assessed
    Then behaviour, compile, source, race, fuzz, allocation and release cases all have passing current evidence
    And every published example compiles
    And any missing category, required case or platform blocks acceptance
    And the implementation is not certified by a specification-only report
```
