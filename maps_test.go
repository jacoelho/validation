package validation_test

import (
	"errors"
	"reflect"
	"strconv"
	"testing"

	"github.com/jacoelho/validation/v2"
)

type mapIssue struct {
	path string
	code validation.Code
}

func mapIssues(err error) []mapIssue {
	issues := validation.Issues(err)
	out := make([]mapIssue, len(issues))
	for i, issue := range issues {
		out[i] = mapIssue{path: validation.FormatPath(issue.Path), code: issue.Code}
	}
	return out
}

func TestMapLengthRulesUseEntryDiagnostics(t *testing.T) {
	tests := []struct {
		name   string
		rule   validation.Rule[map[string]int]
		in     map[string]int
		want   validation.Code
		min    int
		max    int
		hasMax bool
	}{
		{name: "exact", rule: validation.MapLength[map[string]int](2), in: map[string]int{"a": 1}, want: validation.CodeLength, min: 2, max: 2, hasMax: true},
		{name: "minimum", rule: validation.MapMinLength[map[string]int](2), in: map[string]int{"a": 1}, want: validation.CodeMinLength, min: 2},
		{name: "maximum", rule: validation.MapMaxLength[map[string]int](1), in: map[string]int{"a": 1, "b": 2}, want: validation.CodeMaxLength, min: 0, max: 1, hasMax: true},
		{name: "between", rule: validation.MapLengthBetween[map[string]int](2, 3), in: map[string]int{"a": 1}, want: validation.CodeLengthBetween, min: 2, max: 3, hasMax: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rule(tt.in)
			var detail *validation.LengthError
			if !errors.As(err, &detail) {
				t.Fatalf("expected LengthError, got %T", err)
			}
			if detail.Code() != tt.want || detail.Actual() != len(tt.in) || detail.Minimum() != tt.min || detail.Unit() != validation.LengthEntries {
				t.Fatalf("unexpected length detail: code=%q actual=%d minimum=%d unit=%q", detail.Code(), detail.Actual(), detail.Minimum(), detail.Unit())
			}
			gotMax, got := detail.Maximum()
			if got != tt.hasMax || got && gotMax != tt.max {
				t.Fatalf("unexpected maximum: (%d, %t)", gotMax, got)
			}
			if got := mapIssues(err); len(got) != 1 || got[0].path != "$" || got[0].code != tt.want {
				t.Fatalf("issues = %#v", got)
			}
		})
	}
	if validation.MapLength[map[string]int](0)(nil) != nil {
		t.Fatal("nil map with length zero should pass")
	}
}

func TestMapEachReportsEveryEntryFailureInKeyOrder(t *testing.T) {
	var keyCalls, valueCalls int
	keyRule := func(entry validation.Entry[string, int]) error {
		keyCalls++
		if entry.Key == "" {
			return validation.NewViolation(validation.CodeNotEmpty, nil)
		}
		return nil
	}
	valueRule := func(entry validation.Entry[string, int]) error {
		valueCalls++
		if entry.Value < 1 {
			return validation.NewViolation(validation.CodeMin, nil)
		}
		return nil
	}
	err := validation.MapEach[map[string]int](validation.StringKeys[string](), keyRule, valueRule)(map[string]int{
		"b": 0,
		"":  0,
		"a": 1,
	})
	got := mapIssues(err)
	want := []mapIssue{
		{path: `$[""]`, code: validation.CodeNotEmpty},
		{path: `$[""]`, code: validation.CodeMin},
		{path: `$["b"]`, code: validation.CodeMin},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("issues = %#v, want %#v", got, want)
	}
	if keyCalls != 3 || valueCalls != 3 {
		t.Fatalf("calls = key:%d value:%d, want 3 each", keyCalls, valueCalls)
	}
}

func TestMapKeysAndValuesRemainIndependent(t *testing.T) {
	order := validation.StringKeys[string]()
	rule := validation.All(
		validation.MapKeysOneOf[map[string]int](order, "a"),
		validation.MapValues[map[string]int](order, func(value int) error {
			if value < 1 {
				return validation.NewViolation(validation.CodeMin, nil)
			}
			return nil
		}),
	)
	got := mapIssues(rule(map[string]int{"z": 0, "y": 0, "a": 1}))
	want := []mapIssue{
		{path: `$["y"]`, code: validation.CodeOneOf},
		{path: `$["z"]`, code: validation.CodeOneOf},
		{path: `$["y"]`, code: validation.CodeMin},
		{path: `$["z"]`, code: validation.CodeMin},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("issues = %#v, want %#v", got, want)
	}
}

