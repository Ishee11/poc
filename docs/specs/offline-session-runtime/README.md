# Offline Session Runtime

## Status

Proposed implementation specification.

## Purpose

This specification defines the minimal transition of Poker Session Control from an online-first session screen to a resilient local-first runtime for the actions that matter during a live poker game.

The goal is not to copy the full StoicTime synchronization architecture. The goal is to remove network latency from routine session actions, preserve the last usable session state during weak connectivity, and guarantee that accepted local actions are either synchronized or remain visibly pending.

## Product outcome

After the transition, a user who has already opened an active session SHALL be able to record a buy-in or cash-out without waiting for the network. The screen SHALL update immediately, the change SHALL survive a page reload, and the same command SHALL be delivered to the backend after connectivity returns.

The application SHALL also reopen from its installed PWA shell after at least one successful online visit and SHALL display the last locally cached active-session state when the API is temporarily unavailable.

## Scope

The first local-first release covers:

- cached hydration of an already known active session;
- session, session-player and operation snapshots in IndexedDB;
- durable outbox commands;
- stable `request_id` values across retries;
- optimistic buy-in and cash-out projections;
- ordered replay and server reconciliation;
- safe reversal handling for local and server-known operations;
- network timeout and error classification;
- non-destructive handling of failed refreshes;
- offline PWA app-shell caching;
- visible pending, syncing, offline and conflict states;
- backend operation acknowledgements needed to reconcile provisional rows.

## Explicit non-goals

The first release does not make these actions local-first:

- creating or starting a session;
- creating or renaming a player;
- adding a player who is not already known locally;
- expenses and expense closing;
- settlement-transfer editing;
- finishing or reopening a session;
- administration actions;
- authentication and authorization;
- blind-clock commands;
- cross-device real-time collaboration;
- generic conflict resolution comparable to StoicTime.

These actions SHALL remain online-only and SHALL fail calmly when offline.

## Runtime ownership

PostgreSQL remains the authoritative durable server state.

IndexedDB becomes the durable client runtime owner for the currently cached session slice. In-memory `state` remains a rendered copy and interaction cache, not the only durable source of truth.

The service worker owns only application-shell availability. It SHALL NOT cache API responses as domain truth.

## Core invariants

### Local commit before network

For approved local-first commands, the application SHALL atomically persist both:

1. the next local session projection; and
2. the corresponding outbox record.

The UI SHALL render the committed local projection before awaiting any network work.

If IndexedDB cannot safely commit both records, the application SHALL NOT report the action as locally saved.

### Stable command identity

Every queued command SHALL receive one stable `request_id` before its first send attempt. All retries of the same command SHALL use the same `request_id` and the same normalized payload.

A payload change SHALL create a new command and a new `request_id`.

### Ordered replay

Commands for one session SHALL replay in ascending sequence order with at most one in-flight command per session.

A later command SHALL NOT overtake an earlier unresolved command when doing so could change session totals or operation history.

### No silent loss

A timeout, offline state, `5xx`, page reload or browser restart SHALL NOT remove a pending command.

A domain rejection, authorization rejection or reconciliation mismatch SHALL stop affected replay and produce a visible blocked/conflict state. The client SHALL NOT silently discard the local projection.

### Refresh does not erase valid local state

Failed GET requests SHALL preserve the last valid IndexedDB-backed query model. Network failure SHALL NOT replace cached arrays with empty arrays or cached objects with synthetic defaults.

A successful server response containing an empty list MAY replace the cached list because it is confirmed server state.

### Backend remains authoritative after acknowledgement

When replay succeeds, server-returned identifiers and values SHALL replace provisional client values where applicable. The result SHALL first be reconciled into IndexedDB and only then rendered by the UI.

## Local data model

The minimal database SHOULD be named `poker-session-local` and SHOULD initially contain these stores:

### `session_snapshots`

Key: `session_id`.

Required fields:

- `session_id`;
- normalized session object;
- normalized session-player array;
- normalized operation array;
- optional expense and settlement arrays only when already loaded online;
- `server_updated_at` when known;
- `cached_at`;
- `local_revision`;
- `last_server_refresh_status`.

### `outbox`

Key: `request_id`.

Required fields:

- `request_id`;
- `session_id`;
- monotonically increasing `sequence` within the session;
- `kind`;
- normalized payload;
- `created_at`;
- `status`: `pending`, `sending`, `blocked` or `conflict`;
- `attempts`;
- `last_attempt_at`;
- `next_attempt_at`;
- `last_error_kind`;
- optional provisional entity identifiers.

### `sync_state`

Keyed records for:

- active cached session id;
- last successful server refresh;
- last successful replay;
- pending count;
- conflict count;
- schema version and migration metadata.

