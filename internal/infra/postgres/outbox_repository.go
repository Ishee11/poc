package postgres

import (
	"context"

	"github.com/ishee11/poc/internal/usecase"
)

type OutboxRepository struct{}

func NewOutboxRepository() *OutboxRepository {
	return &OutboxRepository{}
}

func (r *OutboxRepository) Save(tx usecase.Tx, event usecase.OutboxEvent) error {
	ctx := context.Background()

	_, err := tx.Exec(ctx, `
		INSERT INTO outbox_events (
			id,
			event_type,
			aggregate_type,
			aggregate_id,
			payload,
			created_at
		)
		VALUES ($1,$2,$3,$4,$5,$6)
	`,
		event.ID,
		event.EventType,
		event.AggregateType,
		event.AggregateID,
		event.Payload,
		event.CreatedAt,
	)

	return err
}
