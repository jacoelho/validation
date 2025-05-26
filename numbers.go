package validation

import (
	"cmp"
)

// NumbersMin validates that the value is greater than or equal to the given minimum.
func NumbersMin[T cmp.Ordered](min T) Rule[T] {
	return func(value T) *Error {
		if value < min {
			return &Error{
				Code:   "min",
				Params: map[string]any{"min": min, "actual": value},
			}
		}
		return nil
	}
}

// NumbersMax validates that the value is less than or equal to the given maximum.
func NumbersMax[T cmp.Ordered](max T) Rule[T] {
	return func(value T) *Error {
		if value > max {
			return &Error{
				Code:   "max",
				Params: map[string]any{"max": max, "actual": value},
			}
		}
		return nil
	}
}

// NumbersBetween validates that the value is between the given minimum and maximum.
func NumbersBetween[T cmp.Ordered](min, max T) Rule[T] {
	return func(value T) *Error {
		if value < min || value > max {
			return &Error{
				Code:   "between",
				Params: map[string]any{"min": min, "max": max, "actual": value},
			}
		}
		return nil
	}
}

// NumbersPositive validates that the value is greater than 0.
func NumbersPositive[T cmp.Ordered]() Rule[T] {
	return func(value T) *Error {
		var zero T
		if cmp.Compare(value, zero) < 0 {
			return &Error{Code: "positive", Params: map[string]any{"value": value}}
		}
		return nil
	}
}

// NumbersNegative validates that the value is less than 0.
func NumbersNegative[T cmp.Ordered]() Rule[T] {
	return func(value T) *Error {
		var zero T
		if cmp.Compare(value, zero) > 0 {
			return &Error{Code: "negative", Params: map[string]any{"value": value}}
		}
		return nil
	}
}
