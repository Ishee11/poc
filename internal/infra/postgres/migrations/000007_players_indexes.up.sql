-- уникальность имени в рамках сессии
CREATE UNIQUE INDEX IF NOT EXISTS players_session_name_uq
ON players_in_session (session_id, name);

-- быстрый поиск по сессии
CREATE INDEX IF NOT EXISTS players_session_idx
ON players_in_session (session_id);
