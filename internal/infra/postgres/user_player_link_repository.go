package postgres

import (
	"context"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
)

type UserPlayerLinkRepository struct{}

func NewUserPlayerLinkRepository() *UserPlayerLinkRepository {
	return &UserPlayerLinkRepository{}
}

func (r *UserPlayerLinkRepository) LinkPlayer(
	tx usecase.Tx,
	userID entity.AuthUserID,
	playerID entity.PlayerID,
) error {
	_, err := tx.Exec(context.Background(), `
		INSERT INTO user_players (user_id, player_id)
		VALUES ($1, $2)
	`, userID, playerID)

	return err
}

func (r *UserPlayerLinkRepository) UnlinkPlayer(
	tx usecase.Tx,
	userID entity.AuthUserID,
	playerID entity.PlayerID,
) error {
	_, err := tx.Exec(context.Background(), `
		DELETE FROM user_players
		WHERE user_id = $1 AND player_id = $2
	`, userID, playerID)

	return err
}

func (r *UserPlayerLinkRepository) ListUserPlayers(
	tx usecase.Tx,
	userID entity.AuthUserID,
) ([]usecase.PlayerDTO, error) {
	rows, err := tx.Query(context.Background(), `
		SELECT p.id, p.name
		FROM user_players up
		JOIN players p ON p.id = up.player_id
		WHERE up.user_id = $1
		ORDER BY p.name ASC, p.id ASC
	`, userID)
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

func (r *UserPlayerLinkRepository) IsPlayerLinked(
	tx usecase.Tx,
	playerID entity.PlayerID,
) (bool, error) {
	var exists bool
	err := tx.QueryRow(context.Background(), `
		SELECT EXISTS (
			SELECT 1
			FROM user_players
			WHERE player_id = $1
		)
	`, playerID).Scan(&exists)

	return exists, err
}

func (r *UserPlayerLinkRepository) IsPlayerLinkedToUser(
	tx usecase.Tx,
	userID entity.AuthUserID,
	playerID entity.PlayerID,
) (bool, error) {
	var exists bool
	err := tx.QueryRow(context.Background(), `
		SELECT EXISTS (
			SELECT 1
			FROM user_players
			WHERE user_id = $1 AND player_id = $2
		)
	`, userID, playerID).Scan(&exists)

	return exists, err
}

func (r *UserPlayerLinkRepository) ListUnlinkedPlayers(
	tx usecase.Tx,
	limit int,
	offset int,
) ([]usecase.PlayerDTO, error) {
	rows, err := tx.Query(context.Background(), `
		SELECT p.id, p.name
		FROM players p
		LEFT JOIN user_players up ON up.player_id = p.id
		WHERE up.player_id IS NULL
		ORDER BY p.name ASC, p.id ASC
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
