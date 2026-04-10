-- +up
CREATE TABLE operations (
    id TEXT PRIMARY KEY,

    request_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    player_id TEXT NOT NULL,

    type TEXT NOT NULL,
    chips BIGINT NOT NULL,

    reference_id TEXT,
    created_at TIMESTAMP NOT NULL
);

-- +down
DROP TABLE operations;
