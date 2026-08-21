package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
	"github.com/jackc/pgx/v5"
)

func (r *AuthRepository) SaveTelegramLoginChallenge(tx usecase.Tx, c *entity.TelegramLoginChallenge) error {
	_, _ = tx.Exec(context.Background(), `DELETE FROM telegram_login_challenges WHERE expires_at < $1`, c.CreatedAt.Add(-time.Hour))
	_, err := tx.Exec(context.Background(), `
		INSERT INTO telegram_login_challenges (
			challenge_hash, browser_binding_hash, verification_code, status, created_at, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`, c.ChallengeHash, c.BrowserBindingHash, c.VerificationCode, c.Status, c.CreatedAt, c.ExpiresAt)
	return err
}

func (r *AuthRepository) FindTelegramLoginChallenge(tx usecase.Tx, challengeHash, bindingHash string, now time.Time) (*entity.TelegramLoginChallenge, error) {
	return findTelegramLoginChallenge(tx, `
		SELECT challenge_hash, browser_binding_hash, verification_code, status,
			COALESCE(telegram_subject, ''), COALESCE(telegram_username, ''), COALESCE(telegram_name, ''),
			created_at, expires_at, approved_at, consumed_at
		FROM telegram_login_challenges
		WHERE challenge_hash = $1 AND browser_binding_hash = $2
	`, challengeHash, bindingHash, now)
}

func (r *AuthRepository) FindTelegramLoginChallengeForBot(tx usecase.Tx, challengeHash string, now time.Time) (*entity.TelegramLoginChallenge, error) {
	return findTelegramLoginChallenge(tx, `
		SELECT challenge_hash, browser_binding_hash, verification_code, status,
			COALESCE(telegram_subject, ''), COALESCE(telegram_username, ''), COALESCE(telegram_name, ''),
			created_at, expires_at, approved_at, consumed_at
		FROM telegram_login_challenges WHERE challenge_hash = $1
	`, challengeHash, now)
}

func (r *AuthRepository) ClaimTelegramLoginChallengeActor(tx usecase.Tx, hash, subject string, now time.Time) (*entity.TelegramLoginChallenge, error) {
	row := tx.QueryRow(context.Background(), `
		UPDATE telegram_login_challenges SET telegram_subject = $2
		WHERE challenge_hash = $1 AND status = 'pending' AND expires_at > $3
			AND (telegram_subject IS NULL OR telegram_subject = $2)
		RETURNING challenge_hash, browser_binding_hash, verification_code, status,
			telegram_subject, COALESCE(telegram_username, ''), COALESCE(telegram_name, ''),
			created_at, expires_at, approved_at, consumed_at
	`, hash, subject, now)
	var c entity.TelegramLoginChallenge
	if err := row.Scan(&c.ChallengeHash, &c.BrowserBindingHash, &c.VerificationCode, &c.Status,
		&c.TelegramSubject, &c.TelegramUsername, &c.TelegramName, &c.CreatedAt, &c.ExpiresAt,
		&c.ApprovedAt, &c.ConsumedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrTelegramChallengeActor
		}
		return nil, err
	}
	return &c, nil
}

func findTelegramLoginChallenge(tx usecase.Tx, query string, args ...any) (*entity.TelegramLoginChallenge, error) {
	var c entity.TelegramLoginChallenge
	err := tx.QueryRow(context.Background(), query, args[:len(args)-1]...).Scan(
		&c.ChallengeHash, &c.BrowserBindingHash, &c.VerificationCode, &c.Status,
		&c.TelegramSubject, &c.TelegramUsername, &c.TelegramName,
		&c.CreatedAt, &c.ExpiresAt, &c.ApprovedAt, &c.ConsumedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, entity.ErrTelegramChallengeInvalid
	}
	if err != nil {
		return nil, err
	}
	now := args[len(args)-1].(time.Time)
	if !c.ExpiresAt.After(now) && c.Status != entity.TelegramLoginChallengeConsumed {
		_, _ = tx.Exec(context.Background(), `
			UPDATE telegram_login_challenges SET status = 'expired'
			WHERE challenge_hash = $1 AND status IN ('pending', 'approved')
		`, c.ChallengeHash)
		c.Status = entity.TelegramLoginChallengeExpired
	}
	return &c, nil
}

