package postgres

import (
	"context"
	"database/sql"

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

// --- НОВОЕ ---

func (r *PlayerRepository) Exists(
	tx usecase.Tx,
	id entity.PlayerID,
) (bool, error) {

	var exists bool

	err := tx.QueryRow(
		context.Background(),
		`SELECT EXISTS (SELECT 1 FROM players WHERE id = $1)`,
		id,
	).Scan(&exists)

	if err != nil {
		return false, err
	}

	return exists, nil
}

func (r *PlayerRepository) GetByID(
	tx usecase.Tx,
	id entity.PlayerID,
) (*entity.Player, error) {

	var name string

	err := tx.QueryRow(
		context.Background(),
		`SELECT name FROM players WHERE id = $1`,
		id,
	).Scan(&name)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, entity.ErrPlayerNotFound
		}
		return nil, err
	}

	return entity.NewPlayer(id, name)
}

func (r *PlayerRepository) List(
	tx usecase.Tx,
	limit int,
	offset int,
) ([]usecase.PlayerDTO, error) {

	rows, err := tx.Query(context.Background(), `
		SELECT id, name
		FROM players
		ORDER BY name ASC, id ASC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]usecase.PlayerDTO, 0)
	for rows.Next() {
		var player usecase.PlayerDTO
		if err := rows.Scan(&player.ID, &player.Name); err != nil {
			return nil, err
		}
		result = append(result, player)
	}

	return result, rows.Err()
}

// --- уже было ---

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
