CREATE TABLE IF NOT EXISTS auth_identities (
    provider TEXT NOT NULL,
    subject TEXT NOT NULL,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    username TEXT,
    display_name TEXT,
    picture_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (provider, subject),
    UNIQUE (provider, user_id)
);

CREATE TABLE IF NOT EXISTS auth_oidc_flows (
    state_hash TEXT PRIMARY KEY,
    mode TEXT NOT NULL CHECK (mode IN ('login', 'link')),
    user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
    code_verifier TEXT NOT NULL,
    nonce TEXT NOT NULL,
    redirect_uri TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_auth_oidc_flows_expires_at
ON auth_oidc_flows(expires_at);
