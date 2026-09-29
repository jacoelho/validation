package validation_test

import (
	"errors"
	"testing"

	v "github.com/jacoelho/validation/v2"
)

func TestNestedStructAllErrors(t *testing.T) {
	type Address struct{ City string }
	type User struct {
		Name    string
		Address Address
		Age     *int
	}
	address := v.Struct(v.Field("city", func(a Address) string { return a.City }, v.NotEmpty[string](), v.RuneMinLength[string](2)))
	user := v.Struct(
		v.Field("name", func(u User) string { return u.Name }, v.NotEmpty[string](), v.RuneMinLength[string](2)),
		v.Field("address", func(u User) Address { return u.Address }, address),
		v.Field("age", func(u User) *int { return u.Age }, v.RequiredPtr(v.Min(18))),
	)
	assertIssues(t, user(User{}), []v.Code{v.CodeNotEmpty, v.CodeRuneMinLength, v.CodeNotEmpty, v.CodeRuneMinLength, v.CodeRequired}, []string{"$.name", "$.name", "$.address.city", "$.address.city", "$.age"})
	age := 20
	if err := user(User{Name: "Ann", Address: Address{City: "Paris"}, Age: &age}); err != nil {
		t.Fatalf("valid user = %v", err)
	}
}

func TestParentConditionGuardsField(t *testing.T) {
	type User struct {
		Premium bool
		Card    string
	}
	rule := v.When(func(u User) bool { return u.Premium }, v.Field("card", func(u User) string { return u.Card }, v.NotEmpty[string]()))
	if got := rule(User{}); got != nil {
		t.Fatalf("basic user = %v", got)
	}
	assertIssues(t, rule(User{Premium: true}), []v.Code{v.CodeNotEmpty}, []string{"$.card"})
}

func TestSavedErrorUnaffectedByLaterCalls(t *testing.T) {
	type Item struct{ Name string }
	rule := v.Field("item", func(i Item) string { return i.Name }, v.NotEmpty[string]())
	first := rule(Item{})
	if first == nil {
		t.Fatal("first failed call returned nil")
	}
	if got := rule(Item{Name: "ok"}); got != nil {
		t.Fatalf("later valid call = %v", got)
	}
	_ = rule(Item{})
	if got := v.Format(first); got != "$.item: not_empty" {
		t.Fatalf("saved error changed: %q", got)
	}
	var coded v.Coded
	if !errors.As(first, &coded) || coded.Code() != v.CodeNotEmpty {
		t.Fatal("coded error not discoverable")
	}
}
