package validation_test

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	validation "github.com/jacoelho/validation/v2"
)

func TestAT_ERRORS_010(t *testing.T) {
	cases := []struct {
		name       string
		failure    func() error
		actual     int
		minimum    int
		maximum    int
		hasMaximum bool
		unit       validation.LengthUnit
	}{
		{"row-1", func() error { return validation.RuneMinLength[string](2)("") }, 0, 2, 0, false, validation.LengthRunes},
		{"row-2", func() error { return validation.ByteMaxLength[string](1)("é") }, 2, 0, 1, true, validation.LengthBytes},
		{"row-3", func() error { return validation.SliceLength[[]int](2)([]int{1, 2, 3}) }, 3, 2, 2, true, validation.LengthElements},
		{"row-4", func() error { return validation.MapLengthBetween[map[string]int](1, 2)(nil) }, 0, 1, 2, true, validation.LengthEntries},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var detail *validation.LengthError
			if !errors.As(tc.failure(), &detail) {
				t.Fatal("missing LengthError")
			}
			maximum, present := detail.Maximum()
			if detail.Actual() != tc.actual || detail.Minimum() != tc.minimum || present != tc.hasMaximum ||
				(present && maximum != tc.maximum) || detail.Unit() != tc.unit {
				t.Fatalf("length detail = actual %d, minimum %d, maximum (%d, %t), unit %q", detail.Actual(), detail.Minimum(), maximum, present, detail.Unit())
			}
		})
	}
}

func TestAT_ERRORS_001(t *testing.T) {
	sentinel := errors.New("root sentinel")
	rule := validation.All(
		validation.Rule[int](func(int) error { return nil }),
		validation.Rule[int](func(int) error { return sentinel }),
		validation.Rule[int](func(int) error { return nil }),
	)

	err := rule(7)
	if err != sentinel {
		t.Fatalf("returned error = %v (%T), want exact sentinel", err, err)
	}
	issues := validation.Issues(err)
	if len(issues) != 1 {
		t.Fatalf("issues = %#v, want one root occurrence", issues)
	}
	if got := validation.FormatPath(issues[0].Path); got != "$" || issues[0].Code != validation.CodeExternal || issues[0].Err != sentinel {
		t.Fatalf("issue = %+v at %q, want external root sentinel", issues[0], got)
	}
}

func TestAT_ERRORS_003(t *testing.T) {
	sentinel := errors.New("amount cause")
	violation := validation.NewViolation(validation.CodeMin, sentinel)
	type input struct{ Amount int }
	rule := validation.Field("amount", func(value input) int { return value.Amount }, validation.Rule[int](func(int) error {
		return violation
	}))

	err := rule(input{Amount: 0})
	if !errors.Is(err, sentinel) {
		t.Fatal("errors.Is did not find the coded violation cause")
	}
	issues := validation.Issues(err)
	if len(issues) != 1 {
		t.Fatalf("issues = %#v, want one occurrence", issues)
	}
	issue := issues[0]
	if validation.FormatPath(issue.Path) != "$.amount" || issue.Code != validation.CodeMin || issue.Err != violation {
		t.Fatalf("issue = %+v, want one min issue at $.amount", issue)
	}
	var coded *validation.Violation
	if !errors.As(issue.Err, &coded) || coded != violation || !errors.Is(issue.Err, sentinel) {
		t.Fatalf("issue error lost coded identity or cause: %T", issue.Err)
	}
}

func TestAT_ERRORS_004(t *testing.T) {
	first := errors.New("first")
	second := errors.New("second")
	rule := validation.Field("amount", func(int) int { return 0 }, validation.Rule[int](func(int) error {
		return fmt.Errorf("application context: %w", errors.Join(first, second))
	}))

	err := rule(0)
	if !errors.Is(err, first) || !errors.Is(err, second) {
		t.Fatal("errors.Is did not find both joined terminals")
	}
	issues := validation.Issues(err)
	want := []struct {
		path string
		code validation.Code
		err  error
	}{
		{path: "$.amount", code: validation.CodeExternal, err: first},
		{path: "$.amount", code: validation.CodeExternal, err: second},
	}
	if len(issues) != len(want) {
		t.Fatalf("issues = %#v, want %#v", issues, want)
	}
	for i, issue := range issues {
		if got := validation.FormatPath(issue.Path); got != want[i].path || issue.Code != want[i].code || issue.Err != want[i].err {
			t.Fatalf("issue[%d] = %+v at %q, want terminal %v at %s", i, issue, got, want[i].err, want[i].path)
		}
	}
}

