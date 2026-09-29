package validation_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	validation "github.com/jacoelho/validation/v2"
)

func TestAcceptanceComparablePrimitiveIntegerRows(t *testing.T) {
	cases := []struct {
		name  string
		rule  validation.Rule[int]
		value int
		code  validation.Code
	}{
		{"AT-COMPARABLE-001/row-1", validation.Equal(3), 3, ""},
		{"AT-COMPARABLE-001/row-2", validation.Equal(3), 4, validation.CodeEqual},
		{"AT-COMPARABLE-001/row-3", validation.NotEqual(3), 3, validation.CodeNotEqual},
		{"AT-COMPARABLE-001/row-4", validation.Zero[int](), 0, ""},
		{"AT-COMPARABLE-001/row-5", validation.Zero[int](), 1, validation.CodeZero},
		{"AT-COMPARABLE-001/row-6", validation.NotZero[int](), 0, validation.CodeNotZero},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.rule(tc.value)
			if tc.code == "" {
				if err != nil {
					t.Fatalf("unexpected failure: %v", err)
				}
				return
			}
			requirePrimitiveCodes(t, err, tc.code)
		})
	}
}

func TestAcceptanceComparableBoolAndStringRules(t *testing.T) {
	t.Run("AT-COMPARABLE-001/row-7", func(t *testing.T) {
		if err := validation.Equal(false)(false); err != nil {
			t.Fatalf("Equal(false) rejected false: %v", err)
		}
	})
	t.Run("AT-COMPARABLE-001/row-8", func(t *testing.T) {
		requirePrimitiveCodes(t, validation.NotZero[bool]()(false), validation.CodeNotZero)
	})
	t.Run("AT-COMPARABLE-001/row-9", func(t *testing.T) {
		if err := validation.OneOf("a", "b", "b")("b"); err != nil {
			t.Fatalf("duplicate OneOf members rejected an allowed value: %v", err)
		}
	})
	t.Run("AT-COMPARABLE-001/row-10", func(t *testing.T) {
		requirePrimitiveCodes(t, validation.OneOf[string]()("a"), validation.CodeOneOf)
	})
	t.Run("AT-COMPARABLE-001/row-11", func(t *testing.T) {
		if err := validation.NotOneOf[string]()("a"); err != nil {
			t.Fatalf("empty NotOneOf rejected a value: %v", err)
		}
	})
	t.Run("AT-COMPARABLE-001/row-12", func(t *testing.T) {
		requirePrimitiveCodes(t, validation.NotOneOf("a", "b")("b"), validation.CodeNotOneOf)
	})
}

func TestAcceptanceComparableNamedValue(t *testing.T) {
	t.Run("AT-COMPARABLE-003", func(t *testing.T) {
		type namedComparable int
		if err := validation.Equal(namedComparable(3))(namedComparable(3)); err != nil {
			t.Fatalf("Equal did not accept a safely comparable named value: %v", err)
		}
	})
}

func TestAcceptanceComparableIEEEEquality(t *testing.T) {
	tests := []struct {
		name  string
		rule  validation.Rule[float64]
		value float64
		want  validation.Code
	}{
		{name: "AT-COMPARABLE-002/row-1 Equal NaN", rule: validation.Equal(math.NaN()), value: math.NaN(), want: validation.CodeEqual},
		{name: "AT-COMPARABLE-002/row-2 OneOf NaN", rule: validation.OneOf(math.NaN()), value: math.NaN(), want: validation.CodeOneOf},
		{name: "AT-COMPARABLE-002/row-3 NotOneOf NaN", rule: validation.NotOneOf(math.NaN()), value: math.NaN(), want: ""},
		{name: "AT-COMPARABLE-002/row-4 signed zero", rule: validation.Equal(0.0), value: math.Copysign(0, -1), want: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.want == "" {
				if err := test.rule(test.value); err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			requirePrimitiveCodes(t, test.rule(test.value), test.want)
		})
	}
}

