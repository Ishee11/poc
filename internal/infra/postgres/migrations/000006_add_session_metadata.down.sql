ALTER TABLE sessions
    DROP COLUMN IF EXISTS finished_at,
    DROP COLUMN IF EXISTS big_blind;
