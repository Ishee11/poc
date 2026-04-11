package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
)

type ProjectionRepository struct{}

func NewProjectionRepository() *ProjectionRepository {
	return &ProjectionRepository{}
}

// --- GetSessionAggregates ---

func (r *ProjectionRepository) GetSessionAggregates(
	tx usecase.Tx,
	sessionID entity.SessionID,
) (usecase.SessionAggregates, error) {

	pgxTx, ok := tx.(pgx.Tx)
	if !ok {
		return usecase.SessionAggregates{}, ErrInvalidTx
	}

	ctx := context.Background()

	row := pgxTx.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(
				CASE
					WHEN o.type = 'buy_in' THEN o.chips
					WHEN o.type = 'reversal' AND ref.type = 'buy_in' THEN -o.chips
					ELSE 0
				END
			), 0),
			COALESCE(SUM(
				CASE
					WHEN o.type = 'cash_out' THEN o.chips
					WHEN o.type = 'reversal' AND ref.type = 'cash_out' THEN -o.chips
					ELSE 0
				END
			), 0)
		FROM operations o
		LEFT JOIN operations ref ON o.reference_id = ref.id
		WHERE o.session_id = $1
	`, sessionID)

	var totalBuyIn int64
	var totalCashOut int64

	err := row.Scan(&totalBuyIn, &totalCashOut)
	if err != nil {
		return usecase.SessionAggregates{}, err
	}

	return usecase.SessionAggregates{
		TotalBuyIn:   totalBuyIn,
		TotalCashOut: totalCashOut,
	}, nil
}

// --- GetPlayerAggregates ---

func (r *ProjectionRepository) GetPlayerAggregates(
	tx usecase.Tx,
	sessionID entity.SessionID,
) (map[entity.PlayerID]usecase.PlayerAggregates, error) {

	pgxTx, ok := tx.(pgx.Tx)
	if !ok {
		return nil, ErrInvalidTx
	}

	ctx := context.Background()

	rows, err := pgxTx.Query(ctx, `
		SELECT
			o.player_id,
			COALESCE(SUM(
				CASE
					WHEN o.type = 'buy_in' THEN o.chips
					WHEN o.type = 'reversal' AND ref.type = 'buy_in' THEN -o.chips
					ELSE 0
				END
			), 0),
			COALESCE(SUM(
				CASE
					WHEN o.type = 'cash_out' THEN o.chips
					WHEN o.type = 'reversal' AND ref.type = 'cash_out' THEN -o.chips
					ELSE 0
				END
			), 0)
		FROM operations o
		LEFT JOIN operations ref ON o.reference_id = ref.id
		WHERE o.session_id = $1
		GROUP BY o.player_id
	`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[entity.PlayerID]usecase.PlayerAggregates)

	for rows.Next() {
		var playerID string
		var buyIn int64
		var cashOut int64

		if err := rows.Scan(&playerID, &buyIn, &cashOut); err != nil {
			return nil, err
		}

		result[entity.PlayerID(playerID)] = usecase.PlayerAggregates{
			BuyIn:   buyIn,
			CashOut: cashOut,
		}
	}

	return result, nil
}

// --- GetLastOperationType ---

func (r *ProjectionRepository) GetLastOperationType(
	tx usecase.Tx,
	sessionID entity.SessionID,
	playerID entity.PlayerID,
) (entity.OperationType, bool, error) {

	pgxTx, ok := tx.(pgx.Tx)
	if !ok {
		return "", false, ErrInvalidTx
	}

	ctx := context.Background()

	row := pgxTx.QueryRow(ctx, `
		SELECT o.type
		FROM operations o
		LEFT JOIN operations rev
			ON rev.reference_id = o.id
			AND rev.type = 'reversal'
		WHERE o.session_id = $1
		  AND o.player_id = $2
		  AND o.type <> 'reversal'
		  AND rev.id IS NULL
		ORDER BY o.created_at DESC
		LIMIT 1
	`, sessionID, playerID)

	var opType string

	err := row.Scan(&opType)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, nil
		}
		return "", false, err
	}

	return entity.OperationType(opType), true, nil
}

// --- ListBySession ---

func (r *ProjectionRepository) ListBySession(
	tx usecase.Tx,
	sessionID entity.SessionID,
	limit int,
	offset int,
) ([]*entity.Operation, error) {

	pgxTx, ok := tx.(pgx.Tx)
	if !ok {
		return nil, ErrInvalidTx
	}

	ctx := context.Background()

	rows, err := pgxTx.Query(ctx, `
		SELECT id, session_id, type, player_id, chips, created_at, reference_id, request_id
		FROM operations
		WHERE session_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, sessionID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*entity.Operation

	for rows.Next() {
		op, err := scanOperation(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, op)
	}

	return result, nil
}
