package validation_test

import (
	"errors"
	"testing"

	v "github.com/jacoelho/validation/v2"
)

type acceptanceMoney struct {
	minor  int
	called *int
	cause  error
}

func (m *acceptanceMoney) Validate() error {
	*m.called++
	if m.minor < 0 {
		return m.cause
	}
	return nil
}

type acceptanceMoneyRecord struct {
	Left  *acceptanceMoney
	Right *acceptanceMoney
}

func TestAT_EXTENSIONS_001_TypedMethodExpressionAsFieldRule(t *testing.T) {
	t.Parallel()

	cause := errors.New("money is invalid")
	calls := 0
	money := &acceptanceMoney{minor: -1, called: &calls, cause: cause}
	validate := v.Rule[*acceptanceMoney]((*acceptanceMoney).Validate)
	rule := v.All(
		v.Field("left", func(record acceptanceMoneyRecord) *acceptanceMoney { return record.Left }, validate),
		v.Field("right", func(record acceptanceMoneyRecord) *acceptanceMoney { return record.Right }, validate),
	)

	err := rule(acceptanceMoneyRecord{Left: money, Right: money})
	if calls != 2 {
		t.Fatalf("Validate calls = %d, want one call per field", calls)
	}
	if !errors.Is(err, cause) {
		t.Fatalf("returned cause is not discoverable: %v", err)
	}
	assertIssues(t, err,
		[]v.Code{v.CodeExternal, v.CodeExternal},
		[]string{"$.left", "$.right"},
	)
}

type acceptanceAmount struct {
	minor int
}

type acceptanceAmountRecord struct {
	Amount acceptanceAmount
}

func TestAT_EXTENSIONS_002_ProjectExtractsOnceWithoutAddingAPathSegment(t *testing.T) {
	t.Parallel()

	getterCalls := 0
	rule := v.Field("amount", func(record acceptanceAmountRecord) acceptanceAmount {
		return record.Amount
	}, v.Project(func(amount acceptanceAmount) int {
		getterCalls++
		return amount.minor
	}, v.Min(100), v.Max(40)))

	err := rule(acceptanceAmountRecord{Amount: acceptanceAmount{minor: 50}})
	if getterCalls != 1 {
		t.Fatalf("project getter calls = %d, want 1", getterCalls)
	}
	assertIssues(t, err,
		[]v.Code{v.CodeMin, v.CodeMax},
		[]string{"$.amount", "$.amount"},
	)
}

func TestAT_EXTENSIONS_003_CheckSuccessfulAlternativesAvoidDiscardedDiagnostics(t *testing.T) {
	var calls [3]int
	factoryCalls := 0
	a, b, c := false, false, true
	rule := v.Check(func(int) bool {
		calls[0]++ // A
		if a {
			return true
		}
		calls[1]++ // B
		if b {
			return true
		}
		calls[2]++ // C
		return c
	}, func(int) error {
		factoryCalls++
		return v.NewViolation("unexpected", nil)
	})

	if err := rule(7); err != nil {
		t.Fatalf("successful alternatives returned %v", err)
	}
	if calls != [3]int{1, 1, 1} || factoryCalls != 0 {
		t.Fatalf("calls = %v, factory calls = %d; want one predicate pass and no factory", calls, factoryCalls)
	}

	result := testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if err := rule(7); err != nil {
				b.Fatal(err)
			}
		}
	})
	if result.AllocsPerOp() != 0 || result.AllocedBytesPerOp() != 0 {
		t.Fatalf("successful Check allocates %d objects and %d bytes per operation", result.AllocsPerOp(), result.AllocedBytesPerOp())
	}
}

func TestAT_EXTENSIONS_005_CheckNegationDoesNotConstructFailure(t *testing.T) {
	t.Parallel()

	forbiddenCalls := 0
	factoryCalls := 0
	rule := v.Check(func(value string) bool {
		forbiddenCalls++
		return value != "forbidden"
	}, func(string) error {
		factoryCalls++
		return v.NewViolation("forbidden", nil)
	})

	if err := rule("safe"); err != nil {
		t.Fatalf("NOT isForbidden returned %v", err)
	}
	if forbiddenCalls != 1 || factoryCalls != 0 {
		t.Fatalf("isForbidden calls = %d, failure factory calls = %d; want 1/0", forbiddenCalls, factoryCalls)
	}
}

type acceptanceRequiredRecord struct {
	Name *string
}

func TestAT_EXTENSIONS_006_ExternalFailureRetainedAlongsideRequiredField(t *testing.T) {
	t.Parallel()

	extraction := errors.New("extraction failed")
	rule := v.All(
		v.Rule[acceptanceRequiredRecord](func(acceptanceRequiredRecord) error { return extraction }),
		v.Field("name", func(record acceptanceRequiredRecord) *string { return record.Name }, v.RequiredPtr[string]()),
	)

	err := rule(acceptanceRequiredRecord{})
	if !errors.Is(err, extraction) {
		t.Fatalf("external extraction failure was lost: %v", err)
	}
	assertIssues(t, err,
		[]v.Code{v.CodeExternal, v.CodeRequired},
		[]string{"$", "$.name"},
	)
}

type acceptanceAge int
type acceptanceLabel string
type acceptanceBlob []byte
type acceptanceNames []string
type acceptanceScores map[string]int

