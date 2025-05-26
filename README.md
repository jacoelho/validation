# Validation

A type-safe, composable validation library for Go that leverages generics to provide compile-time safety and runtime flexibility.

## Features

- Type-safe: Built with Go generics for compile-time type safety
- Composable: Combine validation rules using logical operators (`Or`, `When`, `Unless`)
- Struct validation: Deep validation of nested structs with field-level error reporting
- Collection support: Validate slices and maps with element-level validation
- Rich error context: Detailed error messages with field paths and parameters
- Extensible: Easy to create custom validation rules
- Zero dependencies: Pure Go implementation
- Fatal error handling: Stop validation chains on critical errors

## Installation

```bash
go get github.com/jacoelho/validation
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/jacoelho/validation"
)

type User struct {
    Name  string
    Email string
    Age   int
}

func main() {
    validator := validation.Struct(
        validation.Field("Name", 
            func(u User) string { return u.Name },
            validation.Required[string](),
            validation.StringsRuneMinLength[string](2),
        ),
        validation.Field("Email",
            func(u User) string { return u.Email },
            validation.Required[string](),
            validation.StringsMatchesRegex[string](`^[^@]+@[^@]+\.[^@]+$`),
        ),
        validation.Field("Age",
            func(u User) int { return u.Age },
            validation.NumbersMin(18),
            validation.NumbersMax(120),
        ),
    )

    user := User{Name: "A", Email: "invalid", Age: 15}
    
    if errs := validator.Validate(user); errs.HasErrors() {
        for _, err := range errs {
            fmt.Println(err.Error())
        }
        // Output:
        // min (field: Name) {actual: 1, min: 2}
        // regex (field: Email) {pattern: ^[^@]+@[^@]+\.[^@]+$}
        // min (field: Age) {actual: 15, min: 18}
    }
}
```

## Core Concepts

### Rules

Rules are functions that validate a single value and return an error if validation fails:

```go
type Rule[T any] func(value T) *Error
```

### Validators

Validators coordinate multiple rules and provide structured validation for complex types like structs, slices, and maps.

### Errors

Validation errors include field paths, error codes, and contextual parameters:

```go
type Error struct {
    Field  string                 // Field path (e.g., "User.Address.City")
    Code   string                 // Error code (e.g., "required", "min")
    Params map[string]any         // Additional error parameters
    Fatal  bool                   // Whether to stop validation
}
```

## Validation Rules Reference

### Core Rules

```go
// Basic validation
validation.Required[string]()                    // Not zero value
validation.RequiredZeroable[time.Time]()         // For types with IsZero() method

// Logical operators
validation.RuleNot(validation.Required[string]()) // Negate a rule
validation.Or(rule1, rule2, rule3)               // Any rule must pass
validation.When(condition, rule)                 // Apply rule conditionally
validation.Unless(condition, rule)               // Apply rule unless condition

// Control flow
validation.RuleStopOnError(rule)                 // Stop validation on error
```

### String Validation

```go
validation.StringsNotEmpty[string]()                           // Non-empty
validation.StringsRuneMinLength[string](5)                     // Min rune length
validation.StringsRuneMaxLength[string](100)                   // Max rune length
validation.StringsRuneLengthBetween[string](5, 100)           // Length range
validation.StringsMatchesRegex[string](`^\w+@\w+\.\w+$`)      // Regex pattern
validation.StringsContains[string]("@")                       // Contains substring
validation.StringsAllowed[string]("admin", "user", "guest")   // Whitelist
validation.StringsDisallowed[string]("root", "admin")         // Blacklist
```

### Numeric Validation

```go
validation.NumbersMin(18)                    // Minimum value
validation.NumbersMax(65)                    // Maximum value
validation.NumbersBetween(18, 65)            // Range validation
validation.NumbersPositive[int]()            // Greater than zero
validation.NumbersNegative[int]()            // Less than zero
```

### Time Validation

```go
now := time.Now()
validation.TimeBefore(now)                   // Before specific time
validation.TimeAfter(now.AddDate(-1, 0, 0))  // After specific time
validation.TimeBetween(start, end)           // Between time range
```

### Slice Validation

```go
validation.SlicesMinLength[string](1)                    // Minimum length
validation.SlicesMaxLength[string](10)                   // Maximum length
validation.SlicesLength[string](5)                       // Exact length
validation.SlicesInBetweenLength[string](1, 10)         // Length range
validation.SlicesUnique[string]()                       // All elements unique
validation.SlicesContains[string]("required")           // Contains value
validation.SlicesAllowed[string]("a", "b", "c")         // Element whitelist
validation.SlicesDisallowed[string]("x", "y")           // Element blacklist

// Validate each element
validation.SlicesForEach(
    validation.StringsNotEmpty[string](),
    validation.StringsRuneMaxLength[string](50),
)
```

