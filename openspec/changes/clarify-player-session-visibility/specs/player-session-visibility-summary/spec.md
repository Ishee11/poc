## ADDED Requirements

### Requirement: Player statistics expose authoritative session scopes
The player-detail API SHALL return separate `total_sessions_count` and `visible_sessions_count` aggregates for the requested player and period. The total SHALL count all real sessions contributing to the player's aggregate statistics, while the visible count SHALL count sessions the current viewer is authorized to open.

#### Scenario: Player has no real sessions
- **WHEN** the player has no effective operations in any session in the selected period
- **THEN** both total and visible session counts are zero

#### Scenario: Every session is visible
- **WHEN** the player has 10 contributing sessions and the viewer may access all 10
- **THEN** the API returns total 10 and visible 10

#### Scenario: Some sessions are visible
- **WHEN** the player has 10 contributing sessions and the viewer may access 4
- **THEN** the API returns total 10 and visible 4

#### Scenario: No sessions are visible
- **WHEN** the player has 10 contributing sessions and the viewer may access none
- **THEN** the API returns total 10 and visible 0 even if the player has a non-zero financial result

### Requirement: Hidden session details remain confidential
The system MUST apply the existing session-access rules to the visible count and detailed session array. It MUST NOT include any hidden session identifier, participant, date, status, name, amount, or other hidden-session field in the response.

#### Scenario: Unrelated viewer requests player details
- **WHEN** an unrelated viewer requests statistics containing closed sessions
- **THEN** closed sessions contribute only to permitted all-session aggregates and are absent from the detailed session array

#### Scenario: Visible count matches authorization
- **WHEN** the viewer can open exactly four of ten player sessions
- **THEN** `visible_sessions_count` is four and every returned session record is one of those authorized sessions

### Requirement: Player UI distinguishes total and visible history
The player detail UI SHALL use the explicit total as the primary Sessions value and SHALL explain any difference between total and visible counts without deriving hidden-history state from financial values.

#### Scenario: Complete visibility
- **WHEN** total and visible counts are equal and greater than zero
- **THEN** the UI shows the total session count without an availability warning

#### Scenario: Partial visibility
- **WHEN** total is 10 and visible is 4
- **THEN** the UI shows 10 as the primary count and separately states that 4 sessions are available to view

#### Scenario: Existing history is completely hidden
- **WHEN** total is greater than zero and visible is zero
- **THEN** the UI shows the real total, states that zero sessions are available, and renders an empty state explaining that aggregate statistics include unavailable games

#### Scenario: Player has never played
- **WHEN** total and visible counts are zero
- **THEN** the UI shows zero sessions and a normal no-sessions empty state

### Requirement: Missing total is not represented as zero
The player detail UI MUST treat an absent, non-numeric, or otherwise unavailable explicit total as unknown and MUST make clear that financial aggregates can include history unavailable to the viewer.

#### Scenario: New UI receives a legacy response
- **WHEN** the response has no explicit total session count
- **THEN** the UI renders an em dash for Sessions and neutral history-unavailable explanatory copy instead of zero
