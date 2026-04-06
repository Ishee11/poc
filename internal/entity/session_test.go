package entity

import (
	"errors"
	"testing"

	"github.com/ishee11/poc/internal/entity/valueobject"
)

func newActiveSession() *Session {
	s := NewSession("s1", 2) // 2 chips за 1 money
	_ = s.StartSession()
	return s
}

func mustMoney(t *testing.T, amount int64) valueobject.Money {
	m, err := valueobject.NewMoney(amount)
	if err != nil {
		t.Fatalf("invalid money: %v", err)
	}
	return m
}

func TestSession_Finish(t *testing.T) {
	t.Run("fails if chips remain", func(t *testing.T) {
		s := newActiveSession()

		_ = s.PlayerBuyIn("op1", "p1", mustMoney(t, 100)) // → 200 chips

		err := s.FinishSession()
		if !errors.Is(err, ErrPlayersStillInGame) {
			t.Fatalf("expected ErrPlayersStillInGame, got %v", err)
		}
	})

	t.Run("fails if unbalanced", func(t *testing.T) {
		s := newActiveSession()

		// p1: +200 chips
		_ = s.PlayerBuyIn("op1", "p1", mustMoney(t, 100))

		// p2: +200 chips
		_ = s.PlayerBuyIn("op2", "p2", mustMoney(t, 100))

		// корректный cashout (всё обнулили)
		_ = s.PlayerCashOut("op3", "p1", 200)
		_ = s.PlayerCashOut("op4", "p2", 200)

		// ломаем баланс вручную (доменный инвариант)
		extra := mustMoney(t, 50)
		s.players["p1"].totalMoneyCashedOut =
			s.players["p1"].totalMoneyCashedOut.Add(extra)

		err := s.FinishSession()
		if !errors.Is(err, ErrUnbalancedSession) {
			t.Fatalf("expected ErrUnbalancedSession, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		s := newActiveSession()

		_ = s.PlayerBuyIn("op1", "p1", mustMoney(t, 100)) // 200 chips
		_ = s.PlayerCashOut("op2", "p1", 200)             // обратно 100 money

		if err := s.FinishSession(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if s.Status() != StatusFinished {
			t.Fatalf("expected finished, got %s", s.Status())
		}
	})
}

func TestSession_PlayerResult(t *testing.T) {
	t.Run("fails if not finished", func(t *testing.T) {
		s := newActiveSession()

		_, err := s.PlayerResult("p1")
		if !errors.Is(err, ErrSessionNotFinished) {
			t.Fatalf("expected ErrSessionNotFinished, got %v", err)
		}
	})

	t.Run("player not found", func(t *testing.T) {
		s := newActiveSession()
		_ = s.FinishSession()

		_, err := s.PlayerResult("unknown")
		if !errors.Is(err, ErrPlayerNotFound) {
			t.Fatalf("expected ErrPlayerNotFound, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		s := newActiveSession()

		// p1: внес 100 → получил 200 chips
		_ = s.PlayerBuyIn("op1", "p1", mustMoney(t, 100))

		// вывел все 200 chips → получил обратно 100
		_ = s.PlayerCashOut("op2", "p1", 200)

		_ = s.FinishSession()

		res, err := s.PlayerResult("p1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if res.Amount() != 0 {
			t.Fatalf("expected 0, got %d", res.Amount())
		}
	})
}
