package validation

import (
	"fmt"
)

// MapRule is a function that validates a map of values.
type MapRule[K comparable, V any] func(values map[K]V) Errors

// MapEntryRule is a function that validates an entry in a map.
type MapEntryRule[K comparable, V any] func(key K, value V) *Error

// MapValidator is a validator for maps of values.
type MapValidator[K comparable, V any] struct {
	rules []MapRule[K, V]
}

// Maps creates a new MapValidator with the given rules.
func Maps[K comparable, V any](rules ...MapRule[K, V]) *MapValidator[K, V] {
	return &MapValidator[K, V]{rules: rules}
}

// Validate validates the given values.
func (v *MapValidator[K, V]) Validate(values map[K]V) Errors {
	return v.ValidateWithPrefix(values, "")
}

// ValidateWithPrefix validates the given values with a prefix.
func (v *MapValidator[K, V]) ValidateWithPrefix(values map[K]V, prefix string) Errors {
	var out Errors
	for _, rule := range v.rules {
		ruleErrs := rule(values)
		for _, err := range ruleErrs {
			if err.Field != "" {
				err.Field = joinField(prefix, err.Field)
			} else {
				err.Field = prefix
			}
			out = append(out, err)
			if err.Fatal {
				return out
			}
		}
	}
	return out
}

// MapsForEach validates each entry in the map using the given rules.
func MapsForEach[K comparable, V any](rules ...MapEntryRule[K, V]) MapRule[K, V] {
	return func(values map[K]V) Errors {
		var errs Errors
		for k, v := range values {
			for _, rule := range rules {
				if err := rule(k, v); err != nil {
					err.Field = fmt.Sprintf("%v", k)
					errs = append(errs, err)
					if err.Fatal {
						return errs
					}
				}
			}
		}
		return errs
	}
}

// MapsMinKeys validates that the map has at least the given number of keys.
func MapsMinKeys[K comparable, V any](min int) MapRule[K, V] {
	return func(values map[K]V) Errors {
		if len(values) < min {
			return SingleErrorSlice("", "min", map[string]any{"min": min, "actual": len(values)}, false)
		}
		return nil
	}
}

// MapsMaxKeys validates that the map has at most the given number of keys.
func MapsMaxKeys[K comparable, V any](max int) MapRule[K, V] {
	return func(values map[K]V) Errors {
		if len(values) > max {
			return SingleErrorSlice("", "max", map[string]any{"max": max, "actual": len(values)}, false)
		}
		return nil
	}
}

func MapsLength[K comparable, V any](length int) MapRule[K, V] {
	return func(values map[K]V) Errors {
		if len(values) != length {
			return SingleErrorSlice("", "length", map[string]any{"length": length, "actual": len(values)}, false)
		}
		return nil
	}
}

func MapsLengthBetween[K comparable, V any](min, max int) MapRule[K, V] {
	return func(values map[K]V) Errors {
		if len(values) < min || len(values) > max {
			return SingleErrorSlice("", "between", map[string]any{"min": min, "max": max, "actual": len(values)}, false)
		}
		return nil
	}
}

// MapsKeysOneOf validates that the map has only the given keys.
func MapsKeysOneOf[K comparable, V any](allowed ...K) MapRule[K, V] {
	set := make(map[K]struct{}, len(allowed))
	for _, v := range allowed {
		set[v] = struct{}{}
	}
	return func(m map[K]V) Errors {
		for k := range m {
			if _, ok := set[k]; !ok {
				return SingleErrorSlice("", "one_of", map[string]any{"value": k}, false)
			}
		}
		return nil
	}
}

// MapsKeysNotOneOf validates that the map does not have the given keys.
func MapsKeysNotOneOf[K comparable, V any](disallowed ...K) MapRule[K, V] {
	set := make(map[K]struct{}, len(disallowed))
	for _, v := range disallowed {
		set[v] = struct{}{}
	}
	return func(m map[K]V) Errors {
		for k := range m {
			if _, ok := set[k]; ok {
				return SingleErrorSlice("", "not_one_of", map[string]any{"value": k}, false)
			}
		}
		return nil
	}
}

// MapsValuesOneOf validates that the map has only the given values.
func MapsValuesOneOf[K comparable, V comparable](allowed ...V) MapRule[K, V] {
	set := make(map[V]struct{}, len(allowed))
	for _, v := range allowed {
		set[v] = struct{}{}
	}
	return func(m map[K]V) Errors {
		for _, v := range m {
			if _, ok := set[v]; !ok {
				return SingleErrorSlice("", "one_of", map[string]any{"value": v}, false)
			}
		}
		return nil
	}
}

// MapsValuesNotOneOf validates that the map does not have the given values.
func MapsValuesNotOneOf[K comparable, V comparable](disallowed ...V) MapRule[K, V] {
	set := make(map[V]struct{}, len(disallowed))
	for _, v := range disallowed {
		set[v] = struct{}{}
	}
	return func(m map[K]V) Errors {
		for _, v := range m {
			if _, ok := set[v]; ok {
				return SingleErrorSlice("", "not_one_of", map[string]any{"value": v}, false)
			}
		}
		return nil
	}
}

// MapsKey validates the value of the given key.
func MapsKey[K comparable, V any](key K, rules ...Rule[V]) MapRule[K, V] {
	return func(m map[K]V) Errors {
		v, ok := m[key]
		if !ok {
			return SingleErrorSlice("", "not_found", map[string]any{"key": key}, false)
		}
		var errs Errors
		for _, rule := range rules {
			err := rule(v)
			if err != nil {
				err.Field = fmt.Sprintf("%v", key)
				errs = append(errs, err)
				if err.Fatal {
					return errs
				}
			}
		}
		return errs
	}
}
