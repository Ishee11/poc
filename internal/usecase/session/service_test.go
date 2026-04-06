package session

import (
	"context"
	"errors"
	"testing"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/repo/memory"
)

func newUseCase() *SessionUseCase {
	repo := memory.NewSessionRepository()
	return NewUseCase(repo)
}

func TestSessionUseCase_FullFlow(t *testing.T) {
	uc := newUseCase()
	ctx := context.Background()

	// create
	err := uc.CreateSession(ctx, CreateSessionCommand{
		SessionID: "s1",
		Rate:      10,
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// start
	if err := uc.StartSession(ctx, StartSessionCommand{SessionID: "s1"}); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	// buyin
	if err := uc.BuyIn(ctx, BuyInCommand{
		SessionID:   "s1",
		OperationID: "op1",
		PlayerID:    "p1",
		Chips:       100,
	}); err != nil {
		t.Fatalf("buyin failed: %v", err)
	}

	// cashout
	if err := uc.CashOut(ctx, CashOutCommand{
		SessionID:   "s1",
		OperationID: "op2",
		PlayerID:    "p1",
		Chips:       100,
	}); err != nil {
		t.Fatalf("cashout failed: %v", err)
	}

	// close
	if err := uc.CloseSession(ctx, CloseSessionCommand{SessionID: "s1"}); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	// result
	res, err := uc.GetResult(ctx, GetResultQuery{
		SessionID: "s1",
		PlayerID:  "p1",
	})
	if err != nil {
		t.Fatalf("get result failed: %v", err)
	}

	if res.Amount() != 0 {
		t.Fatalf("expected 0, got %d", res.Amount())
	}
}

func TestSessionUseCase_Idempotency(t *testing.T) {
	uc := newUseCase()
	ctx := context.Background()

	_ = uc.CreateSession(ctx, CreateSessionCommand{SessionID: "s1", Rate: 10})
	_ = uc.StartSession(ctx, StartSessionCommand{SessionID: "s1"})

	cmd := BuyInCommand{
		SessionID:   "s1",
		OperationID: "op1",
		PlayerID:    "p1",
		Chips:       100,
	}

	// первый вызов
	if err := uc.BuyIn(ctx, cmd); err != nil {
		t.Fatalf("buyin failed: %v", err)
	}

	// повтор
	if err := uc.BuyIn(ctx, cmd); err != nil {
		t.Fatalf("buyin second call failed: %v", err)
	}

	// проверяем через result
	_ = uc.CashOut(ctx, CashOutCommand{
		SessionID:   "s1",
		OperationID: "op2",
		PlayerID:    "p1",
		Chips:       100,
	})
	_ = uc.CloseSession(ctx, CloseSessionCommand{SessionID: "s1"})

	res, _ := uc.GetResult(ctx, GetResultQuery{
		SessionID: "s1",
		PlayerID:  "p1",
	})

	if res.Amount() != 0 {
		t.Fatalf("expected 0, got %d", res.Amount())
	}
}

func TestSessionUseCase_NotFound(t *testing.T) {
	uc := newUseCase()
	ctx := context.Background()

	err := uc.StartSession(ctx, StartSessionCommand{SessionID: "unknown"})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if !errors.Is(err, entity.ErrSessionNotFound) &&
		!errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSessionUseCase_BusinessError(t *testing.T) {
	uc := newUseCase()
	ctx := context.Background()

	_ = uc.CreateSession(ctx, CreateSessionCommand{SessionID: "s1", Rate: 10})
	_ = uc.StartSession(ctx, StartSessionCommand{SessionID: "s1"})

	err := uc.CashOut(ctx, CashOutCommand{
		SessionID:   "s1",
		OperationID: "op1",
		PlayerID:    "p1",
		Chips:       100,
	})

	if !errors.Is(err, entity.ErrPlayerNotFound) {
		t.Fatalf("expected ErrPlayerNotFound, got %v", err)
	}
}