func TestAT_ERRORS_006(t *testing.T) {
	first := errors.New("first")
	second := errors.New("second")
	err := validation.All(
		validation.Rule[int](func(int) error { return first }),
		validation.Rule[int](func(int) error { return second }),
	)(0)

	multi, ok := err.(interface{ Unwrap() []error })
	if !ok {
		t.Fatalf("aggregate %T does not expose multi-error unwrapping", err)
	}
	exposed := multi.Unwrap()
	exposed[0] = errors.New("replacement")
	fresh := multi.Unwrap()
	if len(fresh) != 2 || fresh[0] != first || fresh[1] != second {
		t.Fatalf("fresh children = %#v, want original terminals", fresh)
	}
	if !errors.Is(err, first) || !errors.Is(err, second) {
		t.Fatal("mutating exposed children changed original error causes")
	}
}

func TestAT_ERRORS_008(t *testing.T) {
	ordinary := errors.New("ordinary application error")
	var calls [3]int
	rule := validation.All(
		validation.Rule[int](func(int) error { calls[0]++; return ordinary }),
		validation.Rule[int](func(int) error { calls[1]++; return validation.NewViolation(validation.CodeMin, nil) }),
		validation.Rule[int](func(int) error { calls[2]++; return validation.NewViolation(validation.CodeMax, nil) }),
	)

	err := rule(0)
	if calls != [3]int{1, 1, 1} {
		t.Fatalf("rule calls = %v, want one call each", calls)
	}
	issues := validation.Issues(err)
	wantCodes := []validation.Code{validation.CodeExternal, validation.CodeMin, validation.CodeMax}
	if len(issues) != len(wantCodes) {
		t.Fatalf("issues = %#v, want %v", issues, wantCodes)
	}
	for i, want := range wantCodes {
		if issues[i].Code != want || validation.FormatPath(issues[i].Path) != "$" {
			t.Errorf("issue[%d] = %+v, want %q at root", i, issues[i], want)
		}
	}
	if !errors.Is(err, ordinary) {
		t.Fatal("errors.Is did not find ordinary application error")
	}
}

func TestAT_PATHS_001(t *testing.T) {
	type address struct{ City string }
	type input struct{ Addresses []address }
	rule := validation.Field("addresses", func(value input) []address { return value.Addresses }, validation.Each[[]address](
		validation.Field("city", func(value address) string { return value.City }, validation.NotEmpty[string]()),
	))

	issues := validation.Issues(rule(input{Addresses: []address{{City: "London"}, {City: ""}}}))
	if len(issues) != 1 {
		t.Fatalf("issues = %#v, want one issue", issues)
	}
	wantPath := []validation.Segment{
		{Kind: validation.FieldSegment, Name: "addresses"},
		{Kind: validation.IndexSegment, Index: 1},
		{Kind: validation.FieldSegment, Name: "city"},
	}
	if !reflect.DeepEqual(issues[0].Path, wantPath) {
		t.Fatalf("path = %#v, want %#v", issues[0].Path, wantPath)
	}
	if got := validation.FormatPath(issues[0].Path); got != "$.addresses[1].city" {
		t.Fatalf("rendered path = %q, want $.addresses[1].city", got)
	}
}

func TestAT_PATHS_002(t *testing.T) {
	t.Run("row-8", func(t *testing.T) {
		path := []validation.Segment{{Kind: validation.FieldSegment, Name: "2"}}
		if got := validation.FormatPath(path); got != "$.[\"2\"]" {
			t.Fatalf("numeric field rendered ambiguously as %q", got)
		}
	})
}

