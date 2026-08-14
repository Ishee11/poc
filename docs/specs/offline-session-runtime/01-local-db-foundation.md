# Task 01 — Local DB Foundation

## Objective

Introduce a minimal IndexedDB boundary for durable session snapshots, outbox commands and synchronization metadata without changing current user-visible write behavior.

## Depends on

- `README.md` in this directory.

## Allowed implementation scope

Primary paths:

- `web/js/offline-db.js` or an equivalent focused module;
- `web/js/state.js` only for local-runtime status fields;
- `web/js/app.js` only for database initialization;
- focused frontend tests and test helpers;
- `scripts/check-web.mjs` only when required to include new modules.

Avoid changes to Go handlers, service worker behavior and session action handlers in this task.

## Requirements

### Database opening and migration

The application SHALL expose one shared asynchronous database opener.

The database SHALL have an explicit name, integer schema version and upgrade path. Upgrade failures SHALL reject initialization without deleting existing data automatically.

At minimum, create these stores:

- `session_snapshots`, key path `session_id`;
- `outbox`, key path `request_id`;
- `sync_state`, key path `key`.

The `outbox` store SHALL support ordered lookup by session and sequence, either through an index or an equivalent deterministic query.

### Atomic write API

The module SHALL expose a transaction-level operation that accepts:

- the next normalized session snapshot;
- exactly one new or updated outbox command;
- optional sync-state updates.

All supplied records SHALL commit atomically. A failed transaction SHALL leave both snapshot and outbox unchanged.

### Snapshot API

Expose focused functions to:

- read a snapshot by session id;
- write a server-confirmed snapshot;
- write a locally projected snapshot together with an outbox command;
- delete a snapshot only through an explicit function;
- list pending commands for one session in sequence order;
- count pending and blocked commands.

Do not expose raw stores to arbitrary UI modules.

### Record validation

Records read from IndexedDB SHALL be validated before use. Structurally invalid records SHALL be ignored or quarantined and SHALL not be rendered as trusted session state.

Validation SHALL cover required identifiers, arrays, command kind, sequence and status.

### Multi-tab behavior

The first release MAY be single-writer. However, the module SHALL emit a local-runtime change event after successful writes so another part of the same tab can refresh its query model.

The implementation SHOULD reserve a path for `BroadcastChannel`, but cross-tab locking is not required in this task.

## Non-goals

- no optimistic buy-in or cash-out;
- no replay worker;
- no API changes;
- no service-worker caching;
- no conflict UI;
- no data import from localStorage.

## Acceptance scenarios

### Atomic projection and enqueue

- GIVEN a valid snapshot and valid buy-in command;
- WHEN the atomic write API succeeds;
- THEN both records are visible in subsequent reads;
- AND the command is returned in sequence order.

### Transaction failure

- GIVEN a valid existing snapshot;
- AND an invalid outbox record that causes the transaction to abort;
- WHEN the atomic write API is called;
- THEN the original snapshot remains unchanged;
- AND no partial outbox record exists.

### Upgrade preserves data

- GIVEN a database created at the previous schema version with a stored snapshot;
- WHEN the application opens the next schema version;
- THEN the migration completes;
- AND the existing valid snapshot remains readable.

### Unavailable IndexedDB

- GIVEN IndexedDB is unavailable or opening fails;
- WHEN initialization runs;
- THEN the app records local-runtime unavailability;
- AND current online-first behavior remains usable;
- AND the UI does not claim that offline saving is enabled.

## Verification

- unit tests with a fake IndexedDB implementation;
- test atomic commit and rollback;
- test sequence ordering;
- test schema upgrade;
- run the existing frontend import/syntax check.

## Completion output

The implementing agent SHALL report:

- created stores and indexes;
- public module API;
- migration version;
- tests added;
- any browser limitations discovered.
