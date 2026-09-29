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