func TestAcceptanceNumericDegenerateAndExtremeIntegerBoundaries(t *testing.T) {
	t.Run("AT-NUMBERS-001/row-12", func(t *testing.T) {
		if err := validation.Between(5, 5)(5); err != nil {
			t.Fatalf("degenerate inclusive range rejected its only value: %v", err)
		}
	})
	t.Run("AT-NUMBERS-002/row-2", func(t *testing.T) {
		const maxUint64 = ^uint64(0)
		if err := validation.Min[uint64](maxUint64)(maxUint64); err != nil {
			t.Fatalf("maximum uint64 did not satisfy its inclusive minimum: %v", err)
		}
	})
	t.Run("AT-NUMBERS-002/row-4", func(t *testing.T) {
		const minInt64 = -1 << 63
		const maxInt64 = 1<<63 - 1
		if err := validation.Between[int64](minInt64, maxInt64)(maxInt64); err != nil {
			t.Fatalf("maximum int64 did not satisfy the full inclusive range: %v", err)
		}
	})
}

func TestAcceptanceNumericNaNPolicies(t *testing.T) {
	tests := []struct {
		name string
		rule validation.Rule[float64]
		want validation.Code
	}{
		{name: "AT-NUMBERS-003/row-4 greater than", rule: validation.GreaterThan(0.0), want: validation.CodeGreaterThan},
		{name: "AT-NUMBERS-003/row-5 less than", rule: validation.LessThan(0.0), want: validation.CodeLessThan},
		{name: "AT-NUMBERS-003/row-7 non-negative", rule: validation.NonNegative[float64](), want: validation.CodeNonNegative},
		{name: "AT-NUMBERS-003/row-8 negative", rule: validation.Negative[float64](), want: validation.CodeNegative},
		{name: "AT-NUMBERS-003/row-9 non-positive", rule: validation.NonPositive[float64](), want: validation.CodeNonPositive},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requirePrimitiveCodes(t, test.rule(math.NaN()), test.want)
		})
	}
}

func TestAcceptanceNumericFloatPolicies(t *testing.T) {
	tests := []struct {
		name  string
		rule  validation.Rule[float64]
		value float64
		want  validation.Code
	}{
		{name: "AT-NUMBERS-004/row-2 infinity is not NaN", rule: validation.NotNaN[float64](), value: math.Inf(1), want: ""},
		{name: "AT-NUMBERS-004/row-5 negative infinity is not finite", rule: validation.Finite[float64](), value: math.Inf(-1), want: validation.CodeFinite},
		{name: "AT-NUMBERS-004/row-6 finite accepts negative zero", rule: validation.Finite[float64](), value: math.Copysign(0, -1), want: ""},
		{name: "AT-NUMBERS-004/row-9 positive zero is not negative", rule: validation.Negative[float64](), value: 0, want: validation.CodeNegative},
		{name: "AT-NUMBERS-004/row-10 negative zero is non-negative", rule: validation.NonNegative[float64](), value: math.Copysign(0, -1), want: ""},
		{name: "AT-NUMBERS-004/row-13 positive infinity is in infinite range", rule: validation.Between(math.Inf(-1), math.Inf(1)), value: math.Inf(1), want: ""},
		{name: "AT-NUMBERS-004/row-14 NaN is outside every range", rule: validation.Between(math.Inf(-1), math.Inf(1)), value: math.NaN(), want: validation.CodeBetween},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.want == "" {
				if err := test.rule(test.value); err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			requirePrimitiveCodes(t, test.rule(test.value), test.want)
		})
	}
}

func TestAcceptanceNumericCompileConstraints(t *testing.T) {
	tests := []struct {
		name       string
		expression string
	}{
		{name: "AT-NUMBERS-005/row-1 string minimum", expression: `var _ = v.Min[string]("a")`},
		{name: "AT-NUMBERS-005/row-2 string positivity", expression: `var _ = v.Positive[string]()`},
		{name: "AT-NUMBERS-005/row-3 complex range", expression: `var _ = v.Between[complex128](0, 1)`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if !acceptanceCompileFixture(t, test.expression) {
				t.Fatal("unsupported numeric expression compiled successfully")
			}
		})
	}
}

