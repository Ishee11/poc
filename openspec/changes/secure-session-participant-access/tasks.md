## 1. Persistence and domain boundary

- [x] 1.1 Add the idempotent `session_participants` migration and backfill non-reversal operation players.
- [x] 1.2 Add participant repository contracts and PostgreSQL implementation.
- [x] 1.3 Enroll the authenticated starter and successful buy-in player inside their existing transactions.

## 2. Backend authorization and visibility

- [x] 2.1 Separate public/participant read authorization from authenticated participant mutation authorization.
- [x] 2.2 Make finished session list/detail rows public and active rows participant-scoped for user or selected guest player.
- [x] 2.3 Require authentication for session start, player creation, and blind-clock mutations.
- [x] 2.4 Require participant or administrator authorization for every session mutation endpoint.
- [x] 2.5 Add focused use-case, HTTP, and PostgreSQL integration coverage for public, guest, participant, unrelated-user, and administrator cases.

## 3. Frontend read-only experience

- [x] 3.1 Keep guest start visible but disabled with an authentication-required explanation.
- [x] 3.2 Render guest active sessions read-only and prevent offline/local mutation creation.
- [x] 3.3 Change the Russian empty guest-player option from `Без выбора игрока` to `Игрок` and update English copy consistently.
- [x] 3.4 Advance the PWA shell generation and update frontend/service-worker contract tests.

## 4. Documentation and verification

- [x] 4.1 Update Swagger/API documentation for `401`/`403` behavior and guest read semantics.
- [x] 4.2 Run focused tests, `go test ./...`, web checks, network/service-worker tests, strict OpenSpec validation, and `git diff --check`.

## 5. Staged rollout

- [ ] 5.1 Commit only task-owned files and push to `dev`; wait for checks and dev deployment.
- [ ] 5.2 Verify the exact dev read/write matrix and deployed service worker.
- [ ] 5.3 Promote the validated commit to `master`; wait for production deployment.
- [ ] 5.4 Verify production health, public finished reads, selected-player active reads, anonymous mutation rejection, and shell version.
