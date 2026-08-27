CREATE TABLE IF NOT EXISTS session_participants (
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    player_id TEXT NOT NULL REFERENCES players(id) ON DELETE RESTRICT,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (session_id, player_id)
);

INSERT INTO session_participants (session_id, player_id, joined_at)
SELECT o.session_id, o.player_id, MIN(o.created_at)
FROM operations o
WHERE o.type <> 'reversal'
GROUP BY o.session_id, o.player_id
ON CONFLICT (session_id, player_id) DO NOTHING;

CREATE INDEX IF NOT EXISTS idx_session_participants_player
ON session_participants(player_id, session_id);
