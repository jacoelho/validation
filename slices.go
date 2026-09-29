package validation

// SliceLength requires a slice to contain exactly n elements.
func SliceLength[S ~[]E, E any](n int) Rule[S] {
	if n < 0 {
		configurationError("SliceLength")
	}
	return func(values S) error {
		if len(values) == n {
			return nil
		}
		return newLengthError(CodeLength, len(values), n, &n, LengthElements)
	}
}

// SliceMinLength requires a slice to contain at least min elements.
func SliceMinLength[S ~[]E, E any](min int) Rule[S] {
	if min < 0 {
		configurationError("SliceMinLength")
	}
	return func(values S) error {
		if len(values) >= min {
			return nil
		}
		return newLengthError(CodeMinLength, len(values), min, nil, LengthElements)
	}
}

// SliceMaxLength requires a slice to contain at most max elements.
func SliceMaxLength[S ~[]E, E any](max int) Rule[S] {
	if max < 0 {
		configurationError("SliceMaxLength")
	}
	return func(values S) error {
		if len(values) <= max {
			return nil
		}
		return newLengthError(CodeMaxLength, len(values), 0, &max, LengthElements)
	}
}

// SliceLengthBetween requires a slice length in the inclusive interval [min, max].
func SliceLengthBetween[S ~[]E, E any](min, max int) Rule[S] {
	if min < 0 || max < 0 || min > max {
		configurationError("SliceLengthBetween")
	}
	return func(values S) error {
		if len(values) >= min && len(values) <= max {
			return nil
		}
		return newLengthError(CodeLengthBetween, len(values), min, &max, LengthElements)
	}
}

// Each applies every child rule to every element, in index and declaration order.
func Each[S ~[]E, E any](rules ...Rule[E]) Rule[S] {
	children := checkRules("Each", rules)
	return func(values S) error {
		var failures []error
		for i := range values {
			var elementFailures []error
			for _, rule := range children {
				if err := rule(values[i]); err != nil {
					elementFailures = appendFailure(elementFailures, at(Segment{
						Kind:  IndexSegment,
						Index: i,
					}, err))
				}
			}
			failures = appendFailure(failures, combine(elementFailures))
		}
		return combine(failures)
	}
}

// AtIndex applies every child rule to one element. An absent index is a failure
// at that index and does not invoke the child rules.
func AtIndex[S ~[]E, E any](index int, rules ...Rule[E]) Rule[S] {
	if index < 0 {
		configurationError("AtIndex")
	}
	children := checkRules("AtIndex", rules)
	return func(values S) error {
		if index >= len(values) {
			return at(Segment{Kind: IndexSegment, Index: index}, newIndexError(index, len(values)))
		}
		var failures []error
		for _, rule := range children {
			if err := rule(values[index]); err != nil {
				failures = appendFailure(failures, at(Segment{Kind: IndexSegment, Index: index}, err))
			}
		}
		return combine(failures)
	}
}

// SliceContains requires at least one element equal to want.
func SliceContains[S ~[]E, E comparable](want E) Rule[S] {
	return func(values S) error {
		for _, value := range values {
			if value == want {
				return nil
			}
		}
		return NewViolation(CodeContains, nil)
	}
}

// SliceOneOf requires every element to be one of allowed. Each offending
// occurrence is reported independently at its index.
func SliceOneOf[S ~[]E, E comparable](allowed ...E) Rule[S] {
	set := make(map[E]struct{}, len(allowed))
	for _, value := range allowed {
		set[value] = struct{}{}
	}
	return func(values S) error {
		var failures []error
		for i, value := range values {
			if _, ok := set[value]; ok {
				continue
			}
			failures = appendFailure(failures, at(Segment{
				Kind:  IndexSegment,
				Index: i,
			}, NewViolation(CodeOneOf, nil)))
		}
		return combine(failures)
	}
}

// SliceNotOneOf rejects every element that is one of forbidden. Each offending
// occurrence is reported independently at its index.
func SliceNotOneOf[S ~[]E, E comparable](forbidden ...E) Rule[S] {
	set := make(map[E]struct{}, len(forbidden))
	for _, value := range forbidden {
		set[value] = struct{}{}
	}
	return func(values S) error {
		var failures []error
		for i, value := range values {
			if _, ok := set[value]; !ok {
				continue
			}
			failures = appendFailure(failures, at(Segment{
				Kind:  IndexSegment,
				Index: i,
			}, NewViolation(CodeNotOneOf, nil)))
		}
		return combine(failures)
	}
}

// SliceUnique rejects each duplicate occurrence. The diagnostic records the
// earliest equal element, while the issue location identifies the duplicate.
func SliceUnique[S ~[]E, E comparable]() Rule[S] {
	return func(values S) error {
		var failures []error
		for i := range values {
			first := -1
			for j := 0; j < i; j++ {
				if values[j] == values[i] {
					first = j
					break
				}
			}
			if first >= 0 {
				failures = appendFailure(failures, at(Segment{
					Kind:  IndexSegment,
					Index: i,
				}, newDuplicateError(first)))
			}
		}
		return combine(failures)
	}
}
