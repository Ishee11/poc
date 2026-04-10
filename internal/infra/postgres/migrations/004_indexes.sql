-- +up

-- агрегаты по сессии
CREATE INDEX idx_operations_session
ON operations(session_id);

-- last operation игрока
CREATE INDEX idx_operations_session_player_created
ON operations(session_id, player_id, created_at DESC);

-- список операций
CREATE INDEX idx_operations_session_created
ON operations(session_id, created_at DESC);

-- поиск reversal target
CREATE INDEX idx_operations_reference_id
ON operations(reference_id);

-- +down

DROP INDEX idx_operations_session;
DROP INDEX idx_operations_session_player_created;
DROP INDEX idx_operations_session_created;
DROP INDEX idx_operations_reference_id;
