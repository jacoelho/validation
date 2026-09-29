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