func (r *AuthRepository) ApproveTelegramLoginChallenge(tx usecase.Tx, hash, subject, username, displayName string, now time.Time) (*entity.TelegramLoginChallenge, error) {
	c, err := r.FindTelegramLoginChallengeForBot(tx, hash, now)
	if err != nil {
		return nil, err
	}
	if c.Status == entity.TelegramLoginChallengeApproved && c.TelegramSubject == subject {
		return c, nil
	}
	if c.Status != entity.TelegramLoginChallengePending {
		return nil, entity.ErrTelegramChallengeState
	}
	row := tx.QueryRow(context.Background(), `
		UPDATE telegram_login_challenges
		SET status = 'approved', telegram_subject = $2, telegram_username = NULLIF($3, ''),
			telegram_name = NULLIF($4, ''), approved_at = $5
		WHERE challenge_hash = $1 AND status = 'pending' AND expires_at > $5
		RETURNING challenge_hash, browser_binding_hash, verification_code, status,
			telegram_subject, COALESCE(telegram_username, ''), COALESCE(telegram_name, ''),
			created_at, expires_at, approved_at, consumed_at
	`, hash, subject, username, displayName, now)
	var approved entity.TelegramLoginChallenge
	if err := row.Scan(&approved.ChallengeHash, &approved.BrowserBindingHash, &approved.VerificationCode,
		&approved.Status, &approved.TelegramSubject, &approved.TelegramUsername, &approved.TelegramName,
		&approved.CreatedAt, &approved.ExpiresAt, &approved.ApprovedAt, &approved.ConsumedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			current, findErr := r.FindTelegramLoginChallengeForBot(tx, hash, now)
			if findErr == nil && current.Status == entity.TelegramLoginChallengeApproved && current.TelegramSubject == subject {
				return current, nil
			}
			return nil, entity.ErrTelegramChallengeState
		}
		return nil, err
	}
	return &approved, nil
}

func (r *AuthRepository) DenyTelegramLoginChallenge(tx usecase.Tx, hash, subject string, now time.Time) (*entity.TelegramLoginChallenge, error) {
	c, err := r.FindTelegramLoginChallengeForBot(tx, hash, now)
	if err != nil {
		return nil, err
	}
	if c.Status == entity.TelegramLoginChallengeDenied && c.TelegramSubject == subject {
		return c, nil
	}
	if c.Status != entity.TelegramLoginChallengePending {
		return nil, entity.ErrTelegramChallengeState
	}
	row := tx.QueryRow(context.Background(), `
		UPDATE telegram_login_challenges SET status = 'denied', telegram_subject = $2
		WHERE challenge_hash = $1 AND status = 'pending' AND expires_at > $3
		RETURNING challenge_hash, browser_binding_hash, verification_code, status,
			telegram_subject, COALESCE(telegram_username, ''), COALESCE(telegram_name, ''),
			created_at, expires_at, approved_at, consumed_at
	`, hash, subject, now)
	var denied entity.TelegramLoginChallenge
	if err := row.Scan(&denied.ChallengeHash, &denied.BrowserBindingHash, &denied.VerificationCode,
		&denied.Status, &denied.TelegramSubject, &denied.TelegramUsername, &denied.TelegramName,
		&denied.CreatedAt, &denied.ExpiresAt, &denied.ApprovedAt, &denied.ConsumedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			current, findErr := r.FindTelegramLoginChallengeForBot(tx, hash, now)
			if findErr == nil && current.Status == entity.TelegramLoginChallengeDenied && current.TelegramSubject == subject {
				return current, nil
			}
			return nil, entity.ErrTelegramChallengeState
		}
		return nil, err
	}
	return &denied, nil
}

func (r *AuthRepository) LockTelegramLoginChallenge(tx usecase.Tx, hash, bindingHash string, now time.Time) (*entity.TelegramLoginChallenge, error) {
	return findTelegramLoginChallenge(tx, `
		SELECT challenge_hash, browser_binding_hash, verification_code, status,
			COALESCE(telegram_subject, ''), COALESCE(telegram_username, ''), COALESCE(telegram_name, ''),
			created_at, expires_at, approved_at, consumed_at
		FROM telegram_login_challenges
		WHERE challenge_hash = $1 AND browser_binding_hash = $2 FOR UPDATE
	`, hash, bindingHash, now)
}

func (r *AuthRepository) ConsumeTelegramLoginChallenge(tx usecase.Tx, hash string, now time.Time) error {
	tag, err := tx.Exec(context.Background(), `
		UPDATE telegram_login_challenges SET status = 'consumed', consumed_at = $2
		WHERE challenge_hash = $1 AND status = 'approved' AND expires_at > $2
	`, hash, now)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return entity.ErrTelegramChallengeState
	}
	return nil
}
