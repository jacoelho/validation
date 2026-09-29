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
