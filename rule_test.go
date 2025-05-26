package validation_test

import (
	"strings"
	"testing"
	"time"

	"github.com/jacoelho/validation"
)

func TestRequired(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		rule := validation.Required[string]()

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
				errCode: "required",
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
	})

	t.Run("int", func(t *testing.T) {
		rule := validation.Required[int]()

		tests := []struct {
			name    string
			value   int
			wantErr bool
		}{
			{
				name:    "zero int should fail",
				value:   0,
				wantErr: true,
			},
			{
				name:    "positive int should pass",
				value:   5,
				wantErr: false,
			},
			{
				name:    "negative int should pass",
				value:   -5,
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
	})

	t.Run("bool", func(t *testing.T) {
		rule := validation.Required[bool]()

		tests := []struct {
			name    string
			value   bool
			wantErr bool
		}{
			{
				name:    "false bool should fail (zero value)",
				value:   false,
				wantErr: true,
			},
			{
				name:    "true bool should pass",
				value:   true,
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
	})

	t.Run("pointer", func(t *testing.T) {
		rule := validation.Required[*string]()

		value := "test"

		tests := []struct {
			name    string
			value   *string
			wantErr bool
		}{
			{
				name:    "nil pointer should fail",
				value:   nil,
				wantErr: true,
			},
			{
				name:    "non-nil pointer should pass",
				value:   &value,
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
	})
}

func TestRequiredZeroable(t *testing.T) {
	rule := validation.RequiredZeroable[time.Time]()

	now := time.Now()
	var zeroTime time.Time

	tests := []struct {
		name    string
		value   time.Time
		wantErr bool
		errCode string
	}{
		{
			name:    "zero time should fail",
			value:   zeroTime,
			wantErr: true,
			errCode: "required",
		},
		{
			name:    "non-zero time should pass",
			value:   now,
			wantErr: false,
		},
		{
			name:    "epoch time should pass",
			value:   time.Unix(0, 0),
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

func TestRuleNot(t *testing.T) {
	// Create a rule that fails for empty strings
	baseRule := validation.Required[string]()
	notRule := validation.RuleNot(baseRule)

	tests := []struct {
		name    string
		value   string
		wantErr bool
		errCode string
	}{
		{
			name:    "empty string should pass (negated required)",
			value:   "",
			wantErr: false,
		},
		{
			name:    "non-empty string should fail (negated required)",
			value:   "hello",
			wantErr: true,
			errCode: "not",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := notRule(tt.value)
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

func TestRuleStopOnError(t *testing.T) {
	// Create a rule that fails for empty strings
	baseRule := validation.Required[string]()
	stopRule := validation.RuleStopOnError(baseRule)

	tests := []struct {
		name      string
		value     string
		wantErr   bool
		wantFatal bool
		errCode   string
	}{
		{
			name:      "empty string should fail with fatal error",
			value:     "",
			wantErr:   true,
			wantFatal: true,
			errCode:   "required",
		},
		{
			name:      "non-empty string should pass",
			value:     "hello",
			wantErr:   false,
			wantFatal: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := stopRule(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else {
					if err.Code != tt.errCode {
						t.Errorf("expected error code %q, got %q", tt.errCode, err.Code)
					}
					if err.Fatal != tt.wantFatal {
						t.Errorf("expected fatal %v, got %v", tt.wantFatal, err.Fatal)
					}
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got %v", err)
				}
			}
		})
	}
}

func TestOr(t *testing.T) {
	// Create rules for testing
	minLengthRule := validation.StringsRuneMinLength[string](5)
	containsRule := validation.StringsContains[string]("test")

	// Or rule: string must be either >= 5 chars OR contain "test"
	orRule := validation.Or(minLengthRule, containsRule)

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{
			name:    "passes first rule (long enough)",
			value:   "hello world",
			wantErr: false,
		},
		{
			name:    "passes second rule (contains test)",
			value:   "test",
			wantErr: false,
		},
		{
			name:    "passes both rules",
			value:   "this is a test string",
			wantErr: false,
		},
		{
			name:    "fails both rules",
			value:   "hi",
			wantErr: true,
		},
		{
			name:    "empty string fails both rules",
			value:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := orRule(tt.value)
			if tt.wantErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error but got %v", err)
			}
		})
	}
}

func TestOrLastError(t *testing.T) {
	// Test that Or returns the last error when all rules fail
	rule1 := func(value string) *validation.Error {
		return &validation.Error{Code: "first_error"}
	}
	rule2 := func(value string) *validation.Error {
		return &validation.Error{Code: "second_error"}
	}
	rule3 := func(value string) *validation.Error {
		return &validation.Error{Code: "third_error"}
	}

	orRule := validation.Or(rule1, rule2, rule3)
	err := orRule("test")

	if err == nil {
		t.Fatal("expected error but got nil")
	}

	if err.Code != "third_error" {
		t.Errorf("expected last error code 'third_error', got %q", err.Code)
	}
}

func TestWhen(t *testing.T) {
	// Rule that requires non-empty string
	baseRule := validation.Required[string]()

	// Apply rule only when string starts with "admin"
	whenRule := validation.When(
		func(value string) bool {
			return len(value) > 0 && value[0:min(5, len(value))] == "admin"
		},
		baseRule,
	)

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{
			name:    "condition false, empty string should pass",
			value:   "",
			wantErr: false,
		},
		{
			name:    "condition false, user string should pass",
			value:   "user123",
			wantErr: false,
		},
		{
			name:    "condition true, admin string should pass",
			value:   "admin123",
			wantErr: false,
		},
		{
			name:    "condition true but empty admin should fail",
			value:   "admin",
			wantErr: false, // "admin" is not empty, so baseRule passes
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := whenRule(tt.value)
			if tt.wantErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error but got %v", err)
			}
		})
	}
}

func TestWhenWithFailingCondition(t *testing.T) {
	// Test When with a rule that would fail
	minLengthRule := validation.StringsRuneMinLength[string](10)

	// Apply rule only when string contains "validate"
	whenRule := validation.When(
		func(value string) bool {
			return len(value) > 0 && value == "validate"
		},
		minLengthRule,
	)

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{
			name:    "condition false, short string should pass",
			value:   "hi",
			wantErr: false,
		},
		{
			name:    "condition true, but string too short should fail",
			value:   "validate",
			wantErr: true, // "validate" is 8 chars, but rule requires 10
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := whenRule(tt.value)
			if tt.wantErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error but got %v", err)
			}
		})
	}
}

func TestUnless(t *testing.T) {
	// Rule that requires non-empty string
	baseRule := validation.Required[string]()

	// Apply rule unless string starts with "guest"
	unlessRule := validation.Unless(
		func(value string) bool {
			return strings.HasPrefix(value, "guest")
		},
		baseRule,
	)

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{
			name:    "guest string should skip validation",
			value:   "guest123",
			wantErr: false,
		},
		{
			name:    "non-guest string should be validated",
			value:   "user123",
			wantErr: false,
		},
		{
			name:    "non-guest empty string should fail",
			value:   "",
			wantErr: true,
		},
		{
			name:    "empty guest string should skip validation",
			value:   "guest",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := unlessRule(tt.value)
			if tt.wantErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error but got %v", err)
			}
		})
	}
}

