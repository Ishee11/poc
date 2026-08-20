## 1. Persistence and Contracts

- [x] 1.1 Add a guarded migration that fails on multi-player accounts and adds unique ownership on `user_id` without deleting data.
- [x] 1.2 Add singular ownership DTOs, the existing/new player-selection type, stable ownership errors, and repository port changes.
- [x] 1.3 Add PostgreSQL integration coverage for both ownership uniqueness constraints and concurrent claims.

## 2. Self-Service Ownership

- [x] 2.1 Extend available-player queries with sessions count and nullable last-played time, excluding every owned player.
- [x] 2.2 Implement one transactional choose-or-create ownership operation that rejects accounts already linked and maps claim races to stable conflicts.
- [x] 2.3 Require the player-selection object during registration and atomically create the account plus ownership before creating its auth session.
- [x] 2.4 Add `PUT /account/player`, singular `GET /account` fields, the transitional `players` mirror, and the one-time legacy POST alias; disable self-service unlink.
- [x] 2.5 Emit secret-free structured logs for successful self-service ownership changes.
- [x] 2.6 Add unit and HTTP tests for existing-player registration, new-player registration, validation rollback, legacy onboarding, repeat claim, compatibility, and conflict errors.

## 3. Administrator Ownership Management

- [x] 3.1 Add paginated/searchable administrator account queries with nullable player ownership and total count.
- [x] 3.2 Implement transactional administrator replace and idempotent clear operations with row locking and no implicit ownership stealing.
- [x] 3.3 Add role-protected list, replace, and clear endpoints under `/admin/accounts` with stable not-found and conflict responses.
- [x] 3.4 Emit secret-free structured logs for administrator ownership replacement and removal.
- [x] 3.5 Add use-case and HTTP tests for `401`, `403`, listing, first assignment, replacement, same-player idempotency, occupied-player conflict, clear, and unknown targets.

## 4. Registration, Account, and Administrator UI

- [x] 4.1 Extend email registration mode with searchable existing-player selection, player context, new-player creation, and irreversible-choice confirmation; do not add invite fields.
- [x] 4.2 Route authenticated legacy accounts with `onboarding_required` to the same mandatory choose-or-create flow before exposing the normal app UI.
- [x] 4.3 Change the account screen to display one immutable owned player and administrator-contact guidance instead of self-service unlink/relink controls.
- [x] 4.4 Add an admin-only account ownership panel with account search, current player, free-player selection, confirmed replace, confirmed clear, and conflict refresh.
- [x] 4.5 Add Russian and English copy, keyboard-accessible controls, loading/empty/error states, and web tests for both onboarding modes and administrator correction.

## 5. Visibility and Documentation

- [x] 5.1 Add integration scenarios proving guest, owner, and unrelated-account visibility before and after a player is claimed.
- [x] 5.2 Update `docs/AUTH_DESIGN.md`, Swagger artifacts, environment examples, and any stale plural ownership descriptions to the approved one-to-one contract.

## 6. Validation and Rollout

- [x] 6.1 Run formatting, focused ownership tests, `go test ./...`, `go build ./...`, all web checks, network/offline suites, Compose validation, and `git diff --check`.
- [x] 6.2 Run `openspec validate establish-account-player-ownership --strict` and `openspec validate --all --strict` after implementation and documentation updates.
- [ ] 6.3 Deploy to dev and verify existing-player claim, new-player creation, legacy onboarding, race conflict, visibility changes, and administrator correction against the dev database.
- [ ] 6.4 Re-run the production multi-link preflight and existing backup gate, deploy with `AUTH_ENABLED=true`, verify health and ownership flows, then archive the completed OpenSpec change.
