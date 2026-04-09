package valueobject

import (
	"errors"
	"testing"
)

func TestNewChipRate(t *testing.T) {
	tt := []struct {
		name    string
		value   int64
		wantErr error
	}{
		{
			name:  "valid chip rate",
			value: 2,
		},
		{
			name:    "zero chip rate",
			value:   0,
			wantErr: ErrInvalidChips,
		},
		{
			name:    "negative chip rate",
			value:   -1,
			wantErr: ErrInvalidChips,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			rate, err := NewChipRate(tc.value)

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected error %v, got %v", tc.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if rate.Value() != tc.value {
				t.Fatalf("expected %d, got %d", tc.value, rate.Value())
			}
		})
	}
}
