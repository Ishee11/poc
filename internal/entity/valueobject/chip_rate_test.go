package valueobject

import "testing"

func TestChipRate(t *testing.T) {
	tt := []struct {
		name      string
		value     int64
		withPanic bool
	}{
		{name: "valid chip rate", value: 2, withPanic: false},
		{name: "invalid chip rate", value: 0, withPanic: true},
	}
	for _, tc := range tt {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if !tc.withPanic {
						t.Fatal("expected no panic, got panic")
					}
				} else {
					if tc.withPanic {
						t.Fatal("expected panic, got nothing")
					}
				}
			}()

			chipRate := NewChipRate(tc.value)
			if chipRate.value != tc.value {
				t.Errorf("expected chip rate %d, got %d", tc.value, chipRate.value)
			}
		})
	}
}
