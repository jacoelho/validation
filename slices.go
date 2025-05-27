package validation

import (
	"slices"
	"strconv"
)

// SliceRule is a function that validates a slice of values.
type SliceRule[T any] func(values []T) Errors

// SliceValidator is a validator for slices of values.
type SliceValidator[T any] struct {
	rules []SliceRule[T]
}

// Slices creates a new SliceValidator with the given rules.
func Slices[T any](rules ...SliceRule[T]) *SliceValidator[T] {
	return &SliceValidator[T]{rules: rules}
}

// Validate validates the given values.
func (v *SliceValidator[T]) Validate(values []T) Errors {
	return v.ValidateWithPrefix(values, "")
}

// ValidateWithPrefix validates the given values with a prefix.
func (v *SliceValidator[T]) ValidateWithPrefix(values []T, prefix string) Errors {
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

// SlicesMinLength validates that the slice has at least the given length.
func SlicesMinLength[T any](min int) SliceRule[T] {
	return func(values []T) Errors {
		if len(values) < min {
			return SingleErrorSlice("", "min", map[string]any{"min": min, "actual": len(values)}, false)
		}
		return nil
	}
}

// SlicesMaxLength validates that the slice has at most the given length.
func SlicesMaxLength[T any](max int) SliceRule[T] {
	return func(values []T) Errors {
		if len(values) > max {
			return SingleErrorSlice("", "max", map[string]any{"max": max, "actual": len(values)}, false)
		}
		return nil
	}
}

// SlicesInBetweenLength validates that the slice has between the given lengths.
func SlicesInBetweenLength[T any](min, max int) SliceRule[T] {
	return func(values []T) Errors {
		if len(values) < min || len(values) > max {
			return SingleErrorSlice("", "between", map[string]any{"min": min, "max": max, "actual": len(values)}, false)
		}
		return nil
	}
}

// SlicesLength validates that the slice has the given length.
func SlicesLength[T any](length int) SliceRule[T] {
	return func(values []T) Errors {
		if len(values) != length {
			return SingleErrorSlice("", "length", map[string]any{"length": length, "actual": len(values)}, false)
		}
		return nil
	}
}

// SlicesForEach validates each value in the slice using the given rules.
func SlicesForEach[T any](rules ...Rule[T]) SliceRule[T] {
	return func(values []T) Errors {
		var errs Errors
		for i, v := range values {
			for _, rule := range rules {
				if err := rule(v); err != nil {
					err.Field = strconv.Itoa(i)
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

// SlicesUnique validates that the slice has unique values.
func SlicesUnique[T comparable]() SliceRule[T] {
	return func(values []T) Errors {
		seen := make(map[T]struct{})
		for i, v := range values {
			if _, ok := seen[v]; ok {
				return SingleErrorSlice(strconv.Itoa(i), "unique", nil, false)
			}
			seen[v] = struct{}{}
		}
		return nil
	}
}

// SlicesContains validates that the slice contains the given value.
func SlicesContains[T comparable](value T) SliceRule[T] {
	return func(values []T) Errors {
		if slices.Contains(values, value) {
			return nil
		}
		return SingleErrorSlice("", "contains", map[string]any{"value": value}, false)
	}
}

// SlicesOneOf validates that the slice contains only the given values.
func SlicesOneOf[T comparable](allowed ...T) SliceRule[T] {
	set := make(map[T]struct{}, len(allowed))
	for _, v := range allowed {
		set[v] = struct{}{}
	}
	return func(values []T) Errors {
		for i, v := range values {
			if _, ok := set[v]; !ok {
				return SingleErrorSlice(strconv.Itoa(i), "one_of", map[string]any{"value": v}, false)
			}
		}
		return nil
	}
}

// SlicesNotOneOf validates that the slice does not contain the given values.
func SlicesNotOneOf[T comparable](disallowed ...T) SliceRule[T] {
	set := make(map[T]struct{}, len(disallowed))
	for _, v := range disallowed {
		set[v] = struct{}{}
	}
	return func(values []T) Errors {
		for i, v := range values {
			if _, ok := set[v]; ok {
				return SingleErrorSlice(strconv.Itoa(i), "not_one_of", map[string]any{"value": v}, false)
			}
		}
		return nil
	}
}

// SlicesAtIndex validates the value at the given index.
func SlicesAtIndex[T any](index int, rules ...Rule[T]) SliceRule[T] {
	return func(values []T) Errors {
		if index < 0 || index >= len(values) {
			return SingleErrorSlice(strconv.Itoa(index), "index", map[string]any{"index": index}, false)
		}
		v := values[index]
		var errs Errors
		for _, rule := range rules {
			err := rule(v)
			if err != nil {
				err.Field = strconv.Itoa(index)
				errs = append(errs, err)
				if err.Fatal {
					return errs
				}
			}
		}
		return errs
	}
}