### Map Validation

```go
validation.MapsMinKeys[string, string](1)               // Minimum keys
validation.MapsMaxKeys[string, string](10)              // Maximum keys
validation.MapsLength[string, string](5)                // Exact key count
validation.MapsLengthBetween[string, string](1, 10)     // Key count range
validation.MapsKeysAllowed[string, string]("a", "b")    // Key whitelist
validation.MapsKeysDisallowed[string, string]("x")      // Key blacklist
validation.MapsValuesAllowed[string, string]("y", "z")  // Value whitelist
validation.MapsValuesDisallowed[string, string]("bad")  // Value blacklist

// Validate each key-value pair
validation.MapsForEach(func(key, value string) *validation.Error {
    if value == "" {
        return &validation.Error{Code: "empty_value"}
    }
    return nil
})
```

## Struct Validation

### Basic Example

```go
type Person struct {
    Name string
    Age  int
}

validator := validation.Struct(
    validation.Field("Name", 
        func(p Person) string { return p.Name },
        validation.Required[string](),
        validation.StringsRuneMinLength[string](2),
        validation.StringsRuneMaxLength[string](50),
    ),
    validation.Field("Age",
        func(p Person) int { return p.Age },
        validation.NumbersMin(0),
        validation.NumbersMax(150),
    ),
)

person := Person{Name: "John", Age: 30}
errs := validator.Validate(person)
```

### Nested Struct Validation

```go
type Address struct {
    Street string
    City   string
    ZIP    string
}

type User struct {
    Name    string
    Email   string
    Address Address
}

// Create address validator
addressValidator := validation.Struct(
    validation.Field("Street", func(a Address) string { return a.Street },
        validation.Required[string](),
        validation.StringsRuneMinLength[string](5),
    ),
    validation.Field("City", func(a Address) string { return a.City },
        validation.Required[string](),
        validation.StringsRuneMinLength[string](2),
    ),
    validation.Field("ZIP", func(a Address) string { return a.ZIP },
        validation.Required[string](),
        validation.StringsMatchesRegex[string](`^\d{5}(-\d{4})?$`),
    ),
)

// Create user validator with nested address
userValidator := validation.Struct(
    validation.Field("Name", func(u User) string { return u.Name },
        validation.Required[string](),
    ),
    validation.Field("Email", func(u User) string { return u.Email },
        validation.Required[string](),
        validation.StringsMatchesRegex[string](`^[^@]+@[^@]+\.[^@]+$`),
    ),
    validation.StructField("Address", func(u User) Address { return u.Address },
        addressValidator),
)

user := User{
    Name:  "John Doe",
    Email: "john@example.com",
    Address: Address{
        Street: "123 Main St",
        City:   "Anytown",
        ZIP:    "12345",
    },
}

errs := userValidator.Validate(user)
if errs.HasErrors() {
    for _, err := range errs {
        fmt.Printf("Field: %s, Error: %s\n", err.Field, err.Code)
        // Example output: "Field: Address.ZIP, Error: regex"
    }
}
```

### Collection Fields

```go
type User struct {
    Name     string
    Tags     []string
    Settings map[string]string
}

validator := validation.Struct(
    validation.Field("Name", func(u User) string { return u.Name },
        validation.Required[string](),
    ),
    
    // Slice validation
    validation.SliceField("Tags", func(u User) []string { return u.Tags },
        validation.SlicesMinLength[string](1),
        validation.SlicesMaxLength[string](5),
        validation.SlicesUnique[string](),
        validation.SlicesForEach(
            validation.StringsNotEmpty[string](),
            validation.StringsRuneMaxLength[string](20),
        ),
    ),
    
    // Map validation
    validation.MapField("Settings", func(u User) map[string]string { return u.Settings },
        validation.MapsMaxKeys[string, string](10),
        validation.MapsForEach(func(k, v string) *validation.Error {
            if v == "" {
                return &validation.Error{
                    Code:   "empty_setting_value",
                    Params: map[string]any{"key": k},
                }
            }
            return nil
        }),
    ),
)
```

## Error Handling

### Checking for Errors

```go
errs := validator.Validate(data)

// Check if any errors exist
if errs.HasErrors() {
    // Handle validation errors
}

// Check for fatal errors (validation stopped early)
if errs.HasFatalErrors() {
    // Handle critical validation failures
}
```

### Error Formatting