func acceptanceCompileFixture(t *testing.T, expression string) bool {
	t.Helper()
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	mod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	line := strings.SplitN(string(mod), "\n", 2)[0]
	module := strings.TrimPrefix(line, "module ")
	version := "v2.0.0"
	dir := t.TempDir()
	goMod := fmt.Sprintf("module acceptancefixture\n\ngo 1.27\n\nrequire %s %s\nreplace %s => %s\n", module, version, module, root)
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0600); err != nil {
		t.Fatal(err)
	}
	source := fmt.Sprintf("package fixture\n\nimport v %q\n\n%s\n", module, expression)
	if err := os.WriteFile(filepath.Join(dir, "fixture_test.go"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	return cmd.Run() != nil
}

func TestAcceptanceUnicodeByteAndRuneLengths(t *testing.T) {
	tests := []struct {
		name  string
		value string
		bytes int
		runes int
	}{
		{name: "AT-STRINGS_BYTES-001/row-1 empty", value: "", bytes: 0, runes: 0},
		{name: "AT-STRINGS_BYTES-001/row-2 ASCII", value: "A", bytes: 1, runes: 1},
		{name: "AT-STRINGS_BYTES-001/row-3 precomposed", value: "\u00e9", bytes: 2, runes: 1},
		{name: "AT-STRINGS_BYTES-001/row-4 combining sequence", value: "e\u0301", bytes: 3, runes: 2},
		{name: "AT-STRINGS_BYTES-001/row-5 smiling face", value: "🙂", bytes: 4, runes: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := len(test.value); got != test.bytes {
				t.Fatalf("byte length = %d, want %d", got, test.bytes)
			}
			if got := utf8.RuneCountInString(test.value); got != test.runes {
				t.Fatalf("rune count = %d, want %d", got, test.runes)
			}
			if err := validation.ByteLength[string](test.bytes)(test.value); err != nil {
				t.Fatalf("ByteLength rejected the fixture: %v", err)
			}
			if err := validation.RuneLength[string](test.runes)(test.value); err != nil {
				t.Fatalf("RuneLength rejected the fixture: %v", err)
			}
		})
	}
}

func TestAcceptanceInvalidUTF8DoesNotShortCircuitRuneLength(t *testing.T) {
	t.Run("AT-STRINGS_BYTES-002", func(t *testing.T) {
		value := string([]byte{0xff, 0xfe})
		utf8Calls, runeCalls := 0, 0
		utf8Rule := validation.UTF8[string]()
		runeRule := validation.RuneLength[string](2)
		rule := validation.All(
			func(value string) error {
				utf8Calls++
				return utf8Rule(value)
			},
			func(value string) error {
				runeCalls++
				return runeRule(value)
			},
		)
		requirePrimitiveCodes(t, rule(value), validation.CodeUTF8)
		if utf8Calls != 1 || runeCalls != 1 {
			t.Fatalf("rule calls = UTF8:%d RuneLength:%d, want one each", utf8Calls, runeCalls)
		}
	})
}

func TestAcceptanceStringLengthBoundaries(t *testing.T) {
	t.Run("AT-STRINGS_BYTES-003/row-4", func(t *testing.T) {
		if err := validation.ByteLengthBetween[string](2, 2)("é"); err != nil {
			t.Fatalf("ByteLengthBetween rejected the inclusive boundary: %v", err)
		}
	})
	t.Run("AT-STRINGS_BYTES-003/row-5", func(t *testing.T) {
		requirePrimitiveCodes(t, validation.RuneLength[string](2)("é"), validation.CodeRuneLength)
	})
	t.Run("AT-STRINGS_BYTES-003/row-6", func(t *testing.T) {
		if err := validation.RuneMinLength[string](1)("é"); err != nil {
			t.Fatalf("RuneMinLength rejected its inclusive boundary: %v", err)
		}
	})
}

