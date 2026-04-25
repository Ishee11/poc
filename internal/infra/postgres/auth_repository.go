package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
)

type AuthRepository struct{}

func NewAuthRepository() *AuthRepository {
	return &AuthRepository{}
}

func (r *AuthRepository) Save(tx usecase.Tx, user *entity.AuthUser) error {
	_, err := tx.Exec(context.Background(), `
		INSERT INTO users (
			id, email, password_hash, role, status, created_at, updated_at, last_login_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			email = EXCLUDED.email,
			password_hash = EXCLUDED.password_hash,
			role = EXCLUDED.role,
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at,
			last_login_at = EXCLUDED.last_login_at
	`,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.Role,
		user.Status,
		user.CreatedAt,
		user.UpdatedAt,
		user.LastLoginAt,
	)

	return err
}

func (r *AuthRepository) FindUserByID(tx usecase.Tx, id entity.AuthUserID) (*entity.AuthUser, error) {
	row := tx.QueryRow(context.Background(), `
		SELECT id, email, password_hash, role, status, created_at, updated_at, last_login_at
		FROM users
		WHERE id = $1
	`, id)

	return scanAuthUser(row)
}

func (r *AuthRepository) FindUserByEmail(tx usecase.Tx, email string) (*entity.AuthUser, error) {
	row := tx.QueryRow(context.Background(), `
		SELECT id, email, password_hash, role, status, created_at, updated_at, last_login_at
		FROM users
		WHERE lower(email) = lower($1)
	`, email)

	return scanAuthUser(row)
}

func scanAuthUser(row pgx.Row) (*entity.AuthUser, error) {
	var (
		id           entity.AuthUserID
		email        string
		passwordHash string
		role         entity.AuthRole
		status       entity.AuthUserStatus
		createdAt    time.Time
		updatedAt    time.Time
		lastLoginAt  *time.Time
	)

	err := row.Scan(
		&id,
		&email,
		&passwordHash,
		&role,
		&status,
		&createdAt,
		&updatedAt,
		&lastLoginAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrAuthUserNotFound
		}
		return nil, err
	}

	return entity.RestoreAuthUser(
		id,
		email,
		passwordHash,
		role,
		status,
		createdAt,
		updatedAt,
		lastLoginAt,
	)
}

func (r *AuthRepository) UpdateLastLoginAt(tx usecase.Tx, id entity.AuthUserID, at time.Time) error {
	tag, err := tx.Exec(context.Background(), `
		UPDATE users
		SET last_login_at = $2, updated_at = $2
		WHERE id = $1
	`, id, at)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return entity.ErrAuthUserNotFound
	}
	return nil
}

func (r *AuthRepository) SaveSession(tx usecase.Tx, session *entity.AuthSession) error {
	_, err := tx.Exec(context.Background(), `
		INSERT INTO auth_sessions (
			id, user_id, token_hash, user_agent, ip, created_at, last_seen_at, expires_at, revoked_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			token_hash = EXCLUDED.token_hash,
			user_agent = EXCLUDED.user_agent,
			ip = EXCLUDED.ip,
			last_seen_at = EXCLUDED.last_seen_at,
			expires_at = EXCLUDED.expires_at,
			revoked_at = EXCLUDED.revoked_at
	`,
		session.ID,
		session.UserID,
		session.TokenHash,
		nullString(session.UserAgent),
		nullString(session.IP),
		session.CreatedAt,
		session.LastSeenAt,
		session.ExpiresAt,
		session.RevokedAt,
	)

	return err
}

func (r *AuthRepository) FindSessionByTokenHash(tx usecase.Tx, tokenHash string) (*entity.AuthSession, error) {
	row := tx.QueryRow(context.Background(), `
		SELECT id, user_id, token_hash, COALESCE(user_agent, ''), COALESCE(ip, ''),
			created_at, last_seen_at, expires_at, revoked_at
		FROM auth_sessions
		WHERE token_hash = $1 AND revoked_at IS NULL
	`, tokenHash)

	var session entity.AuthSession
	err := row.Scan(
		&session.ID,
		&session.UserID,
		&session.TokenHash,
		&session.UserAgent,
		&session.IP,
		&session.CreatedAt,
		&session.LastSeenAt,
		&session.ExpiresAt,
		&session.RevokedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrAuthSessionNotFound
		}
		return nil, err
	}

	return &session, nil
}

func (r *AuthRepository) TouchSession(tx usecase.Tx, id entity.AuthSessionID, lastSeenAt time.Time) error {
	tag, err := tx.Exec(context.Background(), `
		UPDATE auth_sessions
		SET last_seen_at = $2
		WHERE id = $1 AND revoked_at IS NULL
	`, id, lastSeenAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return entity.ErrAuthSessionNotFound
	}
	return nil
}

func (r *AuthRepository) RevokeSession(tx usecase.Tx, id entity.AuthSessionID, revokedAt time.Time) error {
	tag, err := tx.Exec(context.Background(), `
		UPDATE auth_sessions
		SET revoked_at = $2
		WHERE id = $1 AND revoked_at IS NULL
	`, id, revokedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return entity.ErrAuthSessionNotFound
	}
	return nil
}

func (r *AuthRepository) RevokeSessionByTokenHash(tx usecase.Tx, tokenHash string, revokedAt time.Time) error {
	tag, err := tx.Exec(context.Background(), `
		UPDATE auth_sessions
		SET revoked_at = $2
		WHERE token_hash = $1 AND revoked_at IS NULL
	`, tokenHash, revokedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return entity.ErrAuthSessionNotFound
	}
	return nil
}

func (r *AuthRepository) SaveLoginAttempt(tx usecase.Tx, attempt *entity.LoginAttempt) error {
	_, err := tx.Exec(context.Background(), `
		INSERT INTO login_attempts (id, email, ip, success, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`,
		attempt.ID,
		attempt.Email,
		nullString(attempt.IP),
		attempt.Success,
		attempt.CreatedAt,
	)

	return err
}

func (r *AuthRepository) CountFailedLoginAttempts(tx usecase.Tx, email string, ip string, since time.Time) (int, error) {
	row := tx.QueryRow(context.Background(), `
		SELECT COUNT(*)
		FROM login_attempts
		WHERE success = false
			AND created_at >= $1
			AND (lower(email) = lower($2) OR ip = $3)
	`, since, email, ip)

	var count int
	if err := row.Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

func nullString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
