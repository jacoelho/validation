package validation_test

import (
	"errors"
	"testing"
	"time"

	validation "github.com/jacoelho/validation"
)

func primitiveFixedTime(hour int) time.Time {
	return time.Date(2026, time.September, 29, hour, 0, 0, 0, time.UTC)
}

func TestTimeLayout(t *testing.T) {
	type timestamp string
	rfc3339 := validation.Time[timestamp](time.RFC3339)
	for _, value := range []timestamp{"2026-09-29T08:30:00Z", "2026-09-29T09:30:00.123+01:00"} {
		if err := rfc3339(value); err != nil {
			t.Fatalf("%q should parse as RFC 3339: %v", value, err)
		}
	}
	for _, value := range []timestamp{"", "2026-09-29", "2026-09-29T08:30:00", "2026-02-30T08:30:00Z"} {
		requirePrimitiveCodes(t, rfc3339(value), validation.CodeTime)
	}
	if err := validation.Time[string]("2006-01-02")("2026-09-29"); err != nil {
		t.Fatalf("date-only layout should parse: %v", err)
	}
	if err := validation.Time[string]("15:04")("08:30"); err != nil {
		t.Fatalf("time-only layout should parse: %v", err)
	}
	requirePrimitiveCodes(t, validation.Time[string]("2006-01-02")("08:30"), validation.CodeTime)
	defer func() {
		if got, ok := recover().(*validation.ConfigurationError); !ok || got.Constructor() != "Time" {
			t.Fatalf("empty layout panic = %#v", got)
		}
	}()
	validation.Time[string]("")
}

func TestTimeRelationsAtEqualInstant(t *testing.T) {
	boundary := primitiveFixedTime(8)
	value := time.Date(2026, time.September, 29, 9, 0, 0, 0, time.FixedZone("BST", 3600))
	for _, test := range []struct {
		name string
		rule validation.Rule[time.Time]
		want validation.Code
	}{
		{"before is strict", validation.TimeBefore(boundary), validation.CodeBefore},
		{"before or equal includes boundary", validation.TimeBeforeOrEqual(boundary), ""},
		{"after is strict", validation.TimeAfter(boundary), validation.CodeAfter},
		{"after or equal includes boundary", validation.TimeAfterOrEqual(boundary), ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := test.rule(value)
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

func TestTimeBetweenAndNotZero(t *testing.T) {
	min := primitiveFixedTime(8)
	max := primitiveFixedTime(10)
	rule := validation.TimeBetween(min, max)
	for _, value := range []time.Time{min, primitiveFixedTime(9), max} {
		if err := rule(value); err != nil {
			t.Fatalf("inclusive boundary %v rejected: %v", value, err)
		}
	}
	requirePrimitiveCodes(t, rule(primitiveFixedTime(7)), validation.CodeBetween)
	requirePrimitiveCodes(t, rule(primitiveFixedTime(11)), validation.CodeBetween)

	var bounds *validation.BoundsError[time.Time]
	if !errors.As(rule(primitiveFixedTime(11)), &bounds) {
		t.Fatal("TimeBetween failure did not retain typed bounds")
	}
	lower, lowerInclusive, lowerPresent := bounds.Lower()
	upper, upperInclusive, upperPresent := bounds.Upper()
	if !lowerPresent || !upperPresent || !lowerInclusive || !upperInclusive || !lower.Equal(min) || !upper.Equal(max) {
		t.Fatalf("bounds = (%v, %v, %v), (%v, %v, %v)", lower, lowerInclusive, lowerPresent, upper, upperInclusive, upperPresent)
	}

	if err := validation.TimeNotZero()(time.Time{}); err == nil {
		t.Fatal("zero time unexpectedly passed")
	}
	if err := validation.TimeNotZero()(primitiveFixedTime(9)); err != nil {
		t.Fatalf("nonzero time rejected: %v", err)
	}
}

func TestTimeRangeConfigurationPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected configuration panic")
		}
	}()
	validation.TimeBetween(primitiveFixedTime(10), primitiveFixedTime(8))
}
