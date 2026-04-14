CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    chip_rate BIGINT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('active', 'finished')),
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    total_buy_in BIGINT NOT NULL DEFAULT 0,
    total_cash_out BIGINT NOT NULL DEFAULT 0
);
