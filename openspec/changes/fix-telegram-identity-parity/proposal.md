# Change: Fix Telegram identity parity

## Why

Telegram OIDC returns both an OIDC `sub` and the Telegram profile `id`. The bot challenge receives Bot API `user.id`, but the initial implementation incorrectly assumed that it equalled OIDC `sub`. An existing linked account therefore was not found and a duplicate synthetic-email account could be created.

## What Changes

- Treat the signed OIDC profile `id` as the canonical Telegram identity subject shared with Bot API.
- Retain OIDC `sub` only as a legacy lookup key and migrate an unambiguous legacy identity during OIDC login/link.
- Repair the confirmed `ishee@yandex.ru` / `@semenovv` production duplicate only under strict, fail-closed preconditions.
- Preserve the existing bot challenge, session creation, player ownership, and OIDC fallback behavior.

## Impact

- A bot login and OIDC login for one Telegram user resolve to one Poker account.
- The repair revokes only sessions belonging to the empty synthetic duplicate; the established email account and player ownership remain authoritative.
