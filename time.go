package validation

import "time"

// TimeBefore validates that the time is before the given time.
func TimeBefore(other time.Time) Rule[time.Time] {
	return func(value time.Time) *Error {
		if value.After(other) {
			return &Error{Code: "before", Params: map[string]any{"value": other}}
		}
		return nil
	}
}

// TimeAfter validates that the time is after the given time.
func TimeAfter(other time.Time) Rule[time.Time] {
	return func(value time.Time) *Error {
		if value.Before(other) {
			return &Error{Code: "after", Params: map[string]any{"value": other}}
		}
		return nil
	}
}

// TimeBetween validates that the time is between the given times.
func TimeBetween(min, max time.Time) Rule[time.Time] {
	return func(value time.Time) *Error {
		if value.Before(min) || value.After(max) {
			return &Error{Code: "between", Params: map[string]any{"min": min, "max": max, "value": value}}
		}
		return nil
	}
}
