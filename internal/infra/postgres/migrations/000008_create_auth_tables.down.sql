DROP INDEX IF EXISTS idx_login_attempts_ip_created;
DROP INDEX IF EXISTS idx_login_attempts_email_created;
DROP INDEX IF EXISTS idx_auth_sessions_user_active;
DROP INDEX IF EXISTS idx_auth_sessions_token_hash;

DROP TABLE IF EXISTS login_attempts;
DROP TABLE IF EXISTS auth_sessions;
DROP TABLE IF EXISTS users;