A separate conflict store is optional for the first release. A blocked outbox record with structured details is sufficient until more than one conflict workflow exists.

## Approved command kinds

The first implementation SHALL define an explicit closed registry:

- `buy_in`;
- `cash_out`;
- `reverse_operation`.

No generic endpoint-plus-body queue is allowed. Each kind SHALL have a validator, local projector, API serializer, replay handler and reconciliation rule.

## Command lifecycle

1. Validate the command against the current local snapshot.
2. Generate `request_id`, provisional identifiers and session sequence.
3. Apply the pure local projection.
4. Commit the projection and outbox command in one IndexedDB transaction.
5. Render the new local query model.
6. Trigger replay without blocking interaction.
7. Send the original command with its stable `request_id`.
8. On success, reconcile the acknowledgement and refreshed server slice into IndexedDB.
9. Remove or mark the command processed only inside the same reconciliation transaction.
10. Render the reconciled local query model.

## Read lifecycle

When opening a known session:

1. Read the cached snapshot.
2. Render it immediately when structurally valid.
3. Start a server refresh in the background.
4. On success, merge server state without deleting provisional effects belonging to pending commands.
5. On failure, retain cached state and expose offline/stale status.

When no safe cached snapshot exists, the current online-first loading flow remains the fallback.

## Error policy

### Retryable

- browser offline;
- fetch network failure;
- timeout;
- HTTP `408`, `425`, `429` and `5xx`.

Retryable failures keep the command pending. Automatic retries SHALL use bounded exponential backoff with jitter and SHALL also run on startup, `online` and visible-tab resume.

### Authorization-blocking

- HTTP `401`;
- HTTP `403`.

Replay SHALL stop. The outbox SHALL remain durable. The UI SHALL require restored authorization before retry.

### Domain-blocking

- HTTP `400`, `404` or `409` indicating stale session state, inactive session, missing player, invalid operation target or invariant violation.

The command SHALL become blocked or conflicted with the server error code and details. Later dependent commands for the same session SHALL not replay automatically.

## Reconciliation rules

- A server operation acknowledgement SHALL include enough data to map the provisional operation to the persisted operation.
- Pending commands SHALL remain visible while a background refresh is in flight.
- A refresh SHALL not overwrite rows or totals that include pending local effects with an older server-only projection.
- After each accepted command, the client MAY perform one coalesced session refresh instead of reloading every endpoint after every click.
- Reconciliation SHALL be revision-aware so a refresh started before a newer local command cannot erase that newer projection.

## UX requirements

The session screen SHALL distinguish:

- saved locally and pending;
- currently syncing;
- confirmed by server;
- offline with pending count;
- blocked/conflict requiring attention.

Routine buy-in and cash-out controls SHALL not display a full-screen or section-wide loading state while replay is running.

The user SHALL be prevented from accidental duplicate clicks by local command identity and short-lived control debouncing, not by waiting for the server response.

## Service worker boundary

The service worker SHALL cache versioned application-shell assets required to render the application:

- `/`;
- the current CSS and JavaScript modules;
- manifest;
- required SVG and image assets.

Navigation requests SHOULD use network-first with cached `/` fallback. Versioned static assets SHOULD use cache-first or stale-while-revalidate.

Requests to API routes SHALL bypass domain response caching.

## Delivery order

Implementation SHALL proceed as the following independently reviewable slices:

1. `01-local-db-foundation.md`;
2. `02-network-command-contract.md`;
3. `03-cached-session-hydration.md`;
4. `04-optimistic-buy-in-cash-out.md`;
5. `05-outbox-replay.md`;
6. `06-reverse-and-reconciliation.md`;
7. `07-offline-shell-and-sync-ui.md`;
8. `08-backend-acknowledgements-and-rollout.md`.

An agent implementing one slice SHALL read this document and every listed dependency first. It SHALL not absorb later slices merely because adjacent code is convenient to modify.

## Global verification gate

Before the transition is considered complete:

- existing Go tests SHALL pass;
- existing frontend syntax/import checks SHALL pass;
- focused IndexedDB tests SHALL cover atomic projection-plus-outbox writes;
- focused replay tests SHALL cover retries without duplicate effects;
- browser tests SHALL cover reload while offline with pending commands;
- service-worker tests or scripted checks SHALL confirm offline app-shell startup;
- manual testing SHALL include throttled, offline and intermittent-network modes;
- no online-only action SHALL falsely claim offline success.

## Completion definition

The transition is complete when a user can open an active session online, lose connectivity, record multiple buy-ins and cash-outs with immediate UI updates, reload the application, continue seeing those changes as pending, reconnect, and observe exactly-once server effects followed by confirmed server state without manual data reconstruction.