func TestAT_PATHS_003(t *testing.T) {
	key := "a\"b\\c\n\xff"
	got := validation.FormatPath([]validation.Segment{{Kind: validation.KeySegment, Name: key}})
	want := "$[\"a\\\"b\\\\c\\n\\xff\"]"
	if got != want {
		t.Fatalf("escaped key path = %q, want %q", got, want)
	}
	if strings.Contains(got, "\n") {
		t.Fatal("rendered key contains a raw newline")
	}
}

func TestAT_PATHS_004(t *testing.T) {
	type input struct{ Items []string }
	rule := validation.Field("items", func(value input) []string { return value.Items }, validation.Each[[]string](validation.NotEmpty[string]()))
	err := rule(input{Items: []string{"", ""}})
	issues := validation.Issues(err)
	if len(issues) != 2 {
		t.Fatalf("issues = %#v, want two issues", issues)
	}
	issues[0].Path[0].Name = "changed"
	fresh := validation.Issues(err)
	if len(fresh) != 2 || validation.FormatPath(fresh[0].Path) != "$.items[0]" || validation.FormatPath(fresh[1].Path) != "$.items[1]" {
		t.Fatalf("fresh issues = %#v, want both original paths", fresh)
	}
}

func TestAT_PATHS_005(t *testing.T) {
	type address struct{ City string }
	type input struct{ Address *address }
	rule := validation.Field("address", func(value input) *address { return value.Address }, validation.RequiredPtr(
		validation.Project(func(value address) address { return value },
			validation.Field("city", func(value address) string { return value.City }, validation.NotEmpty[string]()),
		),
	))

	issues := validation.Issues(rule(input{Address: &address{}}))
	if len(issues) != 1 || validation.FormatPath(issues[0].Path) != "$.address.city" {
		t.Fatalf("issues = %#v, want only $.address.city", issues)
	}
	if len(issues[0].Path) != 2 || issues[0].Path[0].Kind != validation.FieldSegment || issues[0].Path[1].Kind != validation.FieldSegment {
		t.Fatalf("path segments = %#v, want only address and city fields", issues[0].Path)
	}
}

func TestAT_SLICES_002(t *testing.T) {
	t.Run("row-3", func(t *testing.T) {
		if err := validation.SliceMaxLength[[]string](0)([]string{}); err != nil {
			t.Fatalf("empty slice max length = %v, want nil", err)
		}
	})
	t.Run("row-4", func(t *testing.T) {
		if got := sliceIssues(validation.SliceLengthBetween[[]string](1, 2)([]string{})); !reflect.DeepEqual(got, []sliceIssue{{path: "$", code: validation.CodeLengthBetween}}) {
			t.Fatalf("empty slice between issues = %#v, want one root length_between issue", got)
		}
	})
	t.Run("row-5", func(t *testing.T) {
		if err := validation.Each[[]string](validation.NotEmpty[string]())(nil); err != nil {
			t.Fatalf("nil slice Each = %v, want nil", err)
		}
	})
}

