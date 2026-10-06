package validation_test

import (
	"errors"
	"math"
	"reflect"
	"regexp"
	"testing"
	"time"

	v "github.com/jacoelho/validation"
)

func TestDefaultFailureReasons(t *testing.T) {
	date := time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"field minimum", v.Min(2).Field("bar", func(n int) int { return n })(1), "$.bar: must be at least 2"},
		{"maximum precision", v.Max(uint64(18446744073709551614))(18446744073709551615), "$: must be at most 18446744073709551614"},
		{"range", v.Between(2, 5)(6), "$: must be between 2 and 5 (inclusive)"},
		{"exclusive lower", v.GreaterThan(2)(2), "$: must be greater than 2"},
		{"exclusive upper", v.LessThan(2)(2), "$: must be less than 2"},
		{"NaN minimum", v.Min(2.0)(math.NaN()), "$: must be at least 2"},
		{"trim and match", v.All(v.Trimmed[string](), v.Match[string](regexp.MustCompile(`^[A-Z][a-z]+$`)))(" Ada"), `$: must not have leading or trailing whitespace; $: must match pattern "^[A-Z][a-z]+$"`},
		{"multiple", v.MultipleOf(5)(12), "$: must be a multiple of 5"},
		{"float multiple", v.FloatMultipleOf(0.5, 0.01)(0.3), "$: must be a finite multiple of 0.5 within tolerance 0.01"},
		{"nonfinite multiple", v.FloatMultipleOf(0.5, 0.01)(math.Inf(1)), "$: must be a finite multiple of 0.5 within tolerance 0.01"},
		{"rune minimum", v.RuneMinLength[string](2)("é"), "$: length must be at least 2 runes (got 1)"},
		{"byte exact", v.ByteLength[string](1)("é"), "$: length must be 1 byte (got 2)"},
		{"byte maximum", v.BytesMaxLength[[]byte](2)([]byte("abc")), "$: length must be at most 2 bytes (got 3)"},
		{"length range", v.RuneLengthBetween[string](2, 4)("x"), "$: length must be between 2 and 4 runes (got 1)"},
		{"duplicate", v.SliceUnique[[]int]()([]int{4, 4}), "$[1]: duplicates element at index 0"},
		{"absent index", v.AtIndex[[]int](3)([]int{1}), "$[3]: index 3 is out of range for length 1"},
		{"slice length", v.SliceMinLength[[]int](2)(nil), "$: length must be at least 2 elements (got 0)"},
		{"contains", v.Contains("needle")("haystack"), `$: must contain "needle"`},
		{"byte prefix", v.BytesHasPrefix([]byte("a\n"))([]byte("b")), `$: must start with "a\n"`},
		{"suffix", v.HasSuffix(".go")("file.txt"), `$: must end with ".go"`},
		{"time layout", v.Time[string]("2006-01-02")("invalid"), `$: must match time layout "2006-01-02"`},
		{"before", v.TimeBefore(date)(date), "$: must be before 2026-09-29 00:00:00 +0000 UTC"},
		{"after inclusive", v.TimeAfterOrEqual(date)(date.Add(-time.Hour)), "$: must be at or after 2026-09-29 00:00:00 +0000 UTC"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Fatal("expected failure")
			}
			if got := v.Format(tt.err); got != tt.want {
				t.Errorf("Format = %q, want %q", got, tt.want)
			}
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error = %q, want %q", got, tt.want)
			}
		})
	}
}

type quotaError struct{ limit int }

func (e quotaError) Error() string              { panic("external Error must not be called") }
func (e quotaError) Code() v.Code               { return "quota" }
func (e quotaError) Parameters() map[string]any { return map[string]any{"limit": e.limit} }

func TestCustomErrorParameters(t *testing.T) {
	failure := quotaError{limit: 3}
	err := v.Field("uploads", func(int) int { return 0 }, v.Rule[int](func(int) error { return failure }))(0)
	issues := v.Issues(err)
	if len(issues) != 1 || issues[0].Code != "quota" || v.FormatPath(issues[0].Path) != "$.uploads" {
		t.Fatalf("issues = %#v", issues)
	}
	detail, ok := issues[0].Err.(v.Parameterized)
	if !ok || !reflect.DeepEqual(detail.Parameters(), map[string]any{"limit": 3}) {
		t.Fatal("lost custom parameters")
	}
	if !errors.Is(err, failure) {
		t.Fatal("lost custom error identity")
	}
	if got := v.Format(err); got != "$.uploads: quota" {
		t.Fatalf("default = %q", got)
	}
	if got := v.NewViolation("taken", nil).Parameters(); len(got) != 0 {
		t.Fatalf("unexpected parameters: %v", got)
	}
}

type customMessageError struct {
	code    v.Code
	message string
}

func (e customMessageError) Error() string   { panic("custom Error must not be called") }
func (e customMessageError) Code() v.Code    { return e.code }
func (e customMessageError) Message() string { return e.message }

var _ v.MessageProvider = customMessageError{}

type uncodedMessageError struct{}

func (uncodedMessageError) Error() string   { panic("external Error must not be called") }
func (uncodedMessageError) Message() string { panic("uncoded Message must not be called") }

