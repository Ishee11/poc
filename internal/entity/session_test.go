package entity

import (
	"testing"

	"github.com/ishee11/poc/internal/entity/valueobject"
)

func TestSession_BuyIn(t *testing.T) {
	chipRate := valueobject.NewChipRate(2)

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
			s := NewSession("s1", chipRate)
			s.status = tc.status

			before := s.TotalBuyIn()

			err := s.BuyIn(tc.chips)

			// проверка: ожидаемая ошибка / её отсутствие
			if !tc.wantErr && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
			if tc.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}

			// ДОБАВЛЕНО: при ошибке состояние не должно меняться
			if tc.wantErr {
				if s.TotalBuyIn() != before {
					t.Fatal("state changed on error")
				}
				return
			}

			// проверка: при успешной операции состояние должно измениться
			if s.TotalBuyIn() != before+tc.chips {
				t.Fatalf("expected buyin %d, got %d", tc.chips, s.TotalBuyIn())
			}

			// ДОБАВЛЕНО: проверка инварианта
			if s.TotalChips() != s.TotalBuyIn() {
				t.Fatal("invalid total chips after buyin")
			}
		})
	}
}

func TestSession_CashOut(t *testing.T) {
	chipRate := valueobject.NewChipRate(2)

	tt := []struct {
		name         string
		chipsCashOut int64
		totalBuyIn   int64
		wantErr      bool
		status       Status
	}{
		{"valid CashOut", 1000, 1000, false, StatusActive},
		{"not enough chips", 1000, 500, true, StatusActive},
		{"negative chips", -1000, 1000, true, StatusActive},
		{"zero chips", 0, 1000, true, StatusActive}, // ДОБАВЛЕНО
		{"invalid status", 1000, 1000, true, StatusFinished},
		{"partial cashout", 300, 1000, false, StatusActive},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			s := NewSession("s1", chipRate)
			s.status = tc.status
			s.totalBuyIn = tc.totalBuyIn

			before := s.TotalCashOut()

			err := s.CashOut(tc.chipsCashOut)

			// проверка: ожидаемая ошибка и отсутствие изменений состояния
			// мы ОЖИДАЕМ ошибку
			if tc.wantErr {
				// если ошибка отсутствует, то тест не пройден
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				// если ошибка присутствует, то состояние не должно измениться
				// если состояние изменилось, то тест не пройден
				if s.TotalCashOut() != before {
					t.Fatal("state changed on error")
				}
				// если ошибка присутствует, то тест пройден
				// нет смысла выполнять код ниже
				return
			}

			// проверка: успешное выполнение без ошибки
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// проверка: состояние изменилось корректно
			if s.TotalCashOut() != before+tc.chipsCashOut {
				t.Fatal("cashout not applied")
			}
			// проверка инварианта
			if s.TotalChips() != tc.totalBuyIn-tc.chipsCashOut {
				t.Fatal("invalid total chips")
			}
		})
	}
}

func TestSession_Finish(t *testing.T) {
	chipRate := valueobject.NewChipRate(2)

	tests := []struct {
		name         string
		status       Status
		totalBuyIn   int64
		totalCashOut int64
		wantErr      bool
	}{
		{"valid finish", StatusActive, 1000, 1000, false},
		{"not active", StatusFinished, 1000, 1000, true},
		{"not settled", StatusActive, 1000, 500, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := NewSession("s1", chipRate)
			s.status = tc.status
			s.totalBuyIn = tc.totalBuyIn
			s.totalCashOut = tc.totalCashOut

			err := s.Finish()

			// CASE: ожидаем ошибку
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				// состояние не должно измениться
				if s.Status() != tc.status {
					t.Fatal("status changed on error")
				}
				return
			}

			// CASE: ожидаем успех
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// состояние должно измениться
			if s.Status() != StatusFinished {
				t.Fatalf("expected status finished, got %s", s.Status())
			}
		})
	}
}

func TestSession_BuyIn_Multiple(t *testing.T) {
	s := NewSession("s1", valueobject.NewChipRate(2))

	_ = s.BuyIn(100)
	_ = s.BuyIn(200)

	if s.TotalBuyIn() != 300 {
		t.Fatalf("expected 300, got %d", s.TotalBuyIn())
	}
}

func TestSession_CashOut_Multiple(t *testing.T) {
	s := NewSession("s1", valueobject.NewChipRate(2))

	_ = s.BuyIn(1000)

	_ = s.CashOut(400)
	_ = s.CashOut(600)

	if s.TotalCashOut() != 1000 {
		t.Fatalf("expected 1000, got %d", s.TotalCashOut())
	}

	if s.TotalChips() != 0 {
		t.Fatal("expected 0 chips")
	}
}

func TestSession_Flow(t *testing.T) {
	s := NewSession("s1", valueobject.NewChipRate(2))

	_ = s.BuyIn(1000)
	_ = s.CashOut(1000)

	err := s.Finish()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if s.Status() != StatusFinished {
		t.Fatal("session not finished")
	}
}

// нельзя завершить если остались фишки (через flow)
func TestSession_Flow_NotSettled(t *testing.T) {
	s := NewSession("s1", valueobject.NewChipRate(2))

	_ = s.BuyIn(1000)
	_ = s.CashOut(500)

	err := s.Finish()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// после Finish операции запрещены
func TestSession_AfterFinish_Operations(t *testing.T) {
	s := NewSession("s1", valueobject.NewChipRate(2))

	_ = s.BuyIn(1000)
	_ = s.CashOut(1000)
	_ = s.Finish()

	if err := s.BuyIn(100); err == nil {
		t.Fatal("expected error on buyin after finish")
	}

	if err := s.CashOut(100); err == nil {
		t.Fatal("expected error on cashout after finish")
	}
}