func TestAT_SLICES_010(t *testing.T) {
	type bytesValue []byte
	input := make(bytesValue, 3, 8)
	copy(input, []byte{1, 2, 3})
	for i := len(input); i < cap(input); i++ {
		input = input[:cap(input)]
		input[i] = byte(0xa0 + i)
		input = input[:3]
	}
	visibleBefore := append([]byte(nil), input...)
	spareBefore := append([]byte(nil), input[3:cap(input)]...)

	rules := []validation.Rule[bytesValue]{
		validation.BytesNotEmpty[bytesValue](),
		validation.BytesLength[bytesValue](3),
		validation.BytesMinLength[bytesValue](2),
		validation.BytesMaxLength[bytesValue](4),
		validation.BytesLengthBetween[bytesValue](2, 4),
		validation.BytesContains[bytesValue](bytesValue{2}),
		validation.BytesHasPrefix[bytesValue](bytesValue{1}),
		validation.BytesHasSuffix[bytesValue](bytesValue{3}),
		validation.BytesUTF8[bytesValue](),
		validation.BytesEqual[bytesValue](bytesValue{1, 2, 3}),
		validation.SliceLength[bytesValue](3),
		validation.SliceMinLength[bytesValue](2),
		validation.SliceMaxLength[bytesValue](4),
		validation.SliceLengthBetween[bytesValue](2, 4),
		validation.SliceContains[bytesValue, byte](2),
		validation.SliceOneOf[bytesValue, byte](1, 2, 3),
		validation.SliceNotOneOf[bytesValue, byte](9),
		validation.SliceUnique[bytesValue, byte](),
	}
	for i, rule := range rules {
		if err := rule(input); err != nil {
			t.Fatalf("rule %d unexpectedly failed: %v", i, err)
		}
	}

	if len(input) != 3 || cap(input) != 8 || !reflect.DeepEqual([]byte(input), visibleBefore) || !reflect.DeepEqual([]byte(input[3:cap(input)]), spareBefore) {
		t.Fatalf("input storage changed: len=%d cap=%d visible=%v spare=%v", len(input), cap(input), input, input[3:cap(input)])
	}

	failed := validation.All(
		validation.BytesContains[bytesValue](bytesValue{9}),
		validation.SliceContains[bytesValue, byte](9),
		validation.SliceUnique[bytesValue, byte](),
	)
	input[2] = 1
	if !retainsByteBackingArray(retainedBytes{input}, input) {
		t.Fatal("backing-array retention oracle missed a direct slice reference")
	}
	if err := failed(input); err == nil {
		t.Fatal("failing built-in byte/slice rules returned nil")
	} else if retainsByteBackingArray(err, input) {
		t.Fatal("built-in diagnostic retains rejected input backing array")
	}
	if !reflect.DeepEqual([]byte(input[3:cap(input)]), spareBefore) {
		t.Fatal("failing built-in rules changed spare-capacity sentinel bytes")
	}
}

type retainedBytes struct{ value []byte }

func (retainedBytes) Error() string { return "retained bytes" }

func retainsByteBackingArray(err error, input []byte) bool {
	start := reflect.ValueOf(input).Pointer()
	end := start + uintptr(cap(input))
	seen := make(map[uintptr]bool)
	var visit func(reflect.Value) bool
	visit = func(value reflect.Value) bool {
		if !value.IsValid() {
			return false
		}
		switch value.Kind() {
		case reflect.Interface:
			return !value.IsNil() && visit(value.Elem())
		case reflect.Pointer:
			if value.IsNil() {
				return false
			}
			pointer := value.Pointer()
			if seen[pointer] {
				return false
			}
			seen[pointer] = true
			return visit(value.Elem())
		case reflect.Slice:
			if value.IsNil() {
				return false
			}
			pointer := value.Pointer()
			if pointer >= start && pointer < end {
				return true
			}
			for i := 0; i < value.Len(); i++ {
				if visit(value.Index(i)) {
					return true
				}
			}
		case reflect.String:
			pointer := value.Pointer()
			return pointer >= start && pointer < end
		case reflect.Struct:
			for i := 0; i < value.NumField(); i++ {
				if visit(value.Field(i)) {
					return true
				}
			}
		}
		return false
	}
	return visit(reflect.ValueOf(err))
}

func TestAT_MAPS_001(t *testing.T) {
	t.Run("row-1", func(t *testing.T) {
		if got := mapIssues(validation.MapRequiredKey[map[string]int]("x", validation.StringKeys[string](), validation.Min[int](1))(nil)); !reflect.DeepEqual(got, []mapIssue{{path: `$["x"]`, code: validation.CodeKeyRequired}}) {
			t.Fatalf("nil required key issues = %#v, want key_required", got)
		}
	})
}

