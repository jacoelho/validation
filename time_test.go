package validation_test

import (
	"testing"
	"time"

	"github.com/jacoelho/validation"
)

func date(year, month, day int) time.Time {
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

func TestTimeBefore(t *testing.T) {
	tests := []struct {
		name     string
		value    time.Time
		other    time.Time
		expected bool
	}{
		{
			name:     "before",
			value:    date(2023, 1, 1),
			other:    date(2023, 1, 2),
			expected: true,
		},
		{
			name:     "equal",
			value:    date(2023, 1, 1),
			other:    date(2023, 1, 1),
			expected: true,
		},
		{
			name:     "after",
			value:    date(2023, 1, 2),
			other:    date(2023, 1, 1),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := validation.TimeBefore(tt.other)
			err := rule(tt.value)
			if (err == nil) != tt.expected {
				t.Errorf("TimeBefore() = %v, want %v", err == nil, tt.expected)
			}
		})
	}
}

func TestTimeAfter(t *testing.T) {
	tests := []struct {
		name     string
		value    time.Time
		other    time.Time
		expected bool
	}{
		{
			name:     "after",
			value:    date(2023, 1, 2),
			other:    date(2023, 1, 1),
			expected: true,
		},
		{
			name:     "equal",
			value:    date(2023, 1, 1),
			other:    date(2023, 1, 1),
			expected: true,
		},
		{
			name:     "before",
			value:    date(2023, 1, 1),
			other:    date(2023, 1, 2),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := validation.TimeAfter(tt.other)
			err := rule(tt.value)
			if (err == nil) != tt.expected {
				t.Errorf("TimeAfter() = %v, want %v", err == nil, tt.expected)
			}
		})
	}
}

func TestTimeBetween(t *testing.T) {
	tests := []struct {
		name     string
		value    time.Time
		min      time.Time
		max      time.Time
		expected bool
	}{
		{
			name:     "between",
			value:    date(2023, 1, 2),
			min:      date(2023, 1, 1),
			max:      date(2023, 1, 3),
			expected: true,
		},
		{
			name:     "equal to min",
			value:    date(2023, 1, 1),
			min:      date(2023, 1, 1),
			max:      date(2023, 1, 3),
			expected: true,
		},
		{
			name:     "equal to max",
			value:    date(2023, 1, 3),
			min:      date(2023, 1, 1),
			max:      date(2023, 1, 3),
			expected: true,
		},
		{
			name:     "before min",
			value:    date(2022, 12, 31),
			min:      date(2023, 1, 1),
			max:      date(2023, 1, 3),
			expected: false,
		},
		{
			name:     "after max",
			value:    date(2023, 1, 4),
			min:      date(2023, 1, 1),
			max:      date(2023, 1, 3),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := validation.TimeBetween(tt.min, tt.max)
			err := rule(tt.value)
			if (err == nil) != tt.expected {
				t.Errorf("TimeBetween() = %v, want %v", err == nil, tt.expected)
			}
		})
	}
}
