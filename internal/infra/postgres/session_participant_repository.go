package postgres

import (
	"context"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
)

type SessionParticipantRepository struct{}

func NewSessionParticipantRepository() *SessionParticipantRepository {
	return &SessionParticipantRepository{}
}

func (r *SessionParticipantRepository) Add(
	tx usecase.Tx,
	sessionID entity.SessionID,
	playerID entity.PlayerID,
) error {
	_, err := tx.Exec(context.Background(), `
		INSERT INTO session_participants (session_id, player_id)
		VALUES ($1, $2)
		ON CONFLICT (session_id, player_id) DO NOTHING
	`, sessionID, playerID)
	return err
}