func TestAcceptanceByteRulesBoundaries(t *testing.T) {
	t.Run("AT-STRINGS_BYTES-005/row-2", func(t *testing.T) {
		if err := validation.BytesNotEmpty[[]byte]()(nil); err == nil {
			t.Fatal("BytesNotEmpty accepted a nil byte slice")
		} else {
			requirePrimitiveCodes(t, err, validation.CodeNotEmpty)
		}
	})
	t.Run("AT-STRINGS_BYTES-005/row-3", func(t *testing.T) {
		value := []byte{0xff, 0xfe}
		want := append([]byte(nil), value...)
		if err := validation.BytesLength[[]byte](2)(value); err != nil {
			t.Fatalf("BytesLength rejected exact length: %v", err)
		}
		if string(value) != string(want) {
			t.Fatalf("byte input changed from %v to %v", want, value)
		}
	})
	t.Run("AT-STRINGS_BYTES-005/row-4", func(t *testing.T) {
		value := []byte{0xff, 0xfe}
		want := append([]byte(nil), value...)
		requirePrimitiveCodes(t, validation.BytesUTF8[[]byte]()(value), validation.CodeUTF8)
		if string(value) != string(want) {
			t.Fatalf("byte input changed from %v to %v", want, value)
		}
	})
	t.Run("AT-STRINGS_BYTES-005/row-8", func(t *testing.T) {
		value := []byte{0x00, 0xff}
		want := append([]byte(nil), value...)
		requirePrimitiveCodes(t, validation.BytesMaxLength[[]byte](1)(value), validation.CodeByteMaxLength)
		if string(value) != string(want) {
			t.Fatalf("byte input changed from %v to %v", want, value)
		}
	})
}

func TestAcceptanceByteImplementationsUseByteRepresentation(t *testing.T) {
	t.Run("AT-STRINGS_BYTES-006", func(t *testing.T) {
		type bytesValue []byte
		value := bytesValue{0x00, 0xff}
		if err := validation.BytesLength[bytesValue](2)(value); err != nil {
			t.Fatalf("named byte slice did not compile through BytesLength: %v", err)
		}
		if err := validation.BytesContains[bytesValue](bytesValue{0xff})(value); err != nil {
			t.Fatalf("named byte slice did not compile through BytesContains: %v", err)
		}

		source, err := os.ReadFile("strings.go")
		if err != nil {
			t.Fatal(err)
		}
		file, err := parser.ParseFile(token.NewFileSet(), "strings.go", source, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Body == nil || !strings.HasPrefix(function.Name.Name, "Bytes") {
				continue
			}
			ast.Inspect(function.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok || len(call.Args) != 1 {
					return true
				}
				identifier, ok := call.Fun.(*ast.Ident)
				argument, argumentIsValue := call.Args[0].(*ast.Ident)
				if ok && identifier.Name == "string" && argumentIsValue && argument.Name == "value" {
					t.Errorf("%s converts the validated byte value through string", function.Name.Name)
				}
				return true
			})
		}
	})
}

func TestAcceptanceTimeZeroRangeBoundary(t *testing.T) {
	t.Run("AT-TIME-002/row-4", func(t *testing.T) {
		min := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
		max := time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)
		requirePrimitiveCodes(t, validation.TimeBetween(min, max)(time.Time{}), validation.CodeBetween)
	})
}

func TestAcceptanceTimeNotZeroHasNoClockDependency(t *testing.T) {
	t.Run("AT-TIME-003", func(t *testing.T) {
		if err := validation.TimeNotZero()(time.Time{}); err == nil {
			t.Fatal("TimeNotZero accepted the zero time")
		}

		source, err := os.ReadFile("time.go")
		if err != nil {
			t.Fatal(err)
		}
		file, err := parser.ParseFile(token.NewFileSet(), "time.go", source, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Name.Name != "TimeNotZero" || function.Body == nil {
				continue
			}
			ast.Inspect(function.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if ok && selector.Sel.Name == "Now" {
					t.Errorf("TimeNotZero reads the current clock")
				}
				return true
			})
		}
	})
}
