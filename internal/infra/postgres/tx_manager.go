package postgres

import (
	"context"
	"runtime"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"

	"github.com/ishee11/poc/internal/usecase"
)

type TxManager struct {
	pool *pgxpool.Pool
}

func NewTxManager(pool *pgxpool.Pool) *TxManager {
	return &TxManager{pool: pool}
}

type txKey struct{}

func (m *TxManager) RunInTx(ctx context.Context, fn func(tx usecase.Tx) error) error {
	ctx, span := otel.Tracer("github.com/ishee11/poc/internal/usecase").Start(ctx, callerUseCaseSpanName())
	defer span.End()

	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.RepeatableRead,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback(ctx)
			panic(r)
		}
	}()

	err = fn(contextTx{ctx: ctx, tx: tx})

	if err != nil {
		_ = tx.Rollback(ctx)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	return nil
}

type contextTx struct {
	ctx context.Context
	tx  pgx.Tx
}

func (t contextTx) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return t.tx.Exec(t.ctx, sql, args...)
}

func (t contextTx) Query(_ context.Context, sql string, args ...any) (pgx.Rows, error) {
	return t.tx.Query(t.ctx, sql, args...)
}

func (t contextTx) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	return t.tx.QueryRow(t.ctx, sql, args...)
}

func callerUseCaseSpanName() string {
	pc, _, _, ok := runtime.Caller(2)
	if !ok {
		return "usecase.transaction"
	}

	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "usecase.transaction"
	}

	name := fn.Name()
	if idx := strings.LastIndex(name, "/internal/usecase."); idx >= 0 {
		name = name[idx+len("/internal/usecase."):]
	}
	name = strings.TrimPrefix(name, "(*")
	name = strings.TrimPrefix(name, "github.com/ishee11/poc/internal/usecase.")
	name = strings.ReplaceAll(name, ").", ".")
	return "usecase." + name
}
