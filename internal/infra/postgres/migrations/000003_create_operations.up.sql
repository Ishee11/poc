CREATE TABLE IF NOT EXISTS operations (
    id TEXT PRIMARY KEY,

    request_id TEXT NOT NULL UNIQUE,
    session_id TEXT NOT NULL,
    player_id TEXT NOT NULL,

    type TEXT NOT NULL CHECK (type IN ('buy_in', 'cash_out', 'reversal')),
    chips BIGINT NOT NULL CHECK (chips > 0),

    reference_id TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT now(),

    FOREIGN KEY (session_id) REFERENCES sessions(id),
    FOREIGN KEY (player_id) REFERENCES players(id),
    FOREIGN KEY (reference_id) REFERENCES operations(id)
);
