package validation_test

import (
	"fmt"
	"math"
	"reflect"
	"testing"
	"time"

	v "github.com/jacoelho/validation/v2"
)

func acceptanceIssues(err error) []struct {
	path string
	code v.Code
} {
	issues := v.Issues(err)
	got := make([]struct {
		path string
		code v.Code
	}, len(issues))
	for i, issue := range issues {
		got[i].path = v.FormatPath(issue.Path)
		got[i].code = issue.Code
	}
	return got
}

func requireAcceptanceIssues(t *testing.T, err error, want ...struct {
	path string
	code v.Code
}) {
	t.Helper()
	got := acceptanceIssues(err)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("issues = %#v, want %#v", got, want)
	}
}

func acceptanceIssueAt(path string, code v.Code) struct {
	path string
	code v.Code
} {
	return struct {
		path string
		code v.Code
	}{path: path, code: code}
}

func TestATCORE002AllFailuresAtOneField(t *testing.T) {
	type input struct{ Name string }
	getterCalls, notEmptyCalls, runeMinCalls := 0, 0, 0
	notEmpty := v.NotEmpty[string]()
	runeMin := v.RuneMinLength[string](2)
	rule := v.Field("name", func(value input) string {
		getterCalls++
		return value.Name
	},
		func(value string) error {
			notEmptyCalls++
			return notEmpty(value)
		},
		func(value string) error {
			runeMinCalls++
			return runeMin(value)
		},
	)

	requireAcceptanceIssues(t, rule(input{}),
		acceptanceIssueAt("$.name", v.CodeNotEmpty),
		acceptanceIssueAt("$.name", v.CodeRuneMinLength),
	)
	if getterCalls != 1 || notEmptyCalls != 1 || runeMinCalls != 1 {
		t.Fatalf("calls = getter:%d not_empty:%d rune_min:%d, want 1/1/1", getterCalls, notEmptyCalls, runeMinCalls)
	}
}

func TestATCORE003IndependentFieldsContinue(t *testing.T) {
	type input struct {
		Name string
		Age  int
		City string
	}
	getters := map[string]int{}
	rule := v.Struct(
		v.Field("name", func(value input) string {
			getters["name"]++
			return value.Name
		}, func(string) error { return v.NewViolation(v.CodeNotEmpty, nil) }),
		v.Field("age", func(value input) int {
			getters["age"]++
			return value.Age
		}, func(int) error { return v.NewViolation(v.CodeMin, nil) }),
		v.Field("city", func(value input) string {
			getters["city"]++
			return value.City
		}, func(string) error { return v.NewViolation(v.CodeNotEmpty, nil) }),
	)

	requireAcceptanceIssues(t, rule(input{}),
		acceptanceIssueAt("$.name", v.CodeNotEmpty),
		acceptanceIssueAt("$.age", v.CodeMin),
		acceptanceIssueAt("$.city", v.CodeNotEmpty),
	)
	if !reflect.DeepEqual(getters, map[string]int{"name": 1, "age": 1, "city": 1}) {
		t.Fatalf("getters = %#v, want one call for every field", getters)
	}
}

func TestATCORE005EmptyCompositions(t *testing.T) {
	getterCalls := 0
	getter := func(value struct{ Name string }) string {
		getterCalls++
		return value.Name
	}
	prepared := struct{ Name string }{Name: "ready"}

	tests := []struct {
		name string
		run  func() error
		want int
	}{
		{name: "AT-CORE-005-row-1-All", run: func() error { return v.All[struct{ Name string }]()(prepared) }, want: 0},
		{name: "AT-CORE-005-row-2-Struct", run: func() error { return v.Struct[struct{ Name string }]()(prepared) }, want: 0},
		{name: "AT-CORE-005-row-3-Field", run: func() error { return v.Field("name", getter)(prepared) }, want: 1},
		{name: "AT-CORE-005-row-4-Project", run: func() error { return v.Project(getter)(prepared) }, want: 1},
		{name: "AT-CORE-005-row-5-Each", run: func() error {
			return v.Each[[]int]()([]int{1, 2, 3})
		}, want: 0},
		{name: "AT-CORE-005-row-6-MapEach", run: func() error {
			return v.MapEach[map[string]int](v.StringKeys[string]())(map[string]int{"a": 1, "b": 2, "c": 3})
		}, want: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			before := getterCalls
			err := test.run()
			if err != nil {
				t.Fatalf("empty composition returned %v", err)
			}
			if got := getterCalls - before; got != test.want {
				t.Fatalf("getter calls = %d, want %d", got, test.want)
			}
		})
	}
}

