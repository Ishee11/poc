package kafka

import (
	"testing"

	kafkago "github.com/segmentio/kafka-go"
)

func TestAuditEventFromMessage(t *testing.T) {
	event, err := auditEventFromMessage(kafkago.Message{
		Value: []byte(`{"operation_id":"op1"}`),
		Headers: []kafkago.Header{
			{Key: "event_id", Value: []byte("evt1")},
			{Key: "event_type", Value: []byte("operation.created")},
			{Key: "aggregate_type", Value: []byte("operation")},
			{Key: "aggregate_id", Value: []byte("op1")},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.EventID != "evt1" || event.EventType != "operation.created" || event.AggregateID != "op1" {
		t.Fatalf("unexpected audit event: %+v", event)
	}
}

func TestAuditEventFromMessage_RequiresHeaders(t *testing.T) {
	_, err := auditEventFromMessage(kafkago.Message{Value: []byte(`{}`)})
	if err == nil {
		t.Fatal("expected missing headers error")
	}
}
