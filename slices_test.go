package validation_test

import (
	"testing"

	"github.com/jacoelho/validation"
)

func TestSlicesMinLength(t *testing.T) {
	rule := validation.SlicesMinLength[string](3)

	tests := []struct {
		name    string
		value   []string
		wantErr bool
		errCode string
	}{
		{
			name:    "empty slice should fail",
			value:   []string{},
			wantErr: true,
			errCode: "min_length",
		},
		{
			name:    "slice shorter than minimum should fail",
			value:   []string{"a", "b"},
			wantErr: true,
			errCode: "min_length",
		},
		{
			name:    "slice equal to minimum should pass",
			value:   []string{"a", "b", "c"},
			wantErr: false,
		},
		{
			name:    "slice longer than minimum should pass",
			value:   []string{"a", "b", "c", "d"},
			wantErr: false,
		},
		{
			name:    "nil slice should fail",
			value:   nil,
			wantErr: true,
			errCode: "min_length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else if len(err) > 0 && err[0].Code != tt.errCode {
					t.Errorf("expected error code %q, got %q", tt.errCode, err[0].Code)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got %v", err)
				}
			}
		})
	}
}

func TestSlicesMaxLength(t *testing.T) {
	rule := validation.SlicesMaxLength[string](3)

	tests := []struct {
		name    string
		value   []string
		wantErr bool
		errCode string
	}{
		{
			name:    "empty slice should pass",
			value:   []string{},
			wantErr: false,
		},
		{
			name:    "slice shorter than maximum should pass",
			value:   []string{"a", "b"},
			wantErr: false,
		},
		{
			name:    "slice equal to maximum should pass",
			value:   []string{"a", "b", "c"},
			wantErr: false,
		},
		{
			name:    "slice longer than maximum should fail",
			value:   []string{"a", "b", "c", "d"},
			wantErr: true,
			errCode: "max",
		},
		{
			name:    "nil slice should pass",
			value:   nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else if len(err) > 0 && err[0].Code != tt.errCode {
					t.Errorf("expected error code %q, got %q", tt.errCode, err[0].Code)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got %v", err)
				}
			}
		})
	}
}

func TestSlicesLength(t *testing.T) {
	rule := validation.SlicesLength[string](3)

	tests := []struct {
		name    string
		value   []string
		wantErr bool
		errCode string
	}{
		{
			name:    "empty slice should fail",
			value:   []string{},
			wantErr: true,
			errCode: "length",
		},
		{
			name:    "slice shorter than required should fail",
			value:   []string{"a", "b"},
			wantErr: true,
			errCode: "length",
		},
		{
			name:    "slice of exact length should pass",
			value:   []string{"a", "b", "c"},
			wantErr: false,
		},
		{
			name:    "slice longer than required should fail",
			value:   []string{"a", "b", "c", "d"},
			wantErr: true,
			errCode: "length",
		},
		{
			name:    "nil slice should fail",
			value:   nil,
			wantErr: true,
			errCode: "length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else if len(err) > 0 && err[0].Code != tt.errCode {
					t.Errorf("expected error code %q, got %q", tt.errCode, err[0].Code)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got %v", err)
				}
			}
		})
	}
}

func TestSlicesInBetweenLength(t *testing.T) {
	rule := validation.SlicesInBetweenLength[string](2, 4)

	tests := []struct {
		name    string
		value   []string
		wantErr bool
		errCode string
	}{
		{
			name:    "empty slice should fail",
			value:   []string{},
			wantErr: true,
			errCode: "between",
		},
		{
			name:    "slice shorter than minimum should fail",
			value:   []string{"a"},
			wantErr: true,
			errCode: "between",
		},
		{
			name:    "slice at minimum should pass",
			value:   []string{"a", "b"},
			wantErr: false,
		},
		{
			name:    "slice within range should pass",
			value:   []string{"a", "b", "c"},
			wantErr: false,
		},
		{
			name:    "slice at maximum should pass",
			value:   []string{"a", "b", "c", "d"},
			wantErr: false,
		},
		{
			name:    "slice longer than maximum should fail",
			value:   []string{"a", "b", "c", "d", "e"},
			wantErr: true,
			errCode: "between",
		},
		{
			name:    "nil slice should fail",
			value:   nil,
			wantErr: true,
			errCode: "between",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else if len(err) > 0 && err[0].Code != tt.errCode {
					t.Errorf("expected error code %q, got %q", tt.errCode, err[0].Code)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got %v", err)
				}
			}
		})
	}
}

func TestSlicesForEach(t *testing.T) {
	rule := validation.SlicesForEach(
		validation.StringsNotEmpty[string](),
		validation.StringsRuneMaxLength[string](5),
	)

	tests := []struct {
		name    string
		value   []string
		wantErr bool
	}{
		{
			name:    "empty slice should pass",
			value:   []string{},
			wantErr: false,
		},
		{
			name:    "all valid elements should pass",
			value:   []string{"a", "bb", "ccc"},
			wantErr: false,
		},
		{
			name:    "empty element should fail",
			value:   []string{"a", "", "c"},
			wantErr: true,
		},
		{
			name:    "too long element should fail",
			value:   []string{"a", "bb", "cccccc"},
			wantErr: true,
		},
		{
			name:    "nil slice should pass",
			value:   nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule(tt.value)
			if tt.wantErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error but got %v", err)
			}
		})
	}
}

