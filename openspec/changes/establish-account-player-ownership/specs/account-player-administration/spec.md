## ADDED Requirements

### Requirement: Administrators can inspect account ownership
The system SHALL provide an administrator-only paginated account listing with ownership context.

#### Scenario: Administrator lists accounts
- **WHEN** an authenticated administrator calls `GET /admin/accounts` with optional query, limit, and offset
- **THEN** the response contains matching accounts, total count, limit, offset, role, status, and nullable current player

#### Scenario: Non-admin lists accounts
- **WHEN** an unauthenticated or non-admin caller requests the account listing
- **THEN** the system returns `401` or `403` without account data

### Requirement: Administrators can atomically replace ownership
The system SHALL allow an administrator to establish or replace an account's player ownership in one transaction without silently stealing another account's player.

#### Scenario: Replace with unowned player
- **WHEN** an administrator assigns an unowned player to an account that already owns another player
- **THEN** the old player becomes unowned and the new player becomes owned by the target account in one committed transaction

#### Scenario: Assign player owned by another account
- **WHEN** an administrator assigns a player owned by a different account
- **THEN** the system returns `409 player_already_linked` and preserves both existing ownership links

#### Scenario: Reassign current player
- **WHEN** an administrator assigns the target account's current player again
- **THEN** the operation succeeds idempotently without changing ownership

#### Scenario: Unknown target
- **WHEN** the target account or player does not exist
- **THEN** the system returns the corresponding stable not-found error without changing ownership

### Requirement: Administrators can clear ownership
The system SHALL allow an administrator to remove an account's ownership link without deleting the account, player, or poker history.

#### Scenario: Clear existing ownership
- **WHEN** an administrator deletes an existing account-player link
- **THEN** the account becomes onboarding-required and the player becomes available for claim

#### Scenario: Clear absent ownership
- **WHEN** an administrator clears an account that has no player
- **THEN** the operation succeeds idempotently

### Requirement: Administrator ownership UI is protected and usable
The system SHALL expose account ownership management only while the current frontend principal has the administrator role.

#### Scenario: Administrator opens account management
- **WHEN** an administrator opens the account screen
- **THEN** the UI shows searchable accounts, current players, free-player choices, and confirmed replace and clear actions

#### Scenario: Regular user opens account screen
- **WHEN** a regular authenticated user opens the account screen
- **THEN** administrator ownership controls are absent

#### Scenario: Administrator action conflicts
- **WHEN** an administrator action receives an ownership conflict
- **THEN** the UI preserves current state, explains the conflict, and refreshes account and available-player data

### Requirement: Administrator ownership changes are auditable
The system SHALL emit structured logs for successful administrator ownership replacement and removal.

#### Scenario: Administrator changes ownership
- **WHEN** an administrator replaces or clears ownership
- **THEN** the log records request ID, operation, actor user ID, target user ID, old player ID, and new player ID without email or authentication secrets
