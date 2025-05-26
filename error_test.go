package validation_test

import (
	"testing"

	"github.com/jacoelho/validation"
)

func TestError(t *testing.T) {
	tests := []struct {
		name     string
		err      *validation.Error
		expected string
	}{
		{
			name: "error with code only",
			err: &validation.Error{
				Code: "required",
			},
			expected: "required",
		},
		{
			name: "error with field",
			err: &validation.Error{
				Code:  "min",
				Field: "Age",
			},
			expected: "min (field: Age)",
		},
		{
			name: "error with params",
			err: &validation.Error{
				Code: "between",
				Params: map[string]any{
					"min":    18,
					"max":    120,
					"actual": 15,
				},
			},
			expected: "between {actual: 15, max: 120, min: 18}",
		},
		{
			name: "error with field and params",
			err: &validation.Error{
				Code:  "length",
				Field: "Name",
				Params: map[string]any{
					"min":    2,
					"actual": 1,
				},
			},
			expected: "length (field: Name) {actual: 1, min: 2}",
		},
		{
			name: "error with fatal flag",
			err: &validation.Error{
				Code:  "invalid",
				Field: "Password",
				Fatal: true,
			},
			expected: "invalid (field: Password)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestErrors(t *testing.T) {
	tests := []struct {
		name     string
		errs     validation.Errors
		expected string
	}{
		{
			name:     "empty errors",
			errs:     validation.Errors{},
			expected: "",
		},
		{
			name: "single error",
			errs: validation.Errors{
				{
					Code:  "required",
					Field: "Name",
				},
			},
			expected: "required (field: Name)",
		},
		{
			name: "multiple errors",
			errs: validation.Errors{
				{
					Code:  "required",
					Field: "Name",
				},
				{
					Code:  "min",
					Field: "Age",
					Params: map[string]any{
						"min":    18,
						"actual": 15,
					},
				},
			},
			expected: "required (field: Name); min (field: Age) {actual: 15, min: 18}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.errs.Error()
			if got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestErrorsHasErrors(t *testing.T) {
	tests := []struct {
		name     string
		errs     validation.Errors
		expected bool
	}{
		{
			name:     "empty errors",
			errs:     validation.Errors{},
			expected: false,
		},
		{
			name: "single error",
			errs: validation.Errors{
				{
					Code: "required",
				},
			},
			expected: true,
		},
		{
			name: "multiple errors",
			errs: validation.Errors{
				{
					Code: "required",
				},
				{
					Code: "min",
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.errs.HasErrors()
			if got != tt.expected {
				t.Errorf("HasErrors() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestErrorsHasFatalErrors(t *testing.T) {
	tests := []struct {
		name     string
		errs     validation.Errors
		expected bool
	}{
		{
			name:     "empty errors",
			errs:     validation.Errors{},
			expected: false,
		},
		{
			name: "non-fatal error",
			errs: validation.Errors{
				{
					Code: "required",
				},
			},
			expected: false,
		},
		{
			name: "fatal error",
			errs: validation.Errors{
				{
					Code:  "required",
					Fatal: true,
				},
			},
			expected: true,
		},
		{
			name: "mixed errors",
			errs: validation.Errors{
				{
					Code: "required",
				},
				{
					Code:  "min",
					Fatal: true,
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.errs.HasFatalErrors()
			if got != tt.expected {
				t.Errorf("HasFatalErrors() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestErrorsFormat(t *testing.T) {
	tests := []struct {
		name     string
		errs     validation.Errors
		format   func(*validation.Error) string
		sep      string
		expected string
	}{
		{
			name:     "empty errors",
			errs:     validation.Errors{},
			format:   func(e *validation.Error) string { return e.Code },
			sep:      ", ",
			expected: "",
		},
		{
			name: "single error with custom format",
			errs: validation.Errors{
				{
					Code:  "required",
					Field: "Name",
				},
			},
			format:   func(e *validation.Error) string { return e.Field + ": " + e.Code },
			sep:      ", ",
			expected: "Name: required",
		},
		{
			name: "multiple errors with custom format",
			errs: validation.Errors{
				{
					Code:  "required",
					Field: "Name",
				},
				{
					Code:  "min",
					Field: "Age",
				},
			},
			format:   func(e *validation.Error) string { return e.Field + ": " + e.Code },
			sep:      " | ",
			expected: "Name: required | Age: min",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.errs.Format(tt.format, tt.sep)
			if got != tt.expected {
				t.Errorf("Format() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestNewErrors(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		code     string
		params   map[string]any
		fatal    bool
		expected validation.Errors
	}{
		{
			name:  "basic error",
			field: "Name",
			code:  "required",
			params: map[string]any{
				"value": "",
			},
			fatal: false,
			expected: validation.Errors{
				{
					Field:  "Name",
					Code:   "required",
					Params: map[string]any{"value": ""},
					Fatal:  false,
				},
			},
		},
		{
			name:  "fatal error",
			field: "Password",
			code:  "invalid",
			params: map[string]any{
				"reason": "too short",
			},
			fatal: true,
			expected: validation.Errors{
				{
					Field:  "Password",
					Code:   "invalid",
					Params: map[string]any{"reason": "too short"},
					Fatal:  true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validation.NewErrors(tt.field, tt.code, tt.params, tt.fatal)
			if len(got) != len(tt.expected) {
				t.Errorf("NewErrors() returned %d errors, want %d", len(got), len(tt.expected))
				return
			}

			err := got[0]
			expected := tt.expected[0]

			if err.Field != expected.Field {
				t.Errorf("Field = %q, want %q", err.Field, expected.Field)
			}
			if err.Code != expected.Code {
				t.Errorf("Code = %q, want %q", err.Code, expected.Code)
			}
			if err.Fatal != expected.Fatal {
				t.Errorf("Fatal = %v, want %v", err.Fatal, expected.Fatal)
			}

			// Compare params
			if len(err.Params) != len(expected.Params) {
				t.Errorf("Params length = %d, want %d", len(err.Params), len(expected.Params))
				return
			}
			for k, v := range expected.Params {
				if gotV, ok := err.Params[k]; !ok || gotV != v {
					t.Errorf("Params[%q] = %v, want %v", k, gotV, v)
				}
			}
		})
	}
}
