## 1. Identity contract

- [x] 1.1 Parse and require signed Telegram OIDC profile `id`.
- [x] 1.2 Resolve bot and OIDC logins through the decimal Telegram user ID.
- [x] 1.3 Migrate an unambiguous legacy OIDC-sub identity without changing its Poker account.

## 2. Production repair

- [x] 2.1 Add an idempotent, fail-closed repair for the confirmed `ishee@yandex.ru` / `@semenovv` duplicate.
- [ ] 2.2 Deploy and verify that bot login returns the established email account and player ownership.

## 3. Validation

- [x] 3.1 Add regression tests for OIDC claim mapping, legacy migration, bot parity, and conflicts.
- [x] 3.2 Run Go, web, OpenSpec, migration, and diff checks.
