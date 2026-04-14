CREATE TABLE IF NOT EXISTS players_in_session (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    name TEXT NOT NULL,
    user_id TEXT NULL,

    created_at TIMESTAMP NOT NULL DEFAULT now()
);
