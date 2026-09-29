package validation_test

import (
	"testing"

	validation "github.com/jacoelho/validation"
)

func TestGenericMethodFieldAndProject(t *testing.T) {
	type person struct{ Name string }
	name := validation.All(validation.NotEmpty[string](), validation.RuneMinLength[string](2))
	get := func(p person) string { return p.Name }

	field := name.Field("name", get)
	var typed validation.Rule[person] = field
	got := validation.Issues(typed(person{}))
	if len(got) != 2 || got[0].Code != validation.CodeNotEmpty || got[1].Code != validation.CodeRuneMinLength ||
		validation.FormatPath(got[0].Path) != "$.name" || validation.FormatPath(got[1].Path) != "$.name" {
		t.Fatalf("field issues = %+v", got)
	}

	projected := name.Project(get)
	got = validation.Issues(projected(person{}))
	if len(got) != 2 || got[0].Code != validation.CodeNotEmpty || got[1].Code != validation.CodeRuneMinLength ||
		validation.FormatPath(got[0].Path) != "$" || validation.FormatPath(got[1].Path) != "$" {
		t.Fatalf("projected issues = %+v", got)
	}
	if err := field(person{Name: "Ada"}); err != nil {
		t.Fatalf("valid field: %v", err)
	}
}

func TestGenericMethodExplicitPresence(t *testing.T) {
	type input struct {
		Value   int
		Present bool
	}
	getterCalls := 0
	getter := func(in input) (int, bool) {
		getterCalls++
		return in.Value, in.Present
	}
	child := validation.All(validation.Min(1), validation.Max(-1))
	optional := child.OptionalValue(getter)
	required := child.RequiredValue(getter)

	if err := optional(input{}); err != nil {
		t.Fatalf("absent optional value: %v", err)
	}
	if getterCalls != 1 {
		t.Fatalf("optional getter calls = %d", getterCalls)
	}
	if got := validation.Issues(required(input{})); len(got) != 1 || got[0].Code != validation.CodeRequired || validation.FormatPath(got[0].Path) != "$" {
		t.Fatalf("absent required issues = %+v", got)
	}
	if getterCalls != 2 {
		t.Fatalf("required getter calls = %d", getterCalls)
	}
	for name, rule := range map[string]validation.Rule[input]{"optional": optional, "required": required} {
		got := validation.Issues(rule(input{Present: true}))
		if len(got) != 2 || got[0].Code != validation.CodeMin || got[1].Code != validation.CodeMax ||
			validation.FormatPath(got[0].Path) != "$" || validation.FormatPath(got[1].Path) != "$" {
			t.Fatalf("%s present zero issues = %+v", name, got)
		}
	}
	if getterCalls != 4 {
		t.Fatalf("total getter calls = %d", getterCalls)
	}
	for name, rule := range map[string]validation.Rule[input]{
		"optional function": validation.OptionalValue(getter, validation.Min(1), validation.Max(-1)),
		"required function": validation.RequiredValue(getter, validation.Min(1), validation.Max(-1)),
	} {
		got := validation.Issues(rule(input{Present: true}))
		if len(got) != 2 || got[0].Code != validation.CodeMin || got[1].Code != validation.CodeMax ||
			validation.FormatPath(got[0].Path) != "$" || validation.FormatPath(got[1].Path) != "$" {
			t.Fatalf("%s present zero issues = %+v", name, got)
		}
	}
	if getterCalls != 6 {
		t.Fatalf("total getter calls after function forms = %d", getterCalls)
	}
}

func TestGenericMethodConfiguration(t *testing.T) {
	var nilRule validation.Rule[int]
	get := func(in struct{ N int }) int { return in.N }
	getPresence := func(in struct{ N int }) (int, bool) { return in.N, true }
	cases := []struct {
		name string
		make func()
	}{
		{"Field", func() { nilRule.Field("n", get) }},
		{"Project", func() { nilRule.Project(get) }},
		{"OptionalValue", func() { nilRule.OptionalValue(getPresence) }},
		{"RequiredValue", func() { nilRule.RequiredValue(getPresence) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				got, ok := recover().(*validation.ConfigurationError)
				if !ok || got.Constructor() != tc.name {
					t.Fatalf("panic = %#v, want %s configuration error", got, tc.name)
				}
			}()
			tc.make()
		})
	}
}
