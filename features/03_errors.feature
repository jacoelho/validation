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
