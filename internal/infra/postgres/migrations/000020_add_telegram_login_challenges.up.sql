CREATE TABLE IF NOT EXISTS telegram_login_challenges (
    challenge_hash TEXT PRIMARY KEY,
    browser_binding_hash TEXT NOT NULL,
    verification_code TEXT NOT NULL CHECK (verification_code ~ '^[0-9]{4}$'),
    status TEXT NOT NULL CHECK (status IN ('pending', 'approved', 'denied', 'expired', 'consumed')),
    telegram_subject TEXT,
    telegram_username TEXT,
    telegram_name TEXT,
    created_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    approved_at TIMESTAMPTZ,
    consumed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_telegram_login_challenges_expires_at
ON telegram_login_challenges(expires_at);
