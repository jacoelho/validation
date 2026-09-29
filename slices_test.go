package validation_test

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/jacoelho/validation/v2"
)

type sliceIssue struct {
	path string
	code validation.Code
}

func sliceIssues(err error) []sliceIssue {
	issues := validation.Issues(err)
	out := make([]sliceIssue, len(issues))
	for i, issue := range issues {
		out[i] = sliceIssue{path: validation.FormatPath(issue.Path), code: issue.Code}
	}
	return out
}

func TestSliceLengthRulesUseElementDiagnostics(t *testing.T) {
	tests := []struct {
		name   string
		rule   validation.Rule[[]string]
		in     []string
		want   validation.Code
		min    int
		max    int
		hasMax bool
	}{
		{name: "exact", rule: validation.SliceLength[[]string](2), in: []string{"a"}, want: validation.CodeLength, min: 2, max: 2, hasMax: true},
		{name: "minimum", rule: validation.SliceMinLength[[]string](2), in: []string{"a"}, want: validation.CodeMinLength, min: 2},
		{name: "maximum", rule: validation.SliceMaxLength[[]string](1), in: []string{"a", "b"}, want: validation.CodeMaxLength, min: 0, max: 1, hasMax: true},
		{name: "between", rule: validation.SliceLengthBetween[[]string](2, 3), in: []string{"a"}, want: validation.CodeLengthBetween, min: 2, max: 3, hasMax: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rule(tt.in)
			var detail *validation.LengthError
			if !errors.As(err, &detail) {
				t.Fatalf("expected LengthError, got %T", err)
			}
			if detail.Code() != tt.want || detail.Actual() != len(tt.in) || detail.Minimum() != tt.min {
				t.Fatalf("unexpected length detail: code=%q actual=%d minimum=%d", detail.Code(), detail.Actual(), detail.Minimum())
			}
			gotMax, got := detail.Maximum()
			if got != tt.hasMax || got && gotMax != tt.max {
				t.Fatalf("unexpected maximum: (%d, %t)", gotMax, got)
			}
			if detail.Unit() != validation.LengthElements {
				t.Fatalf("unit = %q, want elements", detail.Unit())
			}
			if got := sliceIssues(err); len(got) != 1 || got[0].path != "$" || got[0].code != tt.want {
				t.Fatalf("issues = %#v", got)
			}
		})
	}

	if validation.SliceLength[[]string](0)(nil) != nil {
		t.Fatal("nil slice with length zero should pass")
	}
	if validation.SliceMinLength[[]string](1)(nil) == nil {
		t.Fatal("nil slice should fail a positive minimum")
	}
}

func TestSlicesCollectContainerAndElementFailures(t *testing.T) {
	rule := validation.All(
		validation.SliceMaxLength[[]string](1),
		validation.Each[[]string](validation.NotEmpty[string](), validation.RuneMinLength[string](2)),
	)
	got := sliceIssues(rule([]string{"", ""}))
	want := []sliceIssue{
		{path: "$", code: validation.CodeMaxLength},
		{path: "$[0]", code: validation.CodeNotEmpty},
		{path: "$[0]", code: validation.CodeRuneMinLength},
		{path: "$[1]", code: validation.CodeNotEmpty},
		{path: "$[1]", code: validation.CodeRuneMinLength},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("issues = %#v, want %#v", got, want)
	}
}

func TestEachRunsEveryRuleForEveryElementAndCopiesConfiguration(t *testing.T) {
	var calls int
	child := func(value int) error {
		calls++
		if value == 0 {
			return validation.NewViolation(validation.CodeZero, nil)
		}
		return nil
	}
	rules := []validation.Rule[int]{child}
	rule := validation.Each[[]int](rules...)
	rules[0] = nil

	got := sliceIssues(rule([]int{0, 1, 0}))
	want := []sliceIssue{{path: "$[0]", code: validation.CodeZero}, {path: "$[2]", code: validation.CodeZero}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("issues = %#v, want %#v", got, want)
	}
	if calls != 3 {
		t.Fatalf("child calls = %d, want 3", calls)
	}
}

func TestAtIndexGuardsAbsentElementsAndKeepsIndependentRules(t *testing.T) {
	called := false
	panicRule := func(int) error {
		called = true
		panic("out-of-range child invoked")
	}
	rule := validation.All(
		validation.AtIndex[[]int](3, panicRule),
		validation.SliceMinLength[[]int](2),
	)
	got := sliceIssues(rule([]int{1}))
	want := []sliceIssue{{path: "$[3]", code: validation.CodeIndexOutOfRange}, {path: "$", code: validation.CodeMinLength}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("issues = %#v, want %#v", got, want)
	}
	if called {
		t.Fatal("out-of-range child was called")
	}
}

func TestAtIndexRunsAllChildrenAtAValidIndex(t *testing.T) {
	var calls [2]int
	first := func(value string) error {
		calls[0]++
		if value == "" {
			return validation.NewViolation(validation.CodeNotEmpty, nil)
		}
		return nil
	}
	second := func(value string) error {
		calls[1]++
		if len(value) < 2 {
			return validation.NewViolation(validation.CodeRuneMinLength, nil)
		}
		return nil
	}
	got := sliceIssues(validation.AtIndex[[]string](1, first, second)([]string{"ok", ""}))
	want := []sliceIssue{{path: "$[1]", code: validation.CodeNotEmpty}, {path: "$[1]", code: validation.CodeRuneMinLength}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("issues = %#v, want %#v", got, want)
	}
	if calls != [2]int{1, 1} {
		t.Fatalf("calls = %v, want [1 1]", calls)
	}
}

