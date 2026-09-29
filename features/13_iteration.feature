# language: en
@validation_refactor
Feature: Failure-side issue iteration
  Reporting traversal is separate from ordinary Go cause discovery and value execution.

  @AT-ITERATION-001 @REQ-ITER-001 @REQ-ITER-002 @REQ-ERR-005 @REQ-ERR-006 @behaviour
  Scenario: Nil and mixed error trees yield the specified occurrence sequence
    Given an error tree with location wrappers, a coded error with a cause, and a joined pair of uncoded leaves
    When walkIssues is fully consumed
    Then it yields one coded occurrence followed by two external occurrences
    And it does not yield the coded error cause as another issue
    And walking nil yields no occurrences

  @AT-ITERATION-002 @REQ-ITER-003 @behaviour
  Scenario: Early stop does not inspect later siblings
    Given an aggregate whose first child is a coded error
    And a later child has an Unwrap method that panics
    When the first yielded issue causes yield to return false
    Then exactly one issue has been yielded
    And the later child Unwrap method is never called
    And no panic occurs

  @AT-ITERATION-003 @REQ-ITER-003 @REQ-ERR-009 @behaviour
  Scenario: A stopped iterator can be started again from the root
    Given an issue iterator over three independent occurrences of sentinel S
    When one traversal stops after its first occurrence
    And a new traversal consumes that iterator to completion
    Then the new traversal yields all 3 occurrences in original order
    And error identity is not used to suppress repeated occurrences

  @AT-ITERATION-004 @REQ-ITER-004 @REQ-PATH-003 @behaviour
  Scenario: Retaining a yielded path is safe
    Given issues at fields "left" and "right"
    When the consumer saves both yielded Issue values beyond their yield calls
    And modifies the first saved path
    Then the second saved path remains "$.right"
    And another iteration yields both original paths

  @AT-ITERATION-005 @REQ-ITER-001 @REQ-ARCH-007 @source
  Scenario: Reporting does not become part of successful validation
    Given the validation and reporting call graphs
    When the source gate inspects them
    Then only reporting calls walkIssues
    And validation does not depend on iter.Pull, channels or callback collectors
    And failure-side traversal may allocate without weakening the success budget
