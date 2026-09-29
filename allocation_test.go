//go:build !race

package validation_test

import (
	"testing"

	v "github.com/jacoelho/validation/v2"
)

func TestAllocValidCore(t *testing.T) {
	type Input struct {
		Name   string
		Age    int
		Values []int
	}
	input := Input{Name: "Alice", Age: 28, Values: []int{1, 2, 3, 4, 5, 6, 7, 8}}
	rule := v.All(
		v.Field("name", func(i Input) string { return i.Name }, v.NotEmpty[string](), v.RuneMinLength[string](2)),
		v.Field("age", func(i Input) int { return i.Age }, v.Min(18), v.Max(120)),
		v.Field("values", func(i Input) []int { return i.Values }, v.SliceUnique[[]int](), v.Each[[]int](v.Positive[int]())),
	)
	allocs := testing.AllocsPerRun(1000, func() {
		if err := rule(input); err != nil {
			panic(err)
		}
	})
	if allocs != 0 {
		t.Fatalf("valid core allocs/op = %v, want 0", allocs)
	}
}

func TestAllocSuccessfulLastAlternative(t *testing.T) {
	factoryCalls := 0
	rule := v.Check(func(value int) bool { return value < 0 || value == 0 || value == 7 }, func(int) error { factoryCalls++; return v.NewViolation("unexpected", nil) })
	allocs := testing.AllocsPerRun(1000, func() {
		if err := rule(7); err != nil {
			panic(err)
		}
	})
	if allocs != 0 || factoryCalls != 0 {
		t.Fatalf("allocs/op=%v factory calls=%d", allocs, factoryCalls)
	}
}
