# Task 05 — Outbox Replay

## Objective

Deliver pending buy-in and cash-out commands to the backend in deterministic order without blocking interaction or creating duplicate effects.

## Depends on

- Task 01 local database foundation;
- Task 02 network command contract;
- Task 04 optimistic buy-in and cash-out.

## Allowed implementation scope

Primary paths:

- a focused `web/js/offline-sync.js` or equivalent replay module;
- `web/js/app.js` for lifecycle triggers;
- local DB repository functions;
- `web/js/state.js` for replay status;
- focused tests.

Do not implement reverse-operation reconciliation in this task. Do not change backend semantics except where a test fixture requires the existing API contract to be represented accurately.

## Requirements

### Single-flight ordered replay

For each session, replay SHALL select the oldest eligible command by ascending sequence.

At most one command per session SHALL be in flight.

The first implementation MAY serialize replay globally across all sessions. It SHALL not run two replay loops that can send the same command concurrently.

### Lifecycle triggers

Replay SHALL be requested after:

- a local command is committed;
- application startup completes enough local initialization;
- the browser emits `online`;
- the document becomes visible after being hidden;
- a retry delay expires while the app remains active.

Repeated triggers SHALL coalesce into the active replay loop rather than create parallel loops.

### Sending contract

Before sending, replay SHALL reload the latest persisted command and verify:

- status is eligible;
- normalized payload matches the payload fingerprint or deterministic serializer result;
- the command still belongs to the expected session;
- the command has not been superseded or locally cancelled.

The request SHALL use the persisted `request_id` unchanged.

### Sending status durability

Before network send, the command MAY be marked `sending`, but a browser crash SHALL not strand it permanently.

On startup, stale `sending` commands SHALL become retryable `pending` commands after a defined lease timeout.

### Success handling

On an accepted server response, replay SHALL not immediately erase the pending command in isolation.

It SHALL call a reconciliation boundary that atomically:

- records acknowledgement data available from the response;
- updates or confirms the local snapshot;
- removes or marks the command processed;
- updates replay metadata.

Until Task 08 provides richer acknowledgement data, replay MAY perform one coalesced server refresh after success. The command SHALL remain protected from duplicate local removal until reconciliation succeeds.

### Retry handling

Retryable failures SHALL:

- increment attempts;
- set `last_attempt_at`;
- store the classified error kind;
- compute bounded exponential backoff with jitter;
- return command status to `pending`;
- stop the current loop when immediate repeated failure would be wasteful.

The retry schedule SHALL have an upper bound suitable for an interactive PWA. Connectivity lifecycle triggers MAY retry earlier than `next_attempt_at` when explicitly allowed by policy.

### Blocking handling

Authorization failures SHALL stop replay and keep all affected commands durable.

Domain failures SHALL mark the current command `blocked` or `conflict`, preserve server error code/details and stop later commands for the same session.

A blocked earlier command SHALL prevent dependent later commands from replaying automatically.

### Observability

Replay SHALL expose local status usable by UI and diagnostics:

- idle;
- syncing;
- waiting for retry;
- authorization blocked;
- domain blocked;
- last successful replay time;
- pending count.

Sensitive payload values SHALL not be logged unnecessarily.

## Non-goals

- no Background Sync API dependency;
- no service-worker-owned replay;
- no cross-device merging;
- no automatic resolution of domain conflicts;
- no reverse-operation support in this task.

## Acceptance scenarios

### FIFO replay

- GIVEN three pending commands with sequence 1, 2 and 3;
- WHEN replay runs;
- THEN command 1 is accepted before command 2 is sent;
- AND command 2 is accepted before command 3 is sent.

### Retry uses same request id

- GIVEN command 1 times out twice and then succeeds;
- WHEN replay retries it;
- THEN all attempts send the same request id and normalized payload;
- AND only one server effect is expected under backend idempotency.

### Parallel triggers coalesce

- GIVEN startup, `online` and a local commit trigger replay close together;
- WHEN one loop is already active;
- THEN no second send of the same command starts concurrently.

### Authorization block

- GIVEN the first command receives `401`;
- THEN it remains durable;
- AND replay status becomes authorization blocked;
- AND later commands are not sent.

### Domain block

- GIVEN a cash-out receives `409 session_not_active`;
- THEN the command stores the error details and becomes blocked;
- AND later commands for that session remain pending and unsent.

### Crash recovery

- GIVEN a command was marked `sending` and the app closed before completion;
- WHEN the app starts after the sending lease expires;
- THEN the command becomes eligible for retry with the same request id.

## Verification

- deterministic fake transport tests;
- fake timers for backoff;
- concurrent-trigger single-flight test;
- stale sending lease recovery test;
- authorization and domain block tests;
- refresh/reconciliation failure preservation test;
- existing frontend checks.

## Completion output

The implementing agent SHALL report:

- replay trigger list;
- single-flight mechanism;
- backoff policy;
- sending lease duration;
- blocking behavior;
- tests proving stable identity and ordered delivery.