func TestMapKeysUsesKeysAndKeepsEveryChildFailure(t *testing.T) {
	keyCalls := 0
	rule := validation.MapKeys[map[string]string](
		validation.StringKeys[string](),
		func(key string) error {
			keyCalls++
			if key == "" {
				return validation.NewViolation(validation.CodeNotEmpty, nil)
			}
			return nil
		},
		func(key string) error {
			keyCalls++
			if len(key) < 2 {
				return validation.NewViolation(validation.CodeMinLength, nil)
			}
			return nil
		},
	)
	input := map[string]string{"": "valid", "a": "valid", "bb": ""}
	got := mapIssues(rule(input))
	want := []mapIssue{
		{path: `$[""]`, code: validation.CodeNotEmpty},
		{path: `$[""]`, code: validation.CodeMinLength},
		{path: `$["a"]`, code: validation.CodeMinLength},
	}
	if !reflect.DeepEqual(got, want) || keyCalls != 6 {
		t.Fatalf("key issues=%#v calls=%d, want %#v and 6 calls", got, keyCalls, want)
	}
}

func TestMapKeysNotOneOfRejectsOnlyForbiddenKeys(t *testing.T) {
	order := validation.StringKeys[string]()
	rule := validation.MapKeysNotOneOf[map[string]int](order, "z", "a", "z")
	got := mapIssues(rule(map[string]int{"z": 3, "b": 2, "a": 1}))
	want := []mapIssue{
		{path: `$["a"]`, code: validation.CodeNotOneOf},
		{path: `$["z"]`, code: validation.CodeNotOneOf},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("forbidden key issues=%#v, want %#v", got, want)
	}
	if err := validation.MapKeysNotOneOf[map[string]int](order)(map[string]int{"a": 1, "z": 2}); err != nil {
		t.Fatalf("empty forbidden set failed: %v", err)
	}
	if err := rule(nil); err != nil {
		t.Fatalf("nil map failed: %v", err)
	}
}

func TestMapSuccessfulValidationDoesNotCallOrderingCallbacks(t *testing.T) {
	lessCalls, textCalls := 0, 0
	order := validation.KeyOrder[string]{
		Less: func(left, right string) bool {
			lessCalls++
			return left < right
		},
		Text: func(key string) string {
			textCalls++
			return key
		},
	}
	values := make(map[string]int, 1000)
	for i := 0; i < 1000; i++ {
		values[strconv.Itoa(i)] = i
	}
	rule := validation.MapValues[map[string]int](order, func(int) error { return nil })
	if err := rule(values); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lessCalls != 0 || textCalls != 0 {
		t.Fatalf("ordering callbacks = less:%d text:%d, want zero", lessCalls, textCalls)
	}
}

func TestMapFailureGroupsSortAndRenderOnceWithoutReplayingRules(t *testing.T) {
	failed := map[string]bool{"z": true, "a": true, "m": true}
	lessArgs := make([]string, 0)
	textCalls := make(map[string]int)
	order := validation.KeyOrder[string]{
		Less: func(left, right string) bool {
			lessArgs = append(lessArgs, left, right)
			return left < right
		},
		Text: func(key string) string {
			textCalls[key]++
			return key
		},
	}
	var firstCalls, secondCalls int
	first := func(value string) error {
		firstCalls++
		if failed[value] {
			return validation.NewViolation(validation.CodeMin, nil)
		}
		return nil
	}
	second := func(value string) error {
		secondCalls++
		if failed[value] {
			return validation.NewViolation(validation.CodeMax, nil)
		}
		return nil
	}
	values := map[string]string{"z": "z", "a": "a", "m": "m", "pass": "pass"}
	rule := validation.MapValues[map[string]string](order, first, second)
	got := mapIssues(rule(values))
	want := []mapIssue{
		{path: `$["a"]`, code: validation.CodeMin},
		{path: `$["a"]`, code: validation.CodeMax},
		{path: `$["m"]`, code: validation.CodeMin},
		{path: `$["m"]`, code: validation.CodeMax},
		{path: `$["z"]`, code: validation.CodeMin},
		{path: `$["z"]`, code: validation.CodeMax},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("issues = %#v, want %#v", got, want)
	}
	if firstCalls != len(values) || secondCalls != len(values) {
		t.Fatalf("child calls = %d/%d, want %d/%d", firstCalls, secondCalls, len(values), len(values))
	}
	for _, key := range []string{"a", "m", "z"} {
		if textCalls[key] != 1 {
			t.Fatalf("Text(%q) calls = %d, want 1", key, textCalls[key])
		}
	}
	for _, key := range lessArgs {
		if !failed[key] {
			t.Fatalf("Less called with non-failing key %q; calls = %#v", key, lessArgs)
		}
	}
}

