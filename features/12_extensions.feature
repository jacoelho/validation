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