func TestSliceMembershipReportsEveryOffendingOccurrence(t *testing.T) {
	tests := []struct {
		name  string
		rule  validation.Rule[[]string]
		in    []string
		code  validation.Code
		paths []string
	}{
		{name: "one of", rule: validation.SliceOneOf[[]string]("a", "b"), in: []string{"x", "a", "y", "x"}, code: validation.CodeOneOf, paths: []string{"$[0]", "$[2]", "$[3]"}},
		{name: "not one of", rule: validation.SliceNotOneOf[[]string]("x"), in: []string{"x", "a", "x"}, code: validation.CodeNotOneOf, paths: []string{"$[0]", "$[2]"}},
		{name: "empty allowed set", rule: validation.SliceOneOf[[]string](), in: []string{"a", "b"}, code: validation.CodeOneOf, paths: []string{"$[0]", "$[1]"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sliceIssues(tt.rule(tt.in))
			if len(got) != len(tt.paths) {
				t.Fatalf("issue count = %d, want %d (%#v)", len(got), len(tt.paths), got)
			}
			for i, issue := range got {
				if issue.path != tt.paths[i] || issue.code != tt.code {
					t.Fatalf("issue[%d] = %#v, want path %q code %q", i, issue, tt.paths[i], tt.code)
				}
			}
		})
	}

	err := validation.SliceContains[[]string]("x")([]string{"a", "b", "c"})
	got := sliceIssues(err)
	if !reflect.DeepEqual(got, []sliceIssue{{path: "$", code: validation.CodeContains}}) {
		t.Fatalf("contains issues = %#v", got)
	}
}

func TestSliceUniqueReportsEachDuplicateAndEarliestIndex(t *testing.T) {
	input := []string{"a", "b", "a", "a", "b"}
	err := validation.SliceUnique[[]string, string]()(input)
	issues := validation.Issues(err)
	if got, want := len(issues), 3; got != want {
		t.Fatalf("issue count = %d, want %d", got, want)
	}
	wantPaths := []string{"$[2]", "$[3]", "$[4]"}
	wantFirst := []int{0, 0, 1}
	for i, issue := range issues {
		if got := validation.FormatPath(issue.Path); got != wantPaths[i] || issue.Code != validation.CodeUnique {
			t.Fatalf("issue[%d] = %s/%q", i, got, issue.Code)
		}
		var detail *validation.DuplicateError
		if !errors.As(issue.Err, &detail) || detail.FirstIndex() != wantFirst[i] {
			t.Fatalf("issue[%d] duplicate detail = %#v", i, detail)
		}
	}
	if !reflect.DeepEqual(input, []string{"a", "b", "a", "a", "b"}) {
		t.Fatal("uniqueness changed the input")
	}
}

func TestSliceUniqueUsesGoEqualityForNaNAndSignedZero(t *testing.T) {
	input := []float64{math.NaN(), math.NaN(), 0, math.Copysign(0, -1)}
	issues := validation.Issues(validation.SliceUnique[[]float64, float64]()(input))
	if len(issues) != 1 || validation.FormatPath(issues[0].Path) != "$[3]" {
		t.Fatalf("issues = %#v", sliceIssues(validation.SliceUnique[[]float64, float64]()(input)))
	}
	var detail *validation.DuplicateError
	if !errors.As(issues[0].Err, &detail) || detail.FirstIndex() != 2 {
		t.Fatalf("duplicate detail = %#v", detail)
	}
}

func TestSliceNamedTypesCompileAndPreserveNestedLocations(t *testing.T) {
	type Cell string
	type Row []Cell
	type Rows []Row
	rule := validation.Each[Rows](validation.Each[Row](func(value Cell) error {
		if value == "" {
			return validation.NewViolation(validation.CodeNotEmpty, nil)
		}
		return nil
	}))
	got := sliceIssues(rule(Rows{{"ok", "ok"}, {"ok", ""}}))
	want := []sliceIssue{{path: "$[1][1]", code: validation.CodeNotEmpty}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("issues = %#v, want %#v", got, want)
	}
}

func TestSliceConstructorsRejectInvalidConfiguration(t *testing.T) {
	assertConfigurationPanic(t, func() { validation.SliceLength[[]int](-1) })
	assertConfigurationPanic(t, func() { validation.SliceLengthBetween[[]int](2, 1) })
	assertConfigurationPanic(t, func() { validation.AtIndex[[]int](-1) })
	assertConfigurationPanic(t, func() { validation.Each[[]int](nil) })
}

func assertConfigurationPanic(t *testing.T, call func()) {
	t.Helper()
	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatal("expected configuration panic")
		} else if _, ok := recovered.(*validation.ConfigurationError); !ok {
			t.Fatalf("panic type = %T, want *ConfigurationError", recovered)
		}
	}()
	call()
}
