package postgres

import (
	"context"
	"time"

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

func (r *OutboxRepository) FetchPending(tx usecase.Tx, limit int) ([]usecase.OutboxEvent, error) {
	ctx := context.Background()

	rows, err := tx.Query(ctx, `
		SELECT id, event_type, aggregate_type, aggregate_id, payload, created_at
		FROM outbox_events
		WHERE published_at IS NULL
		  AND available_at <= now()
		ORDER BY created_at ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]usecase.OutboxEvent, 0)
	for rows.Next() {
		var event usecase.OutboxEvent
		if err := rows.Scan(
			&event.ID,
			&event.EventType,
			&event.AggregateType,
			&event.AggregateID,
			&event.Payload,
			&event.CreatedAt,
		); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

func (r *OutboxRepository) MarkPublished(tx usecase.Tx, eventID string, publishedAt time.Time) error {
	ctx := context.Background()

	_, err := tx.Exec(ctx, `
		UPDATE outbox_events
		SET published_at = $2,
		    last_error = NULL
		WHERE id = $1
	`, eventID, publishedAt)

	return err
}

func (r *OutboxRepository) MarkFailed(tx usecase.Tx, eventID string, publishErr error, availableAt time.Time) error {
	ctx := context.Background()

	_, err := tx.Exec(ctx, `
		UPDATE outbox_events
		SET attempts = attempts + 1,
		    available_at = $2,
		    last_error = $3
		WHERE id = $1
	`, eventID, availableAt, publishErr.Error())

	return err
}
