package validation

func (e *Violation) Parameters() map[string]any { return nil }

func (e *TextError) Parameters() map[string]any {
	return map[string]any{"constraint": e.constraint}
}

func (e *MultipleOfError[T]) Parameters() map[string]any {
	parameters := map[string]any{"base": e.base}
	if e.hasTolerance {
		parameters["tolerance"] = e.tolerance
	}
	return parameters
}

func (e *BoundsError[T]) Parameters() map[string]any {
	parameters := make(map[string]any)
	if e.hasLower {
		parameters["minimum"] = e.lower.value
		parameters["minimum_inclusive"] = e.lower.inclusive
	}
	if e.hasUpper {
		parameters["maximum"] = e.upper.value
		parameters["maximum_inclusive"] = e.upper.inclusive
	}
	return parameters
}

func (e *LengthError) Parameters() map[string]any {
	parameters := map[string]any{"actual": e.actual, "minimum": e.minimum, "unit": e.unit}
	if e.hasMaximum {
		parameters["maximum"] = e.maximum
	}
	return parameters
}

func (e *DuplicateError) Parameters() map[string]any {
	return map[string]any{"first_index": e.first}
}

func (e *IndexError) Parameters() map[string]any {
	return map[string]any{"index": e.index, "length": e.length}
}
