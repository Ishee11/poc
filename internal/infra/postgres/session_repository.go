package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgconn"
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

	pgxTx, ok := tx.(pgx.Tx)
	if !ok {
		return nil, errors.New("invalid tx type")
	}

	ctx := context.Background()

	row := pgxTx.QueryRow(ctx, `
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

	// восстановление value object
	rate, err := valueobject.NewChipRate(chipRate)
	if err != nil {
		return nil, err
	}

	// создаём entity
	session := entity.NewSession(
		entity.SessionID(id),
		rate,
		createdAt,
	)

	// восстанавливаем статус
	switch status {
	case string(entity.StatusActive):
		// уже active по умолчанию
	case string(entity.StatusFinished):
		_ = session.Finish()
	default:
		return nil, errors.New("unknown session status")
	}

	// восстанавливаем cached aggregates
	// (да, это доступ к приватным полям через методы невозможен → делаем через apply)
	if totalBuyIn > 0 {
		_ = session.BuyIn(totalBuyIn)
	}
	if totalCashOut > 0 {
		_ = session.CashOut(totalCashOut)
	}

	return session, nil
}

// --- Writer ---

func (r *SessionRepository) Save(
	tx usecase.Tx,
	session *entity.Session,
) error {

	pgxTx, ok := tx.(pgx.Tx)
	if !ok {
		return errors.New("invalid tx type")
	}

	ctx := context.Background()

	// пробуем INSERT
	_, err := pgxTx.Exec(ctx, `
		INSERT INTO sessions (
			id, chip_rate, status, created_at, total_buy_in, total_cash_out
		)
		VALUES ($1, $2, $3, $4, $5, $6)
	`,
		session.ID(),
		session.ChipRate().Value(),
		session.Status(),
		session.CreatedAt(),
		session.TotalBuyIn(),
		session.TotalCashOut(),
	)

	if err == nil {
		return nil
	}

	// если уже существует → UPDATE
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		_, err = pgxTx.Exec(ctx, `
			UPDATE sessions
			SET
				status = $2,
				total_buy_in = $3,
				total_cash_out = $4
			WHERE id = $1
		`,
			session.ID(),
			session.Status(),
			session.TotalBuyIn(),
			session.TotalCashOut(),
		)
		return err
	}

	return err
}
