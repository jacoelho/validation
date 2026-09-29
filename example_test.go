package validation_test

import (
	"fmt"

	v "github.com/jacoelho/validation"
)

func Example_nested() {
	type Address struct{ City string }
	type User struct {
		Name      string
		Age       *int
		Addresses []Address
	}
	addressRule := v.All(v.NotEmpty[string]().Field("city", func(a Address) string { return a.City }))
	userRule := v.All(
		v.All(v.NotEmpty[string](), v.RuneMinLength[string](2)).Field("name", func(u User) string { return u.Name }),
		v.OptionalPtr(v.Min(0)).Field("age", func(u User) *int { return u.Age }),
		v.Each[[]Address](addressRule).Field("addresses", func(u User) []Address { return u.Addresses }),
	)
	fmt.Println(v.Format(userRule.Validate(User{Addresses: []Address{{}}})))
	// Output: $.name: must not be empty; $.name: length must be at least 2 runes (got 0); $.addresses[0].city: must not be empty
}

func ExampleParameterized() {
	failure := v.Min(2).Field("bar", func(n int) int { return n })(1)
	for issue := range v.WalkIssues(failure) {
		message := v.DefaultMessage(issue)
		if detail, ok := issue.Err.(v.Parameterized); ok && issue.Code == v.CodeMin {
			message = fmt.Sprintf("deve ser pelo menos %v", detail.Parameters()["minimum"])
		}
		fmt.Printf("%s: %s\n", v.FormatPath(issue.Path), message)
	}
	// Output: $.bar: deve ser pelo menos 2
}
