package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

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

	return mapOwnershipConstraintError(err)
}

func (r *UserPlayerLinkRepository) FindUserPlayer(
	tx usecase.Tx,
	userID entity.AuthUserID,
) (*usecase.PlayerDTO, error) {
	var player usecase.PlayerDTO
	err := tx.QueryRow(context.Background(), `
		SELECT p.id, p.name
		FROM user_players up
		JOIN players p ON p.id = up.player_id
		WHERE up.user_id = $1
	`, userID).Scan(&player.ID, &player.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &player, nil
}

func (r *UserPlayerLinkRepository) FindPlayerOwner(
	tx usecase.Tx,
	playerID entity.PlayerID,
) (*entity.AuthUserID, error) {
	var userID entity.AuthUserID
	err := tx.QueryRow(context.Background(), `
		SELECT user_id
		FROM user_players
		WHERE player_id = $1
	`, playerID).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &userID, nil
}

func (r *UserPlayerLinkRepository) LockUser(tx usecase.Tx, userID entity.AuthUserID) error {
	var lockedID entity.AuthUserID
	err := tx.QueryRow(context.Background(), `
		SELECT id FROM users WHERE id = $1 FOR UPDATE
	`, userID).Scan(&lockedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return entity.ErrAuthUserNotFound
	}
	return err
}

func (r *UserPlayerLinkRepository) LockPlayer(tx usecase.Tx, playerID entity.PlayerID) error {
	var lockedID entity.PlayerID
	err := tx.QueryRow(context.Background(), `
		SELECT id FROM players WHERE id = $1 FOR UPDATE
	`, playerID).Scan(&lockedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return entity.ErrPlayerNotFound
	}
	return err
}

func (r *UserPlayerLinkRepository) ListAccounts(
	tx usecase.Tx,
	query string,
	limit int,
	offset int,
) ([]usecase.AccountOwnershipDTO, int64, error) {
	pattern := "%" + query + "%"
	var total int64
	if err := tx.QueryRow(context.Background(), `
		SELECT COUNT(*)
		FROM users u
		LEFT JOIN user_players up ON up.user_id = u.id
		LEFT JOIN players p ON p.id = up.player_id
		WHERE $1 = '' OR u.email ILIKE $2 OR p.name ILIKE $2
	`, query, pattern).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := tx.Query(context.Background(), `
		SELECT u.id, u.email, u.role, u.status, p.id, p.name
		FROM users u
		LEFT JOIN user_players up ON up.user_id = u.id
		LEFT JOIN players p ON p.id = up.player_id
		WHERE $1 = '' OR u.email ILIKE $2 OR p.name ILIKE $2
		ORDER BY u.email ASC, u.id ASC
		LIMIT $3 OFFSET $4
	`, query, pattern, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	accounts := make([]usecase.AccountOwnershipDTO, 0)
	for rows.Next() {
		var account usecase.AccountOwnershipDTO
		var playerID *entity.PlayerID
		var playerName *string
		if err := rows.Scan(
			&account.ID,
			&account.Email,
			&account.Role,
			&account.Status,
			&playerID,
			&playerName,
		); err != nil {
			return nil, 0, err
		}
		if playerID != nil && playerName != nil {
			account.Player = &usecase.PlayerDTO{ID: *playerID, Name: *playerName}
		}
		accounts = append(accounts, account)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return accounts, total, nil
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
) ([]usecase.AvailablePlayerDTO, error) {
	rows, err := tx.Query(context.Background(), `
		SELECT p.id, p.name,
			COUNT(DISTINCT o.session_id) AS sessions_count,
			MAX(o.created_at) AS last_played_at
		FROM players p
		LEFT JOIN user_players up ON up.player_id = p.id
		LEFT JOIN operations o ON o.player_id = p.id
		WHERE up.player_id IS NULL
		GROUP BY p.id, p.name
		ORDER BY p.name ASC, p.id ASC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]usecase.AvailablePlayerDTO, 0)
	for rows.Next() {
		var player usecase.AvailablePlayerDTO
		if err := rows.Scan(&player.ID, &player.Name, &player.SessionsCount, &player.LastPlayedAt); err != nil {
			return nil, err
		}
		result = append(result, player)
	}

	return result, rows.Err()
}

func mapOwnershipConstraintError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return err
	}
	switch pgErr.ConstraintName {
	case "uq_user_players_user_id":
		return entity.ErrAccountAlreadyLinked
	case "user_players_player_id_key":
		return entity.ErrPlayerAlreadyLinked
	default:
		return err
	}
}
