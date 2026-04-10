-- +up

-- FK на session
ALTER TABLE operations
ADD CONSTRAINT fk_operations_session
FOREIGN KEY (session_id) REFERENCES sessions(id);

-- идемпотентность
CREATE UNIQUE INDEX ux_operations_request_id
ON operations(request_id);

-- защита от двойного reversal
CREATE UNIQUE INDEX ux_operations_reference_id
ON operations(reference_id)
WHERE reference_id IS NOT NULL;

-- базовые проверки
ALTER TABLE operations
ADD CONSTRAINT chk_operations_chips_positive
CHECK (chips > 0);

ALTER TABLE sessions
ADD CONSTRAINT chk_sessions_chip_rate_positive
CHECK (chip_rate > 0);

-- +down

ALTER TABLE operations DROP CONSTRAINT fk_operations_session;

DROP INDEX ux_operations_request_id;
DROP INDEX ux_operations_reference_id;

ALTER TABLE operations DROP CONSTRAINT chk_operations_chips_positive;
ALTER TABLE sessions DROP CONSTRAINT chk_sessions_chip_rate_positive;
