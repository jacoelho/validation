package validation_test

import (
	"errors"
	"reflect"
	"testing"

	v "github.com/jacoelho/validation"
)

func assertIssues(t *testing.T, err error, wantCodes []v.Code, wantPaths []string) {
	t.Helper()
	issues := v.Issues(err)
	if len(issues) != len(wantCodes) {
		t.Fatalf("issues = %v; want %d failures", v.Format(err), len(wantCodes))
	}
	for i, issue := range issues {
		if issue.Code != wantCodes[i] || v.FormatPath(issue.Path) != wantPaths[i] {
			t.Errorf("issue[%d] = %s: %s; want %s: %s", i, v.FormatPath(issue.Path), issue.Code, wantPaths[i], wantCodes[i])
		}
	}
}

func TestAllExhaustiveAndOccurrenceOrder(t *testing.T) {
	first, second := errors.New("first"), errors.New("second")
	calls := 0
	shared := v.Rule[int](func(int) error { calls++; return first })
	rule := v.All(shared, func(int) error { calls++; return second }, shared)
	err := rule.Validate(7)
	if calls != 3 {
		t.Fatalf("calls = %d, want 3", calls)
	}
	assertIssues(t, err, []v.Code{v.CodeExternal, v.CodeExternal, v.CodeExternal}, []string{"$", "$", "$"})
	issues := v.Issues(err)
	if issues[0].Err != first || issues[1].Err != second || issues[2].Err != first {
		t.Fatalf("identity/order = %#v", issues)
	}
	if !errors.Is(err, first) || !errors.Is(err, second) {
		t.Fatal("causes lost")
	}
	if got := v.All[int]()(0); got != nil {
		t.Fatalf("empty All = %v", got)
	}
	if got := v.Struct[int]()(0); got != nil {
		t.Fatalf("empty Struct = %v", got)
	}
}

func TestAllSnapshotsRulesAndNormalizes(t *testing.T) {
	original := errors.New("original")
	rules := []v.Rule[int]{func(int) error { return original }}
	rule := v.All(rules...)
	rules[0] = func(int) error { return nil }
	if got := rule(0); got != original {
		t.Fatalf("single result = %v, want original identity", got)
	}
	if got := v.All(v.Equal(3))(3); got != nil {
		t.Fatalf("success = %v", got)
	}
}

func TestConditionProjectionAndCheck(t *testing.T) {
	type Person struct {
		Active bool
		Name   string
	}
	predicateCalls, getterCalls, failureCalls := 0, 0, 0
	rule := v.When(func(p Person) bool { predicateCalls++; return p.Active },
		v.Field("name", func(p Person) string { getterCalls++; return p.Name },
			v.Check(func(s string) bool { return len(s) >= 2 }, func(string) error { failureCalls++; return v.NewViolation("short", nil) }),
		),
	)
	if got := rule(Person{Name: "x"}); got != nil {
		t.Fatalf("disabled = %v", got)
	}
	if predicateCalls != 1 || getterCalls != 0 || failureCalls != 0 {
		t.Fatalf("disabled calls = %d/%d/%d", predicateCalls, getterCalls, failureCalls)
	}
	err := rule(Person{Active: true, Name: "x"})
	assertIssues(t, err, []v.Code{"short"}, []string{"$.name"})
	if predicateCalls != 2 || getterCalls != 1 || failureCalls != 1 {
		t.Fatalf("enabled calls = %d/%d/%d", predicateCalls, getterCalls, failureCalls)
	}
	project := v.Project(func(p Person) string { getterCalls++; return p.Name }, v.NotEmpty[string]())
	if got := project(Person{Name: "ok"}); got != nil || getterCalls != 2 {
		t.Fatalf("project = %v, getter calls %d", got, getterCalls)
	}
	emptyProject := v.Project(func(p Person) string { getterCalls++; return p.Name })
	if got := emptyProject(Person{}); got != nil || getterCalls != 3 {
		t.Fatalf("empty project = %v, getter calls %d", got, getterCalls)
	}
}

