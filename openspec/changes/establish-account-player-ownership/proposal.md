## Why

Poker accounts and poker players are separate identities, but the current account UI lets any authenticated user attach and detach any unclaimed player and permits one account to own several player records. Enabling public registration now requires a durable ownership contract so a user cannot rotate through player identities or accidentally expose another person's session history.

## What Changes

- Establish a strict one-account-to-one-player and one-player-to-one-account ownership invariant.
- Require registration to either claim one unowned existing player or create and own a new player, without invite codes.
- Provide a mandatory ownership onboarding path for legacy accounts that do not yet have a player.
- Allow a user to claim ownership only once; remove self-service unlink and reassignment.
- Add administrator APIs and UI for listing accounts and atomically replacing or clearing an ownership link.
- Enrich unowned-player choices with enough non-sensitive context to distinguish duplicate names.
- Apply the ownership link to the existing guest/account session-visibility rules and emit structured audit logs for ownership changes.
- **BREAKING**: `POST /auth/register` requires a player selection object.
- **BREAKING**: account ownership becomes singular; self-service `DELETE /account/players` is removed.

## Capabilities

### New Capabilities

- `account-player-ownership`: Registration, legacy onboarding, one-to-one persistence, singular account representation, self-service claim rules, and ownership-derived visibility.
- `account-player-administration`: Administrator account listing, ownership replacement and removal, protected UI controls, and audit logging.

### Modified Capabilities

None. This repository does not yet have canonical OpenSpec capabilities.

## Impact

- PostgreSQL gains a one-player-per-account constraint and ownership migration guard.
- Auth registration, account use cases, repository ports, HTTP DTOs, and visibility integration are updated.
- Public auth/account APIs and the account onboarding UI change; administrator account-management APIs and UI are added.
- Existing Argon2id password hashing, opaque HttpOnly sessions, guest access, and invite-free registration remain in place.
