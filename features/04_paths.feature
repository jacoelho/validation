# language: en
@validation_refactor
Feature: Structured paths and safe reporting
  Paths identify field, key and index boundaries without mutating child errors.

  @AT-PATHS-001 @REQ-PATH-001 @REQ-PATH-002 @REQ-CORE-004 @behaviour
  Scenario: Compose nested field and slice locations
    Given field "addresses" contains Each of an Address rule
    And only element 2 fails NotEmpty on its field "city"
    When the parent is validated
    Then the issue segments are Field("addresses"), Index(2), Field("city")
    And FormatPath returns "$.addresses[2].city"

  @AT-PATHS-002 @REQ-PATH-001 @REQ-PATH-004 @behaviour
  Scenario Outline: Distinguish literal fields, keys and indices
    Given the structured path <segments>
    When FormatPath renders it
    Then the output is <rendering>

    Examples:
      | segments               | rendering |
      | empty path             | $         |
      | Field("name")          | $.name    |
      | Field("a.b")           | $.["a.b"] |
      | Field("a"), Field("b") | $.a.b     |
      | Key("a.b")             | $["a.b"]  |
      | Index(2)               | $[2]      |
      | Key("2")               | $["2"]    |
      | Field("2")             | $.["2"]   |
      | Key("")                | $[""]     |

  @AT-PATHS-003 @REQ-PATH-004 @behaviour
  Scenario: Control characters are escaped without losing path kind
    Given a key contains quote, backslash, newline and the invalid byte 0xff
    When FormatPath renders a Key segment for that key
    Then the quoted form uses Go string-literal escapes for those bytes
    And the output contains no raw newline
    And its segment remains a Key rather than Field or Index

  @AT-PATHS-004 @REQ-PATH-003 @REQ-ERR-003 @REQ-ITER-004 @behaviour
  Scenario: Issue paths are independent snapshots
    Given Issues returns occurrences at "$.items[0]" and "$.items[1]"
    When the caller changes the first returned path and truncates the returned list
    Then the second saved issue path remains "$.items[1]"
    And a fresh Issues call returns both original paths unchanged

  @AT-PATHS-005 @REQ-PATH-002 @REQ-EXT-002 @REQ-PTR-002 @behaviour
  Scenario: Projection and presence wrappers add no synthetic location
    Given field "address" applies RequiredPtr to Project(identity) of a city Field rule
    And the non-nil address has an empty city
    When the parent is validated
    Then the only issue path is "$.address.city"
    And no pointer, projection or synthetic root segment is inserted

  @AT-PATHS-006 @REQ-PATH-004 @REQ-CONF-001 @behaviour
  Scenario Outline: Malformed caller-created segments are not rendered ambiguously
    Given a caller-created path containing <segment>
    When FormatPath is called under a test-only recover
    Then it panics with a ConfigurationError

    Examples:
      | segment                            |
      | an unknown SegmentKind             |
      | an Index segment with value -1     |
      | a Field segment with an empty name |
