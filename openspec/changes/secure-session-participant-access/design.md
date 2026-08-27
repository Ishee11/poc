## Context

The current visibility predicate reconstructs participation from effective operations and account ownership. It intentionally allows some guest reads, but HTTP mutation handlers call the same `RequireView` check. Session creation has no authenticated actor in its command, so a new session has no durable owner or participant until a financial operation is recorded.

## Goals / Non-Goals

**Goals:**

- Use one durable participant set for active-session visibility and writes.
- Keep session creation usable by atomically enrolling the starter's owned player.
- Return `401` for anonymous writes and `403` for authenticated non-participant writes.
- Make finished session details public and active guest views read-only.
- Preserve administrator recovery and maintenance access.

**Non-Goals:**

- Limiting a participant to operations only for their own player.
- Changing one-account-to-one-player ownership.
- Adding a self-service join request or invitation flow.
- Making finished sessions mutable by ordinary participants.

## Decisions

### Participation is explicit and player-based

Add `session_participants(session_id, player_id, joined_at)` with a composite primary key and foreign keys to sessions and players. Player-based membership follows the existing stable poker identity if account ownership is administratively reassigned. The migration inserts every player that has any non-reversal operation in a session; reversals do not erase historical participation.

### Start and first buy-in enroll participants transactionally

`StartSessionCommand` carries the authenticated user ID. The use case resolves that account's single owned player, creates the session, and inserts its participant row in one transaction. A missing ownership link fails closed. `BuyInUseCase` inserts membership in the same transaction as a successful operation so participants added by an existing session participant receive future read/write access.

### Read and write authorization are separate

Read authorization is:

- finished session: everyone;
- active session: administrator, authenticated account whose owned player is a participant, or anonymous viewer whose `guest_player_id` is a participant.

Write authorization is:

- start session: authenticated account with an owned player, or administrator with an owned player;
- session mutation: authenticated participant or administrator, with existing domain status rules still deciding which operations are valid before or after finish;
- global domain mutation without a session identifier (player creation and blind-clock controls): authenticated account or administrator.

When authentication is explicitly disabled by deployment configuration, legacy unrestricted behavior remains available as the existing rollback mode.

### Frontend read-only state mirrors but does not replace the server boundary

The session detail response includes a viewer-scoped `can_mutate` capability. The start control remains visible but disabled for guests and displays authentication-required copy. A session opened without that capability hides/disables operation, expense, settlement, and finish controls and shows a read-only notice. Event handlers also reject mutation attempts before creating offline outbox entries. The backend remains authoritative if DOM controls are altered manually.

### Transaction and compatibility behavior

Membership writes share the existing start or buy-in transaction; no partially created session or operation can exist without its corresponding participant. The migration is additive and idempotent. Existing API response shapes remain compatible; only authorization outcomes and visible record sets change.

## Risks / Trade-offs

- [Existing participant with no recorded operation is omitted] -> The product had no separate membership source before this change; administrators retain access and can add the player through the normal buy-in flow if needed.
- [Guest forges `guest_player_id`] -> This is an explicit view-context selector, not authentication. It grants read-only access to an active session in which that player participates and never grants writes.
- [Offline queue stores a guest write] -> Disable local write controls and guard mutation entry points before local persistence; server replay still returns `401` as a final boundary.
- [Stale PWA keeps old controls] -> Advance the coherent shell generation and verify deployed `sw.js`; server authorization protects clients that have not updated.

## Migration Plan

1. Apply the additive participant migration and backfill existing sessions.
2. Deploy to dev, verify anonymous/public reads plus `401`/`403` writes and authenticated participant writes.
3. Promote the same validated commit to `master` and verify production health, deployed shell version, public finished-session access, guest active read-only access, and anonymous write rejection.
4. Roll back application code by revert. The additive participant table may remain; its down migration is available if a full schema rollback is required.
