## Why

Session visibility currently depends on whether operation participants are linked to accounts, and the same view authorization is reused for writes. As a result, a guest who can open a session can also call mutation endpoints, while some finished sessions remain hidden. Authentication must become the server-enforced write boundary without removing guest read access.

## What Changes

- Make every finished session and its session-detail resources readable by everyone.
- Make active sessions readable only to administrators, authenticated participants, or guests who explicitly selected a participating player.
- Persist participation independently of effective financial operations. The authenticated starter's player joins atomically with session creation, and a player joins on their first buy-in.
- Require authentication for starting sessions and other domain mutations; require active-session participation (or administrator role) for session mutations.
- Render guest session screens read-only, disable session start with an authentication explanation, and shorten the empty guest-player option to `Игрок` in Russian.

## Capabilities

### New Capabilities

- `session-participant-write-access`: authenticated participant membership and server-enforced mutation authorization.

### Modified Capabilities

- `player-session-visibility-summary`: finished records become public while active records use explicit participant membership.

## Impact

- **Breaking API behavior:** unauthenticated domain write requests that previously succeeded now return `401`; authenticated non-participant session writes return `403`.
- Adds a fail-safe, additive `session_participants` table and backfills it from existing non-reversal operations without deleting or rewriting financial data.
- Changes session list/detail SQL predicates, session-start and buy-in transactions, HTTP authorization, frontend controls/copy, Swagger, and service-worker cache generation.
