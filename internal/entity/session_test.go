package entity

import (
	"errors"
	"testing"
)

func newActiveSession() *Session {
	s := NewSession("s1", 10)
	_ = s.StartSession()
	return s
}

func TestSession_Finish(t *testing.T) {
	t.Run("fails if chips remain", func(t *testing.T) {
		s := newActiveSession()

		_ = s.PlayerBuyIn("op1", "p1", 100)

		err := s.FinishSession()
		if !errors.Is(err, ErrPlayersStillInGame) {
			t.Fatalf("expected ErrPlayersStillInGame, got %v", err)
		}
	})

	t.Run("fails if unbalanced", func(t *testing.T) {
		s := newActiveSession()

		_ = s.PlayerBuyIn("op1", "p1", 100)
		_ = s.PlayerBuyIn("op2", "p2", 100)

		_ = s.PlayerCashOut("op3", "p1", 100)
		_ = s.PlayerCashOut("op4", "p2", 100)

		// баланс формально равен → нужно сломать
		money, _ := s.rate.ToMoney(100)
		s.players["p1"].totalMoneyCashedOut = s.players["p1"].totalMoneyCashedOut.Add(money)

		err := s.FinishSession()
		if !errors.Is(err, ErrUnbalancedSession) {
			t.Fatalf("expected ErrUnbalancedSession, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		s := newActiveSession()

		_ = s.PlayerBuyIn("op1", "p1", 100)
		_ = s.PlayerCashOut("op2", "p1", 100)

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

		_ = s.PlayerBuyIn("op1", "p1", 100)
		_ = s.PlayerCashOut("op2", "p1", 100)
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
