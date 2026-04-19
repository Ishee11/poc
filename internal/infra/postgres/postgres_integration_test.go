package postgres

import (
	"context"
	"errors"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/entity/valueobject"
	"github.com/ishee11/poc/internal/usecase"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("PG_URL")
	}
	if dsn == "" {
		t.Skip("DATABASE_URL or PG_URL is not set")
	}
	ensureSafeTestDSN(t, dsn)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("postgres is not available: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("postgres is not available: %v", err)
	}

	if err := RunMigrations(ctx, pool, MigrationsFS); err != nil {
		pool.Close()
		t.Fatalf("run migrations: %v", err)
	}

	t.Cleanup(pool.Close)
	return pool
}

func ensureSafeTestDSN(t *testing.T, dsn string) {
	t.Helper()
	if os.Getenv("ALLOW_DESTRUCTIVE_INTEGRATION_TESTS") == "true" {
		return
	}
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse database dsn: %v", err)
	}
	switch parsed.Hostname() {
	case "127.0.0.1", "localhost", "::1":
		return
	default:
		t.Skipf("refusing to run destructive integration tests against non-local database host %q", parsed.Hostname())
	}
}

func cleanDB(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		TRUNCATE TABLE idempotency_keys, operations, sessions, players
		RESTART IDENTITY CASCADE
	`)
	if err != nil {
		t.Fatalf("clean database: %v", err)
	}
}

func txRun(t *testing.T, pool *pgxpool.Pool, fn func(tx usecase.Tx)) {
	t.Helper()
	txManager := NewTxManager(pool)
	if err := txManager.RunInTx(func(tx usecase.Tx) error {
		fn(tx)
		return nil
	}); err != nil {
		t.Fatalf("run tx: %v", err)
	}
}

func saveTestPlayer(t *testing.T, tx usecase.Tx, id entity.PlayerID, name string) {
	t.Helper()
	player, err := entity.NewPlayer(id, name)
	if err != nil {
		t.Fatal(err)
	}
	if err := NewPlayerRepository().Create(tx, player); err != nil {
		t.Fatal(err)
	}
}

func saveTestSession(t *testing.T, tx usecase.Tx, id entity.SessionID, status entity.Status, chipRate int64) {
	t.Helper()
	rate, err := valueobject.NewChipRate(chipRate)
	if err != nil {
		t.Fatal(err)
	}
	session := entity.RestoreSession(id, rate, 2, entity.CurrencyRUB, status, time.Now().Add(-time.Hour), nil, 0, 0)
	if err := NewSessionRepository().Save(tx, session); err != nil {
		t.Fatal(err)
	}
}

func saveTestOperation(
	t *testing.T,
	tx usecase.Tx,
	id entity.OperationID,
	requestID string,
	sessionID entity.SessionID,
	opType entity.OperationType,
	playerID entity.PlayerID,
	chips int64,
	createdAt time.Time,
) {
	t.Helper()
	op, err := entity.NewOperation(id, requestID, sessionID, opType, playerID, chips, createdAt)
	if err != nil {
		t.Fatal(err)
	}
	if err := NewOperationRepository().Save(tx, op); err != nil {
		t.Fatal(err)
	}
}

func saveTestReversal(
	t *testing.T,
	tx usecase.Tx,
	id entity.OperationID,
	requestID string,
	sessionID entity.SessionID,
	playerID entity.PlayerID,
	chips int64,
	referenceID entity.OperationID,
	createdAt time.Time,
) {
	t.Helper()
	op, err := entity.NewReversalOperation(id, requestID, sessionID, playerID, chips, referenceID, createdAt)
	if err != nil {
		t.Fatal(err)
	}
	if err := NewOperationRepository().Save(tx, op); err != nil {
		t.Fatal(err)
	}
}

func TestOperationRepository_Integration(t *testing.T) {
	pool := testPool(t)
	cleanDB(t, pool)

	txRun(t, pool, func(tx usecase.Tx) {
		saveTestSession(t, tx, "s1", entity.StatusActive, 2)
		saveTestPlayer(t, tx, "p1", "Alice")

		repo := NewOperationRepository()
		op, err := entity.NewOperation("op1", "req1", "s1", entity.OperationBuyIn, "p1", 100, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if err := repo.Save(tx, op); err != nil {
			t.Fatalf("save operation: %v", err)
		}
		if err := repo.Save(tx, op); !errors.Is(err, entity.ErrDuplicateRequest) {
			t.Fatalf("expected duplicate request, got %v", err)
		}

		got, err := repo.GetByRequestID(tx, "req1")
		if err != nil {
			t.Fatalf("get by request id: %v", err)
		}
		if got.ID() != "op1" || got.Chips() != 100 {
			t.Fatalf("unexpected operation: id=%s chips=%d", got.ID(), got.Chips())
		}

		saveTestReversal(t, tx, "op2", "req2", "s1", "p1", 100, "op1", time.Now())
		exists, err := repo.ExistsReversal(tx, "op1")
		if err != nil {
			t.Fatalf("exists reversal: %v", err)
		}
		if !exists {
			t.Fatal("expected reversal to exist")
		}
	})
}

func TestProjectionRepository_Integration(t *testing.T) {
	pool := testPool(t)
	cleanDB(t, pool)

	txRun(t, pool, func(tx usecase.Tx) {
		now := time.Now()
		saveTestSession(t, tx, "s1", entity.StatusActive, 2)
		saveTestPlayer(t, tx, "p1", "Alice")
		saveTestPlayer(t, tx, "p2", "Bob")
		saveTestOperation(t, tx, "op1", "req1", "s1", entity.OperationBuyIn, "p1", 100, now)
		saveTestOperation(t, tx, "op2", "req2", "s1", entity.OperationCashOut, "p1", 40, now.Add(time.Minute))
		saveTestOperation(t, tx, "op3", "req3", "s1", entity.OperationBuyIn, "p2", 50, now.Add(2*time.Minute))
		saveTestReversal(t, tx, "op4", "req4", "s1", "p2", 50, "op3", now.Add(3*time.Minute))

		repo := NewProjectionRepository()
		sessionAgg, err := repo.GetSessionAggregates(tx, "s1")
		if err != nil {
			t.Fatalf("get session aggregates: %v", err)
		}
		if sessionAgg.TotalBuyIn != 100 || sessionAgg.TotalCashOut != 40 {
			t.Fatalf("unexpected session aggregates: %+v", sessionAgg)
		}

		playerAggs, err := repo.GetPlayerAggregates(tx, "s1")
		if err != nil {
			t.Fatalf("get player aggregates: %v", err)
		}
		if playerAggs["p1"].BuyIn != 100 || playerAggs["p1"].CashOut != 40 {
			t.Fatalf("unexpected p1 aggregates: %+v", playerAggs["p1"])
		}
		if playerAggs["p2"].BuyIn != 0 || playerAggs["p2"].CashOut != 0 {
			t.Fatalf("unexpected p2 aggregates after reversal: %+v", playerAggs["p2"])
		}

		opType, found, err := repo.GetLastOperationType(tx, "s1", "p2")
		if err != nil {
			t.Fatalf("get last operation type: %v", err)
		}
		if found {
			t.Fatalf("expected reversed player operation to be ignored, got %s", opType)
		}
	})
}

func TestStatsRepository_Integration(t *testing.T) {
	pool := testPool(t)
	cleanDB(t, pool)

	txRun(t, pool, func(tx usecase.Tx) {
		now := time.Now()
		saveTestSession(t, tx, "s1", entity.StatusFinished, 2)
		saveTestPlayer(t, tx, "p1", "Alice")
		saveTestOperation(t, tx, "op1", "req1", "s1", entity.OperationBuyIn, "p1", 100, now)
		saveTestOperation(t, tx, "op2", "req2", "s1", entity.OperationCashOut, "p1", 40, now.Add(time.Minute))

		repo := NewStatsRepository(pool)
		sessions, err := repo.ListSessions(tx, usecase.SessionStatsFilter{Limit: 10})
		if err != nil {
			t.Fatalf("list sessions: %v", err)
		}
		if len(sessions) != 1 {
			t.Fatalf("expected one session, got %d", len(sessions))
		}
		if sessions[0].PlayerCount != 1 || sessions[0].TotalBuyIn != 100 || sessions[0].TotalCashOut != 40 {
			t.Fatalf("unexpected session stat: %+v", sessions[0])
		}

		player, err := repo.GetPlayerOverall(tx, "p1", usecase.PlayerStatsFilter{})
		if err != nil {
			t.Fatalf("get player overall: %v", err)
		}
		if player.PlayerName != "Alice" || player.ProfitChips != -60 || player.ProfitMoney != -30 {
			t.Fatalf("unexpected player overall: %+v", player)
		}
		if player.TotalBuyInMoney != 50 || player.TotalCashOutMoney != 20 {
			t.Fatalf("unexpected player money totals: %+v", player)
		}
		if player.AvgProfitPerSession != -30 || player.ROIPercent != -60 || player.AvgBuyInPerSession != 100 {
			t.Fatalf("unexpected player performance metrics: %+v", player)
		}

		playerSessions, err := repo.ListPlayerSessions(tx, "p1", usecase.PlayerStatsFilter{Limit: 10})
		if err != nil {
			t.Fatalf("list player sessions: %v", err)
		}
		if len(playerSessions) != 1 || playerSessions[0].ProfitMoney != -30 {
			t.Fatalf("unexpected player sessions: %+v", playerSessions)
		}
	})
}
