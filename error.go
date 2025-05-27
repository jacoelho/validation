package validation

import (
	"fmt"
	"sort"
	"strings"
)

// Error represents a single validation error.
type Error struct {
	Field  string
	Code   string
	Params map[string]any
	Fatal  bool
}

// Error implements the error interface.
func (e *Error) Error() string {
	var sb strings.Builder
	sb.WriteString(e.Code)

	if e.Field != "" {
		sb.WriteString(" (field: ")
		sb.WriteString(e.Field)
		sb.WriteString(")")
	}

	if len(e.Params) > 0 {
		sb.WriteString(" {")
		keys := make([]string, 0, len(e.Params))
		for k := range e.Params {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for i, k := range keys {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("%s: %v", k, e.Params[k]))
		}
		sb.WriteString("}")
	}

	return sb.String()
}

// Errors is a collection of validation errors.
type Errors []*Error

// SingleErrorSlice creates a new Errors instance with a single error.
func SingleErrorSlice(field, code string, params map[string]any, fatal bool) Errors {
	return []*Error{
		{
			Field:  field,
			Code:   code,
			Params: params,
			Fatal:  fatal,
		},
	}
}

// HasErrors reports whether any errors exist.
func (errs Errors) HasErrors() bool {
	return len(errs) > 0
}

// HasFatalErrors reports whether any fatal errors exist.
func (errs Errors) HasFatalErrors() bool {
	if len(errs) == 0 {
		return false
	}
	for _, e := range errs {
		if e.Fatal {
			return true
		}
	}
	return false
}

// Format formats the errors using the given function and separator.
func (errs Errors) Format(f func(e *Error) string, sep string) string {
	switch len(errs) {
	case 0:
		return ""
	case 1:
		return f(errs[0])
	default:
		sb := new(strings.Builder)
		for i, e := range errs {
			if i > 0 {
				sb.WriteString(sep)
			}
			sb.WriteString(f(e))
		}
		return sb.String()
	}
}

// Error returns a concatenated string of all errors.
func (errs Errors) Error() string {
	return errs.Format(func(e *Error) string { return e.Error() }, "; ")
}
