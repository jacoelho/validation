package validation

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// StringsNotEmpty validates that the string is not empty.
func StringsNotEmpty[T ~string]() Rule[T] {
	return func(value T) *Error {
		if value == "" {
			return &Error{
				Code: "not_empty",
			}
		}
		return nil
	}
}

// StringsRuneLengthBetween validates string length (in runes, not bytes) between the given minimum and maximum.
func StringsRuneLengthBetween[T ~string](min, max int) Rule[T] {
	return func(value T) *Error {
		length := utf8.RuneCountInString(string(value))
		if length < min || length > max {
			return &Error{
				Code:   "between",
				Params: map[string]any{"min": min, "max": max, "actual": length},
			}
		}
		return nil
	}
}

// StringsRuneMinLength validates minimum string length in runes.
func StringsRuneMinLength[T ~string](min int) Rule[T] {
	return func(value T) *Error {
		length := utf8.RuneCountInString(string(value))
		if length < min {
			return &Error{
				Code:   "min",
				Params: map[string]any{"min": min, "actual": length},
			}
		}
		return nil
	}
}

// StringsRuneMaxLength validates maximum string length in runes.
func StringsRuneMaxLength[T ~string](max int) Rule[T] {
	return func(value T) *Error {
		length := utf8.RuneCountInString(string(value))
		if length > max {
			return &Error{
				Code:   "max",
				Params: map[string]any{"max": max, "actual": length},
			}
		}
		return nil
	}
}

// StringsMatchesRegex validates string against a regex pattern
func StringsMatchesRegex[T ~string](pattern string) Rule[T] {
	regex := regexp.MustCompile(pattern)
	return func(value T) *Error {
		if !regex.MatchString(string(value)) {
			return &Error{
				Code:   "regex",
				Params: map[string]any{"pattern": pattern},
			}
		}
		return nil
	}
}

// StringsContains validates that the string contains the given substring.
func StringsContains[T ~string](substring T) Rule[T] {
	return func(value T) *Error {
		if !strings.Contains(string(value), string(substring)) {
			return &Error{
				Code:   "contains",
				Params: map[string]any{"substring": substring},
			}
		}
		return nil
	}
}
