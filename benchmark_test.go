package validation_test

import (
	"runtime"
	"strconv"
	"testing"

	v "github.com/jacoelho/validation/v2"
)

func reportBenchmarkContext(b *testing.B, inputItems int) {
	b.Helper()
	b.ReportMetric(float64(inputItems), "input-items")
	b.Logf("compiler=%s platform=%s/%s input-items=%d", runtime.Version(), runtime.GOOS, runtime.GOARCH, inputItems)
}

func BenchmarkValidation(b *testing.B) {
	type Record struct {
		Name  string
		Count int
		Tags  []string
	}
	input := Record{Name: "ready", Count: 4, Tags: []string{"alpha", "beta"}}
	inputItems := 3 + len(input.Tags)
	rule := v.All(
		v.Field("name", func(r Record) string { return r.Name }, v.NotEmpty[string](), v.RuneMaxLength[string](20)),
		v.Field("count", func(r Record) int { return r.Count }, v.Min(0), v.Max(10)),
		v.Field("tags", func(r Record) []string { return r.Tags }, v.Each[[]string](v.NotEmpty[string]())),
	)
	b.Run("record-fields-3-tags-2/valid", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if err := rule(input); err != nil {
				b.Fatal(err)
			}
		}
		reportBenchmarkContext(b, inputItems)
	})
	b.Run("record-fields-3-tags-2/invalid", func(b *testing.B) {
		invalid := input
		invalid.Name = ""
		b.ReportAllocs()
		for b.Loop() {
			if err := rule(invalid); err == nil {
				b.Fatal("missing failure")
			}
		}
		reportBenchmarkContext(b, inputItems)
	})
	b.Run("record-fields-3-tags-2/reporting", func(b *testing.B) {
		invalid := input
		invalid.Name = ""
		err := rule(invalid)
		b.ReportAllocs()
		for b.Loop() {
			_ = v.Format(err)
		}
		reportBenchmarkContext(b, inputItems)
	})
}

var constructedBenchmarkRule v.Rule[int]

func BenchmarkConstruction(b *testing.B) {
	b.Run("membership-4", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			constructedBenchmarkRule = v.All(v.Min(1), v.Max(10), v.OneOf(1, 2, 3, 4))
		}
		reportBenchmarkContext(b, 4)
	})
}

func BenchmarkTraversal(b *testing.B) {
	b.Run("issues-2", func(b *testing.B) {
		rule := v.All(v.NotEmpty[string](), v.RuneMinLength[string](2))
		err := rule("")
		b.ReportAllocs()
		for b.Loop() {
			if len(v.Issues(err)) != 2 {
				b.Fatal("lost issues")
			}
		}
		reportBenchmarkContext(b, 2)
	})
}

func BenchmarkSliceUnique(b *testing.B) {
	for _, size := range []int{8, 64, 1024} {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			values := make([]int, size)
			for i := range values {
				values[i] = i
			}
			rule := v.SliceUnique[[]int]()
			b.ReportAllocs()
			for b.Loop() {
				if err := rule(values); err != nil {
					b.Fatal(err)
				}
			}
			reportBenchmarkContext(b, size)
		})
	}
}

func BenchmarkValidMap(b *testing.B) {
	for _, size := range []int{8, 64, 1024, 10000} {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			values := make(map[string]int, size)
			for i := 0; i < size; i++ {
				values[strconv.Itoa(i)] = i + 1
			}
			rule := v.MapValues[map[string]int](v.StringKeys[string](), v.Positive[int]())
			b.ReportAllocs()
			for b.Loop() {
				if err := rule(values); err != nil {
					b.Fatal(err)
				}
			}
			reportBenchmarkContext(b, size)
		})
	}
}
