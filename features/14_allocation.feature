# language: en
@validation_refactor
Feature: Zero-allocation success and honest cost measurement
  Construction and reporting are outside the budget; hidden first-call work is not.

  @AT-ALLOCATION-001 @REQ-PERF-001 @REQ-PERF-002 @REQ-PERF-004 @REQ-CONF-004 @allocation
  Scenario Outline: All supported valid composition families are allocation-free
    Given a stored preconstructed rule for fixture <fixture>
    And prepared valid inputs and non-allocating callbacks
    When first-call and steady-state allocation probes run on every required native toolchain and platform
    Then every result is nil
    And validation-attributable allocated objects and bytes are both zero

    Examples:
      | fixture                                                          |
      | numeric and comparable scalar catalogue                          |
      | string and byte catalogue                                        |
      | time catalogue                                                   |
      | flat parent with 16 fields                                       |
      | nested parent depth 16                                           |
      | OptionalPtr and RequiredPtr present                              |
      | OptionalPtr absent                                               |
      | OptionalValue absent and RequiredValue present                   |
      | Each at lengths 0, 1, 8, 64, 1024                                |
      | nested named slices                                              |
      | named maps at sizes 0, 1, 8, 64, 1024, 10000                     |
      | membership cardinalities 0, 1, 4, 32, 1024 with valid cases only |
      | scan uniqueness lengths 0, 1, 8, 64, 1024                        |
      | pointer-parent projection of a fixed array                       |
      | Check whose last Boolean alternative succeeds                    |
      | MinBy and BetweenBy with non-allocating comparators              |

  @AT-ALLOCATION-002 @REQ-PERF-006 @REQ-QUAL-006 @allocation
  Scenario: AllocsPerRun rounding cannot conceal rare allocations
    Given a control fixture deliberately allocates once every 100 validation calls
    When the allocation gate inspects total batch allocation and byte deltas
    Then it rejects the fixture
    And an averaged display of 0 allocs/op is not accepted as proof

  @AT-ALLOCATION-003 @REQ-PERF-001 @REQ-PERF-003 @REQ-PERF-006 @allocation
  Scenario: First use cannot hide a lazy successful-path allocation
    Given a control fixture deliberately allocates only on its first validation call
    And the rule has been constructed but never invoked
    When isolated first-call probes run with a stabilised runtime harness
    Then the gate rejects the fixture
    And prior rule execution or pool warm-up is not used to make it pass

  @AT-ALLOCATION-004 @REQ-PERF-002 @REQ-PERF-005 @benchmark
  Scenario: Construction and reporting costs are measured separately
    Given constructors that snapshot rule lists and prepare membership data
    And reporting that allocates paths for invalid results
    When benchmark suites are executed
    Then construction, valid evaluation, invalid evaluation, traversal and formatting have separate result rows
    And each row records compiler, platform, input size, ns/op, B/op and allocs/op
    And valid evaluation includes none of the fixture setup or reporting work

  @AT-ALLOCATION-005 @REQ-PERF-002 @REQ-EXT-001 @allocation
  Scenario: A custom allocating callback does not gain a false guarantee
    Given a custom successful rule calls make and retains a new buffer each time
    When it is composed with allocation-free built-in rules
    Then whole-composition allocation evidence is non-zero
    And documentation attributes the custom cost instead of claiming the composition is allocation-free
    And the equivalent composition with a non-allocating callback still meets the zero gate

  @AT-ALLOCATION-006 @REQ-SLICE-006 @REQ-PERF-005 @benchmark
  Scenario: Uniqueness has an explicit CPU-versus-memory trade-off
    Given SliceUnique validates unique inputs of length 8, 64 and 1024
    When its source and benchmarks are reviewed
    Then the implementation uses a previous-elements scan without per-call maps or input mutation
    And valid runs allocate zero objects and bytes
    And the documented worst-case comparison count is n times (n minus 1) divided by 2
    And no absolute latency SLA is invented from the allocation target

  @AT-ALLOCATION-007 @REQ-PERF-003 @REQ-CONF-003 @REQ-ARCH-007 @source
  Scenario: Source excludes hidden state used to mask allocation
    Given every library-owned runtime source file and applicable build tag
    When the architecture gate inspects the implementation
    Then there is no sync.Pool, mutable per-rule scratch, input cache or reporter-based hot-path ABI
    And fixed membership lookup state is created only during construction and thereafter read-only
