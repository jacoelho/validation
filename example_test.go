package validation_test

import (
	"fmt"

	v "github.com/jacoelho/validation/v2"
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
	// Output: $.name: not_empty; $.name: rune_min_length; $.addresses[0].city: not_empty
}