func TestSlicesUnique(t *testing.T) {
	rule := validation.SlicesUnique[string]()

	tests := []struct {
		name    string
		value   []string
		wantErr bool
		errCode string
	}{
		{
			name:    "empty slice should pass",
			value:   []string{},
			wantErr: false,
		},
		{
			name:    "unique elements should pass",
			value:   []string{"a", "b", "c"},
			wantErr: false,
		},
		{
			name:    "duplicate elements should fail",
			value:   []string{"a", "b", "a"},
			wantErr: true,
			errCode: "unique",
		},
		{
			name:    "multiple duplicates should fail",
			value:   []string{"a", "b", "a", "c", "b"},
			wantErr: true,
			errCode: "unique",
		},
		{
			name:    "nil slice should pass",
			value:   nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else if len(err) > 0 && err[0].Code != tt.errCode {
					t.Errorf("expected error code %q, got %q", tt.errCode, err[0].Code)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got %v", err)
				}
			}
		})
	}
}

func TestSlicesContains(t *testing.T) {
	rule := validation.SlicesContains[string]("test")

	tests := []struct {
		name    string
		value   []string
		wantErr bool
		errCode string
	}{
		{
			name:    "empty slice should fail",
			value:   []string{},
			wantErr: true,
			errCode: "contains",
		},
		{
			name:    "slice containing value should pass",
			value:   []string{"a", "test", "c"},
			wantErr: false,
		},
		{
			name:    "slice without value should fail",
			value:   []string{"a", "b", "c"},
			wantErr: true,
			errCode: "contains",
		},
		{
			name:    "nil slice should fail",
			value:   nil,
			wantErr: true,
			errCode: "contains",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else if len(err) > 0 && err[0].Code != tt.errCode {
					t.Errorf("expected error code %q, got %q", tt.errCode, err[0].Code)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got %v", err)
				}
			}
		})
	}
}

func TestSlicesAllowed(t *testing.T) {
	rule := validation.SlicesAllowed[string]("a", "b", "c")

	tests := []struct {
		name    string
		value   []string
		wantErr bool
		errCode string
	}{
		{
			name:    "empty slice should pass",
			value:   []string{},
			wantErr: false,
		},
		{
			name:    "all allowed values should pass",
			value:   []string{"a", "b", "c"},
			wantErr: false,
		},
		{
			name:    "disallowed value should fail",
			value:   []string{"a", "d", "c"},
			wantErr: true,
			errCode: "allowed",
		},
		{
			name:    "nil slice should pass",
			value:   nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else if len(err) > 0 && err[0].Code != tt.errCode {
					t.Errorf("expected error code %q, got %q", tt.errCode, err[0].Code)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got %v", err)
				}
			}
		})
	}
}

func TestSlicesDisallowed(t *testing.T) {
	rule := validation.SlicesDisallowed[string]("x", "y", "z")

	tests := []struct {
		name    string
		value   []string
		wantErr bool
		errCode string
	}{
		{
			name:    "empty slice should pass",
			value:   []string{},
			wantErr: false,
		},
		{
			name:    "no disallowed values should pass",
			value:   []string{"a", "b", "c"},
			wantErr: false,
		},
		{
			name:    "disallowed value should fail",
			value:   []string{"a", "x", "c"},
			wantErr: true,
			errCode: "disallowed",
		},
		{
			name:    "nil slice should pass",
			value:   nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else if len(err) > 0 && err[0].Code != tt.errCode {
					t.Errorf("expected error code %q, got %q", tt.errCode, err[0].Code)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got %v", err)
				}
			}
		})
	}
}

func TestSlicesWithDifferentTypes(t *testing.T) {
	t.Run("int slices", func(t *testing.T) {
		rule := validation.SlicesMinLength[int](3)
		value := []int{1, 2, 3}
		if err := rule(value); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("float slices", func(t *testing.T) {
		rule := validation.SlicesMaxLength[float64](2)
		value := []float64{1.1, 2.2}
		if err := rule(value); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("struct slices", func(t *testing.T) {
		type Point struct{ X, Y int }
		rule := validation.SlicesLength[Point](2)
		value := []Point{{1, 2}, {3, 4}}
		if err := rule(value); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestSlicesErrorParams(t *testing.T) {
	t.Run("SlicesMinLength error params", func(t *testing.T) {
		rule := validation.SlicesMinLength[string](3)
		err := rule([]string{"a", "b"})

		if err == nil {
			t.Fatal("expected error but got nil")
		}

		if len(err) == 0 {
			t.Fatal("expected error but got empty slice")
		}

		if err[0].Code != "min_length" {
			t.Errorf("expected code 'min_length', got %q", err[0].Code)
		}

		if err[0].Params["min"] != 3 {
			t.Errorf("expected min param to be 3, got %v", err[0].Params["min"])
		}

		if err[0].Params["actual"] != 2 {
			t.Errorf("expected actual param to be 2, got %v", err[0].Params["actual"])
		}
	})

	t.Run("SlicesInBetweenLength error params", func(t *testing.T) {
		rule := validation.SlicesInBetweenLength[string](2, 4)
		err := rule([]string{"a"})

		if err == nil {
			t.Fatal("expected error but got nil")
		}

		if len(err) == 0 {
			t.Fatal("expected error but got empty slice")
		}

		if err[0].Code != "between" {
			t.Errorf("expected code 'between', got %q", err[0].Code)
		}

		if err[0].Params["min"] != 2 {
			t.Errorf("expected min param to be 2, got %v", err[0].Params["min"])
		}

		if err[0].Params["max"] != 4 {
			t.Errorf("expected max param to be 4, got %v", err[0].Params["max"])
		}

		if err[0].Params["actual"] != 1 {
			t.Errorf("expected actual param to be 1, got %v", err[0].Params["actual"])
		}
	})
}
