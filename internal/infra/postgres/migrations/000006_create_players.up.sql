CREATE TABLE players_in_session (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    name TEXT NOT NULL,
    user_id TEXT NULL,

    created_at TIMESTAMP NOT NULL DEFAULT now()
);

-- уникальность имени в рамках сессии
CREATE UNIQUE INDEX players_session_name_uq
ON players_in_session (session_id, name);

-- быстрый поиск по сессии
CREATE INDEX players_session_idx
ON players_in_session (session_id);
