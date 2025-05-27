package validation

// Rule is a function that validates a value.
type Rule[T any] func(value T) *Error

// RuleNot negates the rule.
func RuleNot[T any](rule Rule[T]) Rule[T] {
	return func(value T) *Error {
		if err := rule(value); err != nil {
			return nil
		}
		return &Error{
			Code: "not",
		}
	}
}

// RuleStopOnError stops the validation process if an error occurs.
func RuleStopOnError[T any](rule Rule[T]) Rule[T] {
	return func(value T) *Error {
		if err := rule(value); err != nil {
			err.Fatal = true
			return err
		}
		return nil
	}
}

// Or combines multiple rules, at least one must pass.
// If all rules fail, the last error is returned.
func Or[T any](rules ...Rule[T]) Rule[T] {
	return func(value T) *Error {
		var lastErr *Error
		for _, rule := range rules {
			if err := rule(value); err == nil {
				return nil
			} else {
				lastErr = err
			}
		}
		return lastErr
	}
}

// When applies a rule only if the condition is true.
func When[T any](condition func(T) bool, rule Rule[T]) Rule[T] {
	return func(value T) *Error {
		if condition(value) {
			return rule(value)
		}
		return nil
	}
}

// Unless applies a rule only if the condition is false
func Unless[T any](condition func(T) bool, rule Rule[T]) Rule[T] {
	return func(value T) *Error {
		if !condition(value) {
			return rule(value)
		}
		return nil
	}
}

// NotZero ensures the value is not the zero value for its type
func NotZero[T comparable]() Rule[T] {
	return func(value T) *Error {
		var zero T
		if value == zero {
			return &Error{
				Code: "zero",
			}
		}
		return nil
	}
}

// NotZeroable ensures the value is not the zero value.
// The zero value is determined by the IsZero method.
func NotZeroable[T interface{ IsZero() bool }]() Rule[T] {
	return func(value T) *Error {
		if value.IsZero() {
			return &Error{
				Code: "zero",
			}
		}
		return nil
	}
}

// OneOf validates that the value is one of the given allowed values.
func OneOf[T comparable](allowed ...T) Rule[T] {
	set := make(map[T]struct{}, len(allowed))
	for _, v := range allowed {
		set[v] = struct{}{}
	}
	return func(value T) *Error {
		if _, ok := set[value]; !ok {
			return &Error{
				Code:   "one_of",
				Params: map[string]any{"value": value},
			}
		}
		return nil
	}
}

// NotOneOf validates that the value is not one of the given disallowed values.
func NotOneOf[T comparable](disallowed ...T) Rule[T] {
	set := make(map[T]struct{}, len(disallowed))
	for _, v := range disallowed {
		set[v] = struct{}{}
	}
	return func(value T) *Error {
		if _, ok := set[value]; ok {
			return &Error{
				Code:   "not_one_of",
				Params: map[string]any{"value": value},
			}
		}
		return nil
	}
}
