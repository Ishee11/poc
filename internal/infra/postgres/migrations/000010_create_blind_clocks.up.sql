CREATE TABLE IF NOT EXISTS blind_clocks (
    id TEXT PRIMARY KEY,
    status TEXT NOT NULL CHECK (status IN ('idle', 'running', 'paused', 'finished')),
    started_at TIMESTAMP NULL,
    paused_at TIMESTAMP NULL,
    finished_at TIMESTAMP NULL,
    accumulated_pause_seconds BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS blind_clock_levels (
    clock_id TEXT NOT NULL REFERENCES blind_clocks(id) ON DELETE CASCADE,
    level_index INT NOT NULL,
    small_blind BIGINT NOT NULL,
    big_blind BIGINT NOT NULL,
    duration_seconds BIGINT NOT NULL,
    PRIMARY KEY (clock_id, level_index)
);

CREATE INDEX IF NOT EXISTS idx_blind_clocks_updated_at ON blind_clocks(updated_at DESC);
