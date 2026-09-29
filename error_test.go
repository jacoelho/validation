package validation_test

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	v "github.com/jacoelho/validation"
)

type opaqueError struct{ text string }

func (e *opaqueError) Error() string { panic("external Error method must not be called by Format") }

type nilChildError struct{}

func (*nilChildError) Error() string { panic("external Error method must not be called by Format") }
func (*nilChildError) Unwrap() error { return nil }

func TestSingleUnwrapWithNilChildRemainsExternalTerminal(t *testing.T) {
	terminal := &nilChildError{}
	rule := v.Field("item", func(int) int { return 1 }, v.Rule[int](func(int) error { return terminal }))
	err := rule(0)
	issues := v.Issues(err)
	if len(issues) != 1 || issues[0].Code != v.CodeExternal || issues[0].Err != terminal || v.FormatPath(issues[0].Path) != "$.item" {
		t.Fatalf("nil-child single unwrap issues = %+v", issues)
	}
	if got := v.Format(err); got != "$.item: external" {
		t.Fatalf("safe format = %q", got)
	}
}

func TestErrorTreeInteroperability(t *testing.T) {
	cause := errors.New("private cause")
	coded := v.NewViolation("custom", cause)
	plain := &opaqueError{"private message"}
	rule := v.Field("outer", func(int) int { return 1 }, v.All(
		v.Rule[int](func(int) error { return fmt.Errorf("wrapped: %w", coded) }),
		v.Rule[int](func(int) error { return errors.Join(plain, plain) }),
	))
	err := rule(0)
	if !errors.Is(err, cause) || !errors.Is(err, coded) || !errors.Is(err, plain) {
		t.Fatal("standard unwrapping lost identities")
	}
	var violation *v.Violation
	if !errors.As(err, &violation) || violation != coded {
		t.Fatal("standard errors.As cannot find coded cause")
	}
	issues := v.Issues(err)
	if len(issues) != 3 {
		t.Fatalf("issues = %d, want 3", len(issues))
	}
	wantCodes := []v.Code{"custom", v.CodeExternal, v.CodeExternal}
	for i, issue := range issues {
		if issue.Code != wantCodes[i] || v.FormatPath(issue.Path) != "$.outer" {
			t.Errorf("issue[%d] = %+v", i, issue)
		}
	}
	if issues[0].Err != coded || issues[1].Err != plain || issues[2].Err != plain {
		t.Fatal("terminal identity lost or deduplicated")
	}
	if got := v.Format(err); got != "$.outer: custom; $.outer: external; $.outer: external" {
		t.Fatalf("safe format = %q", got)
	}
	if got := v.Format(nil); got != "" {
		t.Fatalf("Format(nil) = %q", got)
	}
	if got := v.Issues(nil); got != nil {
		t.Fatalf("Issues(nil) = %#v", got)
	}
}

func TestMultiUnwrapIsDefensive(t *testing.T) {
	rule := v.All(v.Rule[int](func(int) error { return errors.New("a") }), v.Rule[int](func(int) error { return errors.New("b") }))
	err := rule(0)
	multi, ok := err.(interface{ Unwrap() []error })
	if !ok {
		t.Fatalf("%T lacks multi Unwrap", err)
	}
	children := multi.Unwrap()
	children[0] = errors.New("changed")
	if got := v.Format(err); got != "$: external; $: external" {
		t.Fatalf("mutated aggregate = %q", got)
	}
}

