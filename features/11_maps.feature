# language: en
@validation_refactor
Feature: Typed map validation with deterministic diagnostics
  Validation order may vary; complete result ordering does not depend on it.

  @AT-MAPS-001 @REQ-MAP-001 @REQ-CORE-009 @behaviour
  Scenario Outline: Map absence differs from a present zero value
    Given <rule> with child Min(1)
    When the map is <input>
    Then the issue codes are <codes>
    And the child rule call count is <calls>

    Examples:
      | rule                | input    | codes        | calls |
      | MapRequiredKey("x") | nil      | key_required | 0     |
      | MapRequiredKey("x") | {"x": 0} | min          | 1     |
      | MapOptionalKey("x") | {}       | none         | 0     |
      | MapOptionalKey("x") | {"x": 0} | min          | 1     |
      | MapRequiredKey("x") | {"x": 1} | none         | 1     |

  @AT-MAPS-002 @REQ-MAP-001 @behaviour
  Scenario: Nil map participates in size validation
    Given All contains MapLength(0) and MapMinLength(1)
    When it validates a nil map
    Then the only issue code is "min_length"
    And both size rules were evaluated

  @AT-MAPS-003 @REQ-MAP-002 @REQ-CORE-004 @behaviour
  Scenario: Report every key and value failure
    Given MapEach uses StringKeys and two rules: key NotEmpty then value Min(1)
    When the map contains entries {"b": 0, "": 0, "a": 1}
    Then the ordered issues are exactly:
      | path  | code      |
      | $[""] | not_empty |
      | $[""] | min       |
      | $["b"]| min       |
    And both rules were called once for each entry

  @AT-MAPS-004 @REQ-MAP-002 @REQ-CORE-002 @behaviour
  Scenario: Forbidden keys are all reported without suppressing value checks
    Given All contains MapKeysOneOf("a") followed by MapValues(Min(1))
    When it validates {"z": 0, "y": 0, "a": 1}
    Then the ordered issues are exactly:
      | path   | code   |
      | $["y"] | one_of |
      | $["z"] | one_of |
      | $["y"] | min    |
      | $["z"] | min    |

  @AT-MAPS-005 @REQ-MAP-003 @REQ-PATH-005 @REQ-CONF-004 @REQ-PERF-001 @allocation
  Scenario: Valid maps allocate no key list or rendering work
    Given MapValues has a KeyOrder with counting Less and Text callbacks
    And all 10000 prepared map values pass a non-allocating rule
    When the map is validated
    Then the result is nil
    And Less and Text are each called zero times
    And successful validation allocates 0 objects and 0 bytes

  @AT-MAPS-006 @REQ-MAP-003 @REQ-PATH-005 @REQ-MAP-005 @behaviour
  Scenario: Sort only failed entry groups and render each group once
    Given 100 map entries and only keys "z", "a" and "m" fail two child rules
    And KeyOrder callbacks record their arguments
    When the map is validated
    Then Less receives only keys from {"a", "m", "z"}
    And Text is called once for each of those 3 keys
    And 6 issues appear in key order "a", "m", "z" with child declaration order
    And no value rule is replayed

  @AT-MAPS-007 @REQ-MAP-003 @REQ-MAP-005 @behaviour
  Scenario: Insertion order does not affect issue order
    Given all 6 insertion permutations of 3 failing string-keyed entries
    When MapValues validates each resulting map 20 times
    Then every Issues result has the same paths and codes in lexical key order
    And the test makes no assertion about callback execution order

  @AT-MAPS-008 @REQ-MAP-004 @REQ-TYPE-009 @compile
  Scenario: Typed non-string keys use an explicit ordering and rendering policy
    Given a named map with int keys and a KeyOrder using numeric less-than and strconv.Itoa
    When keys 10 and 2 both fail
    Then the issue paths are "$["2"]" followed by "$["10"]"
    And no fmt-based or any-based key dispatch is needed

  @AT-MAPS-009 @REQ-MAP-004 @REQ-PATH-001 @REQ-PATH-005 @behaviour
  Scenario: Literal key punctuation is preserved
    Given StringKeys and failing keys "a.b", "a[0]" and ""
    When the map is validated
    Then the 3 keys remain distinct Key segments
    And no key text becomes a nested Field or Index path

  @AT-MAPS-010 @REQ-MAP-006 @REQ-CONF-003 @source
  Scenario: Map validation is read-only
    Given a read-only input map and a reusable validator
    When successful and failing validations are repeated
    Then map membership and every entry value remain unchanged
    And the validator retains no per-input error or map state

  @AT-MAPS-011 @REQ-MAP-001 @REQ-PATH-005 @REQ-CONF-004 @behaviour
  Scenario: Missing required keys render only their own failure
    Given All contains MapRequiredKey("x") and MapOptionalKey("y") with counting key renderers
    And both keys are absent
    When the empty map is validated
    Then exactly one key_required issue has path "$["x"]"
    And the renderer for "x" is called once and the renderer for "y" is never called
    And no child value rule is invoked
