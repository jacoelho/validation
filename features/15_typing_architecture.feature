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
