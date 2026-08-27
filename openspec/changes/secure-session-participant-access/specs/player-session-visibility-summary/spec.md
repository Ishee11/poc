## MODIFIED Requirements

### Requirement: Hidden active session details remain confidential

The system MUST make finished session details readable by every viewer. For an active session, the system MUST return list and detail records only to an administrator, an authenticated account whose owned player participates, or an anonymous viewer whose explicit `guest_player_id` participates. Selecting a player MUST grant read access only and MUST NOT grant mutation authority.

#### Scenario: Guest requests finished sessions
- **WHEN** an unauthenticated viewer requests the session list or a finished session detail without selecting a player
- **THEN** every matching finished session is available

#### Scenario: Guest selects active participant
- **WHEN** an unauthenticated viewer selects a player participating in an active session
- **THEN** that active session is available to list and open read-only

#### Scenario: Guest selects unrelated player
- **WHEN** an unauthenticated viewer selects a player who does not participate in an active session
- **THEN** that active session remains absent and its detail routes return forbidden

#### Scenario: Authenticated participant requests active session
- **WHEN** the authenticated account's owned player participates in an active session
- **THEN** the active session is available to list and open

#### Scenario: Authenticated unrelated viewer requests active session
- **WHEN** the authenticated account's owned player does not participate in an active session
- **THEN** the active session remains absent and its detail routes return forbidden
