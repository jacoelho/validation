package validation_test

import (
	"errors"
	"testing"

	validation "github.com/jacoelho/validation/v2"
)

func TestStringByteAndRuneLengths(t *testing.T) {
	for _, test := range []struct {
		name string
		rule validation.Rule[string]
		text string
		want validation.Code
	}{
		{"byte exact boundary", validation.ByteLength[string](2), "é", ""},
		{"byte minimum rejects", validation.ByteMinLength[string](3), "é", validation.CodeByteMinLength},
		{"byte maximum rejects", validation.ByteMaxLength[string](1), "é", validation.CodeByteMaxLength},
		{"rune exact distinguishes bytes", validation.RuneLength[string](2), "e\u0301", ""},
		{"rune maximum rejects", validation.RuneMaxLength[string](1), "e\u0301", validation.CodeRuneMaxLength},
		{"rune range boundary", validation.RuneLengthBetween[string](2, 2), "e\u0301", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := test.rule(test.text)
			if test.want == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			requirePrimitiveCodes(t, err, test.want)
		})
	}
}

func TestStringInvalidUTF8AndPredicates(t *testing.T) {
	invalid := string([]byte{0xff, 0xfe})
	if err := validation.RuneLength[string](2)(invalid); err != nil {
		t.Fatalf("invalid bytes still count as two replacement runes: %v", err)
	}
	requirePrimitiveCodes(t, validation.UTF8[string]()(invalid), validation.CodeUTF8)

	for _, test := range []struct {
		name string
		rule validation.Rule[string]
		text string
		want validation.Code
	}{
		{"not empty rejects empty", validation.NotEmpty[string](), "", validation.CodeNotEmpty},
		{"space is content", validation.NotEmpty[string](), " ", ""},
		{"contains is case sensitive", validation.Contains("A"), "abc", validation.CodeContains},
		{"empty contains matches", validation.Contains(""), "", ""},
		{"empty prefix matches", validation.HasPrefix(""), "", ""},
		{"empty suffix matches", validation.HasSuffix(""), "", ""},
		{"prefix matches", validation.HasPrefix("ab"), "abc", ""},
		{"suffix matches", validation.HasSuffix("bc"), "abc", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := test.rule(test.text)
			if test.want == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			requirePrimitiveCodes(t, err, test.want)
		})
	}
}

func TestStringLengthDiagnostics(t *testing.T) {
	err := validation.ByteLength[string](2)("x")
	var length *validation.LengthError
	if !errors.As(err, &length) {
		t.Fatalf("%T is not a LengthError", err)
	}
	if length.Actual() != 1 || length.Minimum() != 2 {
		t.Fatalf("length details = actual %d, minimum %d", length.Actual(), length.Minimum())
	}
	if max, ok := length.Maximum(); !ok || max != 2 || length.Unit() != validation.LengthBytes {
		t.Fatalf("length maximum/unit = (%d, %v)/%q", max, ok, length.Unit())
	}
}

func TestByteRulesPreserveNamedSlicesAndCopyConfiguration(t *testing.T) {
	type bytesValue []byte
	part := bytesValue{0xff}
	rule := validation.BytesContains(part)
	part[0] = 0
	if err := rule(bytesValue{0xff}); err != nil {
		t.Fatalf("constructor did not copy byte configuration: %v", err)
	}

	input := bytesValue{0x00, 0xff}
	copyOfInput := append(bytesValue(nil), input...)
	for _, test := range []struct {
		name string
		rule validation.Rule[bytesValue]
		want validation.Code
	}{
		{"equal nil and empty", validation.BytesEqual[bytesValue](nil), ""},
		{"not empty", validation.BytesNotEmpty[bytesValue](), ""},
		{"length", validation.BytesLength[bytesValue](2), ""},
		{"contains", validation.BytesContains[bytesValue](bytesValue{0xff}), ""},
		{"prefix", validation.BytesHasPrefix[bytesValue](bytesValue{0x00}), ""},
		{"suffix", validation.BytesHasSuffix[bytesValue](bytesValue{0xff}), ""},
		{"UTF8 rejects invalid", validation.BytesUTF8[bytesValue](), validation.CodeUTF8},
	} {
		t.Run(test.name, func(t *testing.T) {
			value := input
			if test.name == "equal nil and empty" {
				value = nil
			}
			if test.name == "not empty" {
				value = bytesValue{1}
			}
			if test.name == "UTF8 rejects invalid" {
				value = bytesValue{0xff}
			}
			err := test.rule(value)
			if test.want == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else {
				requirePrimitiveCodes(t, err, test.want)
			}
		})
	}
	if string(input) != string(copyOfInput) {
		t.Fatalf("byte input changed from %v to %v", copyOfInput, input)
	}
}

func TestStringConfigurationPanics(t *testing.T) {
	for _, test := range []struct {
		name string
		make func()
	}{
		{"negative exact length", func() { validation.ByteLength[string](-1) }},
		{"reversed byte range", func() { validation.ByteLengthBetween[string](2, 1) }},
		{"negative rune length", func() { validation.RuneMinLength[string](-1) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("expected configuration panic")
				}
			}()
			test.make()
		})
	}
}
