package validation_test

import (
	"testing"

	"github.com/jacoelho/validation"
)

func TestStringsNotEmpty(t *testing.T) {
	rule := validation.StringsNotEmpty[string]()

	tests := []struct {
		name    string
		value   string
		wantErr bool
		errCode string
	}{
		{
			name:    "empty string should fail",
			value:   "",
			wantErr: true,
			errCode: "not_empty",
		},
		{
			name:    "non-empty string should pass",
			value:   "hello",
			wantErr: false,
		},
		{
			name:    "whitespace string should pass",
			value:   " ",
			wantErr: false,
		},
		{
			name:    "single character should pass",
			value:   "a",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else if err.Code != tt.errCode {
					t.Errorf("expected error code %q, got %q", tt.errCode, err.Code)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got %v", err)
				}
			}
		})
	}
}

func TestStringsRuneMinLength(t *testing.T) {
	rule := validation.StringsRuneMinLength[string](3)

	tests := []struct {
		name    string
		value   string
		wantErr bool
		errCode string
	}{
		{
			name:    "string shorter than minimum should fail",
			value:   "ab",
			wantErr: true,
			errCode: "min",
		},
		{
			name:    "string equal to minimum should pass",
			value:   "abc",
			wantErr: false,
		},
		{
			name:    "string longer than minimum should pass",
			value:   "abcd",
			wantErr: false,
		},
		{
			name:    "empty string should fail",
			value:   "",
			wantErr: true,
			errCode: "min",
		},
		{
			name:    "unicode string with correct rune count should pass",
			value:   "🙂🙃😊", // 3 runes
			wantErr: false,
		},
		{
			name:    "unicode string with insufficient rune count should fail",
			value:   "🙂🙃", // 2 runes
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
				} else if err.Code != tt.errCode {
					t.Errorf("expected error code %q, got %q", tt.errCode, err.Code)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got %v", err)
				}
			}
		})
	}
}

func TestStringsRuneMaxLength(t *testing.T) {
	rule := validation.StringsRuneMaxLength[string](5)

	tests := []struct {
		name    string
		value   string
		wantErr bool
		errCode string
	}{
		{
			name:    "string longer than maximum should fail",
			value:   "abcdef",
			wantErr: true,
			errCode: "max",
		},
		{
			name:    "string equal to maximum should pass",
			value:   "abcde",
			wantErr: false,
		},
		{
			name:    "string shorter than maximum should pass",
			value:   "abcd",
			wantErr: false,
		},
		{
			name:    "empty string should pass",
			value:   "",
			wantErr: false,
		},
		{
			name:    "unicode string within limit should pass",
			value:   "🙂🙃😊🎉🎊", // 5 runes
			wantErr: false,
		},
		{
			name:    "unicode string exceeding limit should fail",
			value:   "🙂🙃😊🎉🎊🚀", // 6 runes
			wantErr: true,
			errCode: "max",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else if err.Code != tt.errCode {
					t.Errorf("expected error code %q, got %q", tt.errCode, err.Code)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got %v", err)
				}
			}
		})
	}
}

func TestStringsRuneLengthBetween(t *testing.T) {
	rule := validation.StringsRuneLengthBetween[string](3, 7)

	tests := []struct {
		name    string
		value   string
		wantErr bool
		errCode string
	}{
		{
			name:    "string shorter than minimum should fail",
			value:   "ab",
			wantErr: true,
			errCode: "between",
		},
		{
			name:    "string equal to minimum should pass",
			value:   "abc",
			wantErr: false,
		},
		{
			name:    "string within range should pass",
			value:   "abcde",
			wantErr: false,
		},
		{
			name:    "string equal to maximum should pass",
			value:   "abcdefg",
			wantErr: false,
		},
		{
			name:    "string longer than maximum should fail",
			value:   "abcdefgh",
			wantErr: true,
			errCode: "between",
		},
		{
			name:    "empty string should fail",
			value:   "",
			wantErr: true,
			errCode: "between",
		},
		{
			name:    "unicode string within range should pass",
			value:   "🙂🙃😊🎉", // 4 runes
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else if err.Code != tt.errCode {
					t.Errorf("expected error code %q, got %q", tt.errCode, err.Code)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got %v", err)
				}
			}
		})
	}
}

