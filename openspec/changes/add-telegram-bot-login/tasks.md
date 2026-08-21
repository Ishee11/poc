## 1. Contracts and persistence

- [x] 1.1 Add challenge states/errors/entities and repository ports with explicit browser binding.
- [x] 1.2 Add PostgreSQL migration/repository using hashed challenge and binding values, TTL indexing, row locking, and atomic terminal transitions.
- [x] 1.3 Extract shared Telegram identity resolution so OIDC and Bot API IDs map to the same account and preserve first-login behavior.

## 2. Backend HTTP and session completion

- [x] 2.1 Add create, status, cancel, and complete endpoints with generic errors, method checks, secure binding cookie, and secret-free logs.
- [x] 2.2 Complete approved challenges in one transaction with existing auth-session semantics and a single consumed transition.
- [x] 2.3 Add bounded create/status/complete rate limits and expose only bot availability/username in public config.

## 3. Telegram bot adapter

- [x] 3.1 Add bot username/token/webhook-secret configuration that reuses the already connected Poker bot without exposing credentials to the frontend.
- [x] 3.2 Add secret-header-protected webhook handling for `/start`, invalid/expired challenges, and inline approve/cancel callbacks.
- [x] 3.3 Add a minimal Bot API sender for code-bearing confirmation messages and callback-query acknowledgements.

## 4. PWA flow

- [x] 4.1 Add API helpers and resumable client state for creating, polling, completing, cancelling, and expiring challenges.
- [x] 4.2 Make bot challenge the primary Telegram login action and open the configured `tg://resolve` link without `t.me` fallback.
- [x] 4.3 Add waiting/code/denied/expired UI, repeat launch, cancel, explicit OIDC fallback, two-second visible polling, and foreground immediate check.
- [x] 4.4 Bump the coherent service-worker shell generation and include every changed module.

## 5. Tests and documentation

- [x] 5.1 Add backend tests for creation entropy/TTL, unknown/expired transitions, approve/deny idempotency, binding isolation, one-use/concurrent completion, session creation, identity reuse/first login, and rate limits.
- [x] 5.2 Add bot tests for valid/invalid/expired start, inline approve/cancel, actor mismatch, and repeat callbacks.
- [x] 5.3 Add frontend tests for create, `tg://`, code/waiting, polling, complete, denied/expired, retry, OIDC fallback, and visibility resume.
- [x] 5.4 Update auth design, environment, Compose/workflows, and operational webhook setup documentation.

## 6. Validation and rollout

- [x] 6.1 Run formatting, focused and full Go tests/build, web/network/service-worker suites, Compose validation, and `git diff --check`.
- [x] 6.2 Run `openspec validate add-telegram-bot-login --strict` and `openspec validate --all --strict`.
- [ ] 6.3 Configure and verify the dedicated bot/webhook in dev across iOS PWA, Android PWA/browser, and Telegram Desktop.
- [ ] 6.4 Deploy production, verify no browser Telegram HTTP request occurs, confirm OIDC fallback, and archive the change after external acceptance.
