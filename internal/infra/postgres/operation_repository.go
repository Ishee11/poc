package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
)

type OperationRepository struct{}

func NewOperationRepository() *OperationRepository {
	return &OperationRepository{}
}

// --- Save ---

func (r *OperationRepository) Save(
	tx usecase.Tx,
	op *entity.Operation,
) error {

	pgxTx, ok := tx.(pgx.Tx)
	if !ok {
		return errors.New("invalid tx type")
	}

	ctx := context.Background()

	cmdTag, err := pgxTx.Exec(ctx, `
		INSERT INTO operations (
			id,
			request_id,
			session_id,
			player_id,
			type,
			chips,
			reference_id,
			created_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (request_id) DO NOTHING
	`,
		op.ID(),
		op.RequestID(),
		op.SessionID(),
		op.PlayerID(),
		op.Type(),
		op.Chips(),
		op.ReferenceID(),
		op.CreatedAt(),
	)

	if err != nil {
		return err
	}

	// если строка не вставилась → это duplicate request
	if cmdTag.RowsAffected() == 0 {
		return entity.ErrDuplicateRequest
	}

	return nil
}

// --- GetByRequestID ---

func (r *OperationRepository) GetByRequestID(
	tx usecase.Tx,
	requestID string,
) (*entity.Operation, error) {

	pgxTx, ok := tx.(pgx.Tx)
	if !ok {
		return nil, errors.New("invalid tx type")
	}

	ctx := context.Background()

	row := pgxTx.QueryRow(ctx, `
		SELECT id, session_id, type, player_id, chips, created_at, reference_id, request_id
		FROM operations
		WHERE request_id = $1
	`, requestID)

	return scanOperation(row)
}

// --- GetByID ---

func (r *OperationRepository) GetByID(
	tx usecase.Tx,
	id entity.OperationID,
) (*entity.Operation, error) {

	pgxTx, ok := tx.(pgx.Tx)
	if !ok {
		return nil, errors.New("invalid tx type")
	}

	ctx := context.Background()

	row := pgxTx.QueryRow(ctx, `
		SELECT id, session_id, type, player_id, chips, created_at, reference_id, request_id
		FROM operations
		WHERE id = $1
	`, id)

	return scanOperation(row)
}

// --- helper ---

func scanOperation(row pgx.Row) (*entity.Operation, error) {
	var (
		id          string
		sessionID   string
		opType      string
		playerID    string
		chips       int64
		createdAt   time.Time
		referenceID *string
		requestID   string
	)

	err := row.Scan(
		&id,
		&sessionID,
		&opType,
		&playerID,
		&chips,
		&createdAt,
		&referenceID,
		&requestID,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrOperationNotFound
		}
		return nil, err
	}

	// обычная операция
	if opType != string(entity.OperationReversal) {
		return entity.NewOperation(
			entity.OperationID(id),
			requestID,
			entity.SessionID(sessionID),
			entity.OperationType(opType),
			entity.PlayerID(playerID),
			chips,
			createdAt,
		)
	}

	// reversal
	if referenceID == nil {
		return nil, entity.ErrInvalidReference
	}

	return entity.NewReversalOperation(
		entity.OperationID(id),
		requestID,
		entity.SessionID(sessionID),
		entity.PlayerID(playerID),
		chips,
		entity.OperationID(*referenceID),
		createdAt,
	)
}
