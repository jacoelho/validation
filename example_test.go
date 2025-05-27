package validation_test

import (
	"fmt"

	"github.com/jacoelho/validation"
)

func Example() {
	// Define a user struct with slices and maps
	type User struct {
		Name     string
		Age      int
		Email    string
		Password string
		Tags     []string
		Settings map[string]string
	}

	// Create a validator for the User struct
	validator := validation.Struct(
		validation.Field("Name", func(u User) string { return u.Name },
			validation.NotZero[string](),
			validation.StringsRuneMaxLength[string](50),
		),
		validation.Field("Age", func(u User) int { return u.Age },
			validation.NumbersMin(18),
			validation.NumbersMax(120),
		),
		validation.Field("Email", func(u User) string { return u.Email },
			validation.NotZero[string](),
			validation.StringsRuneMaxLength[string](100),
		),
		validation.Field("Password", func(u User) string { return u.Password },
			validation.NotZero[string](),
			validation.StringsRuneMinLength[string](8),
		),
		validation.SliceField("Tags", func(u User) []string { return u.Tags },
			validation.SlicesMaxLength[string](5),
			validation.SlicesForEach(
				validation.NotZero[string](),
				validation.StringsRuneMaxLength[string](20),
			),
		),
		validation.MapField("Settings", func(u User) map[string]string { return u.Settings },
			validation.MapsMaxKeys[string, string](5),
			validation.MapsForEach(
				func(k, v string) *validation.Error {
					if v == "" {
						return &validation.Error{
							Code:   "empty_value",
							Field:  k,
							Params: map[string]any{"key": k},
						}
					}
					return nil
				},
			),
			validation.MapsKey("lang", validation.OneOf("en", "fr")),
		),
	)

	// Validate a valid user
	validUser := User{
		Name:     "John Doe",
		Age:      30,
		Email:    "john@example.com",
		Password: "secure123",
		Tags:     []string{"user", "premium"},
		Settings: map[string]string{
			"theme": "dark",
			"lang":  "en",
		},
	}
	err := validator.Validate(validUser)
	fmt.Println("Valid user errors:", len(err))

	// Validate an invalid user
	invalidUser := User{
		Name:     "",
		Age:      15,
		Email:    "invalid-email",
		Password: "short",
		Tags:     []string{"", "very_long_tag_that_exceeds_maximum_length"},
		Settings: map[string]string{
			"theme": "",
			"lang":  "en",
		},
	}
	err = validator.Validate(invalidUser)
	fmt.Println("Invalid user errors:", err)

	// Output:
	// Valid user errors: 0
	// Invalid user errors: zero (field: Name); min (field: Age) {actual: 15, min: 18}; min (field: Password) {actual: 5, min: 8}; zero (field: Tags.0); max (field: Tags.1) {actual: 41, max: 20}; empty_value (field: Settings.theme) {key: theme}
}
