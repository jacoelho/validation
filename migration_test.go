package validation_test

import (
	"errors"
	"testing"

	v "github.com/jacoelho/validation/v2"
)

type legacyError struct{ cause error }

func (e *legacyError) Error() string { return "legacy" }
func (e *legacyError) Unwrap() error { return e.cause }

type legacyErrors []*legacyError

func (es legacyErrors) Error() string { return "legacy failures" }

func adaptLegacyPointer[T any](old func(T) *legacyError) v.Rule[T] {
	return func(value T) error {
		e := old(value)
		if e == nil {
			return nil
		}
		return e
	}
}
func adaptLegacySlice[T any](old func(T) legacyErrors) v.Rule[T] {
	return func(value T) error {
		es := old(value)
		if len(es) == 0 {
			return nil
		}
		var children []error
		for _, e := range es {
			if e != nil {
				children = append(children, e)
			}
		}
		return errors.Join(children...)
	}
}
func TestLegacyAdaptersNormalizeConcreteNil(t *testing.T) {
	pointer := adaptLegacyPointer(func(int) *legacyError { return nil })
	if got := pointer(0); got != nil {
		t.Fatalf("nil pointer = %v", got)
	}
	for _, empty := range []legacyErrors{nil, {}} {
		slice := adaptLegacySlice(func(int) legacyErrors { return empty })
		if got := slice(0); got != nil {
			t.Fatalf("empty slice = %v", got)
		}
	}
	cause := errors.New("cause")
	err := adaptLegacySlice(func(int) legacyErrors { return legacyErrors{{cause: cause}, {cause: errors.New("other")}} })(0)
	if !errors.Is(err, cause) || len(v.Issues(err)) != 2 {
		t.Fatalf("legacy occurrences/causes lost: %v", err)
	}
}
