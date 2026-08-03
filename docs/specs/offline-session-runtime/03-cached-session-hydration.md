# Task 03 — Cached Session Hydration

## Objective

Render the last valid cached session immediately and preserve it when background refreshes fail, without yet making write actions optimistic.

## Depends on

- Task 01 local database foundation;
- Task 02 network classification.

## Allowed implementation scope

Primary paths:

- `web/js/app.js`;
- `web/js/ui/session.js`;
- `web/js/ui/player.js`;
- `web/js/ui/lobby.js` only for the last-known active-session card;
- `web/js/state.js`;
- one focused session-cache/query-model module;
- focused frontend tests.

Do not modify buy-in, cash-out or reverse success flow beyond preserving cached state.

## Requirements

### Cached-first open

When `openSession(sessionId)` is called, the application SHALL first request a validated cached snapshot from IndexedDB.

When a valid snapshot exists, the application SHALL:

- hydrate `state.session`, `state.players`, `state.operations` and any already cached optional arrays;
- render the session screen immediately;
- mark the data as cached/stale until refreshed;
- start network refresh without blocking navigation.

When no valid snapshot exists, retain the current online-first fallback.

### Server refresh persistence

A successful complete refresh SHALL be normalized and written to IndexedDB before replacing the visible cached query model.

The refresh boundary SHALL define what counts as complete. At minimum, session, players and operations are one required session slice. Optional expenses and settlement data MAY retain their previous cached values when their independent request fails.

### Non-destructive failure handling

A failed GET SHALL NOT replace a previously valid local array with `[]` or a valid object with `null`.

The loader SHALL distinguish:

- successful empty server result;
- failed request with cached fallback;
- failed request with no cached fallback.

Only the first case may write and render an empty confirmed list.

### Refresh race protection

Every refresh SHALL capture the session id and local revision observed at start.

Before applying the result, it SHALL verify that:

- the same session is still active in the UI;
- no newer local revision has superseded the refresh.

A stale refresh result MAY be persisted only through a merge that preserves newer local state; otherwise it SHALL be ignored.

### Startup behavior

Application startup SHALL not wait for all lobby and statistics requests before routing to a cached session URL.

When the current route identifies a cached session, route hydration SHOULD begin before non-critical lobby/player-overview requests complete.

### Cache freshness metadata

State SHALL expose at least:

- `sessionDataSource`: `server`, `cache` or `none`;
- `sessionCachedAt`;
- `sessionRefreshStatus`: `idle`, `refreshing`, `failed` or `fresh`.

This task does not require final visual styling, but the metadata SHALL be usable by Task 07.

## Non-goals

- no local-first writes;
- no outbox replay;
- no provisional operations;
- no service-worker changes;
- no offline creation of sessions or players.

## Acceptance scenarios

### Immediate cached render

- GIVEN a valid cached snapshot for session `S`;
- AND the network response is delayed;
- WHEN the user opens `/session/S`;
- THEN cached session content renders without waiting for the delayed API;
- AND refresh status becomes `refreshing`.

### Failed refresh preserves state

- GIVEN cached players and operations are visible;
- WHEN both network requests fail;
- THEN the same players and operations remain visible;
- AND refresh status becomes `failed`;
- AND no empty confirmed snapshot is written.

### Confirmed empty result

- GIVEN cached operations exist;
- WHEN the server successfully returns an empty operations array for the same session;
- THEN the empty array is persisted as server-confirmed state;
- AND the operation list renders empty.

### Route race

- GIVEN refresh for session `A` is in flight;
- WHEN the user opens session `B` before it completes;
- THEN the result for `A` does not overwrite state for `B`.

## Verification

- delayed-promise tests for cached-first rendering;
- failed-refresh preservation tests;
- successful-empty distinction test;
- stale-session and stale-revision race tests;
- existing frontend checks.

## Completion output

The implementing agent SHALL report:

- session-slice completeness rule;
- cache normalization shape;
- refresh race guard;
- current loaders changed from destructive to preserving behavior;
- tests added.