func TestATCORE006ParentConditionsGuardFields(t *testing.T) {
	type input struct {
		Enabled bool
		Left    string
		Right   string
		Outside string
	}

	for _, test := range []struct {
		name       string
		row        int
		when       bool
		useWhen    bool
		childCalls int
		wantIssues int
	}{
		{name: "When true", row: 1, when: true, useWhen: true, childCalls: 2, wantIssues: 3},
		{name: "When false", row: 2, when: false, useWhen: true, childCalls: 0, wantIssues: 1},
		{name: "Unless true", row: 3, when: true, useWhen: false, childCalls: 0, wantIssues: 1},
		{name: "Unless false", row: 4, when: false, useWhen: false, childCalls: 2, wantIssues: 3},
	} {
		t.Run(fmt.Sprintf("AT-CORE-006-row-%d-%s", test.row, test.name), func(t *testing.T) {
			predicateCalls, childCalls := 0, 0
			fieldRule := func(name string) v.Rule[input] {
				return v.Field(name, func(value input) string { return value.Left }, func(string) error {
					childCalls++
					return v.NewViolation(v.CodeNotEmpty, nil)
				})
			}
			conditional := []v.Rule[input]{fieldRule("left"), fieldRule("right")}
			predicate := func(value input) bool {
				predicateCalls++
				return value.Enabled
			}
			var guarded v.Rule[input]
			if test.useWhen {
				guarded = v.When(predicate, conditional...)
			} else {
				guarded = v.Unless(predicate, conditional...)
			}
			rule := v.Struct(
				guarded,
				v.Field("outside", func(value input) string { return value.Outside }, func(string) error {
					return v.NewViolation(v.CodeNotEmpty, nil)
				}),
			)

			if got := len(v.Issues(rule(input{Enabled: test.when}))); got != test.wantIssues {
				t.Fatalf("issues = %d, want %d", got, test.wantIssues)
			}
			if predicateCalls != 1 {
				t.Fatalf("predicate calls = %d, want 1", predicateCalls)
			}
			if childCalls != test.childCalls {
				t.Fatalf("conditional child calls = %d, want %d", childCalls, test.childCalls)
			}
		})
	}
}

func TestATCORE009EmptyValueIsNotOptional(t *testing.T) {
	err := v.RuneMinLength[string](2)("")
	requireAcceptanceIssues(t, err, acceptanceIssueAt("$", v.CodeRuneMinLength))
}

