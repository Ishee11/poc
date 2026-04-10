ALTER TABLE operations DROP CONSTRAINT fk_operations_session;

DROP INDEX ux_operations_request_id;
DROP INDEX ux_operations_reference_id;

ALTER TABLE operations DROP CONSTRAINT chk_operations_chips_positive;
ALTER TABLE sessions DROP CONSTRAINT chk_sessions_chip_rate_positive;
