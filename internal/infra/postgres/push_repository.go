package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ishee11/poc/internal/entity"
)

type BlindClockPushRepository struct {
	pool *pgxpool.Pool
}

func NewBlindClockPushRepository(pool *pgxpool.Pool) *BlindClockPushRepository {
	return &BlindClockPushRepository{pool: pool}
}

func (r *BlindClockPushRepository) UpsertSubscription(subscription entity.BlindClockPushSubscription) error {
	_, err := r.pool.Exec(context.Background(), `
		INSERT INTO blind_clock_push_subscriptions (
			endpoint, key_auth, key_p256dh, user_agent, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (endpoint) DO UPDATE SET
			key_auth = EXCLUDED.key_auth,
			key_p256dh = EXCLUDED.key_p256dh,
			user_agent = EXCLUDED.user_agent,
			updated_at = EXCLUDED.updated_at
	`,
		subscription.Endpoint,
		subscription.KeyAuth,
		subscription.KeyP256DH,
		subscription.UserAgent,
		subscription.CreatedAt,
		subscription.UpdatedAt,
	)

	return err
}

func (r *BlindClockPushRepository) DeleteSubscription(endpoint string) error {
	_, err := r.pool.Exec(context.Background(), `
		DELETE FROM blind_clock_push_subscriptions
		WHERE endpoint = $1
	`, endpoint)
	return err
}

func (r *BlindClockPushRepository) ListSubscriptions() ([]entity.BlindClockPushSubscription, error) {
	rows, err := r.pool.Query(context.Background(), `
		SELECT endpoint, key_auth, key_p256dh, user_agent, created_at, updated_at
		FROM blind_clock_push_subscriptions
		ORDER BY updated_at DESC, created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []entity.BlindClockPushSubscription
	for rows.Next() {
		var item entity.BlindClockPushSubscription
		if err := rows.Scan(
			&item.Endpoint,
			&item.KeyAuth,
			&item.KeyP256DH,
			&item.UserAgent,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}

	return out, rows.Err()
}

func (r *BlindClockPushRepository) HasEvent(eventKey string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(context.Background(), `
		SELECT EXISTS(
			SELECT 1
			FROM blind_clock_push_events
			WHERE event_key = $1
		)
	`, eventKey).Scan(&exists)

	return exists, err
}

func (r *BlindClockPushRepository) SaveEvent(event entity.BlindClockPushEvent) error {
	_, err := r.pool.Exec(context.Background(), `
		INSERT INTO blind_clock_push_events (event_key, clock_id, event_kind, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (event_key) DO NOTHING
	`,
		event.EventKey,
		event.ClockID,
		event.EventKind,
		event.CreatedAt,
	)
	return err
}
