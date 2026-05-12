CREATE TABLE IF NOT EXISTS settlement_transfers (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    from_player_id TEXT NOT NULL,
    to_player_id TEXT NOT NULL,
    amount BIGINT NOT NULL CHECK (amount > 0),
    position INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now(),

    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE,
    FOREIGN KEY (from_player_id) REFERENCES players(id),
    FOREIGN KEY (to_player_id) REFERENCES players(id),
    CHECK (from_player_id <> to_player_id)
);

CREATE INDEX IF NOT EXISTS idx_settlement_transfers_session_position
ON settlement_transfers(session_id, position, created_at);