func TestPresenceGuardsAndIndependentFields(t *testing.T) {
	type Person struct {
		Age  *int
		Name string
	}
	zero := 0
	rule := v.All(
		v.Field("age", func(p Person) *int { return p.Age }, v.RequiredPtr(v.Min(1))),
		v.Field("name", func(p Person) string { return p.Name }, v.NotEmpty[string]()),
	)
	assertIssues(t, rule(Person{}), []v.Code{v.CodeRequired, v.CodeNotEmpty}, []string{"$.age", "$.name"})
	assertIssues(t, rule(Person{Age: &zero}), []v.Code{v.CodeMin, v.CodeNotEmpty}, []string{"$.age", "$.name"})
	if got := v.OptionalPtr(v.Min(1))((*int)(nil)); got != nil {
		t.Fatalf("optional nil = %v", got)
	}
	if got := v.RequiredPtr[int]()((*int)(nil)); got == nil {
		t.Fatal("empty required guard skipped")
	}
	optional := v.OptionalValue(func(p Person) (int, bool) {
		if p.Age == nil {
			return 0, false
		}
		return *p.Age, true
	}, v.Zero[int]())
	if got := optional(Person{Age: &zero}); got != nil {
		t.Fatalf("present zero = %v", got)
	}
	if got := optional(Person{}); got != nil {
		t.Fatalf("absent optional = %v", got)
	}
	required := v.RequiredValue(func(p Person) (int, bool) {
		if p.Age == nil {
			return 0, false
		}
		return *p.Age, true
	})
	assertIssues(t, required(Person{}), []v.Code{v.CodeRequired}, []string{"$"})
}

func TestComparableRules(t *testing.T) {
	tests := []struct {
		name  string
		rule  v.Rule[int]
		input int
		code  v.Code
	}{
		{"equal", v.Equal(3), 4, v.CodeEqual},
		{"not equal", v.NotEqual(3), 3, v.CodeNotEqual},
		{"zero", v.Zero[int](), 1, v.CodeZero},
		{"not zero", v.NotZero[int](), 0, v.CodeNotZero},
		{"one of", v.OneOf(1, 2), 3, v.CodeOneOf},
		{"not one of", v.NotOneOf(1, 2), 2, v.CodeNotOneOf},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { assertIssues(t, tt.rule(tt.input), []v.Code{tt.code}, []string{"$"}) })
	}
	if got := v.NotOneOf[int]()(3); got != nil {
		t.Fatalf("empty NotOneOf = %v", got)
	}
	if got := v.OneOf[int]()(3); got == nil {
		t.Fatal("empty OneOf passed")
	}
	allowed := []int{3}
	rule := v.OneOf(allowed...)
	allowed[0] = 4
	if got := rule(3); got != nil {
		t.Fatalf("membership config changed = %v", got)
	}
}

func TestInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name, constructor string
		construct         func()
	}{
		{"nil child", "All", func() { v.All(v.Rule[int](nil)) }},
		{"nil predicate", "When", func() { v.When[int](nil) }},
		{"nil factory", "Check", func() { v.Check(func(int) bool { return false }, nil) }},
		{"nil getter", "Project", func() { v.Project[int, int](nil) }},
		{"empty field", "Field", func() { v.Field("", func(int) int { return 0 }) }},
		{"nil field getter", "Field", func() { v.Field[int, int]("age", nil) }},
		{"empty code", "NewViolation", func() { v.NewViolation("", nil) }},
		{"nil factory result", "Check", func() { v.Check(func(int) bool { return false }, func(int) error { return nil })(0) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				p := recover()
				ce, ok := p.(*v.ConfigurationError)
				if !ok || ce.Constructor() != tt.constructor {
					t.Errorf("panic = %v, want ConfigurationError(%s)", p, tt.constructor)
				}
			}()
			tt.construct()
		})
	}
}

func TestIssuePathSnapshots(t *testing.T) {
	type Item struct{ Name string }
	rule := v.Field("items", func(items []Item) []Item { return items }, v.Each[[]Item](v.Field("name", func(i Item) string { return i.Name }, v.NotEmpty[string](), v.RuneMinLength[string](2))))
	original := []Item{{Name: ""}, {Name: ""}}
	issues := v.Issues(rule(original))
	want := []string{"$.items[0].name", "$.items[0].name", "$.items[1].name", "$.items[1].name"}
	if len(issues) != len(want) {
		t.Fatalf("issues = %v", issues)
	}
	for i, issue := range issues {
		if got := v.FormatPath(issue.Path); got != want[i] {
			t.Errorf("path[%d] = %s, want %s", i, got, want[i])
		}
	}
	issues[0].Path[0].Name = "changed"
	if got := v.FormatPath(issues[1].Path); got != want[1] {
		t.Errorf("aliased path = %s", got)
	}
	if !reflect.DeepEqual(original, []Item{{Name: ""}, {Name: ""}}) {
		t.Fatal("input changed")
	}
}

