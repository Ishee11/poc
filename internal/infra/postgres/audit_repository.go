package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditEvent struct {
	EventID       string
	EventType     string
	AggregateType string
	AggregateID   string
	Payload       json.RawMessage
	ConsumedAt    time.Time
}

type AuditRepository struct {
	pool *pgxpool.Pool
}

func NewAuditRepository(pool *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{pool: pool}
}

func (r *AuditRepository) Save(ctx context.Context, event AuditEvent) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO audit_events (
			event_id,
			event_type,
			aggregate_type,
			aggregate_id,
			payload,
			consumed_at
		)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (event_id) DO NOTHING
	`,
		event.EventID,
		event.EventType,
		event.AggregateType,
		event.AggregateID,
		event.Payload,
		event.ConsumedAt,
	)

	return err
}
