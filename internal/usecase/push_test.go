package usecase

import (
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ishee11/poc/internal/entity"
)

type fakeBlindClockPushRepo struct {
	upserted      []entity.BlindClockPushSubscription
	subscriptions []entity.BlindClockPushSubscription
}

func (r *fakeBlindClockPushRepo) UpsertSubscription(subscription entity.BlindClockPushSubscription) error {
	r.upserted = append(r.upserted, subscription)
	return nil
}

func (r *fakeBlindClockPushRepo) DeleteSubscription(_ string) error {
	return nil
}

func (r *fakeBlindClockPushRepo) GetSubscription(endpoint string) (*entity.BlindClockPushSubscription, error) {
	for _, subscription := range r.subscriptions {
		if subscription.Endpoint == endpoint {
			item := subscription
			return &item, nil
		}
	}
	return nil, pgx.ErrNoRows
}

func (r *fakeBlindClockPushRepo) ListSubscriptions() ([]entity.BlindClockPushSubscription, error) {
	return r.subscriptions, nil
}

func (r *fakeBlindClockPushRepo) HasEvent(_ string) (bool, error) {
	return false, nil
}

func (r *fakeBlindClockPushRepo) SaveEvent(_ entity.BlindClockPushEvent) error {
	return nil
}

type fakeBlindClockPushSender struct {
	err  error
	sent []entity.BlindClockPushSubscription
}

func (s *fakeBlindClockPushSender) SendTest(subscription entity.BlindClockPushSubscription) error {
	s.sent = append(s.sent, subscription)
	return s.err
}

func TestBlindClockPushServiceSubscribeRejectsFailedDelivery(t *testing.T) {
	repo := &fakeBlindClockPushRepo{}
	sender := &fakeBlindClockPushSender{err: errors.New("BadJwtToken")}
	service := NewBlindClockPushService(repo, sender, BlindClockPushConfig{
		Enabled:   true,
		PublicKey: "public",
	})

	err := service.Subscribe(validPushSubscriptionInput())
	if !errors.Is(err, entity.ErrPushDeliveryFailed) {
		t.Fatalf("subscribe error = %v, want %v", err, entity.ErrPushDeliveryFailed)
	}
	if len(sender.sent) != 1 {
		t.Fatalf("send test count = %d, want 1", len(sender.sent))
	}
	if len(repo.upserted) != 0 {
		t.Fatalf("subscription was saved after failed push delivery: %+v", repo.upserted)
	}
}

func TestBlindClockPushServiceSubscribeSavesAfterSuccessfulDelivery(t *testing.T) {
	repo := &fakeBlindClockPushRepo{}
	sender := &fakeBlindClockPushSender{}
	service := NewBlindClockPushService(repo, sender, BlindClockPushConfig{
		Enabled:   true,
		PublicKey: "public",
	})

	if err := service.Subscribe(validPushSubscriptionInput()); err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if len(sender.sent) != 1 {
		t.Fatalf("send test count = %d, want 1", len(sender.sent))
	}
	if len(repo.upserted) != 1 {
		t.Fatalf("saved subscription count = %d, want 1", len(repo.upserted))
	}
	if repo.upserted[0].Endpoint != "https://web.push.apple.com/test" {
		t.Fatalf("saved endpoint = %q", repo.upserted[0].Endpoint)
	}
	if repo.upserted[0].CreatedAt.IsZero() || repo.upserted[0].UpdatedAt.IsZero() {
		t.Fatalf("saved timestamps were not set")
	}
	if time.Since(repo.upserted[0].CreatedAt) > time.Minute {
		t.Fatalf("saved timestamp is unexpectedly old: %s", repo.upserted[0].CreatedAt)
	}
}

func validPushSubscriptionInput() BlindClockPushSubscriptionInput {
	return BlindClockPushSubscriptionInput{
		Endpoint:        "https://web.push.apple.com/test",
		KeyAuth:         "auth",
		KeyP256DH:       "p256dh",
		UserAgent:       "Safari",
		NotifyWarning60: true,
		NotifyWarning10: true,
	}
}
