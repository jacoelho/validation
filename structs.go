package validation

// fieldValidator is a validator for a field of a struct.
type fieldValidator[T any] interface {
	ValidateWithPrefix(T, string) Errors
}

// StructValidator is a validator for a struct.
type StructValidator[T any] struct {
	fields []fieldValidator[T]
}

// NewStruct creates a new StructValidator with the given fields.
func NewStruct[T any](fields ...fieldValidator[T]) *StructValidator[T] {
	return &StructValidator[T]{fields: fields}
}

// Validate validates the given value.
func (v *StructValidator[T]) Validate(value T) Errors {
	return v.ValidateWithPrefix(value, "")
}

// ValidateWithPrefix validates the given value with a prefix.
func (v *StructValidator[T]) ValidateWithPrefix(value T, prefix string) Errors {
	var out Errors
	for _, field := range v.fields {
		out = append(out, field.ValidateWithPrefix(value, prefix)...)
	}
	return out
}

// FieldAccessor is a field of a struct.
type FieldAccessor[T, F any] struct {
	name  string
	get   func(T) F
	rules []Rule[F]
	inner fieldValidator[F]
}

// Field creates a new FieldAccessor with the given name, getter and rules.
func Field[T, F any](name string, getter func(T) F, rules ...Rule[F]) FieldAccessor[T, F] {
	return FieldAccessor[T, F]{name: name, get: getter, rules: rules}
}

// StructField creates a new FieldAccessor with the given name, getter and validator.
func StructField[T, F any](name string, getter func(T) F, validator *StructValidator[F]) FieldAccessor[T, F] {
	return FieldAccessor[T, F]{
		name:  name,
		get:   getter,
		inner: validator,
	}
}

// SliceField creates a new FieldAccessor with the given name, getter and rules.
func SliceField[T, E any](name string, getter func(T) []E, rules ...SliceRule[E]) FieldAccessor[T, []E] {
	return FieldAccessor[T, []E]{
		name:  name,
		get:   getter,
		inner: NewSliceValidator(rules...),
	}
}

// MapField creates a new FieldAccessor with the given name, getter and rules.
func MapField[T any, K comparable, V any](name string, getter func(T) map[K]V, rules ...MapRule[K, V]) FieldAccessor[T, map[K]V] {
	return FieldAccessor[T, map[K]V]{
		name:  name,
		get:   getter,
		inner: NewMapValidator(rules...),
	}
}

// ValidateWithPrefix validates the given value with a prefix.
func (fa FieldAccessor[T, F]) ValidateWithPrefix(parent T, prefix string) Errors {
	var out Errors
	value := fa.get(parent)

	fieldPath := fa.name
	if prefix != "" {
		fieldPath = prefix + "." + fa.name
	}

	if fa.inner != nil {
		for _, err := range fa.inner.ValidateWithPrefix(value, "") {
			err.Field = joinField(fieldPath, err.Field)
			out = append(out, err)
		}
		return out
	}

	for _, rule := range fa.rules {
		if err := rule(value); err != nil {
			err.Field = fieldPath
			out = append(out, err)
			if err.Fatal {
				break
			}
		}
	}
	return out
}

// joinField joins two field paths.
func joinField(base, child string) string {
	if base == "" {
		return child
	}
	if child == "" {
		return base
	}
	return base + "." + child
}
