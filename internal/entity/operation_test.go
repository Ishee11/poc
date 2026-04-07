package entity

import (
	"testing"
	"time"
)

func TestNewOperation(t *testing.T) {
	now := time.Now()

	tt := []struct {
		name          string
		opType        OperationType
		chips         int64
		wantErr       bool
		expectedError error
	}{
		{
			name:    "buy in success",
			opType:  OperationBuyIn,
			chips:   100,
			wantErr: false,
		},
		{
			name:    "cash out success",
			opType:  OperationCashOut,
			chips:   50,
			wantErr: false,
		},
		{
			name:          "chips zero",
			opType:        OperationBuyIn,
			chips:         0,
			wantErr:       true,
			expectedError: ErrInvalidChips,
		},
		{
			name:          "chips negative",
			opType:        OperationBuyIn,
			chips:         -10,
			wantErr:       true,
			expectedError: ErrInvalidChips,
		},
		{
			name:          "invalid operation type",
			opType:        OperationType("invalid"),
			chips:         100,
			wantErr:       true,
			expectedError: ErrInvalidOperationType,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			op, err := NewOperation(
				"op1",
				"s1",
				tc.opType,
				"p1",
				tc.chips,
				now,
			)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if err != tc.expectedError {
					t.Fatalf("expected %v, got %v", tc.expectedError, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if op == nil {
				t.Fatalf("operation is nil")
			}

			// проверяем что данные записались
			if op.id != "op1" {
				t.Fatalf("wrong id")
			}
			if op.sessionID != "s1" {
				t.Fatalf("wrong sessionID")
			}
			if op.playerID != "p1" {
				t.Fatalf("wrong playerID")
			}
			if op.chips != tc.chips {
				t.Fatalf("wrong chips")
			}
			if op.operationType != tc.opType {
				t.Fatalf("wrong operationType")
			}
			if !op.createdAt.Equal(now) {
				t.Fatalf("wrong createdAt")
			}
		})
	}
}
