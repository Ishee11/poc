package postgres

import (
	"context"
	"testing"

	"github.com/ishee11/poc/internal/infra"
	"github.com/ishee11/poc/internal/usecase"
	"github.com/ishee11/poc/internal/usecase/command"
)

func TestBuyInUseCase_Integration(t *testing.T) {
	pool := setupTestDB(t)

	txManager := NewTxManager(pool)
	sessionRepo := NewSessionRepository()
	opRepo := NewOperationRepository()
	idGen := &infra.UUIDOperationIDGenerator{}
	idempotencyRepo := NewIdempotencyRepository()
	playerRepo := NewPlayerRepository()

	// --- сначала создаем сессию ---
	startUC := usecase.NewStartSessionUseCase(
		sessionRepo,
		sessionRepo,
		txManager,
	)

	err := startUC.Execute(command.StartSessionCommand{
		SessionID: "s1",
		ChipRate:  10,
	})
	if err != nil {
		t.Fatalf("start session failed: %v", err)
	}

	// --- BuyIn ---
	buyUC := usecase.NewBuyInUseCase(
		opRepo,
		sessionRepo,
		sessionRepo,
		txManager,
		idGen,
		idempotencyRepo,
		playerRepo,
	)

	cmd := command.BuyInCommand{
		RequestID: "req-1",
		SessionID: "s1",
		PlayerID:  "p1",
		Chips:     100,
	}

	err = buyUC.Execute(cmd)
	if err != nil {
		t.Fatalf("buy in failed: %v", err)
	}

	// --- проверяем operations ---
	var opCount int
	err = pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM operations WHERE session_id = $1`,
		"s1",
	).Scan(&opCount)
	if err != nil {
		t.Fatalf("query operations failed: %v", err)
	}

	if opCount != 1 {
		t.Fatalf("expected 1 operation, got %d", opCount)
	}

	// --- проверяем session cache ---
	var totalBuyIn int64
	err = pool.QueryRow(context.Background(),
		`SELECT total_buy_in FROM sessions WHERE id = $1`,
		"s1",
	).Scan(&totalBuyIn)
	if err != nil {
		t.Fatalf("query session failed: %v", err)
	}

	if totalBuyIn != 100 {
		t.Fatalf("expected total_buy_in=100, got %d", totalBuyIn)
	}

	// --- повторный вызов (идемпотентность) ---
	err = buyUC.Execute(cmd)
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}

	// operations всё ещё 1
	err = pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM operations WHERE session_id = $1`,
		"s1",
	).Scan(&opCount)
	if err != nil {
		t.Fatalf("query operations failed: %v", err)
	}

	if opCount != 1 {
		t.Fatalf("idempotency broken, expected 1 operation, got %d", opCount)
	}
}
