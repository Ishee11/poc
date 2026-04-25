ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_role_check;

UPDATE users
SET role = 'user'
WHERE role IN ('member', 'viewer');

ALTER TABLE users
    ADD CONSTRAINT users_role_check CHECK (role IN ('admin', 'user'));

CREATE TABLE IF NOT EXISTS user_players (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    player_id TEXT NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, player_id),
    UNIQUE (player_id)
);

CREATE INDEX IF NOT EXISTS idx_user_players_user
ON user_players(user_id, created_at DESC);