func TestATCONSTRUCTION001MalformedConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		row         int
		constructor string
		construct   func()
	}{
		{name: "RuneMinLength(-1)", row: 6, constructor: "RuneMinLength", construct: func() { v.RuneMinLength[string](-1) }},
		{name: "Between(5, 1)", row: 8, constructor: "Between", construct: func() { v.Between(5, 1) }},
		{name: "Min(NaN)", row: 9, constructor: "Min", construct: func() { v.Min(math.NaN()) }},
		{name: "Between(0, NaN)", row: 10, constructor: "Between", construct: func() { v.Between(0.0, math.NaN()) }},
		{name: "MapEach with nil Less", row: 12, constructor: "MapEach", construct: func() {
			v.MapEach[map[string]int](v.KeyOrder[string]{Text: func(string) string { return "" }})
		}},
		{name: "MapEach with nil Text", row: 13, constructor: "MapEach", construct: func() {
			v.MapEach[map[string]int](v.KeyOrder[string]{Less: func(left, right string) bool { return left < right }})
		}},
		{name: "MinBy with a nil comparator", row: 15, constructor: "MinBy", construct: func() { v.MinBy(0, nil) }},
		{name: "Max(NaN)", row: 17, constructor: "Max", construct: func() { v.Max(math.NaN()) }},
		{name: "GreaterThan(NaN)", row: 18, constructor: "GreaterThan", construct: func() { v.GreaterThan(math.NaN()) }},
		{name: "LessThan(NaN)", row: 19, constructor: "LessThan", construct: func() { v.LessThan(math.NaN()) }},
		{name: "TimeBetween reversed", row: 20, constructor: "TimeBetween", construct: func() {
			v.TimeBetween(time.Unix(2, 0), time.Unix(1, 0))
		}},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("AT-CONSTRUCTION-001-row-%d-%s", test.row, test.name), func(t *testing.T) {
			defer func() {
				panicValue := recover()
				configuration, ok := panicValue.(*v.ConfigurationError)
				if !ok {
					t.Fatalf("panic = %v (%T), want ConfigurationError", panicValue, panicValue)
				}
				if configuration.Constructor() != test.constructor {
					t.Fatalf("constructor = %q, want %q", configuration.Constructor(), test.constructor)
				}
			}()
			test.construct()
			t.Fatal("constructor returned without panicking")
		})
	}
}

func TestATCONSTRUCTION003FreezesMembershipAndByteConfiguration(t *testing.T) {
	t.Run("AT-CONSTRUCTION-003-row-1-OneOf", func(t *testing.T) {
		allowed := []string{"a", "b"}
		rule := v.OneOf(allowed...)
		allowed[0], allowed[1] = "x", "y"
		if err := rule("a"); err != nil {
			t.Fatalf("original membership changed: %v", err)
		}
		if err := rule("x"); err == nil {
			t.Fatal("replacement membership unexpectedly accepted")
		}
	})

	t.Run("AT-CONSTRUCTION-003-row-2-BytesHasPrefix", func(t *testing.T) {
		prefix := []byte{'a'}
		rule := v.BytesHasPrefix(prefix)
		prefix[0] = 'z'
		if err := rule([]byte{'a', 'b'}); err != nil {
			t.Fatalf("original byte prefix changed: %v", err)
		}
		if err := rule([]byte{'z', 'b'}); err == nil {
			t.Fatal("replacement byte prefix unexpectedly accepted")
		}
	})
}

