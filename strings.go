package validation

import (
	"bytes"
	"regexp"
	"strings"
	"unicode/utf8"
)

func stringLengthArg(constructor string, n int) {
	if n < 0 {
		configurationError(constructor)
	}
}

func stringLengthRange(constructor string, min, max int) {
	if min < 0 || max < 0 || min > max {
		configurationError(constructor)
	}
}

func copyBytes[B ~[]byte](value B) B {
	if value == nil {
		return nil
	}
	out := make(B, len(value))
	copy(out, value)
	return out
}

// NotEmpty rejects the empty string. Whitespace is retained as content.
func NotEmpty[T ~string]() Rule[T] {
	return func(value T) error {
		if len(value) == 0 {
			return NewViolation(CodeNotEmpty, nil)
		}
		return nil
	}
}

// NotBlank rejects empty and Unicode whitespace-only strings.
func NotBlank[T ~string]() Rule[T] {
	return func(value T) error {
		if strings.TrimSpace(string(value)) == "" {
			return NewViolation(CodeNotBlank, nil)
		}
		return nil
	}
}

// Trimmed rejects leading or trailing Unicode whitespace without changing input.
func Trimmed[T ~string]() Rule[T] {
	return func(value T) error {
		if strings.TrimSpace(string(value)) != string(value) {
			return NewViolation(CodeTrimmed, nil)
		}
		return nil
	}
}

// ByteLength validates the exact byte length of a string.
func ByteLength[T ~string](n int) Rule[T] {
	stringLengthArg("ByteLength", n)
	return func(value T) error {
		actual := len(value)
		if actual != n {
			return newLengthError(CodeByteLength, actual, n, &n, LengthBytes)
		}
		return nil
	}
}

// ByteMinLength validates a string's minimum byte length.
func ByteMinLength[T ~string](min int) Rule[T] {
	stringLengthArg("ByteMinLength", min)
	return func(value T) error {
		actual := len(value)
		if actual < min {
			return newLengthError(CodeByteMinLength, actual, min, nil, LengthBytes)
		}
		return nil
	}
}

// ByteMaxLength validates a string's maximum byte length.
func ByteMaxLength[T ~string](max int) Rule[T] {
	stringLengthArg("ByteMaxLength", max)
	return func(value T) error {
		actual := len(value)
		if actual > max {
			return newLengthError(CodeByteMaxLength, actual, 0, &max, LengthBytes)
		}
		return nil
	}
}

// ByteLengthBetween validates a string's inclusive byte-length interval.
func ByteLengthBetween[T ~string](min, max int) Rule[T] {
	stringLengthRange("ByteLengthBetween", min, max)
	return func(value T) error {
		actual := len(value)
		if actual < min || actual > max {
			return newLengthError(CodeByteLengthBetween, actual, min, &max, LengthBytes)
		}
		return nil
	}
}

// RuneLength validates the exact number of decoded UTF-8 runes.
func RuneLength[T ~string](n int) Rule[T] {
	stringLengthArg("RuneLength", n)
	return func(value T) error {
		actual := utf8.RuneCountInString(string(value))
		if actual != n {
			return newLengthError(CodeRuneLength, actual, n, &n, LengthRunes)
		}
		return nil
	}
}

// RuneMinLength validates a string's minimum rune count.
func RuneMinLength[T ~string](min int) Rule[T] {
	stringLengthArg("RuneMinLength", min)
	return func(value T) error {
		actual := utf8.RuneCountInString(string(value))
		if actual < min {
			return newLengthError(CodeRuneMinLength, actual, min, nil, LengthRunes)
		}
		return nil
	}
}

// RuneMaxLength validates a string's maximum rune count.
func RuneMaxLength[T ~string](max int) Rule[T] {
	stringLengthArg("RuneMaxLength", max)
	return func(value T) error {
		actual := utf8.RuneCountInString(string(value))
		if actual > max {
			return newLengthError(CodeRuneMaxLength, actual, 0, &max, LengthRunes)
		}
		return nil
	}
}

// RuneLengthBetween validates a string's inclusive rune-count interval.
func RuneLengthBetween[T ~string](min, max int) Rule[T] {
	stringLengthRange("RuneLengthBetween", min, max)
	return func(value T) error {
		actual := utf8.RuneCountInString(string(value))
		if actual < min || actual > max {
			return newLengthError(CodeRuneLengthBetween, actual, min, &max, LengthRunes)
		}
		return nil
	}
}

// Contains validates that value contains part as an exact substring.
func Contains[T ~string](part T) Rule[T] {
	want := string(part)
	return func(value T) error {
		if !strings.Contains(string(value), want) {
			return &TextError{code: CodeContains, constraint: want}
		}
		return nil
	}
}

