# Task 04 — Optimistic Buy-In and Cash-Out

## Objective

Make buy-in and cash-out commit locally and render immediately while preserving the exact command needed for later server replay.

## Depends on

- Task 01 local database foundation;
- Task 02 network command contract;
- Task 03 cached session query model.

## Allowed implementation scope

Primary paths:

- `web/js/ui/session.js`;
- `web/js/ui/player.js` only for rendering provisional player values;
- `web/js/state.js`;
- a focused pure projection module;
- the local DB repository from Task 01;
- focused tests.

Replay networking belongs to Task 05. This task MAY trigger a replay hook, but SHALL not own retry scheduling.

## Requirements

### Pure projectors

Implement pure functions for:

- applying buy-in to a normalized local snapshot;
- applying cash-out to a normalized local snapshot.

The projectors SHALL not access DOM, IndexedDB or fetch.

They SHALL validate the same essential invariants the UI can know locally, including positive chips, active session, known player and cash-out eligibility.

### Provisional operation

Each accepted local command SHALL create a provisional operation with:

- a locally generated stable id;
- `request_id`;
- session id;
- player id;
- type;
- chips;
- local creation timestamp;
- `sync_status: "pending"`;
- outbox sequence.

The provisional id SHALL remain stable across reloads until reconciliation.

### Session projection

A local buy-in SHALL immediately update the same visible fields that a successful server refresh would update, including:

- player buy-in;
- player in-game status when applicable;
- player chip and money profit values derivable from current local data;
- session total buy-in;
- session chips on table;
- operation history.

A local cash-out SHALL immediately update:

- player cash-out;
- player in-game status;
- player profit;
- session total cash-out;
- session chips on table;
- operation history.

Calculations SHALL use one shared normalization/projection path rather than duplicating arithmetic across click handlers and renderers.

### Atomic durable command

The click handler SHALL:

1. construct and validate the command;
2. produce the next snapshot with the pure projector;
3. atomically persist snapshot and outbox record;
4. hydrate/render from the persisted or equivalent committed query model;
5. close the modal and report local success;
6. trigger background replay without awaiting it.

The UI SHALL not report success before the IndexedDB transaction commits.

### Duplicate interaction protection

A single modal confirmation SHALL create at most one local command.

The confirmation control SHALL be disabled or guarded while the local IndexedDB transaction is committing. It SHALL be released after commit or failure without waiting for network replay.

### Local persistence failure

When the local transaction fails:

- no optimistic state SHALL remain visible;
- no outbox record SHALL exist;
- the user SHALL see a local-save failure;
- the implementation MAY fall back to the existing online-first request only when explicitly designed and tested to avoid duplicate command identity.

Default behavior SHOULD be to fail calmly rather than silently switch semantics.

### Pending rendering

Provisional operation rows SHALL be visibly distinguishable through data/class attributes usable by Task 07.

Existing totals and player rows SHALL reflect provisional values without requiring special duplicate render branches.

## Non-goals

- no cash-out conflict recovery;
- no reverse behavior;
- no automatic replay loop;
- no optimistic session finish;
- no optimistic player creation;
- no expenses or settlement projections.

## Acceptance scenarios

### Offline buy-in

- GIVEN a cached active session and known player;
- AND the browser is offline;
- WHEN the user records a 2500-chip buy-in;
- THEN the local transaction commits;
- AND totals, player row and operation history update immediately;
- AND one pending outbox command exists;
- AND no network response is required for the visible update.

### Offline cash-out

- GIVEN a player is in game in the local snapshot;
- WHEN the user records a valid cash-out;
- THEN the player becomes locally settled;
- AND local session totals update consistently;
- AND one pending cash-out command exists.

### Reload persistence

- GIVEN one pending buy-in was locally accepted;
- WHEN the page reloads before replay succeeds;
- THEN cached hydration shows the same projected totals and provisional operation;
- AND the same request id remains queued.

### Double confirmation

- GIVEN the user double-clicks modal confirmation;
- WHEN the first local transaction is still committing;
- THEN only one provisional operation and one outbox command are created.

### Invalid local action

- GIVEN a finished session or non-positive chip value;
- WHEN the user submits the action;
- THEN no local projection or outbox command is written.

## Verification

- pure projector table tests;
- arithmetic and player-state tests;
- transaction failure rollback test;
- double-submit test;
- reload-from-snapshot test;
- existing frontend checks.

## Completion output

The implementing agent SHALL report:

- projector inputs and outputs;
- provisional operation shape;
- UI handlers converted;
- local failure behavior;
- tests proving immediate render and durable reload.
