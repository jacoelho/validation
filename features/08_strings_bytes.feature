# language: en
@validation_refactor
Feature: Strings, Unicode and bytes
  Byte counts, rune counts and UTF-8 validity are distinct constraints.

  @AT-STRINGS_BYTES-001 @REQ-TYPE-005 @behaviour
  Scenario Outline: Code points are not bytes or grapheme clusters
    Given the input string <text> has the literal Unicode sequence specified
    When byte and rune length rules evaluate the input
    Then its byte length is <bytes> and rune count is <runes>

    Examples:
      | text                      | bytes | runes |
      | ""                        | 0     | 0     |
      | "A"                       | 1     | 1     |
      | precomposed U+00E9        | 2     | 1     |
      | U+0065 followed by U+0301 | 3     | 2     |
      | U+1F642                   | 4     | 1     |

  @AT-STRINGS_BYTES-002 @REQ-TYPE-005 @REQ-CORE-002 @behaviour
  Scenario: Invalid UTF-8 is checked independently of rune length
    Given a string containing exactly bytes 0xff and 0xfe
    And All contains UTF8 and RuneLength(2)
    When the string is validated
    Then RuneLength(2) succeeds
    And the only issue code is "utf8"
    And both rules execute

  @AT-STRINGS_BYTES-003 @REQ-TYPE-005 @behaviour
  Scenario Outline: String lengths have explicit boundary outcomes
    Given the string rule <rule>
    When it validates <value>
    Then the issue codes are <codes>

    Examples:
      | rule                    | value                     | codes           |
      | ByteLength(2)           | precomposed U+00E9        | none            |
      | ByteMinLength(3)        | precomposed U+00E9        | byte_min_length |
      | ByteMaxLength(1)        | precomposed U+00E9        | byte_max_length |
      | ByteLengthBetween(2, 2) | precomposed U+00E9        | none            |
      | RuneLength(2)           | precomposed U+00E9        | rune_length     |
      | RuneMinLength(1)        | precomposed U+00E9        | none            |
      | RuneMaxLength(1)        | U+0065 followed by U+0301 | rune_max_length |
      | RuneLengthBetween(2, 2) | U+0065 followed by U+0301 | none            |

  @AT-STRINGS_BYTES-004 @REQ-TYPE-006 @REQ-CORE-009 @behaviour
  Scenario Outline: No trimming, case folding or implicit empty-string skip
    Given the string rule <rule>
    When it validates <value>
    Then the issue codes are <codes>

    Examples:
      | rule            | value           | codes     |
      | NotEmpty()      | one ASCII space | none      |
      | Contains("A")   | "abc"           | contains  |
      | Contains("")    | ""              | none      |
      | HasPrefix("")   | ""              | none      |
      | HasSuffix("")   | ""              | none      |
      | HasPrefix("ab") | "abc"           | none      |
      | HasSuffix("bc") | "abc"           | none      |
      | NotEmpty()      | ""              | not_empty |

  @AT-STRINGS_BYTES-005 @REQ-TYPE-007 @REQ-SLICE-007 @behaviour
  Scenario Outline: Dedicated byte rules preserve byte semantics
    Given the byte-slice rule <rule>
    When it validates <value>
    Then the issue codes are <codes>
    And the input backing array is unchanged

    Examples:
      | rule                   | value        | codes           |
      | BytesEqual(empty)      | nil          | none            |
      | BytesNotEmpty()        | nil          | not_empty       |
      | BytesLength(2)         | [0xff, 0xfe] | none            |
      | BytesUTF8()            | [0xff, 0xfe] | utf8            |
      | BytesContains([0xff])  | [0x00, 0xff] | none            |
      | BytesHasPrefix([0x00]) | [0x00, 0xff] | none            |
      | BytesHasSuffix([0xff]) | [0x00, 0xff] | none            |
      | BytesMaxLength(1)      | [0x00, 0xff] | byte_max_length |

  @AT-STRINGS_BYTES-006 @REQ-TYPE-007 @REQ-ARCH-003 @source
  Scenario: Dedicated byte implementations do not copy through strings
    Given all owned runtime byte validation functions
    When the type-aware source gate inspects their call paths
    Then there is no byte-slice-to-string conversion for value validation
    And named byte slices compile against the dedicated byte constructors
