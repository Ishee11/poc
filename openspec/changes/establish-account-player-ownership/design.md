## Context

The repository already separates authentication users (`users`) from poker identities (`players`) through `user_players`. The database prevents one player from being linked to two accounts, but it allows one account to link multiple players. The current account UI lists every unlinked player and lets a user freely link, unlink, and relink. Session visibility already treats linked players as owned identities.

Public email registration is being enabled without invite codes. Ownership must therefore become part of registration and must not let an account rotate through player histories. The current dev database, refreshed from production during deployment, contains three accounts and zero ownership links as of 2026-08-20.

## Goals / Non-Goals

**Goals:**

- Enforce one account to one player and one player to one account in PostgreSQL and application code.
- Make a player selection mandatory for new registration and for legacy-account onboarding.
- Let a user freely claim one unowned player once or create a new player once.
- Restrict later ownership correction to administrators and make corrections atomic and auditable.
- Preserve guest access and the existing ownership-derived visibility model.

**Non-Goals:**

- Invite codes, approval workflows, claim codes, email verification, player merging, aliases, or social login.
- Allowing users to share a player, own multiple player identities, or self-service reassignment.
- Changing poker operations, settlement rules, or offline command semantics.

## Decisions

### Ownership is a database-enforced one-to-one relation

Add a unique constraint on `user_players.user_id` while retaining the existing unique constraint on `player_id`. The migration performs a preflight check and fails without modifying data if any account has multiple rows. A database constraint is required because HTTP prechecks alone cannot prevent concurrent claims.

Alternative considered: keep one account to many players for aliases. Rejected because ownership would grant access to unrelated histories; duplicate identities require a future explicit merge capability.

### Registration and legacy onboarding share one player-selection contract

The wire representation is:

```json
{"mode":"existing","player_id":"player-id"}
```

or:

```json
{"mode":"new","name":"Player name"}
```

`POST /auth/register` requires this object. Account creation plus claiming or creating the player occurs in one transaction. Existing accounts without a link use `PUT /account/player` with the same object. Exactly one mode is accepted. New-player names use the existing player-name validation and ID generator.

Alternative considered: register first and link in a second independent request. Rejected because it creates normal accounts in a partially initialized state and can orphan a new player if linking fails.

### Self-service ownership is write-once

Users may claim only an unowned player and only while their account has no player. The account UI has no unlink or replace action. The legacy `POST /account/players` path delegates to the same one-time claim rule during compatibility; self-service `DELETE /account/players` returns method not allowed.

The UI shows a confirmation before claiming an existing player and states that only an administrator can correct the selection. Free selection is intentionally retained; no administrator approval or claim code is required.

### Account responses become singular without an immediate read break

`GET /account` adds `player` and `onboarding_required`. During the transition it also returns the existing `players` array containing zero or one element. New frontend code consumes the singular field.

Unowned-player entries add `sessions_count` and `last_played_at`; the UI derives a short ID from `player_id`. This is enough to distinguish duplicate names without exposing additional financial data.

### Administrators get an atomic correction boundary

New protected APIs are:

- `GET /admin/accounts?query=&limit=&offset=` returning `accounts`, `total`, `limit`, and `offset`; each account includes id, email, role, status, and nullable player.
- `PUT /admin/accounts/{user_id}/player` with `{ "player_id": "..." }` to atomically replace or establish ownership.
- `DELETE /admin/accounts/{user_id}/player` to idempotently clear ownership.

Replacement locks the target account ownership and target player ownership in one transaction. Assigning a player owned by another account returns `409 player_already_linked`; the API never silently steals ownership. Assigning the current player is idempotent.

The account screen gains an admin-only management panel with account search, current ownership, free-player selection, replace, and clear controls.

### Ownership changes retain the current visibility model

After a claim, sessions containing that player become visible to the owner and hidden from guests unless another existing visibility rule grants access. An authenticated account without a player receives guest-equivalent domain visibility and is routed to onboarding by the frontend.

Every self or administrator ownership mutation writes a structured log containing request ID, operation, actor user ID, target user ID, old player ID, and new player ID. Logs exclude email, password, cookie, and raw session token.

## Risks / Trade-offs

- [A user freely claims the wrong unowned player] -> Show sessions count, last-played date, short ID, and an irreversible-choice confirmation; provide administrator correction.
- [Two users claim the same player concurrently] -> Use transaction checks plus the unique player constraint and return a stable `409 player_already_linked` to the loser.
- [A migration encounters unexpected multi-player accounts] -> Fail closed during preflight and require an explicit data decision; never delete ownership rows automatically.
- [Mandatory onboarding blocks an existing account] -> Permit the same choose-or-create flow after login and keep backend access no broader than guest access until completion.
- [Rollback after deployment] -> `AUTH_ENABLED=false` hides account UI; the additive uniqueness constraint remains safe and does not require rollback.

## Migration Plan

1. Add the guarded uniqueness migration and repository/use-case tests.
2. Implement ownership selection and singular account APIs while retaining the compatible read array and legacy one-time POST alias.
3. Implement registration/onboarding UI and administrator APIs/UI.
4. Deploy to dev, verify existing-player claim, new-player creation, guest/owner visibility, conflict handling, and administrator correction.
5. Re-run production preflight, back up the database through the existing deployment process, and deploy production with `AUTH_ENABLED=true`.

## Open Questions

None. Product choices are fixed for this change.