func TestATCONSTRUCTION004DefersValueCallbacks(t *testing.T) {
	type parent struct{ Value int }
	getterCalls, childCalls, predicateCalls, factoryCalls := 0, 0, 0, 0
	lessCalls, textCalls := 0, 0
	childInt := func(int) error {
		childCalls++
		return nil
	}
	childParent := func(parent) error {
		childCalls++
		return nil
	}
	childString := func(string) error {
		childCalls++
		return nil
	}
	getInt := func(value parent) int {
		getterCalls++
		return value.Value
	}
	getString := func(value parent) string {
		getterCalls++
		return ""
	}
	getPresence := func(value parent) (int, bool) {
		getterCalls++
		return value.Value, true
	}
	predicate := func(parent) bool {
		predicateCalls++
		return true
	}
	factory := func(int) error {
		factoryCalls++
		return v.NewViolation(v.CodeMin, nil)
	}
	order := v.KeyOrder[string]{
		Less: func(left, right string) bool {
			lessCalls++
			return left < right
		},
		Text: func(key string) string {
			textCalls++
			return key
		},
	}

	_ = v.All(v.Rule[parent](childParent))
	_ = v.Struct(v.Rule[parent](childParent))
	_ = v.When(predicate, v.Rule[parent](childParent))
	_ = v.Unless(predicate, v.Rule[parent](childParent))
	_ = v.Check(func(int) bool { predicateCalls++; return true }, factory)
	_ = v.Field("value", getInt, childInt)
	_ = v.Project(getInt, childInt)
	_ = v.OptionalPtr(childInt)
	_ = v.RequiredPtr(childInt)
	_ = v.OptionalValue(getPresence, childInt)
	_ = v.RequiredValue(getPresence, childInt)
	_ = v.Each[[]int](childInt)
	_ = v.AtIndex[[]int](0, childInt)
	_ = v.MapEach[map[string]int](order, func(v.Entry[string, int]) error {
		childCalls++
		return nil
	})
	_ = v.MapKeys[map[string]int](order, func(string) error {
		childCalls++
		return nil
	})
	_ = v.MapValues[map[string]int](order, childInt)
	_ = v.MapRequiredKey[map[string]int]("x", order, childInt)
	_ = v.MapOptionalKey[map[string]int]("x", order, childInt)
	_ = v.Field("text", getString, childString)

	compareCalls := 0
	_ = v.BetweenBy(0, 1, func(left, right int) int {
		compareCalls++
		switch {
		case left < right:
			return -1
		case left > right:
			return 1
		default:
			return 0
		}
	})

	if getterCalls != 0 || childCalls != 0 || predicateCalls != 0 || factoryCalls != 0 || lessCalls != 0 || textCalls != 0 {
		t.Fatalf("callbacks during construction = getter:%d child:%d predicate:%d factory:%d less:%d text:%d; want all zero", getterCalls, childCalls, predicateCalls, factoryCalls, lessCalls, textCalls)
	}
	if compareCalls != 1 {
		t.Fatalf("BetweenBy comparator calls = %d, want 1", compareCalls)
	}
}

func TestATCONSTRUCTION005NilFactoryCannotSucceed(t *testing.T) {
	predicateCalls, factoryCalls := 0, 0
	rule := v.Check(func(int) bool {
		predicateCalls++
		return false
	}, func(int) error {
		factoryCalls++
		return nil
	})

	defer func() {
		panicValue := recover()
		configuration, ok := panicValue.(*v.ConfigurationError)
		if !ok || configuration.Constructor() != "Check" {
			t.Fatalf("panic = %v (%T), want Check ConfigurationError", panicValue, panicValue)
		}
		if predicateCalls != 1 || factoryCalls != 1 {
			t.Fatalf("calls = predicate:%d factory:%d, want 1/1", predicateCalls, factoryCalls)
		}
	}()
	rule(0)
}

type acceptanceTypedNilError struct{}

func (*acceptanceTypedNilError) Error() string { return "typed nil" }

func TestATCONSTRUCTION006TypedNilIsNotReflectivelyRepaired(t *testing.T) {
	var typedNil *acceptanceTypedNilError
	var returned error = typedNil

	err := v.All(v.Rule[int](func(int) error { return returned }))(0)
	if err == nil {
		t.Fatal("typed-nil callback was converted into success")
	}
	issues := v.Issues(err)
	if len(issues) != 1 || issues[0].Code != v.CodeExternal || issues[0].Err != returned {
		t.Fatalf("issues = %#v, want one external issue retaining typed-nil identity", issues)
	}
}

