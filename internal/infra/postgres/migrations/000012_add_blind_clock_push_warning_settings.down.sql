ALTER TABLE blind_clock_push_subscriptions
    DROP COLUMN IF EXISTS notify_warning_60,
    DROP COLUMN IF EXISTS notify_warning_10;
