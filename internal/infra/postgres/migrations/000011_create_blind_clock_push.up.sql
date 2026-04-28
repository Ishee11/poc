CREATE TABLE IF NOT EXISTS blind_clock_push_subscriptions (
    endpoint TEXT PRIMARY KEY,
    key_auth TEXT NOT NULL,
    key_p256dh TEXT NOT NULL,
    user_agent TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS blind_clock_push_events (
    event_key TEXT PRIMARY KEY,
    clock_id UUID NOT NULL REFERENCES blind_clocks(id) ON DELETE CASCADE,
    event_kind TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_blind_clock_push_events_clock_id
    ON blind_clock_push_events (clock_id, created_at DESC);
