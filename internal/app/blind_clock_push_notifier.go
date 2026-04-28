package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ishee11/poc/internal/entity"
	postgres "github.com/ishee11/poc/internal/infra/postgres"
)

const blindClockPushNotifierLock int64 = 41100801

type BlindClockPushNotifier struct {
	pool         *pgxpool.Pool
	clockRepo    *postgres.BlindClockRepository
	pushRepo     *postgres.BlindClockPushRepository
	subject      string
	publicKey    string
	privateKey   string
	warnings     []int64
	pollInterval time.Duration
}

func NewBlindClockPushNotifier(
	pool *pgxpool.Pool,
	clockRepo *postgres.BlindClockRepository,
	pushRepo *postgres.BlindClockPushRepository,
	cfg PushConfig,
) *BlindClockPushNotifier {
	if !cfg.Enabled || cfg.Subject == "" || cfg.PublicKey == "" || cfg.PrivateKey == "" {
		return nil
	}

	interval := cfg.PollInterval
	if interval <= 0 {
		interval = time.Second
	}

	return &BlindClockPushNotifier{
		pool:         pool,
		clockRepo:    clockRepo,
		pushRepo:     pushRepo,
		subject:      cfg.Subject,
		publicKey:    cfg.PublicKey,
		privateKey:   cfg.PrivateKey,
		warnings:     cfg.Warnings,
		pollInterval: interval,
	}
}

func (n *BlindClockPushNotifier) Start(ctx context.Context) {
	if n == nil {
		return
	}

	go n.loop(ctx)
}

func (n *BlindClockPushNotifier) loop(ctx context.Context) {
	ticker := time.NewTicker(n.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := n.tick(ctx); err != nil {
				slog.Warn("blind_clock_push_tick_failed", "err", err)
			}
		}
	}
}

func (n *BlindClockPushNotifier) tick(ctx context.Context) error {
	conn, err := n.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var locked bool
	if err := tx.QueryRow(ctx, `SELECT pg_try_advisory_xact_lock($1)`, blindClockPushNotifierLock).Scan(&locked); err != nil {
		return err
	}
	if !locked {
		return nil
	}

	clock, err := n.clockRepo.FindLatestForUpdate(tx)
	if err == entity.ErrBlindClockNotFound {
		return tx.Commit(ctx)
	}
	if err != nil {
		return err
	}

	now := time.Now()
	if clock.Sync(now) {
		if err := n.clockRepo.Save(tx, clock); err != nil {
			return err
		}
	}

	snapshot := clock.Snapshot(now)
	levels := clock.Levels()
	events := n.collectEvents(clock.ID(), snapshot, levels)

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	for _, event := range events {
		exists, err := n.pushRepo.HasEvent(event.EventKey)
		if err != nil || exists {
			if err != nil {
				slog.Warn("blind_clock_push_event_exists_failed", "event_key", event.EventKey, "err", err)
			}
			continue
		}

		if err := n.deliverEvent(ctx, event); err != nil {
			slog.Warn("blind_clock_push_delivery_failed", "event_key", event.EventKey, "err", err)
			continue
		}

		if err := n.pushRepo.SaveEvent(entity.BlindClockPushEvent{
			ClockID:   clock.ID(),
			EventKey:  event.EventKey,
			EventKind: event.Kind,
			CreatedAt: now,
		}); err != nil {
			slog.Warn("blind_clock_push_event_save_failed", "event_key", event.EventKey, "err", err)
		}
	}

	return nil
}

type blindClockPushEventPayload struct {
	Kind  string `json:"kind"`
	Title string `json:"title"`
	Body  string `json:"body"`
	Tag   string `json:"tag"`
	URL   string `json:"url"`
}

type blindClockDueEvent struct {
	EventKey string
	Kind     string
	Payload  blindClockPushEventPayload
}

