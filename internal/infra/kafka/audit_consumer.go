package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	kafkago "github.com/segmentio/kafka-go"

	"github.com/ishee11/poc/internal/infra/postgres"
)

type AuditEventHandler interface {
	Save(ctx context.Context, event postgres.AuditEvent) error
}

type AuditConsumer struct {
	reader  *kafkago.Reader
	handler AuditEventHandler
}

func NewAuditConsumer(brokers []string, topic string, groupID string, handler AuditEventHandler) *AuditConsumer {
	return &AuditConsumer{
		reader: kafkago.NewReader(kafkago.ReaderConfig{
			Brokers: brokers,
			Topic:   topic,
			GroupID: groupID,
		}),
		handler: handler,
	}
}

func (c *AuditConsumer) Run(ctx context.Context) error {
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			return err
		}

		event, err := auditEventFromMessage(msg)
		if err != nil {
			return err
		}

		if err := c.handler.Save(ctx, event); err != nil {
			return err
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			return err
		}
	}
}

func (c *AuditConsumer) Close() error {
	return c.reader.Close()
}

func auditEventFromMessage(msg kafkago.Message) (postgres.AuditEvent, error) {
	event := postgres.AuditEvent{
		Payload:    json.RawMessage(msg.Value),
		ConsumedAt: time.Now(),
	}

	for _, header := range msg.Headers {
		switch header.Key {
		case "event_id":
			event.EventID = string(header.Value)
		case "event_type":
			event.EventType = string(header.Value)
		case "aggregate_type":
			event.AggregateType = string(header.Value)
		case "aggregate_id":
			event.AggregateID = string(header.Value)
		}
	}

	if event.EventID == "" || event.EventType == "" || event.AggregateType == "" || event.AggregateID == "" {
		return postgres.AuditEvent{}, fmt.Errorf("kafka message is missing audit headers")
	}
	if !json.Valid(event.Payload) {
		return postgres.AuditEvent{}, fmt.Errorf("kafka message payload is not valid json")
	}

	return event, nil
}
