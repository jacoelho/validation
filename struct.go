package validation

// Struct is the struct-oriented spelling of All; it uses the same engine.
func Struct[T any](rules ...Rule[T]) Rule[T] { return All(rules...) }

// Field projects a child rule onto one named parent field.
func (r Rule[F]) Field[P any](name string, get func(P) F) Rule[P] {
	if r == nil || name == "" || get == nil {
		configurationError("Field")
	}
	segment := Segment{Kind: FieldSegment, Name: name}
	return func(parent P) error {
		value := get(parent)
		return at(segment, r(value))
	}
}

// Project adapts a child rule to a parent without adding a path segment.
func (r Rule[V]) Project[P any](get func(P) V) Rule[P] {
	if r == nil || get == nil {
		configurationError("Project")
	}
	return func(parent P) error { return r(get(parent)) }
}

// Field accepts several child rules and delegates projection to their composite.
func Field[P, F any](name string, get func(P) F, rules ...Rule[F]) Rule[P] {
	return allConfigured("Field", rules).Field(name, get)
}

// Project accepts several child rules and delegates projection to their composite.
func Project[P, V any](get func(P) V, rules ...Rule[V]) Rule[P] {
	return allConfigured("Project", rules).Project(get)
}

// OptionalPtr skips children when the pointer is absent.
func OptionalPtr[T any](rules ...Rule[T]) Rule[*T] {
	children := allConfigured("OptionalPtr", rules)
	return func(value *T) error {
		if value == nil {
			return nil
		}
		return children(*value)
	}
}

// RequiredPtr reports absence and otherwise evaluates its children.
func RequiredPtr[T any](rules ...Rule[T]) Rule[*T] {
	children := allConfigured("RequiredPtr", rules)
	return func(value *T) error {
		if value == nil {
			return NewViolation(CodeRequired, nil)
		}
		return children(*value)
	}
}

// OptionalValue evaluates the child only when the parent getter reports presence.
func (r Rule[V]) OptionalValue[P any](get func(P) (V, bool)) Rule[P] {
	if r == nil || get == nil {
		configurationError("OptionalValue")
	}
	return func(parent P) error {
		value, present := get(parent)
		if !present {
			return nil
		}
		return r(value)
	}
}

// RequiredValue reports absence or evaluates the present child value.
func (r Rule[V]) RequiredValue[P any](get func(P) (V, bool)) Rule[P] {
	if r == nil || get == nil {
		configurationError("RequiredValue")
	}
	return func(parent P) error {
		value, present := get(parent)
		if !present {
			return NewViolation(CodeRequired, nil)
		}
		return r(value)
	}
}

// OptionalValue accepts several children and delegates presence to their composite.
func OptionalValue[P, V any](get func(P) (V, bool), rules ...Rule[V]) Rule[P] {
	return allConfigured("OptionalValue", rules).OptionalValue(get)
}

// RequiredValue accepts several children and delegates presence to their composite.
func RequiredValue[P, V any](get func(P) (V, bool), rules ...Rule[V]) Rule[P] {
	return allConfigured("RequiredValue", rules).RequiredValue(get)
}
