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
