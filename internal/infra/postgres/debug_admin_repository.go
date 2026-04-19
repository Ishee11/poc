package postgres

import (
	"context"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
)

type DebugAdminRepository struct{}

func NewDebugAdminRepository() *DebugAdminRepository {
	return &DebugAdminRepository{}
}

func (r *DebugAdminRepository) DeletePlayer(tx usecase.Tx, playerID entity.PlayerID) error {
	ctx := context.Background()

	rows, err := tx.Query(ctx, `
		SELECT DISTINCT session_id
		FROM operations
		WHERE player_id = $1
	`, playerID)
	if err != nil {
		return err
	}

	sessionIDs := make([]entity.SessionID, 0)
	for rows.Next() {
		var sessionID entity.SessionID
		if err := rows.Scan(&sessionID); err != nil {
			rows.Close()
			return err
		}
		sessionIDs = append(sessionIDs, sessionID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	cmd, err := tx.Exec(ctx, `
		DELETE FROM operations
		WHERE reference_id IN (
			SELECT id
			FROM operations
			WHERE player_id = $1
		)
	`, playerID)
	if err != nil {
		return err
	}
	_ = cmd

	if _, err := tx.Exec(ctx, `DELETE FROM operations WHERE player_id = $1`, playerID); err != nil {
		return err
	}

	tag, err := tx.Exec(ctx, `DELETE FROM players WHERE id = $1`, playerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return entity.ErrPlayerNotFound
	}

	for _, sessionID := range sessionIDs {
		if err := recalculateSessionTotals(ctx, tx, sessionID); err != nil {
			return err
		}
	}

	return nil
}

func (r *DebugAdminRepository) DeleteSession(tx usecase.Tx, sessionID entity.SessionID) error {
	ctx := context.Background()

	if _, err := tx.Exec(ctx, `DELETE FROM operations WHERE session_id = $1`, sessionID); err != nil {
		return err
	}

	tag, err := tx.Exec(ctx, `DELETE FROM sessions WHERE id = $1`, sessionID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return entity.ErrSessionNotFound
	}

	return nil
}

func (r *DebugAdminRepository) DeleteSessionFinish(tx usecase.Tx, sessionID entity.SessionID) error {
	ctx := context.Background()

	tag, err := tx.Exec(ctx, `
		UPDATE sessions
		SET status = 'active'
		WHERE id = $1
		  AND status = 'finished'
	`, sessionID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return entity.ErrSessionNotFound
	}

	return nil
}

func recalculateSessionTotals(ctx context.Context, tx usecase.Tx, sessionID entity.SessionID) error {
	_, err := tx.Exec(ctx, `
		WITH totals AS (
			SELECT
				COALESCE(SUM(
					CASE
						WHEN o.type = 'buy_in' THEN o.chips
						WHEN o.type = 'reversal' AND ref.type = 'buy_in' THEN -o.chips
						ELSE 0
					END
				), 0) AS total_buy_in,
				COALESCE(SUM(
					CASE
						WHEN o.type = 'cash_out' THEN o.chips
						WHEN o.type = 'reversal' AND ref.type = 'cash_out' THEN -o.chips
						ELSE 0
					END
				), 0) AS total_cash_out
			FROM operations o
			LEFT JOIN operations ref ON o.reference_id = ref.id
			WHERE o.session_id = $1
		)
		UPDATE sessions
		SET
			total_buy_in = totals.total_buy_in,
			total_cash_out = totals.total_cash_out
		FROM totals
		WHERE sessions.id = $1
	`, sessionID)
	return err
}