func TestStringsMatchesRegex(t *testing.T) {
	emailRule := validation.StringsMatchesRegex[string](`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	phoneRule := validation.StringsMatchesRegex[string](`^\d{3}-\d{3}-\d{4}$`)

	tests := []struct {
		name    string
		rule    validation.Rule[string]
		value   string
		wantErr bool
		errCode string
	}{
		{
			name:    "valid email should pass",
			rule:    emailRule,
			value:   "test@example.com",
			wantErr: false,
		},
		{
			name:    "invalid email should fail",
			rule:    emailRule,
			value:   "invalid-email",
			wantErr: true,
			errCode: "regex",
		},
		{
			name:    "valid phone number should pass",
			rule:    phoneRule,
			value:   "123-456-7890",
			wantErr: false,
		},
		{
			name:    "invalid phone number should fail",
			rule:    phoneRule,
			value:   "123-45-6789",
			wantErr: true,
			errCode: "regex",
		},
		{
			name:    "empty string with email pattern should fail",
			rule:    emailRule,
			value:   "",
			wantErr: true,
			errCode: "regex",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rule(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else if err.Code != tt.errCode {
					t.Errorf("expected error code %q, got %q", tt.errCode, err.Code)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got %v", err)
				}
			}
		})
	}
}

func TestStringsContains(t *testing.T) {
	rule := validation.StringsContains("test")

	tests := []struct {
		name    string
		value   string
		wantErr bool
		errCode string
	}{
		{
			name:    "string containing substring should pass",
			value:   "this is a test string",
			wantErr: false,
		},
		{
			name:    "string not containing substring should fail",
			value:   "this is a sample string",
			wantErr: true,
			errCode: "contains",
		},
		{
			name:    "exact match should pass",
			value:   "test",
			wantErr: false,
		},
		{
			name:    "substring at beginning should pass",
			value:   "testing 123",
			wantErr: false,
		},
		{
			name:    "substring at end should pass",
			value:   "unit test",
			wantErr: false,
		},
		{
			name:    "empty string should fail",
			value:   "",
			wantErr: true,
			errCode: "contains",
		},
		{
			name:    "case sensitive check should fail",
			value:   "this is a TEST string",
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
				} else if err.Code != tt.errCode {
					t.Errorf("expected error code %q, got %q", tt.errCode, err.Code)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got %v", err)
				}
			}
		})
	}
}

func TestStringsCustomType(t *testing.T) {
	type CustomString string

	// Test that the functions work with custom string types
	rule := validation.StringsRuneMinLength[CustomString](3)

	tests := []struct {
		name    string
		value   CustomString
		wantErr bool
	}{
		{
			name:    "custom string type should work",
			value:   CustomString("hello"),
			wantErr: false,
		},
		{
			name:    "short custom string should fail",
			value:   CustomString("hi"),
			wantErr: true,
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

func TestStringsErrorParams(t *testing.T) {
	t.Run("StringsRuneMinLength error params", func(t *testing.T) {
		rule := validation.StringsRuneMinLength[string](5)
		err := rule("abc")

		if err == nil {
			t.Fatal("expected error but got nil")
		}

		if err.Code != "min" {
			t.Errorf("expected code 'min', got %q", err.Code)
		}

		if err.Params["min"] != 5 {
			t.Errorf("expected min param to be 5, got %v", err.Params["min"])
		}

		if err.Params["actual"] != 3 {
			t.Errorf("expected actual param to be 3, got %v", err.Params["actual"])
		}
	})

	t.Run("StringsMatchesRegex error params", func(t *testing.T) {
		pattern := `^\d+$`
		rule := validation.StringsMatchesRegex[string](pattern)
		err := rule("abc")

		if err == nil {
			t.Fatal("expected error but got nil")
		}

		if err.Code != "regex" {
			t.Errorf("expected code 'regex', got %q", err.Code)
		}

		if err.Params["pattern"] != pattern {
			t.Errorf("expected pattern param to be %q, got %v", pattern, err.Params["pattern"])
		}
	})
}
