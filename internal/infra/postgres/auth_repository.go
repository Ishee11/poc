package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
)

type AuthRepository struct{}

func NewAuthRepository() *AuthRepository {
	return &AuthRepository{}
}

func (r *AuthRepository) SaveIdentity(tx usecase.Tx, identity *entity.AuthIdentity) error {
	_, err := tx.Exec(context.Background(), `
		INSERT INTO auth_identities (
			provider, subject, user_id, username, display_name, picture_url, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, identity.Provider, identity.Subject, identity.UserID, nullString(identity.Username),
		nullString(identity.DisplayName), nullString(identity.PictureURL), identity.CreatedAt, identity.UpdatedAt)
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		if pgErr.ConstraintName == "auth_identities_provider_user_id_key" {
			return entity.ErrAuthProviderLinked
		}
		return entity.ErrAuthIdentityLinked
	}
	return err
}

func (r *AuthRepository) ReplaceIdentitySubject(tx usecase.Tx, provider entity.AuthProvider, oldSubject string, identity *entity.AuthIdentity) error {
	tag, err := tx.Exec(context.Background(), `
		UPDATE auth_identities
		SET subject = $1, username = $2, display_name = $3, picture_url = $4, updated_at = $5
		WHERE provider = $6 AND subject = $7 AND user_id = $8
	`, identity.Subject, nullString(identity.Username), nullString(identity.DisplayName),
		nullString(identity.PictureURL), identity.UpdatedAt, provider, oldSubject, identity.UserID)
	if err == nil {
		if tag.RowsAffected() == 0 {
			return entity.ErrAuthIdentityNotFound
		}
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return entity.ErrAuthIdentityLinked
	}
	return err
}

func (r *AuthRepository) FindIdentity(tx usecase.Tx, provider entity.AuthProvider, subject string) (*entity.AuthIdentity, error) {
	row := tx.QueryRow(context.Background(), `
		SELECT provider, subject, user_id, COALESCE(username, ''), COALESCE(display_name, ''),
			COALESCE(picture_url, ''), created_at, updated_at
		FROM auth_identities WHERE provider = $1 AND subject = $2
	`, provider, subject)
	var identity entity.AuthIdentity
	if err := row.Scan(&identity.Provider, &identity.Subject, &identity.UserID, &identity.Username,
		&identity.DisplayName, &identity.PictureURL, &identity.CreatedAt, &identity.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrAuthIdentityNotFound
		}
		return nil, err
	}
	return &identity, nil
}

func (r *AuthRepository) ListIdentities(tx usecase.Tx, userID entity.AuthUserID) ([]entity.AuthIdentity, error) {
	rows, err := tx.Query(context.Background(), `
		SELECT provider, subject, user_id, COALESCE(username, ''), COALESCE(display_name, ''),
			COALESCE(picture_url, ''), created_at, updated_at
		FROM auth_identities WHERE user_id = $1 ORDER BY provider
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	identities := make([]entity.AuthIdentity, 0)
	for rows.Next() {
		var identity entity.AuthIdentity
		if err := rows.Scan(&identity.Provider, &identity.Subject, &identity.UserID, &identity.Username,
			&identity.DisplayName, &identity.PictureURL, &identity.CreatedAt, &identity.UpdatedAt); err != nil {
			return nil, err
		}
		identities = append(identities, identity)
	}
	return identities, rows.Err()
}

func (r *AuthRepository) DeleteIdentity(tx usecase.Tx, userID entity.AuthUserID, provider entity.AuthProvider) error {
	tag, err := tx.Exec(context.Background(), `DELETE FROM auth_identities WHERE user_id = $1 AND provider = $2`, userID, provider)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return entity.ErrAuthIdentityNotFound
	}
	return nil
}

func (r *AuthRepository) SaveOIDCFlow(tx usecase.Tx, flow *entity.AuthOIDCFlow) error {
	_, err := tx.Exec(context.Background(), `
		INSERT INTO auth_oidc_flows (
			state_hash, mode, user_id, code_verifier, nonce, redirect_uri, created_at, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, flow.StateHash, flow.Mode, flow.UserID, flow.CodeVerifier, flow.Nonce, flow.RedirectURI,
		flow.CreatedAt, flow.ExpiresAt)
	return err
}

func (r *AuthRepository) ConsumeOIDCFlow(tx usecase.Tx, stateHash string, now time.Time) (*entity.AuthOIDCFlow, error) {
	row := tx.QueryRow(context.Background(), `
		DELETE FROM auth_oidc_flows
		WHERE state_hash = $1 AND expires_at > $2
		RETURNING state_hash, mode, user_id, code_verifier, nonce, redirect_uri, created_at, expires_at
	`, stateHash, now)
	var flow entity.AuthOIDCFlow
	if err := row.Scan(&flow.StateHash, &flow.Mode, &flow.UserID, &flow.CodeVerifier, &flow.Nonce,
		&flow.RedirectURI, &flow.CreatedAt, &flow.ExpiresAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrOIDCFlowInvalid
		}
		return nil, err
	}
	return &flow, nil
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
