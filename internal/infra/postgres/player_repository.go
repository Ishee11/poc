package postgres

import (
	"context"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
)

type PlayerRepository struct{}

func NewPlayerRepository() *PlayerRepository {
	return &PlayerRepository{}
}

func (r *PlayerRepository) Create(tx usecase.Tx, p *entity.Player) error {
	q := `
		INSERT INTO players (id, name)
		VALUES ($1, $2)
	`

	_, err := tx.Exec(context.Background(), q, p.ID(), p.Name())
	return err
}

func (r *PlayerRepository) ListBySession(
	tx usecase.Tx,
	sessionID entity.SessionID,
) ([]usecase.PlayerDTO, error) {

	rows, err := tx.Query(context.Background(), `
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
