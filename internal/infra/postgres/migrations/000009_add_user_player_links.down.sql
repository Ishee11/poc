DROP INDEX IF EXISTS idx_user_players_user;

DROP TABLE IF EXISTS user_players;

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_role_check;

UPDATE users
SET role = 'member'
WHERE role = 'user';

ALTER TABLE users
    ADD CONSTRAINT users_role_check CHECK (role IN ('admin', 'member', 'viewer'));