func TestComplexRuleCombination(t *testing.T) {
	// Test complex combination of rules
	type User struct {
		Role string
		Name string
	}

	// Complex rule: Name is required unless role is "guest", and if role is "admin", name must be at least 5 chars
	nameRule := validation.Unless(
		func(u User) bool { return u.Role == "guest" },
		validation.When(
			func(u User) bool { return u.Role == "admin" },
			func(u User) *validation.Error {
				if len(u.Name) < 5 {
					return &validation.Error{
						Code:   "min",
						Params: map[string]any{"min": 5, "actual": len(u.Name)},
					}
				}
				return nil
			},
		),
	)

	tests := []struct {
		name    string
		user    User
		wantErr bool
	}{
		{
			name:    "guest with empty name should pass",
			user:    User{Role: "guest", Name: ""},
			wantErr: false,
		},
		{
			name:    "admin with long name should pass",
			user:    User{Role: "admin", Name: "administrator"},
			wantErr: false,
		},
		{
			name:    "admin with short name should fail",
			user:    User{Role: "admin", Name: "bob"},
			wantErr: true,
		},
		{
			name:    "user with any name should pass",
			user:    User{Role: "user", Name: "jo"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := nameRule(tt.user)
			if tt.wantErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error but got %v", err)
			}
		})
	}
}

func TestRuleTypes(t *testing.T) {
	t.Run("different types with Required", func(t *testing.T) {
		stringRule := validation.Required[string]()
		intRule := validation.Required[int]()
		floatRule := validation.Required[float64]()

		// Test string
		if err := stringRule(""); err == nil {
			t.Error("expected empty string to fail")
		}
		if err := stringRule("test"); err != nil {
			t.Error("expected non-empty string to pass")
		}

		// Test int
		if err := intRule(0); err == nil {
			t.Error("expected zero int to fail")
		}
		if err := intRule(42); err != nil {
			t.Error("expected non-zero int to pass")
		}

		// Test float
		if err := floatRule(0.0); err == nil {
			t.Error("expected zero float to fail")
		}
		if err := floatRule(3.14); err != nil {
			t.Error("expected non-zero float to pass")
		}
	})
}
