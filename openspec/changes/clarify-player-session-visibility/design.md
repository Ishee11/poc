## Context

`GET /stats/player` currently combines two data scopes. `GetPlayerOverall` aggregates every effective operation for the player in the requested period and therefore produces the real session count and financial result. `ListPlayerSessions` uses the existing guest/user/admin visibility predicate and returns only viewable session details, capped at 100. The frontend renders the overall `player.sessions_count`, but labels the disclosure from `sessions.length` and omits the history section entirely when that array is empty.

The existing access model already permits aggregate player statistics while withholding closed-session details. This change must explain that boundary without changing it or inferring hidden history from financial values.

## Goals / Non-Goals

**Goals:**

- Give the player-detail response authoritative, separately named total and viewer-visible session counts for the selected period.
- Keep financial aggregates and the primary Sessions value aligned to all real player sessions.
- Keep detailed session records strictly filtered by the current access predicate.
- Render complete, partial, none-visible, truly-empty, and unavailable states explicitly.
- Preserve compatibility for consumers of `player.sessions_count`.

**Non-Goals:**

- Changing guest, user, owner, or administrator authorization rules.
- Returning identifiers, dates, names, statuses, amounts, participants, or any other fields for hidden sessions.
- Adding session-list pagination or changing player ranking semantics.
- Deriving visibility from profit, loss, or any other financial aggregate.

## Decisions

### Counts are response metadata with explicit scope

`GET /stats/player` adds top-level `total_sessions_count` and `visible_sessions_count`. The existing `player.sessions_count` remains as a compatibility alias for the total; new UI code consumes the explicit top-level fields. Top-level placement keeps viewer-dependent visibility metadata separate from the all-session player aggregate.

Alternative considered: reinterpret `player.sessions_count` as the visible count. Rejected because it would recreate the misleading mismatch with total financial statistics and break existing consumers.

### The repository counts visible sessions directly

Add a repository operation that performs `COUNT(DISTINCT session_id)` for the requested player and date range using the same authorization predicate as `ListPlayerSessions`. The use case obtains the overall aggregate, visible count, and visible details inside its existing read transaction. It does not use `len(sessions)`, because the detail list has a safety limit and count semantics must remain correct independently of representation limits.

Alternative considered: count the returned array. Rejected because it confuses list size with the number of sessions a viewer may access.

### Aggregate disclosure stays within the established privacy boundary

The total count is derived from the same all-session population already disclosed through `player.sessions_count`, totals, averages, and PnL. Hidden rows remain absent from `sessions`, and the new count query returns only one integer. No authorization predicate or session-detail route changes.

Alternative considered: suppress the total for non-owners. Rejected because it would conflict with the existing product model that intentionally exposes aggregate player statistics and would require a broader authorization product decision.

### UI state is based only on explicit counts

The primary Sessions card renders `total_sessions_count`. When `visible < total`, it also renders a localized viewer-availability line. With `total > 0` and `visible = 0`, the history area renders an explanatory empty state. With `total = 0`, it renders a normal no-sessions state. If the explicit total is absent or invalid, it renders an em dash and neutral unavailable copy; financial hints explain that aggregates can include unavailable history. No branch checks whether profit is non-zero.

The disclosure continues to show only records in `sessions`; its label uses the explicit visible count. If the count exceeds the current 100-record detail limit, the count remains authoritative while the list remains bounded.

### Compatibility and documentation

The contract is additive and needs no database migration. Generated Swagger artifacts document both new fields and the compatibility meaning of `player.sessions_count`. The service worker shell cache version is advanced because frontend modules and copy change, preventing production clients from retaining the old presentation.

## Risks / Trade-offs

- [Visibility predicate drifts between count and list] -> Centralize the SQL predicate shape as closely as the current query style permits and cover guest, owner, unrelated user, and admin cases with integration tests.
- [A total count reveals that hidden sessions exist] -> This is no broader than the existing all-session count and financial aggregates; return only an integer and no hidden-row metadata.
- [Visible count exceeds the detail-list cap] -> Keep count semantics truthful and avoid claiming that the returned array is exhaustive; pagination remains a future change.
- [New frontend is briefly served by an old backend] -> Treat missing explicit counts as unavailable, never as zero.
- [Old cached frontend misses the copy] -> Bump the shell cache version and verify the deployed service worker.

## Migration Plan

1. Add and validate the additive response fields and tests locally.
2. Deploy the commit from `dev`, verify API privacy/count semantics and UI states in dev.
3. Promote the same commit to `master`, wait for the production workflow, then verify `/health`, the production API contract, and deployed service-worker version.
4. Roll back by reverting the commit and redeploying; no database rollback is needed.

## Open Questions

None.
