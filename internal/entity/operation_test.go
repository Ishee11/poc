package entity

import (
	"errors"
	"testing"
	"time"
)

func TestNewOperation(t *testing.T) {
	now := time.Now()

	tt := []struct {
		name          string
		requestID     string
		opType        OperationType
		chips         int64
		wantErr       bool
		expectedError error
	}{
		{
			name:      "buy in success",
			requestID: "req1",
			opType:    OperationBuyIn,
			chips:     100,
			wantErr:   false,
		},
		{
			name:      "cash out success",
			requestID: "req1",
			opType:    OperationCashOut,
			chips:     50,
			wantErr:   false,
		},
		{
			name:          "chips zero",
			requestID:     "req1",
			opType:        OperationBuyIn,
			chips:         0,
			wantErr:       true,
			expectedError: ErrInvalidChips,
		},
		{
			name:          "chips negative",
			requestID:     "req1",
			opType:        OperationBuyIn,
			chips:         -10,
			wantErr:       true,
			expectedError: ErrInvalidChips,
		},
		{
			name:          "invalid operation type",
			requestID:     "req1",
			opType:        OperationType("invalid"),
			chips:         100,
			wantErr:       true,
			expectedError: ErrInvalidOperationType,
		},
		{
			name:          "reversal cannot be created via NewOperation",
			requestID:     "req1",
			opType:        OperationReversal,
			chips:         100,
			wantErr:       true,
			expectedError: ErrInvalidOperationType,
		},
		{
			name:          "empty requestID",
			requestID:     "",
			opType:        OperationBuyIn,
			chips:         100,
			wantErr:       true,
			expectedError: ErrInvalidRequestID,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			op, err := NewOperation(
				"op1",
				tc.requestID,
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
				if !errors.Is(err, tc.expectedError) {
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

			if op.ID() != "op1" {
				t.Fatalf("wrong id")
			}
			if op.SessionID() != "s1" {
				t.Fatalf("wrong sessionID")
			}
			if op.PlayerID() != "p1" {
				t.Fatalf("wrong playerID")
			}
			if op.Chips() != tc.chips {
				t.Fatalf("wrong chips")
			}
			if op.Type() != tc.opType {
				t.Fatalf("wrong operationType")
			}
			if !op.CreatedAt().Equal(now) {
				t.Fatalf("wrong createdAt")
			}
			if op.ReferenceID() != nil {
				t.Fatalf("referenceID should be nil")
			}
			if op.RequestID() != tc.requestID {
				t.Fatalf("wrong requestID")
			}
		})
	}
}

func TestNewReversalOperation(t *testing.T) {
	now := time.Now()

	refID := OperationID("ref1")

	tt := []struct {
		name          string
		requestID     string
		chips         int64
		referenceID   OperationID
		wantErr       bool
		expectedError error
	}{
		{
			name:        "success",
			requestID:   "req1",
			chips:       100,
			referenceID: refID,
			wantErr:     false,
		},
		{
			name:          "chips zero",
			requestID:     "req1",
			chips:         0,
			referenceID:   refID,
			wantErr:       true,
			expectedError: ErrInvalidChips,
		},
		{
			name:          "empty referenceID",
			requestID:     "req1",
			chips:         100,
			referenceID:   "",
			wantErr:       true,
			expectedError: ErrInvalidReference,
		},
		{
			name:          "empty requestID",
			requestID:     "",
			chips:         100,
			referenceID:   refID,
			wantErr:       true,
			expectedError: ErrInvalidRequestID,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			op, err := NewReversalOperation(
				"op1",
				tc.requestID,
				"s1",
				"p1",
				tc.chips,
				tc.referenceID,
				now,
			)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !errors.Is(err, tc.expectedError) {
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

			if op.ID() != "op1" {
				t.Fatalf("wrong id")
			}
			if op.SessionID() != "s1" {
				t.Fatalf("wrong sessionID")
			}
			if op.PlayerID() != "p1" {
				t.Fatalf("wrong playerID")
			}
			if !op.CreatedAt().Equal(now) {
				t.Fatalf("wrong createdAt")
			}
			if op.Type() != OperationReversal {
				t.Fatalf("wrong operation type")
			}
			if op.RequestID() != tc.requestID {
				t.Fatalf("wrong requestID")
			}

			if op.ReferenceID() == nil {
				t.Fatalf("referenceID is nil")
			}
			if *op.ReferenceID() != tc.referenceID {
				t.Fatalf("wrong referenceID")
			}
			if op.Chips() != tc.chips {
				t.Fatalf("wrong chips")
			}
		})
	}
}