func (n *BlindClockPushNotifier) collectEvents(
	clockID entity.BlindClockID,
	snapshot entity.BlindClockSnapshot,
	levels []entity.BlindClockLevel,
) []blindClockDueEvent {
	if len(levels) == 0 || snapshot.CurrentLevelIndex < 0 || snapshot.CurrentLevelIndex >= len(levels) {
		return nil
	}

	current := levels[snapshot.CurrentLevelIndex]
	levelNumber := snapshot.CurrentLevelIndex + 1
	var out []blindClockDueEvent
	window := n.eventWindowSeconds()

	if snapshot.Status == entity.BlindClockStatusRunning && withinEventWindow(snapshot.RemainingSeconds, current.DurationSeconds, window) {
		out = append(out, blindClockDueEvent{
			EventKey: fmt.Sprintf("level-start:%s:%d", clockID, levelNumber),
			Kind:     "level_started",
			Payload: blindClockPushEventPayload{
				Kind:  "level_started",
				Title: fmt.Sprintf("Level %d started", levelNumber),
				Body:  fmt.Sprintf("Blinds %d / %d", current.SmallBlind, current.BigBlind),
				Tag:   fmt.Sprintf("blind-clock-level-%d", levelNumber),
				URL:   "/blinds/presentation",
			},
		})
	}

	if snapshot.Status == entity.BlindClockStatusRunning {
		for _, warning := range n.warnings {
			if !withinEventWindow(snapshot.RemainingSeconds, warning, window) {
				continue
			}

			out = append(out, blindClockDueEvent{
				EventKey: fmt.Sprintf("warning:%s:%d:%d", clockID, levelNumber, warning),
				Kind:     "warning",
				Payload: blindClockPushEventPayload{
					Kind:  "warning",
					Title: fmt.Sprintf("%d seconds left", warning),
					Body:  fmt.Sprintf("Level %d · %d / %d", levelNumber, current.SmallBlind, current.BigBlind),
					Tag:   fmt.Sprintf("blind-clock-warning-%d", warning),
					URL:   "/blinds/presentation",
				},
			})
		}
	}

	if snapshot.Status == entity.BlindClockStatusFinished {
		out = append(out, blindClockDueEvent{
			EventKey: fmt.Sprintf("finished:%s", clockID),
			Kind:     "finished",
			Payload: blindClockPushEventPayload{
				Kind:  "finished",
				Title: "Blind structure finished",
				Body:  fmt.Sprintf("Last level %d · %d / %d", levelNumber, current.SmallBlind, current.BigBlind),
				Tag:   "blind-clock-finished",
				URL:   "/blinds/presentation",
			},
		})
	}

	return out
}

func (n *BlindClockPushNotifier) eventWindowSeconds() int64 {
	window := int64(n.pollInterval / time.Second)
	if n.pollInterval%time.Second != 0 {
		window += 1
	}
	if window < 3 {
		window = 3
	}
	return window
}

func withinEventWindow(remaining, target, window int64) bool {
	if target < 0 || window <= 0 {
		return false
	}

	lowerBound := target - window + 1
	if lowerBound < 0 {
		lowerBound = 0
	}

	return remaining <= target && remaining >= lowerBound
}

func (n *BlindClockPushNotifier) deliverEvent(ctx context.Context, event blindClockDueEvent) error {
	subscriptions, err := n.pushRepo.ListSubscriptions()
	if err != nil {
		return err
	}
	if len(subscriptions) == 0 {
		return nil
	}

	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return err
	}

	delivered := false
	for _, subscription := range subscriptions {
		resp, err := webpush.SendNotificationWithContext(ctx, payload, &webpush.Subscription{
			Endpoint: subscription.Endpoint,
			Keys: webpush.Keys{
				Auth:   subscription.KeyAuth,
				P256dh: subscription.KeyP256DH,
			},
		}, &webpush.Options{
			Subscriber:      n.subject,
			VAPIDPublicKey:  n.publicKey,
			VAPIDPrivateKey: n.privateKey,
			TTL:             120,
		})
		if err != nil {
			slog.Warn("blind_clock_push_send_failed", "endpoint", subscription.Endpoint, "err", err)
			continue
		}

		_ = resp.Body.Close()

		if resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusNotFound {
			_ = n.pushRepo.DeleteSubscription(subscription.Endpoint)
			delivered = true
			continue
		}

		if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
			delivered = true
			continue
		}

		slog.Warn("blind_clock_push_rejected", "endpoint", subscription.Endpoint, "status", resp.StatusCode)
	}

	if !delivered {
		return fmt.Errorf("no_successful_push_delivery")
	}

	return nil
}
