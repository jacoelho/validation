package validation

import (
	"fmt"
	"strconv"
	"strings"
)

// DefaultMessage returns the built-in English message without a path prefix.
// Custom formatters can use it as a fallback without formatting the error tree again.
func DefaultMessage(issue Issue) string {
	if detail, ok := issue.Err.(interface{ failureMessage() string }); ok {
		return detail.failureMessage()
	}
	switch issue.Code {
	case CodeRequired:
		return "value is required"
	case CodeKeyRequired:
		return "key is required"
	case CodeEqual:
		return "must equal the expected value"
	case CodeNotEqual:
		return "must differ from the forbidden value"
	case CodeZero:
		return "must be zero"
	case CodeNotZero:
		return "must not be zero"
	case CodeOneOf:
		return "must be one of the allowed values"
	case CodeNotOneOf:
		return "must not be one of the forbidden values"
	case CodeMultipleOf:
		return "must be a multiple of the configured base"
	case CodeMin:
		return "must meet the minimum"
	case CodeMax:
		return "must not exceed the maximum"
	case CodeBetween:
		return "must be within the allowed range"
	case CodeGreaterThan:
		return "must exceed the lower bound"
	case CodeLessThan:
		return "must be below the upper bound"
	case CodePositive:
		return "must be greater than zero"
	case CodeNonNegative:
		return "must be greater than or equal to zero"
	case CodeNegative:
		return "must be less than zero"
	case CodeNonPositive:
		return "must be less than or equal to zero"
	case CodeNotNaN:
		return "must not be NaN"
	case CodeFinite:
		return "must be finite"
	case CodeNotEmpty:
		return "must not be empty"
	case CodeNotBlank:
		return "must not be blank"
	case CodeTrimmed:
		return "must not have leading or trailing whitespace"
	case CodeContains:
		return "must contain the required content"
	case CodePrefix:
		return "must start with the required prefix"
	case CodeSuffix:
		return "must end with the required suffix"
	case CodeMatch:
		return "must match the required pattern"
	case CodeUTF8:
		return "must be valid UTF-8"
	case CodeTime:
		return "must match the required time layout"
	default:
		return string(issue.Code)
	}
}

func (e *BoundsError[T]) failureMessage() string {
	if e.hasLower && e.hasUpper {
		return fmt.Sprintf("must be between %v and %v (inclusive)", e.lower.value, e.upper.value)
	}
	if e.hasLower {
		switch e.code {
		case CodeAfter:
			return fmt.Sprintf("must be after %v", e.lower.value)
		case CodeAfterOrEqual:
			return fmt.Sprintf("must be at or after %v", e.lower.value)
		}
		if e.lower.inclusive {
			return fmt.Sprintf("must be at least %v", e.lower.value)
		}
		return fmt.Sprintf("must be greater than %v", e.lower.value)
	}
	if e.hasUpper {
		switch e.code {
		case CodeBefore:
			return fmt.Sprintf("must be before %v", e.upper.value)
		case CodeBeforeOrEqual:
			return fmt.Sprintf("must be at or before %v", e.upper.value)
		}
		if e.upper.inclusive {
			return fmt.Sprintf("must be at most %v", e.upper.value)
		}
		return fmt.Sprintf("must be less than %v", e.upper.value)
	}
	return string(e.code)
}

func (e *LengthError) failureMessage() string {
	if !e.hasMaximum {
		return fmt.Sprintf("length must be at least %d %s (got %d)", e.minimum, lengthUnit(e.unit, e.minimum), e.actual)
	}
	if e.minimum == e.maximum {
		return fmt.Sprintf("length must be %d %s (got %d)", e.minimum, lengthUnit(e.unit, e.minimum), e.actual)
	}
	if e.code == CodeMaxLength || e.code == CodeByteMaxLength || e.code == CodeRuneMaxLength {
		return fmt.Sprintf("length must be at most %d %s (got %d)", e.maximum, lengthUnit(e.unit, e.maximum), e.actual)
	}
	return fmt.Sprintf("length must be between %d and %d %s (got %d)", e.minimum, e.maximum, lengthUnit(e.unit, e.maximum), e.actual)
}

func (e *DuplicateError) failureMessage() string {
	return fmt.Sprintf("duplicates element at index %d", e.first)
}

func (e *IndexError) failureMessage() string {
	return fmt.Sprintf("index %d is out of range for length %d", e.index, e.length)
}

func lengthUnit(unit LengthUnit, count int) string {
	if count == 1 {
		if unit == LengthEntries {
			return "entry"
		}
		return strings.TrimSuffix(string(unit), "s")
	}
	return string(unit)
}

func (e *TextError) failureMessage() string {
	constraint := strconv.Quote(e.constraint)
	switch e.code {
	case CodeContains:
		return "must contain " + constraint
	case CodePrefix:
		return "must start with " + constraint
	case CodeSuffix:
		return "must end with " + constraint
	case CodeMatch:
		return "must match pattern " + constraint
	case CodeTime:
		return "must match time layout " + constraint
	default:
		return string(e.code)
	}
}

func (e *MultipleOfError[T]) failureMessage() string {
	if e.hasTolerance {
		return fmt.Sprintf("must be a finite multiple of %v within tolerance %v", e.base, e.tolerance)
	}
	return fmt.Sprintf("must be a multiple of %v", e.base)
}