func TestATPRESENCE001PresenceAndZeroAreIndependent(t *testing.T) {
	for _, test := range []struct {
		name      string
		row       int
		adapter   func(...v.Rule[int]) v.Rule[*int]
		input     *int
		wantCodes []v.Code
		wantCalls int
	}{
		{name: "OptionalPtr nil", row: 1, adapter: v.OptionalPtr[int], wantCodes: nil, wantCalls: 0},
		{name: "RequiredPtr nil", row: 2, adapter: v.RequiredPtr[int], wantCodes: []v.Code{v.CodeRequired}, wantCalls: 0},
		{name: "OptionalPtr zero", row: 3, adapter: v.OptionalPtr[int], input: intPointer(0), wantCodes: nil, wantCalls: 1},
		{name: "RequiredPtr zero", row: 4, adapter: v.RequiredPtr[int], input: intPointer(0), wantCodes: nil, wantCalls: 1},
		{name: "RequiredPtr negative", row: 5, adapter: v.RequiredPtr[int], input: intPointer(-1), wantCodes: []v.Code{v.CodeMin}, wantCalls: 1},
	} {
		t.Run(fmt.Sprintf("AT-PRESENCE-001-row-%d-%s", test.row, test.name), func(t *testing.T) {
			calls := 0
			child := func(value int) error {
				calls++
				return v.Min(0)(value)
			}
			err := test.adapter(child)(test.input)
			issues := v.Issues(err)
			var gotCodes []v.Code
			for _, issue := range issues {
				gotCodes = append(gotCodes, issue.Code)
			}
			if !reflect.DeepEqual(gotCodes, test.wantCodes) || calls != test.wantCalls {
				t.Fatalf("codes = %v, calls = %d; want codes %v, calls %d", gotCodes, calls, test.wantCodes, test.wantCalls)
			}
		})
	}
}

func intPointer(value int) *int { return &value }

func TestATPRESENCE002MissingPointerSkipsOnlyItsChildren(t *testing.T) {
	type input struct {
		Address *int
		Age     int
	}
	childCalled := false
	rule := v.Struct(
		v.Field("address", func(value input) *int { return value.Address }, v.RequiredPtr(v.Rule[int](func(int) error {
			childCalled = true
			panic("nil child was evaluated")
		}))),
		v.Field("age", func(value input) int { return value.Age }, v.Min(0)),
	)

	requireAcceptanceIssues(t, rule(input{Age: -1}),
		acceptanceIssueAt("$.address", v.CodeRequired),
		acceptanceIssueAt("$.age", v.CodeMin),
	)
	if childCalled {
		t.Fatal("missing pointer child was evaluated")
	}
}

func TestATPRESENCE003NestedPointerLayersRemainGuarded(t *testing.T) {
	minCalls := 0
	rule := v.RequiredPtr(v.RequiredPtr(v.Rule[int](func(value int) error {
		minCalls++
		return v.Min(0)(value)
	})))
	var inner *int
	outer := &inner

	requireAcceptanceIssues(t, rule(outer), acceptanceIssueAt("$", v.CodeRequired))
	if minCalls != 0 {
		t.Fatalf("Min child calls = %d, want 0", minCalls)
	}
}

func TestATPRESENCE004NullableProjectionSuppliesPresence(t *testing.T) {
	type input struct {
		Value   int
		Present bool
	}
	for _, test := range []struct {
		name      string
		row       int
		required  bool
		value     int
		present   bool
		wantCodes []v.Code
		wantCalls int
	}{
		{name: "OptionalValue absent", row: 1, value: -1, present: false, wantCodes: nil, wantCalls: 0},
		{name: "RequiredValue absent", row: 2, required: true, present: false, wantCodes: []v.Code{v.CodeRequired}, wantCalls: 0},
		{name: "RequiredValue zero", row: 3, required: true, value: 0, present: true, wantCodes: nil, wantCalls: 1},
		{name: "OptionalValue negative", row: 4, value: -1, present: true, wantCodes: []v.Code{v.CodeMin}, wantCalls: 1},
	} {
		t.Run(fmt.Sprintf("AT-PRESENCE-004-row-%d-%s", test.row, test.name), func(t *testing.T) {
			getterCalls, childCalls := 0, 0
			getter := func(value input) (int, bool) {
				getterCalls++
				return value.Value, value.Present
			}
			child := func(value int) error {
				childCalls++
				return v.Min(0)(value)
			}
			var rule v.Rule[input]
			if test.required {
				rule = v.RequiredValue(getter, child)
			} else {
				rule = v.OptionalValue(getter, child)
			}
			issues := v.Issues(rule(input{Value: test.value, Present: test.present}))
			var gotCodes []v.Code
			for _, issue := range issues {
				gotCodes = append(gotCodes, issue.Code)
			}
			if !reflect.DeepEqual(gotCodes, test.wantCodes) || getterCalls != 1 || childCalls != test.wantCalls {
				t.Fatalf("codes = %v, getter calls = %d, child calls = %d; want %v, 1, %d", gotCodes, getterCalls, childCalls, test.wantCodes, test.wantCalls)
			}
		})
	}
}

