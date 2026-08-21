DO $$
DECLARE
    target_user_id TEXT;
    legacy_subject TEXT;
    canonical_subject TEXT;
    duplicate_user_id TEXT;
    duplicate_count INTEGER;
BEGIN
    SELECT id INTO target_user_id
    FROM users
    WHERE LOWER(email) = 'ishee@yandex.ru';

    IF target_user_id IS NULL THEN
        RETURN;
    END IF;

    SELECT subject INTO legacy_subject
    FROM auth_identities
    WHERE provider = 'telegram'
      AND user_id = target_user_id
      AND LOWER(COALESCE(username, '')) = 'semenovv';

    IF legacy_subject IS NULL OR legacy_subject ~ '^[0-9]+$' THEN
        RETURN;
    END IF;

    SELECT COUNT(*), MIN(ai.subject), MIN(ai.user_id)
    INTO duplicate_count, canonical_subject, duplicate_user_id
    FROM auth_identities ai
    JOIN users duplicate_user ON duplicate_user.id = ai.user_id
    WHERE ai.provider = 'telegram'
      AND ai.user_id <> target_user_id
      AND ai.subject ~ '^[0-9]+$'
      AND LOWER(COALESCE(ai.username, '')) = 'semenovv'
      AND duplicate_user.email LIKE 'telegram-%@telegram.invalid';

    IF duplicate_count = 0 THEN
        RETURN;
    END IF;
    IF duplicate_count <> 1 THEN
        RAISE EXCEPTION 'ambiguous Telegram identity repair for ishee@yandex.ru';
    END IF;
    IF EXISTS (SELECT 1 FROM user_players WHERE user_id = duplicate_user_id) THEN
        RAISE EXCEPTION 'refusing to merge Telegram duplicate with player ownership';
    END IF;
    IF (SELECT COUNT(*) FROM auth_identities WHERE user_id = duplicate_user_id) <> 1 THEN
        RAISE EXCEPTION 'refusing to merge Telegram duplicate with additional identities';
    END IF;

    DELETE FROM auth_sessions WHERE user_id = duplicate_user_id;
    DELETE FROM auth_oidc_flows WHERE user_id = duplicate_user_id;
    DELETE FROM auth_identities
    WHERE provider = 'telegram' AND subject = canonical_subject AND user_id = duplicate_user_id;

    UPDATE auth_identities
    SET subject = canonical_subject, updated_at = NOW()
    WHERE provider = 'telegram' AND subject = legacy_subject AND user_id = target_user_id;

    DELETE FROM users
    WHERE id = duplicate_user_id
      AND email LIKE 'telegram-%@telegram.invalid';
END $$;
