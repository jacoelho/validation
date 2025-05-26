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

// NewMapValidator creates a new MapValidator with the given rules.
func NewMapValidator[K comparable, V any](rules ...MapRule[K, V]) *MapValidator[K, V] {
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
			return NewErrors("", "min", map[string]any{"min": min, "actual": len(values)}, false)
		}
		return nil
	}
}

// MapsMaxKeys validates that the map has at most the given number of keys.
func MapsMaxKeys[K comparable, V any](max int) MapRule[K, V] {
	return func(values map[K]V) Errors {
		if len(values) > max {
			return NewErrors("", "max", map[string]any{"max": max, "actual": len(values)}, false)
		}
		return nil
	}
}

func MapsLength[K comparable, V any](length int) MapRule[K, V] {
	return func(values map[K]V) Errors {
		if len(values) != length {
			return NewErrors("", "length", map[string]any{"length": length, "actual": len(values)}, false)
		}
		return nil
	}
}

func MapsLengthBetween[K comparable, V any](min, max int) MapRule[K, V] {
	return func(values map[K]V) Errors {
		if len(values) < min || len(values) > max {
			return NewErrors("", "between", map[string]any{"min": min, "max": max, "actual": len(values)}, false)
		}
		return nil
	}
}

// MapsKeysAllowed validates that the map has only the given keys.
func MapsKeysAllowed[K comparable, V any](allowed ...K) MapRule[K, V] {
	allowedSet := make(map[K]struct{}, len(allowed))
	for _, v := range allowed {
		allowedSet[v] = struct{}{}
	}
	return func(m map[K]V) Errors {
		for k := range m {
			if _, ok := allowedSet[k]; !ok {
				return NewErrors("", "allowed", map[string]any{"value": k}, false)
			}
		}
		return nil
	}
}

// MapsValuesAllowed validates that the map has only the given values.
func MapsValuesAllowed[K comparable, V comparable](allowed ...V) MapRule[K, V] {
	allowedSet := make(map[V]struct{}, len(allowed))
	for _, v := range allowed {
		allowedSet[v] = struct{}{}
	}
	return func(m map[K]V) Errors {
		for _, v := range m {
			if _, ok := allowedSet[v]; !ok {
				return NewErrors("", "allowed", map[string]any{"value": v}, false)
			}
		}
		return nil
	}
}

// MapsValuesDisallowed validates that the map does not have the given values.
func MapsValuesDisallowed[K comparable, V comparable](disallowed ...V) MapRule[K, V] {
	disallowedSet := make(map[V]struct{}, len(disallowed))
	for _, v := range disallowed {
		disallowedSet[v] = struct{}{}
	}
	return func(m map[K]V) Errors {
		for _, v := range m {
			if _, ok := disallowedSet[v]; ok {
				return NewErrors("", "disallowed", map[string]any{"value": v}, false)
			}
		}
		return nil
	}
}

// MapsKeysDisallowed validates that the map does not have the given keys.
func MapsKeysDisallowed[K comparable, V any](disallowed ...K) MapRule[K, V] {
	disallowedSet := make(map[K]struct{}, len(disallowed))
	for _, v := range disallowed {
		disallowedSet[v] = struct{}{}
	}
	return func(m map[K]V) Errors {
		for k := range m {
			if _, ok := disallowedSet[k]; ok {
				return NewErrors("", "disallowed", map[string]any{"value": k}, false)
			}
		}
		return nil
	}
}
