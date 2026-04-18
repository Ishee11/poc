package app

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	postgres "github.com/ishee11/poc/internal/infra/postgres"
)

type DB struct {
	Pool *pgxpool.Pool
}

func NewDB(ctx context.Context, dsn string) (*DB, error) {
	pool, err := connectWithRetry(ctx, dsn)
	if err != nil {
		return nil, err
	}

	if err := runMigrations(ctx, pool); err != nil {
		return nil, err
	}

	return &DB{Pool: pool}, nil
}

func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}

// --- internal ---

func connectWithRetry(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	var (
		pool *pgxpool.Pool
		err  error
	)

	for i := 0; i < 10; i++ {
		pool, err = pgxpool.New(ctx, dsn)
		if err == nil {
			if pingErr := pool.Ping(ctx); pingErr == nil {
				log.Println("connected to db")
				return pool, nil
			} else {
				err = pingErr
			}
		}

		log.Println("waiting for db...")
		time.Sleep(2 * time.Second)
	}

	return nil, fmt.Errorf("failed to connect to db: %w", err)
}

func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	if err := postgres.RunMigrations(ctx, pool, postgres.MigrationsFS); err != nil {
		return fmt.Errorf("migrations failed: %w", err)
	}

	log.Println("migrations applied")
	return nil
}
