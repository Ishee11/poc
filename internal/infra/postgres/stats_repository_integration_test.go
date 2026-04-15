package postgres

import (
	"testing"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
	"github.com/ishee11/poc/internal/usecase/command"
)

func TestStatsRepository_Integration(t *testing.T) {
	pool := setupTestDB(t)

	txManager := NewTxManager(pool)
	sessionRepo := NewSessionRepository()
	opRepo := NewOperationRepository()
	projectionRepo := NewProjectionRepository()
	idempotencyRepo := NewIdempotencyRepository()
	statsRepo := NewStatsRepository()
	playerRepo := NewPlayerRepository()

	idGen := &staticOperationIDGenerator{
		ids: []entity.OperationID{"op-1", "op-2", "op-3", "op-4", "op-5"},
	}

	// --- helper ---
	helper := usecase.NewHelper(
		sessionRepo,
		sessionRepo,
		playerRepo,
		opRepo,
		idGen,
	)

	startUC := usecase.NewStartSessionUseCase(sessionRepo, sessionRepo, txManager)

	buyInUC := usecase.NewBuyInUseCase(
		helper,
		txManager,
		idempotencyRepo,
	)

	cashOutUC := usecase.NewCashOutUseCase(
		helper,
		projectionRepo,
		projectionRepo,
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

	// --- setup ---
	if err := startUC.Execute(command.StartSessionCommand{SessionID: "s1", ChipRate: 10}); err != nil {
		t.Fatalf("start session 1 failed: %v", err)
	}
	if err := startUC.Execute(command.StartSessionCommand{SessionID: "s2", ChipRate: 20}); err != nil {
		t.Fatalf("start session 2 failed: %v", err)
	}

	if err := buyInUC.Execute(command.BuyInCommand{
		RequestID:  "req-1",
		SessionID:  entity.SessionID("s1"),
		PlayerName: "p1",
		Chips:      100,
	}); err != nil {
		t.Fatalf("buy in 1 failed: %v", err)
	}

	if err := cashOutUC.Execute(command.CashOutCommand{
		RequestID:  "req-2",
		SessionID:  entity.SessionID("s1"),
		PlayerName: "p1",
		Chips:      40,
	}); err != nil {
		t.Fatalf("cash out 1 failed: %v", err)
	}

	if err := reverseUC.Execute(command.ReverseOperationCommand{
		RequestID:         "req-3",
		TargetOperationID: "op-2",
	}); err != nil {
		t.Fatalf("reverse failed: %v", err)
	}

	if err := cashOutUC.Execute(command.CashOutCommand{
		RequestID:  "req-4",
		SessionID:  entity.SessionID("s1"),
		PlayerName: "p1",
		Chips:      20,
	}); err != nil {
		t.Fatalf("cash out 2 failed: %v", err)
	}

	if err := buyInUC.Execute(command.BuyInCommand{
		RequestID:  "req-5",
		SessionID:  entity.SessionID("s2"),
		PlayerName: "p2",
		Chips:      200,
	}); err != nil {
		t.Fatalf("buy in 2 failed: %v", err)
	}

	// --- assertions ---
	err := txManager.RunInTx(func(tx usecase.Tx) error {
		sessions, err := statsRepo.ListSessions(tx, usecase.SessionStatsFilter{Limit: 10})
		if err != nil {
			return err
		}
		if len(sessions) != 2 {
			t.Fatalf("expected 2 sessions, got %d", len(sessions))
		}

		players, err := statsRepo.ListPlayers(tx, usecase.PlayerStatsFilter{Limit: 10})
		if err != nil {
			return err
		}
		if len(players) != 2 {
			t.Fatalf("expected 2 players, got %d", len(players))
		}

		player, err := statsRepo.GetPlayerOverall(tx, "p1", usecase.PlayerStatsFilter{Limit: 10})
		if err != nil {
			return err
		}
		if player.TotalBuyIn != 100 || player.TotalCashOut != 20 || player.ProfitMoney != -8 {
			t.Fatalf("unexpected player totals: %+v", player)
		}

		playerSessions, err := statsRepo.ListPlayerSessions(tx, "p1", usecase.PlayerStatsFilter{Limit: 10})
		if err != nil {
			return err
		}
		if len(playerSessions) != 1 {
			t.Fatalf("expected 1 player session, got %d", len(playerSessions))
		}
		if playerSessions[0].ProfitChips != -80 || playerSessions[0].ProfitMoney != -8 {
			t.Fatalf("unexpected player session totals: %+v", playerSessions[0])
		}

		return nil
	})
	if err != nil {
		t.Fatalf("stats query failed: %v", err)
	}
}
