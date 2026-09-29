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
