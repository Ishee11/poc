## ADDED Requirements

### Requirement: Account and player ownership is one-to-one
The system SHALL allow an account to own at most one player and a player to be owned by at most one account, with both invariants enforced by PostgreSQL constraints.

#### Scenario: First ownership link
- **WHEN** an unlinked account claims an unowned player
- **THEN** the system stores exactly one ownership row for that account and player

#### Scenario: Concurrent claim
- **WHEN** two accounts concurrently claim the same unowned player
- **THEN** exactly one claim succeeds and the other returns `409 player_already_linked`

#### Scenario: Unsafe migration state
- **WHEN** the ownership migration finds an account linked to more than one player
- **THEN** the migration fails without deleting or rewriting any ownership row

### Requirement: Registration includes player ownership
The system SHALL require invite-free email registration to include either one unowned existing player or one new player name, and SHALL create the account and ownership link atomically.

#### Scenario: Register with existing player
- **WHEN** valid registration credentials select an unowned existing player
- **THEN** the system creates the account, links that player, creates the session, and returns the authenticated user

#### Scenario: Register with new player
- **WHEN** valid registration credentials provide a valid new player name
- **THEN** the system creates the account, creates the player, links them, creates the session, and returns the authenticated user

#### Scenario: Registration omits player selection
- **WHEN** registration provides neither a valid existing-player selection nor a valid new-player selection
- **THEN** the system rejects the request with a stable validation error and creates neither account nor player

#### Scenario: Registration ownership conflict
- **WHEN** the selected existing player becomes owned before registration commits
- **THEN** the system returns `409 player_already_linked` and creates no account

#### Scenario: Registration contains invite data
- **WHEN** a client registers without any invite code
- **THEN** the system evaluates only credentials and player selection and does not require an invite

### Requirement: Legacy accounts complete ownership onboarding
The system SHALL expose mandatory choose-or-create onboarding for an authenticated account that has no player.

#### Scenario: Legacy account signs in
- **WHEN** an authenticated account has no ownership link
- **THEN** `GET /account` returns `player: null` and `onboarding_required: true`, and the frontend routes to ownership onboarding

#### Scenario: Legacy account completes onboarding
- **WHEN** an unlinked authenticated account submits a valid selection to `PUT /account/player`
- **THEN** the system claims or creates the player atomically and returns an account with `onboarding_required: false`

#### Scenario: Unlinked account bypasses frontend onboarding
- **WHEN** an authenticated account without a player calls domain APIs directly
- **THEN** its data visibility is no broader than guest visibility

### Requirement: Self-service ownership is write-once
The system SHALL prevent an account owner from unlinking or replacing a player after the first successful ownership link.

#### Scenario: Account attempts a second claim
- **WHEN** an account that already owns a player calls the self-service ownership endpoint
- **THEN** the system returns `409 account_already_linked` and leaves the existing ownership unchanged

#### Scenario: Account attempts self-service unlink
- **WHEN** a non-admin user calls the legacy self-service unlink route
- **THEN** the system returns method not allowed and preserves ownership

#### Scenario: Legacy claim endpoint
- **WHEN** an unlinked account calls the legacy `POST /account/players` route with an unowned player
- **THEN** the system applies the same one-time claim rules as `PUT /account/player`

### Requirement: Account ownership responses are singular and compatible
The system SHALL represent current ownership as a nullable singular player while retaining the legacy player array during the transition.

#### Scenario: Linked account response
- **WHEN** a linked account requests `GET /account`
- **THEN** the response contains the owned `player`, `onboarding_required: false`, and a `players` array containing that same player

#### Scenario: Unlinked account response
- **WHEN** an unlinked account requests `GET /account`
- **THEN** the response contains `player: null`, `onboarding_required: true`, and an empty `players` array

### Requirement: Player choices are distinguishable
The system SHALL list only unowned players for self-service selection and SHALL provide non-sensitive context for distinguishing players with the same name.

#### Scenario: Available-player list
- **WHEN** an account opens ownership onboarding
- **THEN** each available option includes player ID, name, sessions count, and nullable last-played time

#### Scenario: Owned player is excluded
- **WHEN** a player is already owned by any account
- **THEN** that player does not appear in the self-service available-player list

### Requirement: Ownership controls session visibility
The system SHALL use the ownership link in the existing guest and account visibility rules.

#### Scenario: Guest visibility after claim
- **WHEN** an unowned player with historical sessions becomes owned
- **THEN** guests can no longer access those sessions through visibility-filtered APIs

#### Scenario: Owner visibility after claim
- **WHEN** an account owns a player with historical sessions
- **THEN** that account can access those sessions through visibility-filtered APIs

#### Scenario: Other account visibility
- **WHEN** another account owns no participant in the claimed player's session
- **THEN** that account cannot access the session unless an existing public visibility rule independently grants access

### Requirement: Self-service ownership changes are auditable
The system SHALL emit a structured ownership-change log without authentication secrets or email addresses.

#### Scenario: Successful self claim
- **WHEN** an account claims or creates its player
- **THEN** the log records request ID, operation, actor user ID, target user ID, old player ID, and new player ID