```go
// Default error formatting
fmt.Println(errs.Error())
// Output: "required (field: Name); min (field: Age) {actual: 15, min: 18}"

// Custom formatting
customFormat := errs.Format(func(e *validation.Error) string {
    return fmt.Sprintf("%s: %s", e.Field, e.Code)
}, "\n")
fmt.Println(customFormat)
// Output:
// Name: required
// Age: min

// Individual error details
for _, err := range errs {
    fmt.Printf("Field: %s\n", err.Field)
    fmt.Printf("Code: %s\n", err.Code)
    fmt.Printf("Params: %v\n", err.Params)
    fmt.Printf("Fatal: %t\n", err.Fatal)
}
```

## Custom Validation Rules

### Simple Custom Rule

```go
func ValidateEmail() validation.Rule[string] {
    return func(value string) *validation.Error {
        if !strings.Contains(value, "@") || !strings.Contains(value, ".") {
            return &validation.Error{
                Code: "invalid_email",
                Params: map[string]any{
                    "value": value,
                },
            }
        }
        return nil
    }
}

// Usage
validation.Field("Email", func(u User) string { return u.Email },
    ValidateEmail(),
)
```

### Parameterized Custom Rule

```go
func StringContainsAny(substrings ...string) validation.Rule[string] {
    return func(value string) *validation.Error {
        for _, substr := range substrings {
            if strings.Contains(value, substr) {
                return nil
            }
        }
        return &validation.Error{
            Code: "missing_required_substring",
            Params: map[string]any{
                "value":      value,
                "substrings": substrings,
            },
        }
    }
}

// Usage
validation.Field("Description", func(p Product) string { return p.Description },
    StringContainsAny("premium", "deluxe", "pro"),
)
```

### Custom Rule with External Dependencies

```go
func ValidateUniqueUsername(userRepo UserRepository) validation.Rule[string] {
    return func(username string) *validation.Error {
        exists, err := userRepo.UsernameExists(username)
        if err != nil {
            return &validation.Error{
                Code: "username_check_failed",
                Params: map[string]any{"error": err.Error()},
                Fatal: true, // Stop validation on system errors
            }
        }
        if exists {
            return &validation.Error{
                Code: "username_taken",
                Params: map[string]any{"username": username},
            }
        }
        return nil
    }
}
```

## Advanced Usage

### Conditional Validation

```go
type User struct {
    Type       string
    CreditCard string
    BankAccount string
}

validator := validation.Struct(
    validation.Field("Type", func(u User) string { return u.Type },
        validation.StringsAllowed[string]("basic", "premium"),
    ),
    
    // Require credit card for premium users
    validation.Field("CreditCard", func(u User) string { return u.CreditCard },
        validation.When(
            func(u User) bool { return u.Type == "premium" },
            validation.Required[string](),
        ),
    ),
    
    // Bank account is optional for basic users, forbidden for premium
    validation.Field("BankAccount", func(u User) string { return u.BankAccount },
        validation.Unless(
            func(u User) bool { return u.Type == "premium" },
            validation.Required[string](),
        ),
    ),
)
```

### Complex Logical Validation

```go
// Either email or phone must be provided
validation.Field("ContactInfo", func(u User) User { return u },
    validation.Or(
        validation.Field("Email", func(u User) string { return u.Email },
            validation.Required[string](),
        ),
        validation.Field("Phone", func(u User) string { return u.Phone },
            validation.Required[string](),
        ),
    ),
)
```

### Fatal Error Handling

```go
validation.Field("Password", func(u User) string { return u.Password },
    // Stop validation if password is missing
    validation.RuleStopOnError(validation.Required[string]()),
    
    // These rules only run if password is present
    validation.StringsRuneMinLength[string](8),
    validation.StringsMatchesRegex[string](`[A-Z]`), // Must contain uppercase
    validation.StringsMatchesRegex[string](`[0-9]`), // Must contain number
)
```

### Validation Groups

```go
// Create reusable validation groups
var (
    emailRules = []validation.Rule[string]{
        validation.Required[string](),
        validation.StringsMatchesRegex[string](`^[^@]+@[^@]+\.[^@]+$`),
        validation.StringsRuneMaxLength[string](100),
    }
    
    passwordRules = []validation.Rule[string]{
        validation.Required[string](),
        validation.StringsRuneMinLength[string](8),
        validation.StringsMatchesRegex[string](`[A-Z]`),
        validation.StringsMatchesRegex[string](`[a-z]`),
        validation.StringsMatchesRegex[string](`[0-9]`),
    }
)

userValidator := validation.Struct(
    validation.Field("Email", func(u User) string { return u.Email }, emailRules...),
    validation.Field("Password", func(u User) string { return u.Password }, passwordRules...),
)
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.


