package entity

import (
	"testing"
	"time"

	"github.com/ishee11/poc/internal/entity/valueobject"
)

func mustRate(t *testing.T) valueobject.ChipRate {
	r, err := valueobject.NewChipRate(2)
	if err != nil {
		t.Fatalf("failed to create chip rate: %v", err)
	}
	return r
}

func newSession(t *testing.T) *Session {
	return NewSession("s1", mustRate(t), 2, CurrencyRUB, time.Now())
}

func TestSession_BuyIn(t *testing.T) {
	tt := []struct {
		name    string
		chips   int64
		wantErr bool
		status  Status
	}{
		{"valid buyin", 1000, false, StatusActive},
		{"zero chips", 0, true, StatusActive},
		{"negative chips", -1000, true, StatusActive},
		{"invalid status", 1000, true, StatusFinished},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			s := newSession(t)
			s.status = tc.status

			before := s.TotalBuyIn()

			err := s.BuyIn(tc.chips)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if s.TotalBuyIn() != before {
					t.Fatal("state changed on error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if s.TotalBuyIn() != before+tc.chips {
				t.Fatalf("expected buyin %d, got %d", tc.chips, s.TotalBuyIn())
			}

			if s.TotalChips() != s.TotalBuyIn() {
				t.Fatal("invalid total chips after buyin")
			}
		})
	}
}

func TestSession_CashOut(t *testing.T) {
	tt := []struct {
		name         string
		chipsCashOut int64
		totalBuyIn   int64
		wantErr      bool
		status       Status
	}{
		{"valid CashOut", 1000, 1000, false, StatusActive},
		{"negative chips", -1000, 1000, true, StatusActive},
		{"zero chips", 0, 1000, true, StatusActive},
		{"invalid status", 1000, 1000, true, StatusFinished},
		{"partial cashout", 300, 1000, false, StatusActive},
		{"cashout exceeds table chips (allowed in entity)", 1000, 500, false, StatusActive},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			s := newSession(t)
			s.status = tc.status
			s.totalBuyInCache = tc.totalBuyIn

			before := s.TotalCashOut()

			err := s.CashOut(tc.chipsCashOut)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if s.TotalCashOut() != before {
					t.Fatal("state changed on error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if s.TotalCashOut() != before+tc.chipsCashOut {
				t.Fatal("cashout not applied")
			}

			if s.TotalChips() != tc.totalBuyIn-tc.chipsCashOut {
				t.Fatal("invalid total chips")
			}
		})
	}
}

func TestSession_Finish(t *testing.T) {
	tests := []struct {
		name    string
		status  Status
		wantErr bool
	}{
		{"valid finish", StatusActive, false},
		{"not active", StatusFinished, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := newSession(t)
			s.status = tc.status

			err := s.Finish(time.Now())

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if s.Status() != tc.status {
					t.Fatal("status changed on error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if s.Status() != StatusFinished {
				t.Fatalf("expected status finished, got %s", s.Status())
			}
		})
	}
}

func TestSession_BuyIn_Multiple(t *testing.T) {
	s := newSession(t)

	if err := s.BuyIn(100); err != nil {
		t.Fatal(err)
	}
	if err := s.BuyIn(200); err != nil {
		t.Fatal(err)
	}

	if s.TotalBuyIn() != 300 {
		t.Fatalf("expected 300, got %d", s.TotalBuyIn())
	}
}

func TestSession_CashOut_Multiple(t *testing.T) {
	s := newSession(t)

	if err := s.BuyIn(1000); err != nil {
		t.Fatal(err)
	}

	if err := s.CashOut(400); err != nil {
		t.Fatal(err)
	}
	if err := s.CashOut(600); err != nil {
		t.Fatal(err)
	}

	if s.TotalCashOut() != 1000 {
		t.Fatalf("expected 1000, got %d", s.TotalCashOut())
	}

	if s.TotalChips() != 0 {
		t.Fatal("expected 0 chips")
	}
}

