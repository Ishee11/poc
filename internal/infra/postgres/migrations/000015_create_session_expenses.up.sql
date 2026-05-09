CREATE TABLE IF NOT EXISTS session_expenses (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    title TEXT NOT NULL,
    amount BIGINT NOT NULL CHECK (amount > 0),
    created_at TIMESTAMP NOT NULL DEFAULT now(),

    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS session_expense_participants (
    expense_id TEXT NOT NULL,
    player_id TEXT NOT NULL,

    PRIMARY KEY (expense_id, player_id),
    FOREIGN KEY (expense_id) REFERENCES session_expenses(id) ON DELETE CASCADE,
    FOREIGN KEY (player_id) REFERENCES players(id)
);

CREATE TABLE IF NOT EXISTS session_expense_payments (
    expense_id TEXT NOT NULL,
    player_id TEXT NOT NULL,
    amount BIGINT NOT NULL CHECK (amount > 0),

    PRIMARY KEY (expense_id, player_id),
    FOREIGN KEY (expense_id) REFERENCES session_expenses(id) ON DELETE CASCADE,
    FOREIGN KEY (player_id) REFERENCES players(id)
);

CREATE INDEX IF NOT EXISTS idx_session_expenses_session_created
ON session_expenses(session_id, created_at DESC);
