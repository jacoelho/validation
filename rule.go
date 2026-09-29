// Package validation composes typed rules and reports structured failures.
package validation

// Rule is the synchronous, typed validation contract.
type Rule[T any] func(T) error

func (r Rule[T]) Validate(value T) error { return r(value) }

func checkRules[T any](constructor string, rules []Rule[T]) []Rule[T] {
	for _, rule := range rules {
		if rule == nil {
			configurationError(constructor)
		}
	}
	return append([]Rule[T](nil), rules...)
}

// All evaluates every configured rule in declaration order.
func All[T any](rules ...Rule[T]) Rule[T] { return allConfigured("All", rules) }

func allConfigured[T any](constructor string, rules []Rule[T]) Rule[T] {
	configured := checkRules(constructor, rules)
	return func(value T) error {
		var failures []error
		for _, rule := range configured {
			failures = appendFailure(failures, rule(value))
		}
		return combine(failures)
	}
}

// When evaluates its child group when the predicate holds.
func When[T any](predicate func(T) bool, rules ...Rule[T]) Rule[T] {
	if predicate == nil {
		configurationError("When")
	}
	children := allConfigured("When", rules)
	return func(value T) error {
		if !predicate(value) {
			return nil
		}
		return children(value)
	}
}

// Unless evaluates its child group when the predicate does not hold.
func Unless[T any](predicate func(T) bool, rules ...Rule[T]) Rule[T] {
	if predicate == nil {
		configurationError("Unless")
	}
	children := allConfigured("Unless", rules)
	return func(value T) error {
		if predicate(value) {
			return nil
		}
		return children(value)
	}
}

// Check creates one constraint from a predicate and a lazy failure factory.
func Check[T any](predicate func(T) bool, failure func(T) error) Rule[T] {
	if predicate == nil || failure == nil {
		configurationError("Check")
	}
	return func(value T) error {
		if predicate(value) {
			return nil
		}
		err := failure(value)
		if err == nil {
			configurationError("Check")
		}
		return err
	}
}

func Equal[T comparable](want T) Rule[T] {
	return func(value T) error {
		if value == want {
			return nil
		}
		return NewViolation(CodeEqual, nil)
	}
}
func NotEqual[T comparable](other T) Rule[T] {
	return func(value T) error {
		if value != other {
			return nil
		}
		return NewViolation(CodeNotEqual, nil)
	}
}
func Zero[T comparable]() Rule[T] {
	var zero T
	return func(value T) error {
		if value == zero {
			return nil
		}
		return NewViolation(CodeZero, nil)
	}
}
func NotZero[T comparable]() Rule[T] {
	var zero T
	return func(value T) error {
		if value != zero {
			return nil
		}
		return NewViolation(CodeNotZero, nil)
	}
}
func OneOf[T comparable](allowed ...T) Rule[T] {
	set := make(map[T]struct{}, len(allowed))
	for _, value := range allowed {
		set[value] = struct{}{}
	}
	return func(value T) error {
		if _, ok := set[value]; ok {
			return nil
		}
		return NewViolation(CodeOneOf, nil)
	}
}
func NotOneOf[T comparable](forbidden ...T) Rule[T] {
	set := make(map[T]struct{}, len(forbidden))
	for _, value := range forbidden {
		set[value] = struct{}{}
	}
	return func(value T) error {
		if _, ok := set[value]; !ok {
			return nil
		}
		return NewViolation(CodeNotOneOf, nil)
	}
}
