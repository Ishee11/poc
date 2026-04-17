package postgres

import (
	"context"
	"testing"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
	"github.com/ishee11/poc/internal/usecase/command"
)

type staticOperationIDGenerator struct {
	ids []entity.OperationID
	idx int
}

func (g *staticOperationIDGenerator) New() entity.OperationID {
	id := g.ids[g.idx]
	g.idx++
	return id
}

func TestCashOutAfterReversal_Integration(t *testing.T) {
	pool := setupTestDB(t)

	txManager := NewTxManager(pool)
	sessionRepo := NewSessionRepository()
	opRepo := NewOperationRepository()
	projectionRepo := NewProjectionRepository()
	idempotencyRepo := NewIdempotencyRepository()
	playerRepo := NewPlayerRepository()

	idGen := &staticOperationIDGenerator{
		ids: []entity.OperationID{"op-1", "op-2", "op-3", "op-4"},
	}

	// --- helper ---
	helper := usecase.NewHelper(
		sessionRepo,
		sessionRepo,
		playerRepo,
		opRepo,
		idGen,
	)

	startUC := usecase.NewStartSessionUseCase(
		sessionRepo,
		sessionRepo,
		txManager,
	)

	buyInUC := usecase.NewBuyInUseCase(
		helper,
		txManager,
		idempotencyRepo,
	)

	cashOutUC := usecase.NewCashOutUseCase(
		helper,
		projectionRepo, // playerStateReader
		projectionRepo, // projection
		txManager,
		idempotencyRepo,
	)

	reverseUC := usecase.NewReverseOperationUseCase(
		opRepo,
		opRepo,
		opRepo,
		sessionRepo,
		sessionRepo,
		txManager,
		idGen,
		idempotencyRepo,
	)

	if err := startUC.Execute(command.StartSessionCommand{
		SessionID: "s1",
		ChipRate:  10,
	}); err != nil {
		t.Fatalf("start session failed: %v", err)
	}

	ctx := context.Background()

	_, err := pool.Exec(ctx,
		`INSERT INTO players (id, name) VALUES ($1, $2)`,
		"p1", "player1",
	)
	if err != nil {
		t.Fatalf("failed to create player: %v", err)
	}

	if err := buyInUC.Execute(command.BuyInCommand{
		RequestID: "req-buy-1",
		SessionID: entity.SessionID("s1"),
		PlayerID:  "p1",
		Chips:     100,
	}); err != nil {
		t.Fatalf("buy in failed: %v", err)
	}

	if err := cashOutUC.Execute(command.CashOutCommand{
		RequestID: "req-cash-1",
		SessionID: entity.SessionID("s1"),
		PlayerID:  "p1",
		Chips:     40,
	}); err != nil {
		t.Fatalf("cash out failed: %v", err)
	}

	if err := reverseUC.Execute(command.ReverseOperationCommand{
		RequestID:         "req-rev-1",
		TargetOperationID: "op-2",
	}); err != nil {
		t.Fatalf("reverse operation failed: %v", err)
	}

	if err := cashOutUC.Execute(command.CashOutCommand{
		RequestID: "req-cash-2",
		SessionID: entity.SessionID("s1"),
		PlayerID:  "p1",
		Chips:     30,
	}); err != nil {
		t.Fatalf("cash out after reversal failed: %v", err)
	}
}
