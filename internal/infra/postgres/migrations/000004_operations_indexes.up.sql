-- aggregates
CREATE INDEX IF NOT EXISTS idx_operations_session
ON operations(session_id);

-- last operation
CREATE INDEX IF NOT EXISTS idx_operations_session_player_created
ON operations(session_id, player_id, created_at DESC);

-- list operations
CREATE INDEX IF NOT EXISTS idx_operations_session_created
ON operations(session_id, created_at DESC);

-- reference lookup
CREATE INDEX IF NOT EXISTS idx_operations_reference_id
ON operations(reference_id);
