package usecase

import (
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

type OutboxWriter interface {
	Save(tx Tx, event OutboxEvent) error
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