func TestATPRESENCE005InputOwnsAbsentNullAndValueStates(t *testing.T) {
	type state uint8
	const (
		omitted state = iota
		null
		value
	)
	type input struct {
		State state
		Value int
	}

	rule := v.When(func(item input) bool { return item.State != omitted },
		v.RequiredValue(func(item input) (int, bool) {
			return item.Value, item.State == value
		}, v.Min(0)),
	)
	for _, test := range []struct {
		name      string
		input     input
		wantCodes []v.Code
	}{
		{name: "omitted", input: input{State: omitted, Value: -1}, wantCodes: nil},
		{name: "null", input: input{State: null, Value: 0}, wantCodes: []v.Code{v.CodeRequired}},
		{name: "value", input: input{State: value, Value: 0}, wantCodes: nil},
		{name: "invalid value", input: input{State: value, Value: -1}, wantCodes: []v.Code{v.CodeMin}},
	} {
		t.Run("AT-PRESENCE-005-"+test.name, func(t *testing.T) {
			issues := v.Issues(rule(test.input))
			var gotCodes []v.Code
			for _, issue := range issues {
				gotCodes = append(gotCodes, issue.Code)
			}
			if !reflect.DeepEqual(gotCodes, test.wantCodes) {
				t.Fatalf("codes = %v, want %v", gotCodes, test.wantCodes)
			}
		})
	}
}

func TestATPRESENCE006EmptyChildGroupsKeepStructuralGuards(t *testing.T) {
	order := v.StringKeys[string]()
	tests := []struct {
		name      string
		row       int
		rule      func() error
		wantCodes []v.Code
	}{
		{name: "RequiredPtr nil", row: 1, rule: func() error { return v.RequiredPtr[int]()(nil) }, wantCodes: []v.Code{v.CodeRequired}},
		{name: "OptionalPtr nil", row: 2, rule: func() error { return v.OptionalPtr[int]()(nil) }, wantCodes: nil},
		{name: "RequiredValue absent", row: 3, rule: func() error {
			return v.RequiredValue(func(int) (string, bool) { return "", false })(0)
		}, wantCodes: []v.Code{v.CodeRequired}},
		{name: "MapRequiredKey empty", row: 4, rule: func() error {
			return v.MapRequiredKey[map[string]int]("x", order)(map[string]int{})
		}, wantCodes: []v.Code{v.CodeKeyRequired}},
		{name: "MapOptionalKey empty", row: 5, rule: func() error {
			return v.MapOptionalKey[map[string]int]("x", order)(map[string]int{})
		}, wantCodes: nil},
		{name: "AtIndex out of range", row: 6, rule: func() error {
			return v.AtIndex[[]int](2)([]int{1})
		}, wantCodes: []v.Code{v.CodeIndexOutOfRange}},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("AT-PRESENCE-006-row-%d-%s", test.row, test.name), func(t *testing.T) {
			issues := v.Issues(test.rule())
			var gotCodes []v.Code
			for _, issue := range issues {
				gotCodes = append(gotCodes, issue.Code)
			}
			if !reflect.DeepEqual(gotCodes, test.wantCodes) {
				t.Fatalf("codes = %v, want %v", gotCodes, test.wantCodes)
			}
		})
	}
}