func TestUnlessAndPanicPropagation(t *testing.T) {
	calls := 0
	rule := v.Unless(func(n int) bool { calls++; return n == 0 }, v.Positive[int](), v.LessThan(10))
	if err := rule(0); err != nil || calls != 1 {
		t.Fatalf("disabled result=%v calls=%d", err, calls)
	}
	assertIssues(t, rule(-1), []v.Code{v.CodePositive}, []string{"$"})
	if calls != 2 {
		t.Fatalf("predicate calls = %d", calls)
	}
	panicValue := errors.New("callback panic")
	panics := v.All(v.Rule[int](func(int) error { panic(panicValue) }), v.Rule[int](func(int) error { t.Fatal("later rule ran after panic"); return nil }))
	defer func() {
		if got := recover(); got != panicValue {
			t.Errorf("panic = %v, want original", got)
		}
	}()
	panics(0)
}

func TestNestedPointerAndEmptyGroups(t *testing.T) {
	var absent *int
	presentLayer := &absent
	rule := v.RequiredPtr(v.RequiredPtr(v.Zero[int]()))
	assertIssues(t, rule(presentLayer), []v.Code{v.CodeRequired}, []string{"$"})
	if got := v.OptionalPtr(v.RequiredPtr(v.Zero[int]()))((**int)(nil)); got != nil {
		t.Fatalf("outer optional = %v", got)
	}
	if got := v.When(func(int) bool { return true })(0); got != nil {
		t.Fatalf("empty enabled group = %v", got)
	}
	called := 0
	field := v.Field("x", func(n int) int { called++; return n })
	if got := field(1); got != nil || called != 1 {
		t.Fatalf("empty field = %v, getter calls %d", got, called)
	}
}

type testPointerError struct{}

func (*testPointerError) Error() string { return "test pointer error" }

func TestCustomTypedNilIsNotSuccess(t *testing.T) {
	// A typed nil converted to error is still a non-nil interface. Reporting it
	// is outside the extension contract; evaluation must not silently erase it.
	var pointer *testPointerError
	var err error = pointer
	rule := v.All(v.Rule[int](func(int) error { return err }))
	if got := rule(0); got == nil {
		t.Fatal("typed-nil callback was normalized to success")
	}
}

func TestNoFailureCapAcrossLargeSlice(t *testing.T) {
	firstCalls, secondCalls := 0, 0
	first := v.Rule[int](func(int) error { firstCalls++; return v.NewViolation("first", nil) })
	second := v.Rule[int](func(int) error { secondCalls++; return v.NewViolation("second", nil) })
	values := make([]int, 257)
	issues := v.Issues(v.Each[[]int](first, second)(values))
	if len(issues) != 514 || firstCalls != 257 || secondCalls != 257 {
		t.Fatalf("issues=%d calls=%d/%d, want 514 and 257/257", len(issues), firstCalls, secondCalls)
	}
	for i := 0; i < 257; i++ {
		for child, code := range []v.Code{"first", "second"} {
			issue := issues[2*i+child]
			if issue.Code != code || len(issue.Path) != 1 || issue.Path[0].Kind != v.IndexSegment || issue.Path[0].Index != i {
				t.Fatalf("issue[%d]=%+v, want %s at %d", 2*i+child, issue, code, i)
			}
		}
	}
}

func TestValidErrorBoundaryStaysLiteralNil(t *testing.T) {
	rule := v.All(v.Equal(3), v.NotZero[int](), v.Min(2))
	returnError := func() error { return rule.Validate(3) }
	if err := returnError(); err != nil || v.Issues(err) != nil || v.Format(err) != "" {
		t.Fatalf("valid boundary returned %v", err)
	}
}

func TestCheckUsesOneCompoundPredicate(t *testing.T) {
	type Contact struct{ Email, Phone string }
	a, b, c, factory := 0, 0, 0, 0
	check := v.Check(func(contact Contact) bool {
		a++
		if contact.Email != "" {
			return true
		}
		b++
		if contact.Phone == "special" {
			return true
		}
		c++
		return contact.Email == "" && contact.Phone == "ok"
	}, func(Contact) error { factory++; return v.NewViolation("contact_required", nil) })
	if err := check(Contact{Phone: "ok"}); err != nil {
		t.Fatalf("last alternative success=%v", err)
	}
	if a != 1 || b != 1 || c != 1 || factory != 0 {
		t.Fatalf("success calls=%d/%d/%d factory=%d", a, b, c, factory)
	}
	assertIssues(t, check(Contact{}), []v.Code{"contact_required"}, []string{"$"})
	if a != 2 || b != 2 || c != 2 || factory != 1 {
		t.Fatalf("failure calls=%d/%d/%d factory=%d", a, b, c, factory)
	}
}