func TestCustomDefaultMessages(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"custom message", customMessageError{"quota", "upload limit reached"}, "upload limit reached"},
		{"known code override", customMessageError{v.CodeRequired, "choose an account"}, "choose an account"},
		{"known code empty message", customMessageError{v.CodeRequired, ""}, "value is required"},
		{"unknown code empty message", customMessageError{"quota", ""}, "quota"},
		{"whitespace message", customMessageError{"quota", " \t "}, " \t "},
		{"uncoded message", uncodedMessageError{}, "external"},
		{"coded cause message", v.NewViolation("outer", customMessageError{"quota", "private cause message"}), "outer"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := v.Field("uploads", func(count int) int { return count }, v.Rule[int](func(int) error { return tt.err }))
			err := rule(4)
			issues := v.Issues(err)
			if len(issues) != 1 {
				t.Fatalf("issue count = %d, want 1", len(issues))
			}
			if got := v.DefaultMessage(issues[0]); got != tt.want {
				t.Errorf("DefaultMessage = %q, want %q", got, tt.want)
			}
			if got, want := v.Format(err), "$.uploads: "+tt.want; got != want {
				t.Errorf("Format = %q, want %q", got, want)
			}
		})
	}
}

func TestParameterSnapshots(t *testing.T) {
	cases := []struct {
		err  error
		want map[string]any
	}{
		{v.Between(2, 5)(1), map[string]any{"minimum": 2, "minimum_inclusive": true, "maximum": 5, "maximum_inclusive": true}},
		{v.GreaterThan(2)(2), map[string]any{"minimum": 2, "minimum_inclusive": false}},
		{v.Max(uint64(18446744073709551614))(18446744073709551615), map[string]any{"maximum": uint64(18446744073709551614), "maximum_inclusive": true}},
		{v.RuneLengthBetween[string](2, 4)("x"), map[string]any{"minimum": 2, "maximum": 4, "actual": 1, "unit": v.LengthRunes}},
		{v.Match[string](regexp.MustCompile("abc"))("x"), map[string]any{"constraint": "abc"}},
		{v.MultipleOf(5)(2), map[string]any{"base": 5}},
		{v.FloatMultipleOf(0.5, 0.0)(0.3), map[string]any{"base": 0.5, "tolerance": 0.0}},
		{v.SliceUnique[[]int]()([]int{4, 4}), map[string]any{"first_index": 0}},
		{v.AtIndex[[]int](3)([]int{1}), map[string]any{"index": 3, "length": 1}},
	}
	for _, tt := range cases {
		issue := v.Issues(tt.err)[0]
		detail, ok := issue.Err.(v.Parameterized)
		if !ok {
			t.Fatalf("missing parameters on %T", issue.Err)
		}
		got := detail.Parameters()
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s parameters = %#v, want %#v", issue.Code, got, tt.want)
		}
		clear(got)
		got["changed"] = true
		if !reflect.DeepEqual(detail.Parameters(), tt.want) {
			t.Fatal("parameters share map storage")
		}
	}
}

func TestTextConstraintDetails(t *testing.T) {
	prefix := []byte{'a', 0xff}
	rule := v.BytesHasPrefix(prefix)
	prefix[0] = 'b'
	tests := []struct {
		err        error
		code       v.Code
		constraint string
	}{
		{v.Match[string](regexp.MustCompile(`^[a-z]+$`))("1"), v.CodeMatch, `^[a-z]+$`},
		{v.Contains("needle")("hay"), v.CodeContains, "needle"},
		{rule([]byte("x")), v.CodePrefix, "a\xff"},
		{v.HasSuffix(".go")("a.txt"), v.CodeSuffix, ".go"},
		{v.Time[string]("2006-01-02")("invalid"), v.CodeTime, "2006-01-02"},
	}
	for _, tt := range tests {
		detail, ok := errors.AsType[*v.TextError](tt.err)
		if !ok {
			t.Fatalf("missing TextError: %T", tt.err)
		}
		if detail.Code() != tt.code || detail.Constraint() != tt.constraint {
			t.Errorf("constraint = %q/%q, want %q/%q", detail.Code(), detail.Constraint(), tt.code, tt.constraint)
		}
	}
}

func TestMultipleDetailsRetainTypeAndTolerance(t *testing.T) {
	type count uint64
	const base count = 18446744073709551614
	detail, ok := errors.AsType[*v.MultipleOfError[count]](v.MultipleOf(base)(1))
	if !ok || detail.Base() != base {
		t.Fatal("lost named type or base precision")
	}
	if _, present := detail.Tolerance(); present {
		t.Fatal("integer multiple has tolerance")
	}
	for _, tolerance := range []float64{0, 0.01} {
		detail, ok := errors.AsType[*v.MultipleOfError[float64]](v.FloatMultipleOf(0.5, tolerance)(0.3))
		if !ok || detail.Base() != 0.5 {
			t.Fatal("missing floating base")
		}
		if got, present := detail.Tolerance(); !present || got != tolerance {
			t.Fatalf("tolerance = %v/%t, want %v/true", got, present, tolerance)
		}
	}
}
