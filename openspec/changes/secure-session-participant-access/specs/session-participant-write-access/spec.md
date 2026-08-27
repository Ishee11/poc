## ADDED Requirements

### Requirement: Session participation is durable

The system SHALL persist a session's participating players independently of current effective balances. Starting a session SHALL atomically enroll the authenticated starter's owned player, and a successful first buy-in for another player SHALL enroll that player.

#### Scenario: Authenticated player starts a session
- **WHEN** an authenticated account with an owned player starts a valid session
- **THEN** the session and participant membership are committed together

#### Scenario: Existing participant adds another player
- **WHEN** an authorized participant records a successful buy-in for a player not yet in the session
- **THEN** that player becomes a durable participant in the same transaction

#### Scenario: Operation is reversed
- **WHEN** every financial operation for a participating player is later reversed
- **THEN** the player's participation remains recorded

### Requirement: Authentication protects domain writes

The server MUST reject anonymous domain mutations regardless of frontend state. Starting sessions, creating players, changing blind-clock state, recording or reversing operations, changing expenses or settlements, and finishing sessions SHALL require a valid authenticated session.

#### Scenario: Guest calls session start directly
- **WHEN** an unauthenticated client posts a valid session-start request
- **THEN** the server returns `401` and creates no session

#### Scenario: Guest alters the page controls
- **WHEN** an unauthenticated client removes frontend disabled attributes and calls a mutation endpoint
- **THEN** the server still returns `401` and changes no domain data

### Requirement: Session writes require participation

The server MUST allow an ordinary session mutation only when the authenticated account's owned player participates in that session, and MUST retain administrator maintenance authority. A non-participant SHALL receive `403`. Existing domain status rules SHALL still decide which mutations are valid for active or finished sessions.

#### Scenario: Authenticated participant changes an active session
- **WHEN** an authenticated participant submits a valid buy-in, cash-out, reversal, expense, settlement, or finish command
- **THEN** normal domain validation and processing continue

#### Scenario: Authenticated unrelated user changes an active session
- **WHEN** an authenticated account whose player is not a participant calls an active-session mutation endpoint
- **THEN** the server returns `403` and changes no session data

### Requirement: Guest session UI is explicitly read-only

The frontend SHALL keep session reading available to authorized guest context while preventing mutation interactions and explaining the authentication boundary.

#### Scenario: Guest sees session start
- **WHEN** no account is authenticated
- **THEN** the start control remains visible but disabled and explains that login is required

#### Scenario: Guest opens an active session through selected player
- **WHEN** a guest selects a participating player and opens that active session
- **THEN** the session data is visible, mutation controls are unavailable, and a read-only notice is shown
