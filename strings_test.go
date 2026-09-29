package validation_test

import (
	"errors"
	"regexp"
	"testing"

	validation "github.com/jacoelho/validation"
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

func TestWhitespaceRules(t *testing.T) {
	type label string
	notBlank := validation.NotBlank[label]()
	trimmed := validation.Trimmed[label]()
	for _, value := range []label{"", " \t\n", "\u00a0"} {
		requirePrimitiveCodes(t, notBlank(value), validation.CodeNotBlank)
	}
	for _, value := range []label{"name", "name with spaces", "  name  "} {
		if err := notBlank(value); err != nil {
			t.Fatalf("%q should not be blank: %v", value, err)
		}
	}
	for _, value := range []label{"", "name", "name with spaces"} {
		if err := trimmed(value); err != nil {
			t.Fatalf("%q should be trimmed: %v", value, err)
		}
	}
	for _, value := range []label{" name", "name\u00a0", "\tname\n"} {
		requirePrimitiveCodes(t, trimmed(value), validation.CodeTrimmed)
	}
	invalidUTF8 := label(string([]byte{0xff}))
	if err := notBlank(invalidUTF8); err != nil {
		t.Fatalf("UTF-8 validation should remain separate: %v", err)
	}
	requirePrimitiveCodes(t, validation.UTF8[label]()(invalidUTF8), validation.CodeUTF8)
	combined := validation.All(
		validation.NotBlank[string](),
		validation.Trimmed[string](),
		validation.Match[string](regexp.MustCompile(`^[A-Z]+$`)),
	)
	requirePrimitiveCodes(t, combined(" "), validation.CodeNotBlank, validation.CodeTrimmed, validation.CodeMatch)
}

func TestMatch(t *testing.T) {
	type label string
	containsDigits := validation.Match[label](regexp.MustCompile(`[0-9]+`))
	if err := containsDigits("item42end"); err != nil {
		t.Fatalf("substring should match: %v", err)
	}
	requirePrimitiveCodes(t, containsDigits("item"), validation.CodeMatch)
	wholeDigits := validation.Match[label](regexp.MustCompile(`^[0-9]+$`))
	requirePrimitiveCodes(t, wholeDigits("item42"), validation.CodeMatch)
	if err := wholeDigits("42"); err != nil {
		t.Fatalf("anchored expression should match entire input: %v", err)
	}
	if err := validation.Match[string](regexp.MustCompile(`^$`))(""); err != nil {
		t.Fatalf("empty input should follow the pattern: %v", err)
	}
	defer func() {
		if got, ok := recover().(*validation.ConfigurationError); !ok || got.Constructor() != "Match" {
			t.Fatalf("nil pattern panic = %#v", got)
		}
	}()
	validation.Match[string](nil)
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
