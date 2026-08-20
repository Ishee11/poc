DO $$
BEGIN
    IF EXISTS (
        SELECT user_id
        FROM user_players
        GROUP BY user_id
        HAVING COUNT(*) > 1
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '23514',
            MESSAGE = 'cannot establish one-to-one account player ownership: an account owns multiple players';
    END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS uq_user_players_user_id
ON user_players(user_id);
