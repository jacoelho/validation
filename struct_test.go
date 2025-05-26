package validation_test

import (
	"testing"

	"github.com/jacoelho/validation"
)

type User struct {
	Name     string
	Age      int
	Email    string
	Address  Address
	Tags     []string
	Settings map[string]string
}

type Address struct {
	Street  string
	City    string
	Country string
}

func TestStructValidation(t *testing.T) {
	validator := validation.NewStruct(
		validation.Field("Name", func(u User) string { return u.Name },
			validation.StringsNotEmpty[string](),
			validation.StringsRuneMaxLength[string](50),
		),
		validation.Field("Age", func(u User) int { return u.Age },
			validation.NumbersMin(18),
			validation.NumbersMax(120),
		),
		validation.Field("Email", func(u User) string { return u.Email },
			validation.StringsNotEmpty[string](),
			validation.StringsRuneMaxLength[string](100),
		),
		validation.StructField("Address", func(u User) Address { return u.Address },
			validation.NewStruct(
				validation.Field("Street", func(a Address) string { return a.Street },
					validation.StringsNotEmpty[string](),
				),
				validation.Field("City", func(a Address) string { return a.City },
					validation.StringsNotEmpty[string](),
				),
				validation.Field("Country", func(a Address) string { return a.Country },
					validation.StringsNotEmpty[string](),
				),
			),
		),
		validation.SliceField("Tags", func(u User) []string { return u.Tags },
			validation.SlicesMaxLength[string](5),
			validation.SlicesForEach(
				validation.StringsNotEmpty[string](),
				validation.StringsRuneMaxLength[string](20),
			),
		),
		validation.MapField("Settings", func(u User) map[string]string { return u.Settings },
			validation.MapsMaxKeys[string, string](10),
			validation.MapsForEach(
				func(k, v string) *validation.Error {
					if v == "" {
						return &validation.Error{
							Code:   "empty_value",
							Params: map[string]any{"key": k},
							Field:  k,
						}
					}
					return nil
				},
			),
		),
	)

	tests := []struct {
		name     string
		user     User
		wantErr  bool
		errCode  string
		errField string
	}{
		{
			name: "valid user",
			user: User{
				Name:  "John Doe",
				Age:   30,
				Email: "john@example.com",
				Address: Address{
					Street:  "123 Main St",
					City:    "New York",
					Country: "USA",
				},
				Tags: []string{"user", "premium"},
				Settings: map[string]string{
					"theme": "dark",
					"lang":  "en",
				},
			},
			wantErr: false,
		},
		{
			name: "empty name",
			user: User{
				Name:  "",
				Age:   30,
				Email: "john@example.com",
				Address: Address{
					Street:  "123 Main St",
					City:    "New York",
					Country: "USA",
				},
			},
			wantErr:  true,
			errCode:  "not_empty",
			errField: "Name",
		},
		{
			name: "invalid age",
			user: User{
				Name:  "John Doe",
				Age:   15,
				Email: "john@example.com",
				Address: Address{
					Street:  "123 Main St",
					City:    "New York",
					Country: "USA",
				},
			},
			wantErr:  true,
			errCode:  "min",
			errField: "Age",
		},
		{
			name: "empty address fields",
			user: User{
				Name:  "John Doe",
				Age:   30,
				Email: "john@example.com",
				Address: Address{
					Street:  "",
					City:    "",
					Country: "",
				},
			},
			wantErr:  true,
			errCode:  "not_empty",
			errField: "Address.Street",
		},
		{
			name: "too many tags",
			user: User{
				Name:  "John Doe",
				Age:   30,
				Email: "john@example.com",
				Address: Address{
					Street:  "123 Main St",
					City:    "New York",
					Country: "USA",
				},
				Tags: []string{"1", "2", "3", "4", "5", "6"},
			},
			wantErr:  true,
			errCode:  "max",
			errField: "Tags",
		},
		{
			name: "empty setting value",
			user: User{
				Name:  "John Doe",
				Age:   30,
				Email: "john@example.com",
				Address: Address{
					Street:  "123 Main St",
					City:    "New York",
					Country: "USA",
				},
				Settings: map[string]string{
					"theme": "",
				},
			},
			wantErr:  true,
			errCode:  "empty_value",
			errField: "Settings.theme",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.user)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
					return
				}
				if len(err) == 0 {
					t.Error("expected error slice to be non-empty")
					return
				}
				if err[0].Code != tt.errCode {
					t.Errorf("expected error code %q, got %q", tt.errCode, err[0].Code)
				}
				if err[0].Field != tt.errField {
					t.Errorf("expected error field %q, got %q", tt.errField, err[0].Field)
				}
			} else {
				if len(err) > 0 {
					t.Errorf("expected no error but got %v", err)
				}
			}
		})
	}
}