func TestAT_MAPS_002(t *testing.T) {
	var calls [2]int
	rule := validation.All(
		validation.Rule[map[string]int](func(values map[string]int) error { calls[0]++; return validation.MapLength[map[string]int](0)(values) }),
		validation.Rule[map[string]int](func(values map[string]int) error {
			calls[1]++
			return validation.MapMinLength[map[string]int](1)(values)
		}),
	)
	got := mapIssues(rule(nil))
	if !reflect.DeepEqual(got, []mapIssue{{path: "$", code: validation.CodeMinLength}}) {
		t.Fatalf("nil map length issues = %#v, want min_length", got)
	}
	if calls != [2]int{1, 1} {
		t.Fatalf("length rule calls = %v, want one each", calls)
	}
}

func TestAT_MAPS_005(t *testing.T) {
	values := make(map[string]int, 10000)
	for i := 0; i < 10000; i++ {
		values[strconv.Itoa(i)] = i
	}
	var lessCalls, textCalls int
	order := validation.KeyOrder[string]{
		Less: func(left, right string) bool { lessCalls++; return left < right },
		Text: func(key string) string { textCalls++; return key },
	}
	rule := validation.MapValues[map[string]int](order, func(int) error { return nil })
	allocs := testing.AllocsPerRun(100, func() {
		if err := rule(values); err != nil {
			t.Fatalf("valid map = %v", err)
		}
	})
	if allocs != 0 || lessCalls != 0 || textCalls != 0 {
		t.Fatalf("valid map allocs=%v less calls=%d text calls=%d, want zero", allocs, lessCalls, textCalls)
	}
}

func TestAT_MAPS_007(t *testing.T) {
	order := validation.StringKeys[string]()
	rule := validation.MapValues[map[string]int](order, func(int) error {
		return validation.NewViolation(validation.CodeMin, nil)
	})
	permutations := [][]string{
		{"a", "m", "z"}, {"a", "z", "m"}, {"m", "a", "z"},
		{"m", "z", "a"}, {"z", "a", "m"}, {"z", "m", "a"},
	}
	want := []mapIssue{
		{path: `$["a"]`, code: validation.CodeMin},
		{path: `$["m"]`, code: validation.CodeMin},
		{path: `$["z"]`, code: validation.CodeMin},
	}
	for permutation, keys := range permutations {
		values := make(map[string]int, len(keys))
		for _, key := range keys {
			values[key] = 0
		}
		for repeat := 0; repeat < 20; repeat++ {
			if got := mapIssues(rule(values)); !reflect.DeepEqual(got, want) {
				t.Fatalf("permutation %d repeat %d issues = %#v, want %#v", permutation, repeat, got, want)
			}
		}
	}
}

func TestAT_MAPS_010(t *testing.T) {
	order := validation.StringKeys[string]()
	rule := validation.MapValues[map[string]int](order, func(value int) error {
		if value == 0 {
			return validation.NewViolation(validation.CodeMin, nil)
		}
		return nil
	})
	failing := map[string]int{"a": 0, "b": 1}
	passing := map[string]int{"ok": 1}
	failingBefore := map[string]int{"a": 0, "b": 1}

	first := mapIssues(rule(failing))
	if err := rule(passing); err != nil {
		t.Fatalf("passing map = %v", err)
	}
	second := mapIssues(rule(failing))
	want := []mapIssue{{path: `$["a"]`, code: validation.CodeMin}}
	if !reflect.DeepEqual(first, want) || !reflect.DeepEqual(second, want) {
		t.Fatalf("reusable results = first:%#v second:%#v, want %#v", first, second, want)
	}
	if !reflect.DeepEqual(failing, failingBefore) || !reflect.DeepEqual(passing, map[string]int{"ok": 1}) {
		t.Fatalf("map values changed: failing=%#v passing=%#v", failing, passing)
	}

	delete(failing, "a")
	failing["c"] = 0
	if got := mapIssues(rule(failing)); !reflect.DeepEqual(got, []mapIssue{{path: `$["c"]`, code: validation.CodeMin}}) {
		t.Fatalf("reused validator retained old map state: %#v", got)
	}
	if _, ok := failing["a"]; ok {
		t.Fatal("validator restored or mutated map membership")
	}
}
