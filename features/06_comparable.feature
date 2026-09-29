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
