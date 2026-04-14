package postgres

import (
	"context"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
)

func resetDB(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	ctx := context.Background()

	_, err := pool.Exec(ctx, `
		DROP SCHEMA public CASCADE;
		CREATE SCHEMA public;
	`)
	if err != nil {
		t.Fatalf("failed to reset schema: %v", err)
	}
}

func applyMigrations(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	// migrate работает через database/sql
	db := stdlib.OpenDB(*pool.Config().ConnConfig)

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		t.Fatalf("failed to create driver: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://./migrations",
		"postgres",
		driver,
	)
	if err != nil {
		t.Fatalf("failed to create migrate instance: %v", err)
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		t.Fatalf("failed to apply migrations: %v", err)
	}
}

func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx := context.Background()

	dsn := "postgres://postgres:postgres@localhost:5432/poc_test?sslmode=disable"

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("failed to connect db: %v", err)
	}

	//resetDB(t, pool) // пересоздаем базу
	applyMigrations(t, pool)

	// чистим данные (не схему!)
	_, _ = pool.Exec(ctx, "TRUNCATE idempotency_keys, operations, sessions CASCADE")

	return pool
}

func TestStartSessionUseCase_Integration(t *testing.T) {
	pool := setupTestDB(t)

	txManager := NewTxManager(pool)
	sessionRepo := NewSessionRepository()

	uc := usecase.NewStartSessionUseCase(
		sessionRepo,
		sessionRepo,
		txManager,
	)

	cmd := usecase.StartSessionCommand{
		SessionID: entity.SessionID("s1"),
		ChipRate:  10,
	}

	// --- первый вызов ---
	err := uc.Execute(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var count int
	err = pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM sessions WHERE id = $1`,
		"s1",
	).Scan(&count)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if count != 1 {
		t.Fatalf("expected 1 session, got %d", count)
	}

	// --- второй вызов ---
	err = uc.Execute(cmd)
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}

	err = pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM sessions WHERE id = $1`,
		"s1",
	).Scan(&count)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if count != 1 {
		t.Fatalf("idempotency broken, expected 1 session, got %d", count)
	}
}
