package kafka

import (
	"context"
	"time"

	kafkago "github.com/segmentio/kafka-go"

	"github.com/ishee11/poc/internal/usecase"
)

type OutboxPublisher struct {
	writer *kafkago.Writer
}

func NewOutboxPublisher(brokers []string, topic string) *OutboxPublisher {
	return &OutboxPublisher{
		writer: &kafkago.Writer{
			Addr:         kafkago.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafkago.Hash{},
			BatchTimeout: 10 * time.Millisecond,
		},
	}
}

func (p *OutboxPublisher) Publish(ctx context.Context, event usecase.OutboxEvent) error {
	return p.writer.WriteMessages(ctx, kafkago.Message{
		Key:   []byte(event.AggregateID),
		Value: event.Payload,
		Headers: []kafkago.Header{
			{Key: "event_id", Value: []byte(event.ID)},
			{Key: "event_type", Value: []byte(event.EventType)},
			{Key: "aggregate_type", Value: []byte(event.AggregateType)},
			{Key: "aggregate_id", Value: []byte(event.AggregateID)},
			{Key: "created_at", Value: []byte(event.CreatedAt.Format(time.RFC3339Nano))},
		},
	})
}

func (p *OutboxPublisher) Close() error {
	return p.writer.Close()
}
