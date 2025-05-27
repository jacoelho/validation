package validation_test

import (
	"reflect"
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

func TestMapsKeysOneOf(t *testing.T) {
	rule := validation.MapsKeysOneOf[string, string]("a", "b", "c")

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
			errCode: "one_of",
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

func TestMapsValuesOneOf(t *testing.T) {
	rule := validation.MapsValuesOneOf[string]("1", "2", "3")

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
			errCode: "one_of",
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

func TestMapsValuesNotOneOf(t *testing.T) {
	rule := validation.MapsValuesNotOneOf[string]("x", "y", "z")

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
			errCode: "not_one_of",
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

func TestMapsKeysNotOneOf(t *testing.T) {
	rule := validation.MapsKeysNotOneOf[string, string]("x", "y", "z")

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
			errCode: "not_one_of",
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

func TestMapsKeyValidation(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		rules     []validation.Rule[string]
		value     map[string]string
		wantErr   bool
		errCode   string
		errField  string
		errParams map[string]any
	}{
		{
			name:      "key not found in map",
			key:       "missing",
			rules:     []validation.Rule[string]{validation.NotZero[string]()},
			value:     map[string]string{"existing": "value"},
			wantErr:   true,
			errCode:   "not_found",
			errField:  "",
			errParams: map[string]any{"key": "missing"},
		},
		{
			name:    "key exists with valid value",
			key:     "name",
			rules:   []validation.Rule[string]{validation.NotZero[string]()},
			value:   map[string]string{"name": "John"},
			wantErr: false,
		},
		{
			name:     "key exists with invalid value",
			key:      "name",
			rules:    []validation.Rule[string]{validation.NotZero[string]()},
			value:    map[string]string{"name": ""},
			wantErr:  true,
			errCode:  "zero",
			errField: "name",
		},
		{
			name: "multiple rules all pass",
			key:  "name",
			rules: []validation.Rule[string]{
				validation.NotZero[string](),
				validation.StringsRuneMaxLength[string](10),
			},
			value:   map[string]string{"name": "John"},
			wantErr: false,
		},
		{
			name: "multiple rules first fails",
			key:  "name",
			rules: []validation.Rule[string]{
				validation.NotZero[string](),
				validation.StringsRuneMaxLength[string](10),
			},
			value:    map[string]string{"name": ""},
			wantErr:  true,
			errCode:  "zero",
			errField: "name",
		},
		{
			name: "multiple rules second fails",
			key:  "name",
			rules: []validation.Rule[string]{
				validation.NotZero[string](),
				validation.StringsRuneMaxLength[string](3),
			},
			value:     map[string]string{"name": "John"},
			wantErr:   true,
			errCode:   "max",
			errField:  "name",
			errParams: map[string]any{"max": 3, "actual": 4},
		},
		{
			name: "fatal error stops validation",
			key:  "name",
			rules: []validation.Rule[string]{
				validation.RuleStopOnError(validation.NotZero[string]()),
				validation.StringsRuneMaxLength[string](3),
			},
			value:    map[string]string{"name": ""},
			wantErr:  true,
			errCode:  "zero",
			errField: "name",
		},
		{
			name: "custom validation rule",
			key:  "status",
			rules: []validation.Rule[string]{
				func(value string) *validation.Error {
					if value != "active" && value != "inactive" {
						return &validation.Error{
							Code:   "invalid_status",
							Params: map[string]any{"value": value},
						}
					}
					return nil
				},
			},
			value:     map[string]string{"status": "pending"},
			wantErr:   true,
			errCode:   "invalid_status",
			errField:  "status",
			errParams: map[string]any{"value": "pending"},
		},
		{
			name:      "nil map with key validation",
			key:       "name",
			rules:     []validation.Rule[string]{validation.NotZero[string]()},
			value:     nil,
			wantErr:   true,
			errCode:   "not_found",
			errField:  "",
			errParams: map[string]any{"key": "name"},
		},
		{
			name:      "empty map with key validation",
			key:       "name",
			rules:     []validation.Rule[string]{validation.NotZero[string]()},
			value:     map[string]string{},
			wantErr:   true,
			errCode:   "not_found",
			errField:  "",
			errParams: map[string]any{"key": "name"},
		},
		{
			name: "multiple keys with same validation",
			key:  "name",
			rules: []validation.Rule[string]{
				validation.NotZero[string](),
				validation.StringsRuneMaxLength[string](5),
			},
			value: map[string]string{
				"name": "John",
				"age":  "30",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := validation.MapsKey(tt.key, tt.rules...)
			err := rule(tt.value)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if len(err) == 0 {
					t.Error("expected error, got empty slice")
					return
				}
				if err[0].Code != tt.errCode {
					t.Errorf("expected error code %q, got %q", tt.errCode, err[0].Code)
				}
				if err[0].Field != tt.errField {
					t.Errorf("expected error field %q, got %q", tt.errField, err[0].Field)
				}
				if tt.errParams != nil && !reflect.DeepEqual(err[0].Params, tt.errParams) {
					t.Errorf("expected error params %v, got %v", tt.errParams, err[0].Params)
				}
			} else {
				if len(err) > 0 {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}
