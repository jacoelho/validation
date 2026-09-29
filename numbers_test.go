package validation_test

import (
	"errors"
	"math"
	"testing"

	validation "github.com/jacoelho/validation/v2"
)

func primitiveIssueCodes(err error) []validation.Code {
	issues := validation.Issues(err)
	codes := make([]validation.Code, len(issues))
	for i, issue := range issues {
		codes[i] = issue.Code
	}
	return codes
}

func requirePrimitiveCodes(t *testing.T, err error, want ...validation.Code) {
	t.Helper()
	got := primitiveIssueCodes(err)
	if len(got) != len(want) {
		t.Fatalf("issue codes = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("issue codes = %v, want %v", got, want)
		}
	}
}

func TestNumericBounds(t *testing.T) {
	tests := []struct {
		name  string
		rule  validation.Rule[int]
		value int
		want  validation.Code
	}{
		{"min rejects below", validation.Min(5), 4, validation.CodeMin},
		{"min includes boundary", validation.Min(5), 5, ""},
		{"max rejects above", validation.Max(5), 6, validation.CodeMax},
		{"max includes boundary", validation.Max(5), 5, ""},
		{"greater than excludes boundary", validation.GreaterThan(5), 5, validation.CodeGreaterThan},
		{"greater than accepts above", validation.GreaterThan(5), 6, ""},
		{"less than excludes boundary", validation.LessThan(5), 5, validation.CodeLessThan},
		{"less than accepts below", validation.LessThan(5), 4, ""},
		{"between includes lower", validation.Between(2, 5), 2, ""},
		{"between includes upper", validation.Between(2, 5), 5, ""},
		{"between rejects above", validation.Between(2, 5), 6, validation.CodeBetween},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rule(tt.value)
			if tt.want == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			requirePrimitiveCodes(t, err, tt.want)
		})
	}
}

func TestNumericExactIntegerAndFloatPolicies(t *testing.T) {
	if err := validation.Min[uint64](9007199254740993)(9007199254740992); err == nil {
		t.Fatal("exact uint64 comparison unexpectedly passed")
	}
	const minInt64 = int64(-1 << 63)
	if err := validation.Max[int64](minInt64)(minInt64); err != nil {
		t.Fatalf("int64 minimum should pass: %v", err)
	}

	for _, test := range []struct {
		name string
		rule validation.Rule[float64]
		want validation.Code
	}{
		{"min NaN", validation.Min(0.0), validation.CodeMin},
		{"max NaN", validation.Max(1.0), validation.CodeMax},
		{"between NaN", validation.Between(0.0, 1.0), validation.CodeBetween},
		{"positive NaN", validation.Positive[float64](), validation.CodePositive},
		{"not NaN", validation.NotNaN[float64](), validation.CodeNotNaN},
		{"finite NaN", validation.Finite[float64](), validation.CodeFinite},
		{"finite infinity", validation.Finite[float64](), validation.CodeFinite},
	} {
		t.Run(test.name, func(t *testing.T) {
			value := math.NaN()
			if test.name == "finite infinity" {
				value = math.Inf(1)
			}
			requirePrimitiveCodes(t, test.rule(value), test.want)
		})
	}

	for _, test := range []struct {
		name  string
		rule  validation.Rule[float64]
		value float64
		want  validation.Code
	}{
		{"positive negative zero", validation.Positive[float64](), math.Copysign(0, -1), validation.CodePositive},
		{"negative positive zero", validation.Negative[float64](), 0, validation.CodeNegative},
	} {
		t.Run(test.name, func(t *testing.T) {
			requirePrimitiveCodes(t, test.rule(test.value), test.want)
		})
	}

	if err := validation.Min(0.0)(math.Inf(1)); err != nil {
		t.Fatalf("positive infinity should satisfy Min: %v", err)
	}
	if err := validation.Max(0.0)(math.Inf(-1)); err != nil {
		t.Fatalf("negative infinity should satisfy Max: %v", err)
	}
}

func TestNumericBoundsDiagnostics(t *testing.T) {
	err := validation.Min(5)(4)
	var bounds *validation.BoundsError[int]
	if !errors.As(err, &bounds) {
		t.Fatalf("%T is not a BoundsError", err)
	}
	lower, inclusive, present := bounds.Lower()
	if lower != 5 || !inclusive || !present {
		t.Fatalf("lower bound = (%v, %v, %v)", lower, inclusive, present)
	}
	if _, _, present := bounds.Upper(); present {
		t.Fatal("Min unexpectedly reported an upper bound")
	}
}

func TestBetweenByUsesComparatorSignsAndBothComparisons(t *testing.T) {
	type amount struct{ minor int64 }
	var calls int
	compare := func(left, right amount) int {
		calls++
		switch {
		case left.minor < right.minor:
			return -7
		case left.minor > right.minor:
			return 9
		default:
			return 0
		}
	}
	rule := validation.BetweenBy(amount{100}, amount{200}, compare)
	if calls != 1 {
		t.Fatalf("construction comparisons = %d, want 1", calls)
	}
	for _, test := range []struct {
		value amount
		want  validation.Code
	}{
		{amount{99}, validation.CodeBetween},
		{amount{100}, ""},
		{amount{200}, ""},
		{amount{201}, validation.CodeBetween},
	} {
		before := calls
		err := rule(test.value)
		if calls-before != 2 {
			t.Fatalf("value comparisons = %d, want 2", calls-before)
		}
		if test.want == "" {
			if err != nil {
				t.Fatalf("%v: unexpected error: %v", test.value, err)
			}
		} else {
			requirePrimitiveCodes(t, err, test.want)
		}
	}
}

func TestNumericConfigurationPanics(t *testing.T) {
	for _, name := range []string{"Between reversed", "Min NaN", "Max NaN"} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("expected configuration panic")
				}
			}()
			switch name {
			case "Between reversed":
				validation.Between(2, 1)
			case "Min NaN":
				validation.Min(math.NaN())
			case "Max NaN":
				validation.Max(math.NaN())
			}
		})
	}
}
