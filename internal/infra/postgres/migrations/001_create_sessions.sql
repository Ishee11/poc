-- +up
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    chip_rate BIGINT NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,

    total_buy_in BIGINT NOT NULL DEFAULT 0,
    total_cash_out BIGINT NOT NULL DEFAULT 0
);

-- +down
DROP TABLE sessions;
