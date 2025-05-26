package validation_test

import (
	"testing"

	"github.com/jacoelho/validation"
)

func TestMapsMinKeys(t *testing.T) {
	rule := validation.MapsMinKeys[string, string](3)

	tests := []struct {
		name    string
		value   map[string]string
		wantErr bool
		errCode string
	}{
		{
			name:    "empty map should fail",
			value:   map[string]string{},
			wantErr: true,
			errCode: "min",
		},
		{
			name:    "map with fewer keys should fail",
			value:   map[string]string{"a": "1", "b": "2"},
			wantErr: true,
			errCode: "min",
		},
		{
			name:    "map with exact keys should pass",
			value:   map[string]string{"a": "1", "b": "2", "c": "3"},
			wantErr: false,
		},
		{
			name:    "map with more keys should pass",
			value:   map[string]string{"a": "1", "b": "2", "c": "3", "d": "4"},
			wantErr: false,
		},
		{
			name:    "nil map should fail",
			value:   nil,
			wantErr: true,
			errCode: "min",
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

func TestMapsMaxKeys(t *testing.T) {
	rule := validation.MapsMaxKeys[string, string](3)

	tests := []struct {
		name    string
		value   map[string]string
		wantErr bool
		errCode string
	}{
		{
			name:    "empty map should pass",
			value:   map[string]string{},
			wantErr: false,
		},
		{
			name:    "map with fewer keys should pass",
			value:   map[string]string{"a": "1", "b": "2"},
			wantErr: false,
		},
		{
			name:    "map with exact keys should pass",
			value:   map[string]string{"a": "1", "b": "2", "c": "3"},
			wantErr: false,
		},
		{
			name:    "map with more keys should fail",
			value:   map[string]string{"a": "1", "b": "2", "c": "3", "d": "4"},
			wantErr: true,
			errCode: "max",
		},
		{
			name:    "nil map should pass",
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

func TestMapsLength(t *testing.T) {
	rule := validation.MapsLength[string, string](3)

	tests := []struct {
		name    string
		value   map[string]string
		wantErr bool
		errCode string
	}{
		{
			name:    "empty map should fail",
			value:   map[string]string{},
			wantErr: true,
			errCode: "length",
		},
		{
			name:    "map with fewer keys should fail",
			value:   map[string]string{"a": "1", "b": "2"},
			wantErr: true,
			errCode: "length",
		},
		{
			name:    "map with exact keys should pass",
			value:   map[string]string{"a": "1", "b": "2", "c": "3"},
			wantErr: false,
		},
		{
			name:    "map with more keys should fail",
			value:   map[string]string{"a": "1", "b": "2", "c": "3", "d": "4"},
			wantErr: true,
			errCode: "length",
		},
		{
			name:    "nil map should fail",
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

func TestMapsLengthBetween(t *testing.T) {
	rule := validation.MapsLengthBetween[string, string](2, 4)

	tests := []struct {
		name    string
		value   map[string]string
		wantErr bool
		errCode string
	}{
		{
			name:    "empty map should fail",
			value:   map[string]string{},
			wantErr: true,
			errCode: "between",
		},
		{
			name:    "map with fewer keys should fail",
			value:   map[string]string{"a": "1"},
			wantErr: true,
			errCode: "between",
		},
		{
			name:    "map at minimum should pass",
			value:   map[string]string{"a": "1", "b": "2"},
			wantErr: false,
		},
		{
			name:    "map within range should pass",
			value:   map[string]string{"a": "1", "b": "2", "c": "3"},
			wantErr: false,
		},
		{
			name:    "map at maximum should pass",
			value:   map[string]string{"a": "1", "b": "2", "c": "3", "d": "4"},
			wantErr: false,
		},
		{
			name:    "map with more keys should fail",
			value:   map[string]string{"a": "1", "b": "2", "c": "3", "d": "4", "e": "5"},
			wantErr: true,
			errCode: "between",
		},
		{
			name:    "nil map should fail",
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

func TestMapsKeysAllowed(t *testing.T) {
	rule := validation.MapsKeysAllowed[string, string]("a", "b", "c")

	tests := []struct {
		name    string
		value   map[string]string
		wantErr bool
		errCode string
	}{
		{
			name:    "empty map should pass",
			value:   map[string]string{},
			wantErr: false,
		},
		{
			name:    "map with all allowed keys should pass",
			value:   map[string]string{"a": "1", "b": "2", "c": "3"},
			wantErr: false,
		},
		{
			name:    "map with disallowed key should fail",
			value:   map[string]string{"a": "1", "d": "2"},
			wantErr: true,
			errCode: "allowed",
		},
		{
			name:    "nil map should pass",
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

func TestMapsValuesAllowed(t *testing.T) {
	rule := validation.MapsValuesAllowed[string, string]("1", "2", "3")

	tests := []struct {
		name    string
		value   map[string]string
		wantErr bool
		errCode string
	}{
		{
			name:    "empty map should pass",
			value:   map[string]string{},
			wantErr: false,
		},
		{
			name:    "map with all allowed values should pass",
			value:   map[string]string{"a": "1", "b": "2", "c": "3"},
			wantErr: false,
		},
		{
			name:    "map with disallowed value should fail",
			value:   map[string]string{"a": "1", "b": "4"},
			wantErr: true,
			errCode: "allowed",
		},
		{
			name:    "nil map should pass",
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

func TestMapsValuesDisallowed(t *testing.T) {
	rule := validation.MapsValuesDisallowed[string, string]("x", "y", "z")

	tests := []struct {
		name    string
		value   map[string]string
		wantErr bool
		errCode string
	}{
		{
			name:    "empty map should pass",
			value:   map[string]string{},
			wantErr: false,
		},
		{
			name:    "map with no disallowed values should pass",
			value:   map[string]string{"a": "1", "b": "2", "c": "3"},
			wantErr: false,
		},
		{
			name:    "map with disallowed value should fail",
			value:   map[string]string{"a": "1", "b": "x"},
			wantErr: true,
			errCode: "disallowed",
		},
		{
			name:    "nil map should pass",
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

func TestMapsKeysDisallowed(t *testing.T) {
	rule := validation.MapsKeysDisallowed[string, string]("x", "y", "z")

	tests := []struct {
		name    string
		value   map[string]string
		wantErr bool
		errCode string
	}{
		{
			name:    "empty map should pass",
			value:   map[string]string{},
			wantErr: false,
		},
		{
			name:    "map with no disallowed keys should pass",
			value:   map[string]string{"a": "1", "b": "2", "c": "3"},
			wantErr: false,
		},
		{
			name:    "map with disallowed key should fail",
			value:   map[string]string{"a": "1", "x": "2"},
			wantErr: true,
			errCode: "disallowed",
		},
		{
			name:    "nil map should pass",
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

func TestMapsForEach(t *testing.T) {
	rule := validation.MapsForEach(
		func(k, v string) *validation.Error {
			if v == "" {
				return &validation.Error{
					Code:   "empty_value",
					Params: map[string]any{"key": k},
				}
			}
			return nil
		},
	)

	tests := []struct {
		name    string
		value   map[string]string
		wantErr bool
		errCode string
	}{
		{
			name:    "empty map should pass",
			value:   map[string]string{},
			wantErr: false,
		},
		{
			name:    "map with all non-empty values should pass",
			value:   map[string]string{"a": "1", "b": "2", "c": "3"},
			wantErr: false,
		},
		{
			name:    "map with empty value should fail",
			value:   map[string]string{"a": "1", "b": ""},
			wantErr: true,
			errCode: "empty_value",
		},
		{
			name:    "nil map should pass",
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

func TestMapsWithDifferentTypes(t *testing.T) {
	t.Run("int keys", func(t *testing.T) {
		rule := validation.MapsMinKeys[int, string](2)
		value := map[int]string{1: "a", 2: "b"}
		if err := rule(value); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("float values", func(t *testing.T) {
		rule := validation.MapsMaxKeys[string, float64](2)
		value := map[string]float64{"a": 1.1, "b": 2.2}
		if err := rule(value); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("struct values", func(t *testing.T) {
		type Point struct{ X, Y int }
		rule := validation.MapsLength[string, Point](2)
		value := map[string]Point{"a": {1, 2}, "b": {3, 4}}
		if err := rule(value); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestMapsErrorParams(t *testing.T) {
	t.Run("MapsMinKeys error params", func(t *testing.T) {
		rule := validation.MapsMinKeys[string, string](3)
		err := rule(map[string]string{"a": "1", "b": "2"})

		if err == nil {
			t.Fatal("expected error but got nil")
		}

		if len(err) == 0 {
			t.Fatal("expected error but got empty slice")
		}

		if err[0].Code != "min" {
			t.Errorf("expected code 'min', got %q", err[0].Code)
		}

		if err[0].Params["min"] != 3 {
			t.Errorf("expected min param to be 3, got %v", err[0].Params["min"])
		}

		if err[0].Params["actual"] != 2 {
			t.Errorf("expected actual param to be 2, got %v", err[0].Params["actual"])
		}
	})

	t.Run("MapsLengthBetween error params", func(t *testing.T) {
		rule := validation.MapsLengthBetween[string, string](2, 4)
		err := rule(map[string]string{"a": "1"})

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
