## Why

Player detail currently combines all-session financial aggregates with a visibility-filtered session list, so users can see a profitable player beside an apparently empty history without any explanation. The API and UI need explicit total-versus-visible session semantics while preserving the existing authorization boundary.

## What Changes

- Add an explicit aggregate count of player sessions visible to the current viewer alongside the existing all-session count.
- Keep overall player statistics based on every real session in the selected period, and keep the detailed session list filtered by the current access rules.
- Present the all-session count as the primary `Sessions` value and explain partial or zero visibility without exposing hidden session fields.
- Add an informative empty state when sessions exist but none are viewable, and an unavailable fallback for clients that receive no authoritative total.
- Update API documentation and backend/frontend tests for total, partial, zero, and fully visible histories.
- The API change is additive; there is no breaking migration or authorization change.

## Capabilities

### New Capabilities

- `player-session-visibility-summary`: Defines authoritative total and viewer-visible player session counts, privacy boundaries, and corresponding UI states.

### Modified Capabilities

None. This repository has no canonical player-statistics capability yet.

## Impact

- Player statistics use case, repository port, PostgreSQL stats queries, and HTTP response contract.
- Player detail rendering, Russian and English interface copy, and web checks.
- Swagger artifacts and access-model documentation.
- No schema migration, new dependency, or expansion of session access is required.
