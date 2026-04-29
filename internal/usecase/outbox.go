package usecase

import (
	"encoding/json"
	"time"
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
