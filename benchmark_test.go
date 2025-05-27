package validation_test

import (
	"testing"

	"github.com/jacoelho/validation"
)

func BenchmarkValidation(b *testing.B) {
	type User struct {
		Name     string
		Age      int
		Email    string
		Password string
		Tags     []string
		Settings map[string]string
	}

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
		),
	)

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

	b.Run("ValidUser", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			_ = validator.Validate(validUser)
		}
	})

	b.Run("InvalidUser", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			_ = validator.Validate(invalidUser)
		}
	})
}
