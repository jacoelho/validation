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
