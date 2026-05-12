package postgres

import (
	"context"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
)

type SettlementTransferRepository struct{}

func NewSettlementTransferRepository() *SettlementTransferRepository {
	return &SettlementTransferRepository{}
}

func (r *SettlementTransferRepository) ListBySession(tx usecase.Tx, sessionID entity.SessionID) ([]usecase.SettlementTransfer, error) {
	rows, err := tx.Query(context.Background(), `
		SELECT id, session_id, from_player_id, to_player_id, amount, position
		FROM settlement_transfers
		WHERE session_id = $1
		ORDER BY position ASC, created_at ASC
	`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	transfers := make([]usecase.SettlementTransfer, 0)
	for rows.Next() {
		var transfer usecase.SettlementTransfer
		if err := rows.Scan(
			&transfer.ID,
			&transfer.SessionID,
			&transfer.From,
			&transfer.To,
			&transfer.Amount,
			&transfer.Position,
		); err != nil {
			return nil, err
		}
		transfers = append(transfers, transfer)
	}
	return transfers, rows.Err()
}

func (r *SettlementTransferRepository) ReplaceBySession(tx usecase.Tx, sessionID entity.SessionID, transfers []usecase.SettlementTransfer) error {
	if _, err := tx.Exec(context.Background(), `
		DELETE FROM settlement_transfers
		WHERE session_id = $1
	`, sessionID); err != nil {
		return err
	}

	for _, transfer := range transfers {
		if _, err := tx.Exec(context.Background(), `
			INSERT INTO settlement_transfers (id, session_id, from_player_id, to_player_id, amount, position)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, transfer.ID, sessionID, transfer.From, transfer.To, transfer.Amount, transfer.Position); err != nil {
			return err
		}
	}

	return nil
}
