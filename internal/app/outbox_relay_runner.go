package app

import (
	"context"
	"log/slog"
	"time"

	"github.com/ishee11/poc/internal/usecase"
)

type OutboxRelayRunner struct {
	relay    *usecase.OutboxRelay
	interval time.Duration
	batch    int
}

func NewOutboxRelayRunner(relay *usecase.OutboxRelay, interval time.Duration, batch int) *OutboxRelayRunner {
	if interval <= 0 {
		interval = time.Second
	}
	if batch <= 0 {
		batch = 100
	}

	return &OutboxRelayRunner{
		relay:    relay,
		interval: interval,
		batch:    batch,
	}
}

func (r *OutboxRelayRunner) Start(ctx context.Context) {
	if r == nil {
		return
	}

	go r.loop(ctx)
}

func (r *OutboxRelayRunner) loop(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			result, err := r.relay.DispatchOnce(ctx, r.batch)
			if err != nil {
				slog.Warn("outbox_relay_failed", "err", err)
				continue
			}
			if result.Published > 0 || result.Failed > 0 {
				slog.Info("outbox_relay_dispatched", "published", result.Published, "failed", result.Failed)
			}
		}
	}
}
