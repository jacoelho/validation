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
	referenceTime := date(2023, 1, 2)
	rule := validation.TimeBefore(referenceTime)

	tests := []struct {
		name    string
		value   time.Time
		wantErr bool
		errCode string
	}{
		{
			name:    "time before reference should pass",
			value:   date(2023, 1, 1),
			wantErr: false,
		},
		{
			name:    "time equal to reference should fail",
			value:   date(2023, 1, 2),
			wantErr: true,
			errCode: "before",
		},
		{
			name:    "time after reference should fail",
			value:   date(2023, 1, 3),
			wantErr: true,
			errCode: "before",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else if err.Code != tt.errCode {
					t.Errorf("expected error code %q, got %q", tt.errCode, err.Code)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got %v", err)
				}
			}
		})
	}
}

func TestTimeBeforeOrEqual(t *testing.T) {
	referenceTime := date(2023, 1, 2)
	rule := validation.TimeBeforeOrEqual(referenceTime)

	tests := []struct {
		name    string
		value   time.Time
		wantErr bool
		errCode string
	}{
		{
			name:    "time before reference should pass",
			value:   date(2023, 1, 1),
			wantErr: false,
		},
		{
			name:    "time equal to reference should pass",
			value:   date(2023, 1, 2),
			wantErr: false,
		},
		{
			name:    "time after reference should fail",
			value:   date(2023, 1, 3),
			wantErr: true,
			errCode: "before",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else if err.Code != tt.errCode {
					t.Errorf("expected error code %q, got %q", tt.errCode, err.Code)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got %v", err)
				}
			}
		})
	}
}

func TestTimeAfter(t *testing.T) {
	referenceTime := date(2023, 1, 2)
	rule := validation.TimeAfter(referenceTime)

	tests := []struct {
		name    string
		value   time.Time
		wantErr bool
		errCode string
	}{
		{
			name:    "time after reference should pass",
			value:   date(2023, 1, 3),
			wantErr: false,
		},
		{
			name:    "time equal to reference should fail",
			value:   date(2023, 1, 2),
			wantErr: true,
			errCode: "after",
		},
		{
			name:    "time before reference should fail",
			value:   date(2023, 1, 1),
			wantErr: true,
			errCode: "after",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else if err.Code != tt.errCode {
					t.Errorf("expected error code %q, got %q", tt.errCode, err.Code)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got %v", err)
				}
			}
		})
	}
}

func TestTimeAfterOrEqual(t *testing.T) {
	referenceTime := date(2023, 1, 2)
	rule := validation.TimeAfterOrEqual(referenceTime)

	tests := []struct {
		name    string
		value   time.Time
		wantErr bool
		errCode string
	}{
		{
			name:    "time after reference should pass",
			value:   date(2023, 1, 3),
			wantErr: false,
		},
		{
			name:    "time equal to reference should pass",
			value:   date(2023, 1, 2),
			wantErr: false,
		},
		{
			name:    "time before reference should fail",
			value:   date(2023, 1, 1),
			wantErr: true,
			errCode: "after",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else if err.Code != tt.errCode {
					t.Errorf("expected error code %q, got %q", tt.errCode, err.Code)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got %v", err)
				}
			}
		})
	}
}

func TestTimeBetween(t *testing.T) {
	minTime := date(2023, 1, 1)
	maxTime := date(2023, 1, 3)
	rule := validation.TimeBetween(minTime, maxTime)

	tests := []struct {
		name    string
		value   time.Time
		wantErr bool
		errCode string
	}{
		{
			name:    "time within range should pass",
			value:   date(2023, 1, 2),
			wantErr: false,
		},
		{
			name:    "time equal to minimum should pass",
			value:   date(2023, 1, 1),
			wantErr: false,
		},
		{
			name:    "time equal to maximum should pass",
			value:   date(2023, 1, 3),
			wantErr: false,
		},
		{
			name:    "time before minimum should fail",
			value:   date(2022, 12, 31),
			wantErr: true,
			errCode: "between",
		},
		{
			name:    "time after maximum should fail",
			value:   date(2023, 1, 4),
			wantErr: true,
			errCode: "between",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else if err.Code != tt.errCode {
					t.Errorf("expected error code %q, got %q", tt.errCode, err.Code)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got %v", err)
				}
			}
		})
	}
}

func TestTimeErrorParams(t *testing.T) {
	referenceTime := date(2023, 1, 2)

	t.Run("TimeBefore error params", func(t *testing.T) {
		rule := validation.TimeBefore(referenceTime)
		err := rule(date(2023, 1, 3))

		if err == nil {
			t.Fatal("expected error but got nil")
		}

		if err.Code != "before" {
			t.Errorf("expected code 'before', got %q", err.Code)
		}

		if err.Params["value"] != referenceTime {
			t.Errorf("expected value param to be %v, got %v", referenceTime, err.Params["value"])
		}
	})

	t.Run("TimeAfter error params", func(t *testing.T) {
		rule := validation.TimeAfter(referenceTime)
		err := rule(date(2023, 1, 1))

		if err == nil {
			t.Fatal("expected error but got nil")
		}

		if err.Code != "after" {
			t.Errorf("expected code 'after', got %q", err.Code)
		}

		if err.Params["value"] != referenceTime {
			t.Errorf("expected value param to be %v, got %v", referenceTime, err.Params["value"])
		}
	})

	t.Run("TimeBetween error params", func(t *testing.T) {
		minTime := date(2023, 1, 1)
		maxTime := date(2023, 1, 3)
		rule := validation.TimeBetween(minTime, maxTime)
		testTime := date(2023, 1, 5)
		err := rule(testTime)

		if err == nil {
			t.Fatal("expected error but got nil")
		}

		if err.Code != "between" {
			t.Errorf("expected code 'between', got %q", err.Code)
		}

		if err.Params["min"] != minTime {
			t.Errorf("expected min param to be %v, got %v", minTime, err.Params["min"])
		}

		if err.Params["max"] != maxTime {
			t.Errorf("expected max param to be %v, got %v", maxTime, err.Params["max"])
		}

		if err.Params["value"] != testTime {
			t.Errorf("expected value param to be %v, got %v", testTime, err.Params["value"])
		}
	})
}

func TestTimeEdgeCases(t *testing.T) {
	t.Run("zero time validation", func(t *testing.T) {
		var zeroTime time.Time
		futureTime := time.Now().Add(24 * time.Hour)

		rule := validation.TimeBefore(futureTime)
		err := rule(zeroTime)

		if err != nil {
			t.Errorf("expected zero time to pass before future time, got error: %v", err)
		}
	})

	t.Run("same time validation", func(t *testing.T) {
		sameTime := time.Now()

		beforeRule := validation.TimeBefore(sameTime)
		if beforeRule(sameTime) == nil {
			t.Error("expected TimeBefore to fail for same time")
		}

		afterRule := validation.TimeAfter(sameTime)
		if afterRule(sameTime) == nil {
			t.Error("expected TimeAfter to fail for same time")
		}
	})

	t.Run("microsecond precision", func(t *testing.T) {
		baseTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
		microsecondLater := baseTime.Add(time.Microsecond)

		rule := validation.TimeBefore(baseTime)
		err := rule(microsecondLater)

		if err == nil {
			t.Error("expected microsecond later time to fail TimeBefore")
		}
	})
}
