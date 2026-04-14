package postgres

import (
	"context"
	"errors"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
	"github.com/jackc/pgx/v5"
)

type PlayerRepository struct{}

func NewPlayerRepository() *PlayerRepository {
	return &PlayerRepository{}
}

func (r *PlayerRepository) GetOrCreate(
	tx usecase.Tx,
	sessionID entity.SessionID,
	name string,
) (entity.PlayerID, error) {

	pgxTx, ok := tx.(pgx.Tx)
	if !ok {
		return "", errors.New("invalid tx type")
	}

	ctx := context.Background()

	var id string

	err := pgxTx.QueryRow(ctx, `
		INSERT INTO players_in_session (id, session_id, name)
		VALUES (gen_random_uuid()::text, $1, $2)
		ON CONFLICT (session_id, name)
		DO UPDATE SET name = EXCLUDED.name
		RETURNING id
	`, sessionID, name).Scan(&id)

	if err != nil {
		return "", err
	}

	return entity.PlayerID(id), nil
}

func (r *PlayerRepository) ListBySession(
	tx usecase.Tx,
	sessionID entity.SessionID,
) ([]usecase.PlayerDTO, error) {

	pgxTx, ok := tx.(pgx.Tx)
	if !ok {
		return nil, ErrInvalidTx
	}

	rows, err := pgxTx.Query(context.Background(), `
		SELECT id, name
		FROM players_in_session
		WHERE session_id = $1
		ORDER BY created_at ASC
	`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []usecase.PlayerDTO

	for rows.Next() {
		var p usecase.PlayerDTO
		if err := rows.Scan(&p.ID, &p.Name); err != nil {
			return nil, err
		}
		result = append(result, p)
	}

	return result, rows.Err()
}