func TestSession_ReverseBuyIn(t *testing.T) {
	tt := []struct {
		name       string
		chips      int64
		totalBuyIn int64
		wantErr    bool
		status     Status
	}{
		{"valid reverse buyin", 400, 1000, false, StatusActive},
		{"zero chips", 0, 1000, true, StatusActive},
		{"negative chips", -100, 1000, true, StatusActive},
		{"more than total buyin", 1200, 1000, true, StatusActive},
		{"invalid status", 400, 1000, true, StatusFinished},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			s := newSession(t)
			s.status = tc.status
			s.totalBuyInCache = tc.totalBuyIn

			before := s.TotalBuyIn()

			err := s.ReverseBuyIn(tc.chips)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if s.TotalBuyIn() != before {
					t.Fatal("state changed on error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if s.TotalBuyIn() != before-tc.chips {
				t.Fatalf("expected buyin %d, got %d", before-tc.chips, s.TotalBuyIn())
			}
		})
	}
}

func TestSession_ReverseCashOut(t *testing.T) {
	tt := []struct {
		name         string
		chips        int64
		totalBuyIn   int64
		totalCashOut int64
		wantErr      bool
		status       Status
	}{
		{"valid reverse cashout", 400, 1000, 700, false, StatusActive},
		{"zero chips", 0, 1000, 700, true, StatusActive},
		{"negative chips", -100, 1000, 700, true, StatusActive},
		{"more than total cashout", 800, 1000, 700, true, StatusActive},
		{"invalid status", 400, 1000, 700, true, StatusFinished},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			s := newSession(t)
			s.status = tc.status
			s.totalBuyInCache = tc.totalBuyIn
			s.totalCashOutCache = tc.totalCashOut

			beforeCashOut := s.TotalCashOut()

			err := s.ReverseCashOut(tc.chips)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if s.TotalCashOut() != beforeCashOut {
					t.Fatal("state changed on error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if s.TotalCashOut() != beforeCashOut-tc.chips {
				t.Fatalf("expected cashout %d, got %d", beforeCashOut-tc.chips, s.TotalCashOut())
			}

			if s.TotalChips() != tc.totalBuyIn-s.TotalCashOut() {
				t.Fatal("invalid total chips")
			}
		})
	}
}

func TestSession_Flow(t *testing.T) {
	s := newSession(t)

	if err := s.BuyIn(1000); err != nil {
		t.Fatal(err)
	}
	if err := s.CashOut(1000); err != nil {
		t.Fatal(err)
	}

	if err := s.Finish(time.Now()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if s.Status() != StatusFinished {
		t.Fatal("session not finished")
	}
}

func TestSession_AfterFinish_Operations(t *testing.T) {
	s := newSession(t)

	if err := s.BuyIn(1000); err != nil {
		t.Fatal(err)
	}
	if err := s.CashOut(1000); err != nil {
		t.Fatal(err)
	}
	if err := s.Finish(time.Now()); err != nil {
		t.Fatal(err)
	}

	if err := s.BuyIn(100); err == nil {
		t.Fatal("expected error on buyin after finish")
	}

	if err := s.CashOut(100); err == nil {
		t.Fatal("expected error on cashout after finish")
	}
}

func TestSession_CashOut_CanGoNegative(t *testing.T) {
	s := newSession(t)

	if err := s.BuyIn(500); err != nil {
		t.Fatal(err)
	}
	if err := s.CashOut(1000); err != nil {
		t.Fatal(err)
	}

	if s.TotalChips() != -500 {
		t.Fatalf("expected -500, got %d", s.TotalChips())
	}
}

func TestSession_TotalChips_Consistency(t *testing.T) {
	s := newSession(t)

	if err := s.BuyIn(1000); err != nil {
		t.Fatal(err)
	}
	if err := s.CashOut(300); err != nil {
		t.Fatal(err)
	}

	if s.TotalChips() != 700 {
		t.Fatalf("expected 700, got %d", s.TotalChips())
	}
}
