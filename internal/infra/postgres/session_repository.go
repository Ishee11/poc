package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/entity/valueobject"
	"github.com/ishee11/poc/internal/usecase"
)

type SessionRepository struct{}

func NewSessionRepository() *SessionRepository {
	return &SessionRepository{}
}

// --- Reader ---

func (r *SessionRepository) FindByID(
	tx usecase.Tx,
	sessionID entity.SessionID,
) (*entity.Session, error) {

	ctx := context.Background()

	row := tx.QueryRow(ctx, `
		SELECT id, chip_rate, big_blind, currency, status, created_at, finished_at, total_buy_in, total_cash_out
		FROM sessions
		WHERE id = $1
`, sessionID)

	var (
		id           string
		chipRate     int64
		bigBlind     int64
		currency     entity.Currency
		status       string
		createdAt    time.Time
		finishedAt   *time.Time
		totalBuyIn   int64
		totalCashOut int64
	)

	err := row.Scan(
		&id,
		&chipRate,
		&bigBlind,
		&currency,
		&status,
		&createdAt,
		&finishedAt,
		&totalBuyIn,
		&totalCashOut,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrSessionNotFound
		}
		return nil, err
	}

	rate, err := valueobject.NewChipRate(chipRate)
	if err != nil {
		return nil, err
	}

	return entity.RestoreSession(
		entity.SessionID(id),
		rate,
		bigBlind,
		currency,
		entity.Status(status),
		createdAt,
		finishedAt,
		totalBuyIn,
		totalCashOut,
	), nil
}

// --- Writer ---

func (r *SessionRepository) Save(
	tx usecase.Tx,
	session *entity.Session,
) error {

	ctx := context.Background()

	_, err := tx.Exec(ctx, `
		INSERT INTO sessions (
			id, chip_rate, big_blind, currency, status, created_at, finished_at, total_buy_in, total_cash_out
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			finished_at = EXCLUDED.finished_at,
			total_buy_in = EXCLUDED.total_buy_in,
			total_cash_out = EXCLUDED.total_cash_out
	`,
		session.ID(),
		session.ChipRate().Value(),
		session.BigBlind(),
		session.Currency(),
		session.Status(),
		session.CreatedAt(),
		session.FinishedAt(),
		session.TotalBuyIn(),
		session.TotalCashOut(),
	)

	return err
}

func (r *SessionRepository) FindByIDForUpdate(
	tx usecase.Tx,
	id entity.SessionID,
) (*entity.Session, error) {

	row := tx.QueryRow(
		context.Background(),
		`
		SELECT id, chip_rate, big_blind, currency, status, created_at, finished_at, total_buy_in, total_cash_out
		FROM sessions
		WHERE id = $1
		FOR UPDATE
		`,
		id,
	)

	var (
		sessionID    entity.SessionID
		chipRate     int64
		bigBlind     int64
		currency     entity.Currency
		status       entity.Status
		createdAt    time.Time
		finishedAt   *time.Time
		totalBuyIn   int64
		totalCashOut int64
	)

	err := row.Scan(
		&sessionID,
		&chipRate,
		&bigBlind,
		&currency,
		&status,
		&createdAt,
		&finishedAt,
		&totalBuyIn,
		&totalCashOut,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, entity.ErrSessionNotFound
		}
		return nil, err
	}

	rate, err := valueobject.NewChipRate(chipRate)
	if err != nil {
		return nil, err
	}

	return entity.RestoreSession(
		sessionID,
		rate,
		bigBlind,
		currency,
		status,
		createdAt,
		finishedAt,
		totalBuyIn,
		totalCashOut,
	), nil
}
