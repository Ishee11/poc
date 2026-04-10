-- FK
ALTER TABLE operations
ADD CONSTRAINT fk_operations_session
FOREIGN KEY (session_id) REFERENCES sessions(id);

-- idempotency
CREATE UNIQUE INDEX ux_operations_request_id
ON operations(request_id);

-- reversal protection
CREATE UNIQUE INDEX ux_operations_reference_id
ON operations(reference_id)
WHERE reference_id IS NOT NULL;

-- checks
ALTER TABLE operations
ADD CONSTRAINT chk_operations_chips_positive
CHECK (chips > 0);

ALTER TABLE sessions
ADD CONSTRAINT chk_sessions_chip_rate_positive
CHECK (chip_rate > 0);