func TestMapKeyPunctuationRemainsLiteral(t *testing.T) {
	values := map[string]int{"a.b": 0, "a[0]": 0, "": 0}
	rule := validation.MapValues[map[string]int](validation.StringKeys[string](), func(int) error {
		return validation.NewViolation(validation.CodeMin, nil)
	})
	got := mapIssues(rule(values))
	want := []mapIssue{
		{path: `$[""]`, code: validation.CodeMin},
		{path: `$["a.b"]`, code: validation.CodeMin},
		{path: `$["a[0]"]`, code: validation.CodeMin},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("issues = %#v, want %#v", got, want)
	}
}

func TestMapRequiredAndOptionalKeyPresence(t *testing.T) {
	textCalls := make(map[string]int)
	order := validation.KeyOrder[string]{
		Less: func(left, right string) bool { return left < right },
		Text: func(key string) string {
			textCalls[key]++
			return key
		},
	}
	firstCalls, secondCalls := 0, 0
	first := func(value int) error {
		firstCalls++
		if value == 0 {
			return validation.NewViolation(validation.CodeMin, nil)
		}
		return nil
	}
	second := func(value int) error {
		secondCalls++
		if value == 0 {
			return validation.NewViolation(validation.CodeMax, nil)
		}
		return nil
	}
	rule := validation.All(
		validation.MapRequiredKey[map[string]int]("x", order, first, second),
		validation.MapOptionalKey[map[string]int]("y", order, first, second),
	)
	got := mapIssues(rule(map[string]int{}))
	want := []mapIssue{{path: `$["x"]`, code: validation.CodeKeyRequired}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("issues = %#v, want %#v", got, want)
	}
	if firstCalls != 0 || secondCalls != 0 || textCalls["x"] != 1 || textCalls["y"] != 0 {
		t.Fatalf("calls = first:%d second:%d text:%v, want no children and x only", firstCalls, secondCalls, textCalls)
	}

	got = mapIssues(rule(map[string]int{"x": 0, "y": 0}))
	want = []mapIssue{
		{path: `$["x"]`, code: validation.CodeMin},
		{path: `$["x"]`, code: validation.CodeMax},
		{path: `$["y"]`, code: validation.CodeMin},
		{path: `$["y"]`, code: validation.CodeMax},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("present zero issues = %#v, want %#v", got, want)
	}
	if firstCalls != 2 || secondCalls != 2 || textCalls["x"] != 2 || textCalls["y"] != 1 {
		t.Fatalf("present calls = first:%d second:%d text:%v", firstCalls, secondCalls, textCalls)
	}
}

func TestMapNamedAndIntegerKeyTypesCompile(t *testing.T) {
	type Key int
	type Values map[Key]string
	order := validation.KeyOrder[Key]{
		Less: func(left, right Key) bool { return left < right },
		Text: func(key Key) string { return strconv.Itoa(int(key)) },
	}
	rule := validation.MapValues[Values](order, func(value string) error {
		if value == "" {
			return validation.NewViolation(validation.CodeNotEmpty, nil)
		}
		return nil
	})
	got := mapIssues(rule(Values{10: "", 2: ""}))
	want := []mapIssue{{path: `$["2"]`, code: validation.CodeNotEmpty}, {path: `$["10"]`, code: validation.CodeNotEmpty}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("issues = %#v, want %#v", got, want)
	}
}

func TestMapRulesDoNotMutateInput(t *testing.T) {
	values := map[string]int{"a": 0, "b": 1}
	want := map[string]int{"a": 0, "b": 1}
	rule := validation.MapValues[map[string]int](validation.StringKeys[string](), func(value int) error {
		if value == 0 {
			return validation.NewViolation(validation.CodeMin, nil)
		}
		return nil
	})
	_ = rule(values)
	if !reflect.DeepEqual(values, want) {
		t.Fatalf("map changed: got %#v, want %#v", values, want)
	}
}

func TestMapConstructorsRejectInvalidConfiguration(t *testing.T) {
	assertMapConfigurationPanic(t, func() { validation.MapLength[map[string]int](-1) })
	assertMapConfigurationPanic(t, func() { validation.MapLengthBetween[map[string]int](2, 1) })
	assertMapConfigurationPanic(t, func() { validation.MapValues[map[string]int](validation.KeyOrder[string]{}) })
	assertMapConfigurationPanic(t, func() { validation.MapEach[map[string]int](validation.StringKeys[string](), nil) })
}

func assertMapConfigurationPanic(t *testing.T, call func()) {
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