func TestFormatPathGrammar(t *testing.T) {
	cases := []struct {
		path []v.Segment
		want string
	}{
		{nil, "$"},
		{[]v.Segment{{Kind: v.FieldSegment, Name: "name"}}, "$.name"},
		{[]v.Segment{{Kind: v.FieldSegment, Name: "a.b"}}, `$.["a.b"]`},
		{[]v.Segment{{Kind: v.FieldSegment, Name: "a"}, {Kind: v.FieldSegment, Name: "b"}}, "$.a.b"},
		{[]v.Segment{{Kind: v.KeySegment, Name: "a.b"}}, `$["a.b"]`},
		{[]v.Segment{{Kind: v.KeySegment, Name: "2"}}, `$["2"]`},
		{[]v.Segment{{Kind: v.IndexSegment, Index: 2}}, "$[2]"},
		{[]v.Segment{{Kind: v.KeySegment, Name: ""}}, `$[""]`},
		{[]v.Segment{{Kind: v.FieldSegment, Name: "a\n\xff"}}, `$.["a\n\xff"]`},
	}
	for _, tt := range cases {
		if got := v.FormatPath(tt.path); got != tt.want {
			t.Errorf("FormatPath(%#v) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestMalformedPathPanics(t *testing.T) {
	for _, segment := range []v.Segment{{Kind: v.FieldSegment}, {Kind: v.IndexSegment, Index: -1}, {Kind: 99}} {
		func() {
			defer func() {
				if _, ok := recover().(*v.ConfigurationError); !ok {
					t.Errorf("segment %+v did not panic with ConfigurationError", segment)
				}
			}()
			v.FormatPath([]v.Segment{segment})
		}()
	}
}

func TestTypedErrorDetails(t *testing.T) {
	err := v.RuneMinLength[string](2)("x")
	var length *v.LengthError
	if !errors.As(err, &length) {
		t.Fatal("missing LengthError")
	}
	if length.Code() != v.CodeRuneMinLength || length.Actual() != 1 || length.Minimum() != 2 || length.Unit() != v.LengthRunes {
		t.Fatalf("length details = %+v", length)
	}
	if _, present := length.Maximum(); present {
		t.Fatal("minimum-only length has maximum")
	}
	err = v.Min(5)(3)
	var bounds *v.BoundsError[int]
	if !errors.As(err, &bounds) {
		t.Fatal("missing BoundsError[int]")
	}
	lower, inclusive, present := bounds.Lower()
	if lower != 5 || !inclusive || !present {
		t.Fatalf("lower = %d/%t/%t", lower, inclusive, present)
	}
	if _, _, present := bounds.Upper(); present {
		t.Fatal("Min has upper bound")
	}
}

func TestIssueSnapshotIndependence(t *testing.T) {
	rule := v.Field("name", func(string) string { return "" }, v.NotEmpty[string](), v.NotEmpty[string]())
	err := rule("x")
	first := v.Issues(err)
	second := v.Issues(err)
	first[0].Path[0].Name = "changed"
	first[0].Code = "changed"
	if !slices.EqualFunc(second, v.Issues(err), func(a, b v.Issue) bool {
		return a.Code == b.Code && slices.Equal(a.Path, b.Path)
	}) {
		t.Fatal("issue mutation changed source tree")
	}
	if second[0].Path[0].Name != "name" || first[1].Path[0].Name != "name" {
		t.Fatal("issue paths share storage")
	}
}

func TestErrorWrapperVariants(t *testing.T) {
	a := errors.New("A")
	type custom struct{ error }
	b := &custom{errors.New("B")}
	base := v.All(
		v.Field("a", func(int) int { return 0 }, v.Rule[int](func(int) error { return a })),
		v.Field("b", func(int) int { return 0 }, v.Rule[int](func(int) error { return b })),
	)(0)
	third := errors.New("C")
	variants := []struct {
		name string
		wrap func(error) error
	}{
		{"none", func(err error) error { return err }},
		{"fmt", func(err error) error { return fmt.Errorf("outer: %w", err) }},
		{"join", func(err error) error { return errors.Join(err, third) }},
		{"fmt join", func(err error) error { return fmt.Errorf("outer: %w", errors.Join(err, third)) }},
	}
	for _, variant := range variants {
		t.Run(variant.name, func(t *testing.T) {
			err := variant.wrap(base)
			var found *custom
			if !errors.Is(err, a) || !errors.As(err, &found) || found != b {
				t.Fatalf("wrapper lost identities: %v", err)
			}
			if multi, ok := base.(interface{ Unwrap() []error }); ok {
				for i, child := range multi.Unwrap() {
					if child == nil {
						t.Fatalf("nil child %d", i)
					}
				}
			}
		})
	}
}

func TestSharedErrorHasSeparateImmutableLocations(t *testing.T) {
	shared := errors.New("shared")
	rule := v.All(
		v.Field("left", func(int) int { return 0 }, v.Rule[int](func(int) error { return shared })),
		v.Field("right", func(int) int { return 0 }, v.Rule[int](func(int) error { return shared })),
	)
	first := rule(0)
	assertIssues(t, first, []v.Code{v.CodeExternal, v.CodeExternal}, []string{"$.left", "$.right"})
	second := rule(0)
	assertIssues(t, second, []v.Code{v.CodeExternal, v.CodeExternal}, []string{"$.left", "$.right"})
	if got := v.Format(first); got != "$.left: external; $.right: external" {
		t.Fatalf("saved result = %q", got)
	}
	if !errors.Is(first, shared) || !errors.Is(second, shared) {
		t.Fatal("shared identity lost")
	}
}

func TestFormatDoesNotCallExternalErrorOrShowCause(t *testing.T) {
	secret := "very-secret-value"
	rule := v.All(
		v.Field("password", func(int) int { return 0 }, v.Rule[int](func(int) error { return &opaqueError{secret} })),
		v.Rule[int](func(int) error { return v.NewViolation(v.CodeNotEmpty, errors.New(secret)) }),
	)
	if got := v.Format(rule(0)); got != "$.password: external; $: not_empty" || strings.Contains(got, secret) {
		t.Fatalf("unsafe format = %q", got)
	}
}

func TestBoundsErrorRetainsUint64Precision(t *testing.T) {
	const limit uint64 = 9007199254740992
	err := v.Max(limit)(limit + 1)
	var bounds *v.BoundsError[uint64]
	if !errors.As(err, &bounds) {
		t.Fatalf("missing BoundsError[uint64]: %T", err)
	}
	upper, inclusive, present := bounds.Upper()
	if upper != limit || !inclusive || !present {
		t.Fatalf("upper=%d inclusive=%t present=%t", upper, inclusive, present)
	}
}
