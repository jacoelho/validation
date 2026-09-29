package validation_test

import (
	"errors"
	"testing"
	"time"

	validation "github.com/jacoelho/validation/v2"
)

func primitiveFixedTime(hour int) time.Time {
	return time.Date(2026, time.September, 29, hour, 0, 0, 0, time.UTC)
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
