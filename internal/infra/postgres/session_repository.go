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
		SELECT id, chip_rate, status, created_at, total_buy_in, total_cash_out
		FROM sessions
		WHERE id = $1
`, sessionID)

	var (
		id           string
		chipRate     int64
		status       string
		createdAt    time.Time
		totalBuyIn   int64
		totalCashOut int64
	)

	err := row.Scan(
		&id,
		&chipRate,
		&status,
		&createdAt,
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
		entity.Status(status),
		createdAt,
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
			id, chip_rate, status, created_at, total_buy_in, total_cash_out
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			total_buy_in = EXCLUDED.total_buy_in,
			total_cash_out = EXCLUDED.total_cash_out
	`,
		session.ID(),
		session.ChipRate().Value(),
		session.Status(),
		session.CreatedAt(),
		session.TotalBuyIn(),
		session.TotalCashOut(),
	)

	return err
}
