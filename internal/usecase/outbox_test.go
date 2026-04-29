package usecase

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeOutboxRelayRepo struct {
	events       []OutboxEvent
	publishedIDs []string
	failedIDs    []string
}

func (r *fakeOutboxRelayRepo) FetchPending(_ Tx, _ int) ([]OutboxEvent, error) {
	return r.events, nil
}

func (r *fakeOutboxRelayRepo) MarkPublished(_ Tx, eventID string, _ time.Time) error {
	r.publishedIDs = append(r.publishedIDs, eventID)
	return nil
}

func (r *fakeOutboxRelayRepo) MarkFailed(_ Tx, eventID string, _ error, _ time.Time) error {
	r.failedIDs = append(r.failedIDs, eventID)
	return nil
}

type fakeOutboxPublisher struct {
	fail map[string]bool
}

func (p fakeOutboxPublisher) Publish(_ context.Context, event OutboxEvent) error {
	if p.fail[event.ID] {
		return errors.New("publish failed")
	}
	return nil
}

func TestOutboxRelay_DispatchOnce(t *testing.T) {
	repo := &fakeOutboxRelayRepo{
		events: []OutboxEvent{
			{ID: "evt1", EventType: OutboxEventOperationCreated},
			{ID: "evt2", EventType: OutboxEventOperationCreated},
		},
	}
	relay := NewOutboxRelay(
		repo,
		fakeTxManager{},
		fakeOutboxPublisher{fail: map[string]bool{"evt2": true}},
		fakeClock{now: time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)},
	)

	result, err := relay.DispatchOnce(context.Background(), 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Published != 1 || result.Failed != 1 {
		t.Fatalf("unexpected dispatch result: %+v", result)
	}
	if len(repo.publishedIDs) != 1 || repo.publishedIDs[0] != "evt1" {
		t.Fatalf("unexpected published ids: %+v", repo.publishedIDs)
	}
	if len(repo.failedIDs) != 1 || repo.failedIDs[0] != "evt2" {
		t.Fatalf("unexpected failed ids: %+v", repo.failedIDs)
	}
}
