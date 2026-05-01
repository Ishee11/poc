package usecase

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ishee11/poc/internal/entity"
)

const (
	OutboxEventOperationCreated  = "operation.created"
	OutboxEventOperationReversed = "operation.reversed"
	OutboxEventSessionFinished   = "session.finished"

	OutboxAggregateOperation = "operation"
	OutboxAggregateSession   = "session"
)

type OutboxEvent struct {
	ID            string
	EventType     string
	AggregateType string
	AggregateID   string
	Payload       json.RawMessage
	CreatedAt     time.Time
}

type OutboxDispatchResult struct {
	Published int
	Failed    int
}

type OutboxWriter interface {
	Save(tx Tx, event OutboxEvent) error
}

type OutboxRelayRepository interface {
	FetchPending(tx Tx, limit int) ([]OutboxEvent, error)
	MarkPublished(tx Tx, eventID string, publishedAt time.Time) error
	MarkFailed(tx Tx, eventID string, err error, availableAt time.Time) error
}

type OutboxPublisher interface {
	Publish(ctx context.Context, event OutboxEvent) error
}

type OutboxRelay struct {
	repo      OutboxRelayRepository
	txManager TxManager
	publisher OutboxPublisher
	clock     Clock
}

func NewOutboxRelay(
	repo OutboxRelayRepository,
	txManager TxManager,
	publisher OutboxPublisher,
	clock Clock,
) *OutboxRelay {
	return &OutboxRelay{
		repo:      repo,
		txManager: txManager,
		publisher: publisher,
		clock:     clock,
	}
}

func (r *OutboxRelay) DispatchOnce(ctx context.Context, limit int) (OutboxDispatchResult, error) {
	if limit <= 0 {
		limit = 100
	}

	result := OutboxDispatchResult{}
	err := r.txManager.RunInTx(ctx, func(tx Tx) error {
		events, err := r.repo.FetchPending(tx, limit)
		if err != nil {
			return err
		}

		for _, event := range events {
			if err := r.publisher.Publish(ctx, event); err != nil {
				result.Failed++
				if markErr := r.repo.MarkFailed(tx, event.ID, err, r.clock.Now().Add(time.Minute)); markErr != nil {
					return markErr
				}
				continue
			}

			result.Published++
			if err := r.repo.MarkPublished(tx, event.ID, r.clock.Now()); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return OutboxDispatchResult{}, err
	}

	return result, nil
}

type operationOutboxPayload struct {
	OperationID          entity.OperationID   `json:"operation_id"`
	RequestID            string               `json:"request_id"`
	SessionID            entity.SessionID     `json:"session_id"`
	PlayerID             entity.PlayerID      `json:"player_id"`
	Type                 entity.OperationType `json:"type"`
	Chips                int64                `json:"chips"`
	ReferenceOperationID *entity.OperationID  `json:"reference_operation_id,omitempty"`
	CreatedAt            time.Time            `json:"created_at"`
}

type sessionFinishedOutboxPayload struct {
	RequestID  string           `json:"request_id"`
	SessionID  entity.SessionID `json:"session_id"`
	FinishedAt time.Time        `json:"finished_at"`
}

func NewOperationCreatedOutboxEvent(op *entity.Operation) (OutboxEvent, error) {
	return newOperationOutboxEvent(OutboxEventOperationCreated, op)
}

func NewOperationReversedOutboxEvent(op *entity.Operation) (OutboxEvent, error) {
	return newOperationOutboxEvent(OutboxEventOperationReversed, op)
}

func NewSessionFinishedOutboxEvent(requestID string, sessionID entity.SessionID, finishedAt time.Time) (OutboxEvent, error) {
	payload, err := json.Marshal(sessionFinishedOutboxPayload{
		RequestID:  requestID,
		SessionID:  sessionID,
		FinishedAt: finishedAt,
	})
	if err != nil {
		return OutboxEvent{}, err
	}

	return OutboxEvent{
		ID:            OutboxEventSessionFinished + ":" + string(sessionID),
		EventType:     OutboxEventSessionFinished,
		AggregateType: OutboxAggregateSession,
		AggregateID:   string(sessionID),
		Payload:       payload,
		CreatedAt:     finishedAt,
	}, nil
}

func newOperationOutboxEvent(eventType string, op *entity.Operation) (OutboxEvent, error) {
	payload, err := json.Marshal(operationOutboxPayload{
		OperationID:          op.ID(),
		RequestID:            op.RequestID(),
		SessionID:            op.SessionID(),
		PlayerID:             op.PlayerID(),
		Type:                 op.Type(),
		Chips:                op.Chips(),
		ReferenceOperationID: op.ReferenceID(),
		CreatedAt:            op.CreatedAt(),
	})
	if err != nil {
		return OutboxEvent{}, err
	}

	return OutboxEvent{
		ID:            eventType + ":" + string(op.ID()),
		EventType:     eventType,
		AggregateType: OutboxAggregateOperation,
		AggregateID:   string(op.ID()),
		Payload:       payload,
		CreatedAt:     op.CreatedAt(),
	}, nil
}
