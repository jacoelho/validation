# language: en
@validation_refactor
Feature: Verification, traceability and migration
  Only current-revision implementation evidence can close the release gate.

  @AT-QUALITY_RELEASE-001 @REQ-CONF-003 @REQ-ERR-003 @REQ-MAP-006 @REQ-QUAL-002 @race
  Scenario: Concurrent reuse does not mutate rules, inputs or errors
    Given one preconstructed validator with shared immutable sentinel errors
    And prepared read-only structs, slices and maps containing valid and invalid cases
    When 32 goroutines each validate and inspect 100 cases under the race detector
    Then no race is reported
    And each case has the expected complete issue sequence
    And no prior result or input changes

  @AT-QUALITY_RELEASE-002 @REQ-QUAL-001 @REQ-QUAL-006 @REQ-PERF-001 @release
  Scenario: Native toolchain matrix is recorded rather than assumed
    Given the release declares Go 1.27 minimum
    When release evidence is collected
    Then each required native compiler and platform pair has correctness and allocation results
    And the exact compiler patch, OS and architecture are recorded
    And cross-compilation without native execution is not counted as allocation evidence

  @AT-QUALITY_RELEASE-003 @REQ-QUAL-003 @REQ-SLICE-007 @REQ-MAP-005 @fuzz
  Scenario: Bounded properties use independent oracles
    Given bounded generated inputs for paths, numeric bounds, slices and maps
    When property and fuzz tests compare the implementation with independent oracles
    Then issue count, order and codes match the oracle
    And inputs remain unchanged
    And map insertion permutations preserve diagnostic order
    And failures retain their reproducible seed or corpus entry

  @AT-QUALITY_RELEASE-004 @REQ-QUAL-005 @traceability
  Scenario: Every requirement and example row has acceptance routing
    Given this specification and its tagged feature files
    When the specification checker runs
    Then every requirement has one or more matching scenario IDs
    And scenario IDs are unique and reference only existing requirements
    And each Scenario Outline has complete non-empty Examples rows
    And each scenario records a verification category

  @AT-QUALITY_RELEASE-005 @REQ-QUAL-005 @REQ-MIG-004 @release
  Scenario: A scenario with no implementation binding is not a pass
    Given a required scenario has no Go test or step-definition binding
    When implementation acceptance is summarised
    Then that case is reported as not executed rather than passed
    And release acceptance is blocked
    And successful specification parsing does not override the missing result

  @AT-QUALITY_RELEASE-006 @REQ-QUAL-006 @REQ-MIG-004 @release
  Scenario: Reject evidence from another code or specification revision
    Given a passing result records an older code commit or different feature-file hash
    When the current release checks its evidence manifest
    Then that result is excluded from the current acceptance gate
    And implementation acceptance remains incomplete until matching evidence is supplied

  @AT-QUALITY_RELEASE-007 @REQ-MIG-001 @REQ-MIG-002 @REQ-MIG-003 @REQ-ARCH-005 @release
  Scenario: Migration does not retain the old engines or fail-fast semantics
    Given the migration guide and the refactored public API
    When old-to-new examples and exported symbols are reviewed
    Then Rule returns error and fields, structs and collections use the one typed engine
    And Fatal, RuleStopOnError, arbitrary error-based Or/RuleNot and regex constructors are removed
    And guidance covers all-error results, pointer presence, paths, naming and concrete nil adaptation
    And compatibility does not reintroduce context, reflection or allocating successful alternatives

  @AT-QUALITY_RELEASE-008 @REQ-MIG-002 @REQ-ERR-007 @REQ-CORE-001 @compile
  Scenario: Legacy nil values are normalised before error conversion
    Given explicit migration examples adapt a legacy pointer result and a legacy error-slice result
    When those legacy functions return a nil pointer or a nil or empty slice
    Then each adapter returns a literal nil error interface
    And non-empty legacy failures remain discoverable through ordinary wrapping
    And the examples use concrete nil checks rather than reflection

  @AT-QUALITY_RELEASE-009 @REQ-MIG-004 @REQ-QUAL-004 @REQ-QUAL-005 @REQ-QUAL-006 @release
  Scenario: The full gate requires all verification categories
    Given a release candidate for this exact specification revision
    When completion is assessed
    Then behaviour, compile, source, race, fuzz, allocation and release cases all have passing current evidence
    And every published example compiles
    And any missing category, required case or platform blocks acceptance
    And the implementation is not certified by a specification-only report