type acceptanceDocument struct {
	Age    acceptanceAge
	Label  acceptanceLabel
	Blob   acceptanceBlob
	Names  acceptanceNames
	Scores acceptanceScores
}

func TestAT_TYPING_ARCHITECTURE_001_NamedValuesComposeAsTypedRules(t *testing.T) {
	t.Parallel()

	ageRule := v.Min(acceptanceAge(18))
	labelRule := v.NotEmpty[acceptanceLabel]()
	blobRule := v.BytesNotEmpty[acceptanceBlob]()
	namesRule := v.Each[acceptanceNames](v.NotEmpty[string]())
	scoresRule := v.MapValues[acceptanceScores](v.StringKeys[string](), v.Positive[int]())
	rule := v.Struct(
		v.Field("age", func(document acceptanceDocument) acceptanceAge { return document.Age }, ageRule),
		v.Field("label", func(document acceptanceDocument) acceptanceLabel { return document.Label }, labelRule),
		v.Field("blob", func(document acceptanceDocument) acceptanceBlob { return document.Blob }, blobRule),
		v.Field("names", func(document acceptanceDocument) acceptanceNames { return document.Names }, namesRule),
		v.Field("scores", func(document acceptanceDocument) acceptanceScores { return document.Scores }, scoresRule),
	)

	valid := acceptanceDocument{
		Age:    21,
		Label:  "ready",
		Blob:   acceptanceBlob{1},
		Names:  acceptanceNames{"Ada", "Lin"},
		Scores: acceptanceScores{"one": 1, "two": 2},
	}
	if err := rule(valid); err != nil {
		t.Fatalf("named values failed typed composition: %v", err)
	}
	if got := v.All(
		v.Field("age", func(document acceptanceDocument) acceptanceAge { return document.Age }, ageRule),
		v.Field("label", func(document acceptanceDocument) acceptanceLabel { return document.Label }, labelRule),
		v.Field("blob", func(document acceptanceDocument) acceptanceBlob { return document.Blob }, blobRule),
		v.Field("names", func(document acceptanceDocument) acceptanceNames { return document.Names }, namesRule),
		v.Field("scores", func(document acceptanceDocument) acceptanceScores { return document.Scores }, scoresRule),
	)(valid); got != nil {
		t.Fatalf("All composition failed where Struct succeeded: %v", got)
	}
}

type acceptanceArrayParent struct {
	Values [4]int
}

func TestAT_TYPING_ARCHITECTURE_002_PointerParentArrayProjectionPreservesStorage(t *testing.T) {
	parent := &acceptanceArrayParent{Values: [4]int{1, 2, 3, 4}}
	rule := v.Field("values", func(parent *acceptanceArrayParent) []int {
		return parent.Values[:]
	}, v.Each[[]int](v.Positive[int]()))

	if err := rule(parent); err != nil {
		t.Fatalf("array projection failed: %v", err)
	}
	if parent.Values != ([4]int{1, 2, 3, 4}) {
		t.Fatalf("array changed after validation: %v", parent.Values)
	}

	result := testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if err := rule(parent); err != nil {
				b.Fatal(err)
			}
		}
	})
	if result.AllocsPerOp() != 0 || result.AllocedBytesPerOp() != 0 {
		t.Fatalf("valid array projection allocates %d objects and %d bytes per operation", result.AllocsPerOp(), result.AllocedBytesPerOp())
	}
}

func TestAT_TYPING_ARCHITECTURE_006_ZeroRuleIsConfigurationErrorAndDirectNilCallPanics(t *testing.T) {
	t.Parallel()

	var zero v.Rule[int]
	func() {
		defer func() {
			caught := recover()
			configuration, ok := caught.(*v.ConfigurationError)
			if !ok || configuration.Constructor() != "All" {
				t.Errorf("All(nil rule) panic = %v, want ConfigurationError(All)", caught)
			}
		}()
		v.All(zero)
	}()

	func() {
		defer func() {
			if caught := recover(); caught == nil {
				t.Error("direct nil Rule call did not panic")
			} else if _, ok := caught.(*v.ConfigurationError); ok {
				t.Errorf("direct nil Rule call produced configuration panic: %v", caught)
			}
		}()
		zero(0)
	}()
}

func TestAT_ALLOCATION_005_CustomSuccessfulCallbackAllocationIsMeasured(t *testing.T) {
	var sink []byte
	allocating := v.All(v.NotEmpty[string](), v.Rule[string](func(string) error {
		sink = make([]byte, 1)
		return nil
	}))
	nonAllocating := v.All(v.NotEmpty[string](), v.Rule[string](func(string) error { return nil }))

	custom := testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if err := allocating("ok"); err != nil {
				b.Fatal(err)
			}
		}
	})
	if custom.AllocsPerOp() <= 0 || custom.AllocedBytesPerOp() <= 0 {
		t.Fatalf("allocating callback measured %d objects and %d bytes per operation", custom.AllocsPerOp(), custom.AllocedBytesPerOp())
	}

	fixed := testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if err := nonAllocating("ok"); err != nil {
				b.Fatal(err)
			}
		}
	})
	if fixed.AllocsPerOp() != 0 || fixed.AllocedBytesPerOp() != 0 {
		t.Fatalf("non-allocating callback measured %d objects and %d bytes per operation", fixed.AllocsPerOp(), fixed.AllocedBytesPerOp())
	}
	sink = nil
	_ = sink
}