// HasPrefix validates that value begins with prefix.
func HasPrefix[T ~string](prefix T) Rule[T] {
	want := string(prefix)
	return func(value T) error {
		if !strings.HasPrefix(string(value), want) {
			return &TextError{code: CodePrefix, constraint: want}
		}
		return nil
	}
}

// HasSuffix validates that value ends with suffix.
func HasSuffix[T ~string](suffix T) Rule[T] {
	want := string(suffix)
	return func(value T) error {
		if !strings.HasSuffix(string(value), want) {
			return &TextError{code: CodeSuffix, constraint: want}
		}
		return nil
	}
}

// Match requires a match by the caller's compiled expression. Anchor the
// expression when the entire value must match.
func Match[T ~string](pattern *regexp.Regexp) Rule[T] {
	if pattern == nil {
		configurationError("Match")
	}
	return func(value T) error {
		if !pattern.MatchString(string(value)) {
			return &TextError{code: CodeMatch, constraint: pattern.String()}
		}
		return nil
	}
}

// UTF8 rejects malformed UTF-8 strings.
func UTF8[T ~string]() Rule[T] {
	return func(value T) error {
		if !utf8.ValidString(string(value)) {
			return NewViolation(CodeUTF8, nil)
		}
		return nil
	}
}

// BytesNotEmpty rejects nil and empty byte slices.
func BytesNotEmpty[B ~[]byte]() Rule[B] {
	return func(value B) error {
		if len(value) == 0 {
			return NewViolation(CodeNotEmpty, nil)
		}
		return nil
	}
}

// BytesLength validates exact byte-slice length.
func BytesLength[B ~[]byte](n int) Rule[B] {
	stringLengthArg("BytesLength", n)
	return func(value B) error {
		actual := len(value)
		if actual != n {
			return newLengthError(CodeByteLength, actual, n, &n, LengthBytes)
		}
		return nil
	}
}

// BytesMinLength validates a byte slice's minimum length.
func BytesMinLength[B ~[]byte](min int) Rule[B] {
	stringLengthArg("BytesMinLength", min)
	return func(value B) error {
		actual := len(value)
		if actual < min {
			return newLengthError(CodeByteMinLength, actual, min, nil, LengthBytes)
		}
		return nil
	}
}

// BytesMaxLength validates a byte slice's maximum length.
func BytesMaxLength[B ~[]byte](max int) Rule[B] {
	stringLengthArg("BytesMaxLength", max)
	return func(value B) error {
		actual := len(value)
		if actual > max {
			return newLengthError(CodeByteMaxLength, actual, 0, &max, LengthBytes)
		}
		return nil
	}
}

// BytesLengthBetween validates an inclusive byte-slice length interval.
func BytesLengthBetween[B ~[]byte](min, max int) Rule[B] {
	stringLengthRange("BytesLengthBetween", min, max)
	return func(value B) error {
		actual := len(value)
		if actual < min || actual > max {
			return newLengthError(CodeByteLengthBetween, actual, min, &max, LengthBytes)
		}
		return nil
	}
}

// BytesContains validates that value contains part as a byte subsequence.
func BytesContains[B ~[]byte](part B) Rule[B] {
	want := copyBytes(part)
	return func(value B) error {
		if !bytes.Contains([]byte(value), []byte(want)) {
			return &TextError{code: CodeContains, constraint: string(want)}
		}
		return nil
	}
}

// BytesHasPrefix validates that value begins with prefix.
func BytesHasPrefix[B ~[]byte](prefix B) Rule[B] {
	want := copyBytes(prefix)
	return func(value B) error {
		if !bytes.HasPrefix([]byte(value), []byte(want)) {
			return &TextError{code: CodePrefix, constraint: string(want)}
		}
		return nil
	}
}

// BytesHasSuffix validates that value ends with suffix.
func BytesHasSuffix[B ~[]byte](suffix B) Rule[B] {
	want := copyBytes(suffix)
	return func(value B) error {
		if !bytes.HasSuffix([]byte(value), []byte(want)) {
			return &TextError{code: CodeSuffix, constraint: string(want)}
		}
		return nil
	}
}

// BytesUTF8 rejects malformed UTF-8 byte sequences.
func BytesUTF8[B ~[]byte]() Rule[B] {
	return func(value B) error {
		if !utf8.Valid([]byte(value)) {
			return NewViolation(CodeUTF8, nil)
		}
		return nil
	}
}

// BytesEqual compares byte content. Nil and empty slices compare equal.
func BytesEqual[B ~[]byte](want B) Rule[B] {
	configured := copyBytes(want)
	return func(value B) error {
		if bytes.Equal([]byte(value), []byte(configured)) {
			return nil
		}
		return NewViolation(CodeEqual, nil)
	}
}
