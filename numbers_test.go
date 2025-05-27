package validation_test

import (
	"testing"

	"github.com/jacoelho/validation"
)

func TestNumbersMin(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		rule := validation.NumbersMin(10)

		tests := []struct {
			name    string
			value   int
			wantErr bool
			errCode string
		}{
			{
				name:    "value below minimum should fail",
				value:   5,
				wantErr: true,
				errCode: "min",
			},
			{
				name:    "value equal to minimum should pass",
				value:   10,
				wantErr: false,
			},
			{
				name:    "value above minimum should pass",
				value:   15,
				wantErr: false,
			},
			{
				name:    "negative value below minimum should fail",
				value:   -5,
				wantErr: true,
				errCode: "min",
			},
			{
				name:    "zero below minimum should fail",
				value:   0,
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
	})

	t.Run("float64", func(t *testing.T) {
		rule := validation.NumbersMin(10.5)

		tests := []struct {
			name    string
			value   float64
			wantErr bool
		}{
			{
				name:    "float below minimum should fail",
				value:   10.4,
				wantErr: true,
			},
			{
				name:    "float equal to minimum should pass",
				value:   10.5,
				wantErr: false,
			},
			{
				name:    "float above minimum should pass",
				value:   10.6,
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

	t.Run("string", func(t *testing.T) {
		rule := validation.NumbersMin("apple")

		tests := []struct {
			name    string
			value   string
			wantErr bool
		}{
			{
				name:    "string below minimum should fail",
				value:   "aaa",
				wantErr: true,
			},
			{
				name:    "string equal to minimum should pass",
				value:   "apple",
				wantErr: false,
			},
			{
				name:    "string above minimum should pass",
				value:   "banana",
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

func TestNumbersMax(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		rule := validation.NumbersMax(100)

		tests := []struct {
			name    string
			value   int
			wantErr bool
			errCode string
		}{
			{
				name:    "value above maximum should fail",
				value:   105,
				wantErr: true,
				errCode: "max",
			},
			{
				name:    "value equal to maximum should pass",
				value:   100,
				wantErr: false,
			},
			{
				name:    "value below maximum should pass",
				value:   95,
				wantErr: false,
			},
			{
				name:    "negative value below maximum should pass",
				value:   -5,
				wantErr: false,
			},
			{
				name:    "zero below maximum should pass",
				value:   0,
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

	t.Run("float32", func(t *testing.T) {
		rule := validation.NumbersMax(float32(99.9))

		tests := []struct {
			name    string
			value   float32
			wantErr bool
		}{
			{
				name:    "float above maximum should fail",
				value:   100.0,
				wantErr: true,
			},
			{
				name:    "float equal to maximum should pass",
				value:   99.9,
				wantErr: false,
			},
			{
				name:    "float below maximum should pass",
				value:   99.8,
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

func TestNumbersBetween(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		rule := validation.NumbersBetween(10, 20)

		tests := []struct {
			name    string
			value   int
			wantErr bool
			errCode string
		}{
			{
				name:    "value below range should fail",
				value:   5,
				wantErr: true,
				errCode: "between",
			},
			{
				name:    "value equal to minimum should pass",
				value:   10,
				wantErr: false,
			},
			{
				name:    "value within range should pass",
				value:   15,
				wantErr: false,
			},
			{
				name:    "value equal to maximum should pass",
				value:   20,
				wantErr: false,
			},
			{
				name:    "value above range should fail",
				value:   25,
				wantErr: true,
				errCode: "between",
			},
			{
				name:    "negative value below range should fail",
				value:   -5,
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

	t.Run("float64", func(t *testing.T) {
		rule := validation.NumbersBetween(1.5, 2.5)

		tests := []struct {
			name    string
			value   float64
			wantErr bool
		}{
			{
				name:    "float below range should fail",
				value:   1.4,
				wantErr: true,
			},
			{
				name:    "float at minimum should pass",
				value:   1.5,
				wantErr: false,
			},
			{
				name:    "float within range should pass",
				value:   2.0,
				wantErr: false,
			},
			{
				name:    "float at maximum should pass",
				value:   2.5,
				wantErr: false,
			},
			{
				name:    "float above range should fail",
				value:   2.6,
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
	})

	t.Run("edge case - same min and max", func(t *testing.T) {
		rule := validation.NumbersBetween(5, 5)

		tests := []struct {
			name    string
			value   int
			wantErr bool
		}{
			{
				name:    "value equal to both min and max should pass",
				value:   5,
				wantErr: false,
			},
			{
				name:    "value below should fail",
				value:   4,
				wantErr: true,
			},
			{
				name:    "value above should fail",
				value:   6,
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
	})
}

func TestNumbersPositive(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		rule := validation.NumbersPositive[int]()

		tests := []struct {
			name    string
			value   int
			wantErr bool
			errCode string
		}{
			{
				name:    "positive value should pass",
				value:   5,
				wantErr: false,
			},
			{
				name:    "zero should fail",
				value:   0,
				wantErr: true,
				errCode: "positive",
			},
			{
				name:    "negative value should fail",
				value:   -5,
				wantErr: true,
				errCode: "positive",
			},
			{
				name:    "large positive value should pass",
				value:   1000000,
				wantErr: false,
			},
			{
				name:    "large negative value should fail",
				value:   -1000000,
				wantErr: true,
				errCode: "positive",
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

	t.Run("float64", func(t *testing.T) {
		rule := validation.NumbersPositive[float64]()

		tests := []struct {
			name    string
			value   float64
			wantErr bool
		}{
			{
				name:    "positive float should pass",
				value:   5.5,
				wantErr: false,
			},
			{
				name:    "zero float should fail",
				value:   0.0,
				wantErr: true,
			},
			{
				name:    "negative float should fail",
				value:   -5.5,
				wantErr: true,
			},
			{
				name:    "very small positive float should pass",
				value:   0.0001,
				wantErr: false,
			},
			{
				name:    "very small negative float should fail",
				value:   -0.0001,
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
	})
}

func TestNumbersNonNegative(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		rule := validation.NumbersNonNegative[int]()

		tests := []struct {
			name    string
			value   int
			wantErr bool
			errCode string
		}{
			{
				name:    "positive value should pass",
				value:   5,
				wantErr: false,
			},
			{
				name:    "zero should pass",
				value:   0,
				wantErr: false,
			},
			{
				name:    "negative value should fail",
				value:   -5,
				wantErr: true,
				errCode: "non_negative",
			},
			{
				name:    "large positive value should pass",
				value:   1000000,
				wantErr: false,
			},
			{
				name:    "large negative value should fail",
				value:   -1000000,
				wantErr: true,
				errCode: "non_negative",
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

	t.Run("float64", func(t *testing.T) {
		rule := validation.NumbersNonNegative[float64]()

		tests := []struct {
			name    string
			value   float64
			wantErr bool
		}{
			{
				name:    "positive float should pass",
				value:   5.5,
				wantErr: false,
			},
			{
				name:    "zero float should pass",
				value:   0.0,
				wantErr: false,
			},
			{
				name:    "negative float should fail",
				value:   -5.5,
				wantErr: true,
			},
			{
				name:    "very small positive float should pass",
				value:   0.0001,
				wantErr: false,
			},
			{
				name:    "very small negative float should fail",
				value:   -0.0001,
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
	})
}

func TestNumbersNegative(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		rule := validation.NumbersNegative[int]()

		tests := []struct {
			name    string
			value   int
			wantErr bool
			errCode string
		}{
			{
				name:    "negative value should pass",
				value:   -5,
				wantErr: false,
			},
			{
				name:    "zero should fail",
				value:   0,
				wantErr: true,
				errCode: "negative",
			},
			{
				name:    "positive value should fail",
				value:   5,
				wantErr: true,
				errCode: "negative",
			},
			{
				name:    "large negative value should pass",
				value:   -1000000,
				wantErr: false,
			},
			{
				name:    "large positive value should fail",
				value:   1000000,
				wantErr: true,
				errCode: "negative",
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

	t.Run("float32", func(t *testing.T) {
		rule := validation.NumbersNegative[float32]()

		tests := []struct {
			name    string
			value   float32
			wantErr bool
		}{
			{
				name:    "negative float should pass",
				value:   -5.5,
				wantErr: false,
			},
			{
				name:    "zero float should fail",
				value:   0.0,
				wantErr: true,
			},
			{
				name:    "positive float should fail",
				value:   5.5,
				wantErr: true,
			},
			{
				name:    "very small negative float should pass",
				value:   -0.0001,
				wantErr: false,
			},
			{
				name:    "very small positive float should fail",
				value:   0.0001,
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
	})
}

func TestNumbersNonPositive(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		rule := validation.NumbersNonPositive[int]()

		tests := []struct {
			name    string
			value   int
			wantErr bool
			errCode string
		}{
			{
				name:    "negative value should pass",
				value:   -5,
				wantErr: false,
			},
			{
				name:    "zero should pass",
				value:   0,
				wantErr: false,
			},
			{
				name:    "positive value should fail",
				value:   5,
				wantErr: true,
				errCode: "non_positive",
			},
			{
				name:    "large negative value should pass",
				value:   -1000000,
				wantErr: false,
			},
			{
				name:    "large positive value should fail",
				value:   1000000,
				wantErr: true,
				errCode: "non_positive",
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

	t.Run("float32", func(t *testing.T) {
		rule := validation.NumbersNonPositive[float32]()

		tests := []struct {
			name    string
			value   float32
			wantErr bool
		}{
			{
				name:    "negative float should pass",
				value:   -5.5,
				wantErr: false,
			},
			{
				name:    "zero float should pass",
				value:   0.0,
				wantErr: false,
			},
			{
				name:    "positive float should fail",
				value:   5.5,
				wantErr: true,
			},
			{
				name:    "very small negative float should pass",
				value:   -0.0001,
				wantErr: false,
			},
			{
				name:    "very small positive float should fail",
				value:   0.0001,
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
	})
}

func TestNumbersErrorParams(t *testing.T) {
	t.Run("NumbersMin error params", func(t *testing.T) {
		rule := validation.NumbersMin(10)
		err := rule(5)

		if err == nil {
			t.Fatal("expected error but got nil")
		}

		if err.Code != "min" {
			t.Errorf("expected code 'min', got %q", err.Code)
		}

		if err.Params["min"] != 10 {
			t.Errorf("expected min param to be 10, got %v", err.Params["min"])
		}

		if err.Params["actual"] != 5 {
			t.Errorf("expected actual param to be 5, got %v", err.Params["actual"])
		}
	})

	t.Run("NumbersMax error params", func(t *testing.T) {
		rule := validation.NumbersMax(100)
		err := rule(150)

		if err == nil {
			t.Fatal("expected error but got nil")
		}

		if err.Code != "max" {
			t.Errorf("expected code 'max', got %q", err.Code)
		}

		if err.Params["max"] != 100 {
			t.Errorf("expected max param to be 100, got %v", err.Params["max"])
		}

		if err.Params["actual"] != 150 {
			t.Errorf("expected actual param to be 150, got %v", err.Params["actual"])
		}
	})

	t.Run("NumbersBetween error params", func(t *testing.T) {
		rule := validation.NumbersBetween(10, 20)
		err := rule(5)

		if err == nil {
			t.Fatal("expected error but got nil")
		}

		if err.Code != "between" {
			t.Errorf("expected code 'between', got %q", err.Code)
		}

		if err.Params["min"] != 10 {
			t.Errorf("expected min param to be 10, got %v", err.Params["min"])
		}

		if err.Params["max"] != 20 {
			t.Errorf("expected max param to be 20, got %v", err.Params["max"])
		}

		if err.Params["actual"] != 5 {
			t.Errorf("expected actual param to be 5, got %v", err.Params["actual"])
		}
	})

	t.Run("NumbersPositive error params", func(t *testing.T) {
		rule := validation.NumbersPositive[int]()
		err := rule(-5)

		if err == nil {
			t.Fatal("expected error but got nil")
		}

		if err.Code != "positive" {
			t.Errorf("expected code 'positive', got %q", err.Code)
		}

		if err.Params["value"] != -5 {
			t.Errorf("expected value param to be -5, got %v", err.Params["value"])
		}
	})

	t.Run("NumbersNonNegative error params", func(t *testing.T) {
		rule := validation.NumbersNonNegative[int]()
		err := rule(-5)

		if err == nil {
			t.Fatal("expected error but got nil")
		}

		if err.Code != "non_negative" {
			t.Errorf("expected code 'non_negative', got %q", err.Code)
		}

		if err.Params["value"] != -5 {
			t.Errorf("expected value param to be -5, got %v", err.Params["value"])
		}
	})

	t.Run("NumbersNegative error params", func(t *testing.T) {
		rule := validation.NumbersNegative[int]()
		err := rule(5)

		if err == nil {
			t.Fatal("expected error but got nil")
		}

		if err.Code != "negative" {
			t.Errorf("expected code 'negative', got %q", err.Code)
		}

		if err.Params["value"] != 5 {
			t.Errorf("expected value param to be 5, got %v", err.Params["value"])
		}
	})

	t.Run("NumbersNonPositive error params", func(t *testing.T) {
		rule := validation.NumbersNonPositive[int]()
		err := rule(5)

		if err == nil {
			t.Fatal("expected error but got nil")
		}

		if err.Code != "non_positive" {
			t.Errorf("expected code 'non_positive', got %q", err.Code)
		}

		if err.Params["value"] != 5 {
			t.Errorf("expected value param to be 5, got %v", err.Params["value"])
		}
	})
}

func TestNumbersWithDifferentTypes(t *testing.T) {
	t.Run("different integer types", func(t *testing.T) {
		// Test with different integer types
		int8Rule := validation.NumbersMin(int8(10))
		int16Rule := validation.NumbersMin(int16(100))
		int32Rule := validation.NumbersMin(int32(1000))
		int64Rule := validation.NumbersMin(int64(10000))
		uintRule := validation.NumbersMin(uint(10))

		if err := int8Rule(int8(5)); err == nil {
			t.Error("expected int8 rule to fail")
		}
		if err := int8Rule(int8(15)); err != nil {
			t.Error("expected int8 rule to pass")
		}

		if err := int16Rule(int16(50)); err == nil {
			t.Error("expected int16 rule to fail")
		}
		if err := int16Rule(int16(150)); err != nil {
			t.Error("expected int16 rule to pass")
		}

		if err := int32Rule(int32(500)); err == nil {
			t.Error("expected int32 rule to fail")
		}
		if err := int32Rule(int32(1500)); err != nil {
			t.Error("expected int32 rule to pass")
		}

		if err := int64Rule(int64(5000)); err == nil {
			t.Error("expected int64 rule to fail")
		}
		if err := int64Rule(int64(15000)); err != nil {
			t.Error("expected int64 rule to pass")
		}

		if err := uintRule(uint(5)); err == nil {
			t.Error("expected uint rule to fail")
		}
		if err := uintRule(uint(15)); err != nil {
			t.Error("expected uint rule to pass")
		}
	})

	t.Run("different float types", func(t *testing.T) {
		float32Rule := validation.NumbersMax(float32(10.5))
		float64Rule := validation.NumbersMax(float64(20.5))

		if err := float32Rule(float32(15.5)); err == nil {
			t.Error("expected float32 rule to fail")
		}
		if err := float32Rule(float32(5.5)); err != nil {
			t.Error("expected float32 rule to pass")
		}

		if err := float64Rule(float64(25.5)); err == nil {
			t.Error("expected float64 rule to fail")
		}
		if err := float64Rule(float64(15.5)); err != nil {
			t.Error("expected float64 rule to pass")
		}
	})
}

func TestNumbersEdgeCases(t *testing.T) {
	t.Run("zero values", func(t *testing.T) {
		minRule := validation.NumbersMin(0)
		maxRule := validation.NumbersMax(0)
		betweenRule := validation.NumbersBetween(0, 0)

		// Test with zero
		if err := minRule(0); err != nil {
			t.Error("expected zero to pass min(0) rule")
		}
		if err := maxRule(0); err != nil {
			t.Error("expected zero to pass max(0) rule")
		}
		if err := betweenRule(0); err != nil {
			t.Error("expected zero to pass between(0,0) rule")
		}
	})

	t.Run("negative ranges", func(t *testing.T) {
		rule := validation.NumbersBetween(-10, -5)

		tests := []struct {
			value   int
			wantErr bool
		}{
			{-15, true},  // below range
			{-10, false}, // at min
			{-8, false},  // within range
			{-5, false},  // at max
			{-3, true},   // above range
			{0, true},    // positive value
		}

		for _, tt := range tests {
			err := rule(tt.value)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for value %d", tt.value)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error for value %d, got %v", tt.value, err)
			}
		}
	})
}
