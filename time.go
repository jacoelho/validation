package validation

import "time"

// TimeBefore validates that value is strictly before other.
func TimeBefore(other time.Time) Rule[time.Time] {
	return func(value time.Time) error {
		if !value.Before(other) {
			return upperBound(CodeBefore, other, false)
		}
		return nil
	}
}

// TimeBeforeOrEqual validates that value is before or equal to other.
func TimeBeforeOrEqual(other time.Time) Rule[time.Time] {
	return func(value time.Time) error {
		if value.After(other) {
			return upperBound(CodeBeforeOrEqual, other, true)
		}
		return nil
	}
}

// TimeAfter validates that value is strictly after other.
func TimeAfter(other time.Time) Rule[time.Time] {
	return func(value time.Time) error {
		if !value.After(other) {
			return lowerBound(CodeAfter, other, false)
		}
		return nil
	}
}

// TimeAfterOrEqual validates that value is after or equal to other.
func TimeAfterOrEqual(other time.Time) Rule[time.Time] {
	return func(value time.Time) error {
		if value.Before(other) {
			return lowerBound(CodeAfterOrEqual, other, true)
		}
		return nil
	}
}

// TimeBetween validates that value is in the inclusive instant interval
// [min, max].
func TimeBetween(min, max time.Time) Rule[time.Time] {
	if min.After(max) {
		configurationError("TimeBetween")
	}
	return func(value time.Time) error {
		if value.Before(min) || value.After(max) {
			return rangeBounds(CodeBetween, min, max)
		}
		return nil
	}
}

// TimeNotZero rejects the zero time using time.Time.IsZero.
func TimeNotZero() Rule[time.Time] {
	return func(value time.Time) error {
		if value.IsZero() {
			return NewViolation(CodeNotZero, nil)
		}
		return nil
	}
}
