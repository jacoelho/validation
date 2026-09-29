package validation

import "math"

// Signed is the set of signed integer types accepted by numeric rules.
type Signed interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

// Unsigned is the set of unsigned integer types accepted by numeric rules.
type Unsigned interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

// Integer is the set of integer types accepted by numeric rules.
type Integer interface {
	Signed | Unsigned
}

// Float is the set of floating-point types accepted by numeric rules.
type Float interface {
	~float32 | ~float64
}

// Number is the set of numeric types accepted by numeric rules.
type Number interface {
	Integer | Float
}

func numberBound[T Number](constructor string, value T) {
	if value != value { // The only Number value that is not equal to itself.
		configurationError(constructor)
	}
}

func numberRange[T Number](constructor string, min, max T) {
	if min != min || max != max || min > max {
		configurationError(constructor)
	}
}

func numberNaN[T Number](value T) bool { return value != value }

func lowerBound[T any](code Code, value T, inclusive bool) error {
	return newBoundsError(code, &bound[T]{value: value, inclusive: inclusive}, nil)
}

func upperBound[T any](code Code, value T, inclusive bool) error {
	return newBoundsError(code, nil, &bound[T]{value: value, inclusive: inclusive})
}

func rangeBounds[T any](code Code, min, max T) error {
	return newBoundsError(
		code,
		&bound[T]{value: min, inclusive: true},
		&bound[T]{value: max, inclusive: true},
	)
}

// Min validates that value is greater than or equal to min.
func Min[T Number](min T) Rule[T] {
	numberBound("Min", min)
	return func(value T) error {
		if numberNaN(value) || value < min {
			return lowerBound(CodeMin, min, true)
		}
		return nil
	}
}

// Max validates that value is less than or equal to max.
func Max[T Number](max T) Rule[T] {
	numberBound("Max", max)
	return func(value T) error {
		if numberNaN(value) || value > max {
			return upperBound(CodeMax, max, true)
		}
		return nil
	}
}

// Between validates that value is in the inclusive range [min, max].
func Between[T Number](min, max T) Rule[T] {
	numberRange("Between", min, max)
	return func(value T) error {
		if numberNaN(value) || value < min || value > max {
			return rangeBounds(CodeBetween, min, max)
		}
		return nil
	}
}

// GreaterThan validates that value is strictly greater than min.
func GreaterThan[T Number](min T) Rule[T] {
	numberBound("GreaterThan", min)
	return func(value T) error {
		if numberNaN(value) || value <= min {
			return lowerBound(CodeGreaterThan, min, false)
		}
		return nil
	}
}

// LessThan validates that value is strictly less than max.
func LessThan[T Number](max T) Rule[T] {
	numberBound("LessThan", max)
	return func(value T) error {
		if numberNaN(value) || value >= max {
			return upperBound(CodeLessThan, max, false)
		}
		return nil
	}
}

// Positive validates that value is strictly greater than zero.
func Positive[T Number]() Rule[T] {
	return func(value T) error {
		var zero T
		if numberNaN(value) || value <= zero {
			return NewViolation(CodePositive, nil)
		}
		return nil
	}
}

// NonNegative validates that value is greater than or equal to zero.
func NonNegative[T Number]() Rule[T] {
	return func(value T) error {
		var zero T
		if numberNaN(value) || value < zero {
			return NewViolation(CodeNonNegative, nil)
		}
		return nil
	}
}

// Negative validates that value is strictly less than zero.
func Negative[T Number]() Rule[T] {
	return func(value T) error {
		var zero T
		if numberNaN(value) || value >= zero {
			return NewViolation(CodeNegative, nil)
		}
		return nil
	}
}

// NonPositive validates that value is less than or equal to zero.
func NonPositive[T Number]() Rule[T] {
	return func(value T) error {
		var zero T
		if numberNaN(value) || value > zero {
			return NewViolation(CodeNonPositive, nil)
		}
		return nil
	}
}

// NotNaN rejects a NaN floating-point value. Infinities are valid.
func NotNaN[T Float]() Rule[T] {
	return func(value T) error {
		if value != value {
			return NewViolation(CodeNotNaN, nil)
		}
		return nil
	}
}

// Finite rejects NaN and both positive and negative infinity.
func Finite[T Float]() Rule[T] {
	return func(value T) error {
		if value != value || math.IsInf(float64(value), 0) {
			return NewViolation(CodeFinite, nil)
		}
		return nil
	}
}

// MinBy validates value >= min using a caller-supplied comparator.
func MinBy[T any](min T, compare func(T, T) int) Rule[T] {
	if compare == nil {
		configurationError("MinBy")
	}
	return func(value T) error {
		if compare(value, min) < 0 {
			return lowerBound(CodeMin, min, true)
		}
		return nil
	}
}

// MaxBy validates value <= max using a caller-supplied comparator.
func MaxBy[T any](max T, compare func(T, T) int) Rule[T] {
	if compare == nil {
		configurationError("MaxBy")
	}
	return func(value T) error {
		if compare(value, max) > 0 {
			return upperBound(CodeMax, max, true)
		}
		return nil
	}
}

// BetweenBy validates value in the inclusive range [min, max] using a
// caller-supplied comparator. Both value comparisons run for every input.
func BetweenBy[T any](min, max T, compare func(T, T) int) Rule[T] {
	if compare == nil {
		configurationError("BetweenBy")
	}
	if compare(min, max) > 0 {
		configurationError("BetweenBy")
	}
	return func(value T) error {
		below := compare(value, min) < 0
		above := compare(value, max) > 0
		if below || above {
			return rangeBounds(CodeBetween, min, max)
		}
		return nil
	}
}