func TestStructValidationWithPrefix(t *testing.T) {
	validator := validation.NewStruct(
		validation.Field("Name", func(u User) string { return u.Name },
			validation.StringsNotEmpty[string](),
		),
	)

	tests := []struct {
		name     string
		user     User
		prefix   string
		wantErr  bool
		errField string
	}{
		{
			name: "with prefix",
			user: User{
				Name: "",
			},
			prefix:   "user",
			wantErr:  true,
			errField: "user.Name",
		},
		{
			name: "without prefix",
			user: User{
				Name: "",
			},
			prefix:   "",
			wantErr:  true,
			errField: "Name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateWithPrefix(tt.user, tt.prefix)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
					return
				}
				if len(err) == 0 {
					t.Error("expected error slice to be non-empty")
					return
				}
				if err[0].Field != tt.errField {
					t.Errorf("expected error field %q, got %q", tt.errField, err[0].Field)
				}
			} else {
				if len(err) > 0 {
					t.Errorf("expected no error but got %v", err)
				}
			}
		})
	}
}

func TestNestedStructValidation(t *testing.T) {
	type Company struct {
		Name    string
		Address Address
	}

	type Employee struct {
		Name    string
		Company Company
	}

	validator := validation.NewStruct(
		validation.Field("Name", func(e Employee) string { return e.Name },
			validation.StringsNotEmpty[string](),
		),
		validation.StructField("Company", func(e Employee) Company { return e.Company },
			validation.NewStruct(
				validation.Field("Name", func(c Company) string { return c.Name },
					validation.StringsNotEmpty[string](),
				),
				validation.StructField("Address", func(c Company) Address { return c.Address },
					validation.NewStruct(
						validation.Field("Street", func(a Address) string { return a.Street },
							validation.StringsNotEmpty[string](),
						),
					),
				),
			),
		),
	)

	tests := []struct {
		name     string
		employee Employee
		wantErr  bool
		errField string
	}{
		{
			name: "valid employee",
			employee: Employee{
				Name: "John Doe",
				Company: Company{
					Name: "Acme Inc",
					Address: Address{
						Street: "123 Main St",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "empty employee name",
			employee: Employee{
				Name: "",
				Company: Company{
					Name: "Acme Inc",
					Address: Address{
						Street: "123 Main St",
					},
				},
			},
			wantErr:  true,
			errField: "Name",
		},
		{
			name: "empty company name",
			employee: Employee{
				Name: "John Doe",
				Company: Company{
					Name: "",
					Address: Address{
						Street: "123 Main St",
					},
				},
			},
			wantErr:  true,
			errField: "Company.Name",
		},
		{
			name: "empty address street",
			employee: Employee{
				Name: "John Doe",
				Company: Company{
					Name: "Acme Inc",
					Address: Address{
						Street: "",
					},
				},
			},
			wantErr:  true,
			errField: "Company.Address.Street",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.employee)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
					return
				}
				if len(err) == 0 {
					t.Error("expected error slice to be non-empty")
					return
				}
				if err[0].Field != tt.errField {
					t.Errorf("expected error field %q, got %q", tt.errField, err[0].Field)
				}
			} else {
				if len(err) > 0 {
					t.Errorf("expected no error but got %v", err)
				}
			}
		})
	}
}

func TestStructValidationWithFatalErrors(t *testing.T) {
	validator := validation.NewStruct(
		validation.Field("Name", func(u User) string { return u.Name },
			validation.StringsNotEmpty[string](),
		),
		validation.Field("Age", func(u User) int { return u.Age },
			validation.NumbersMin[int](18),
		),
	)

	tests := []struct {
		name    string
		user    User
		wantErr bool
		errLen  int
	}{
		{
			name: "multiple errors",
			user: User{
				Name: "",
				Age:  15,
			},
			wantErr: true,
			errLen:  2,
		},
		{
			name: "valid user",
			user: User{
				Name: "John Doe",
				Age:  30,
			},
			wantErr: false,
			errLen:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.user)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
					return
				}
				if len(err) != tt.errLen {
					t.Errorf("expected %d errors, got %d", tt.errLen, len(err))
				}
			} else {
				if len(err) > 0 {
					t.Errorf("expected no error but got %v", err)
				}
			}
		})
	}
}
