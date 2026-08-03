# Task 08 — Backend Acknowledgements and Rollout

## Objective

Provide the minimum backend contract needed for deterministic client reconciliation, harden replay safety, and roll out the local-first session runtime behind explicit verification gates.

## Depends on

- Tasks 01–07;
- existing backend idempotency and operation persistence behavior.

## Allowed implementation scope

Primary paths:

- operation HTTP handlers and DTOs;
- buy-in, cash-out and reverse use cases only as needed to return acknowledgements;
- idempotency repository/use case only for narrowly specified hardening;
- Swagger/OpenAPI documentation;
- backend and frontend contract tests;
- a narrowly scoped frontend feature flag or rollout guard;
- runbook/release notes.

Do not broaden this task into a general API redesign.

## Requirements

### Operation acknowledgement

Successful buy-in and cash-out responses SHALL return a JSON acknowledgement rather than an empty success body.

The acknowledgement SHALL include at least:

- `request_id`;
- persisted `operation_id`;
- `session_id`;
- `player_id`;
- operation `type`;
- `chips`;
- persisted `created_at`.

Successful reverse SHALL additionally include:

- persisted reverse `operation_id`;
- `target_operation_id`;
- `request_id`;
- the effective reversed operation details needed by the client.

The response MAY include updated session totals or a compact session revision when this can be returned without duplicating query logic.

### Idempotent duplicate acknowledgement

When the backend receives the same `request_id` and the same normalized payload again, it SHALL return an acknowledgement representing the already accepted operation rather than a success with no mapping data.

This requires the idempotency boundary to retain or recover the original result.

### Payload mismatch protection

When an existing `request_id` is reused with a materially different normalized payload, the backend SHALL reject the request with a stable conflict error such as `idempotency_payload_mismatch`.

The backend SHALL not interpret this as an accepted duplicate.

The normalized fingerprint SHOULD include endpoint/command kind and all domain fields that define the effect.

### Atomicity

The idempotency record, operation persistence, session mutation and stored acknowledgement SHALL commit in one database transaction.

A failed business transaction SHALL not leave a committed idempotency record that prevents a valid retry unless that failure result is intentionally stored and returned by contract.

### Backward compatibility

Existing clients that only check `res.ok` SHALL continue to work with the new JSON success body.

HTTP success status SHOULD remain stable unless a deliberate API version decision is documented.

### Client reconciliation

The frontend SHALL use the acknowledgement to:

- map provisional operation id to server operation id;
- mark the exact request accepted;
- replace provisional timestamps or values with persisted values;
- reconcile IndexedDB atomically;
- avoid an immediate full refresh when acknowledgement data is sufficient.

A coalesced background refresh MAY still run for validation and derived statistics.

### Feature flag

The local-first write path SHALL be protected by an explicit rollout guard with a safe online-first fallback.

The flag SHALL default according to the deployment plan and SHALL be inspectable in diagnostics. An unknown flag state SHALL not silently enable local-first writes.

### Rollout stages

Recommended rollout:

1. ship IndexedDB and cached reads with local-first writes disabled;
2. enable local-first writes in development;
3. test offline/reconnect and duplicate replay with a disposable session;
4. enable for a limited production cohort or explicit local setting;
5. monitor blocked commands, replay attempts and mismatch errors;
6. enable by default only after the verification gate passes.

### Diagnostics

Add enough structured diagnostics to determine:

- request id;
- command kind;
- session id;
- replay attempt number;
- acknowledgement result;
- idempotent duplicate versus newly created effect;
- payload mismatch;
- reconciliation failure.

Logs SHALL not expose authentication secrets.

## Non-goals

- no event-sourcing conversion;
- no WebSocket synchronization;
- no generic idempotency framework for every endpoint;
- no offline support for non-approved command kinds;
- no automatic destructive conflict resolution.

## Acceptance scenarios

### New operation acknowledgement

- GIVEN a valid new buy-in request;
- WHEN the backend commits it;
- THEN the response contains the persisted operation id and original request id;
- AND the client can replace its provisional operation deterministically.

### Same request replay

- GIVEN the backend already accepted request `R` with payload `P`;
- WHEN request `R` with payload `P` is sent again;
- THEN the backend returns the same logical acknowledgement;
- AND no second operation or session mutation is created.

### Changed payload rejection

- GIVEN request `R` was accepted for 2500 chips;
- WHEN request `R` is reused for 5000 chips;
- THEN the backend returns `409 idempotency_payload_mismatch` or an equivalent stable conflict;
- AND session state remains unchanged by the second payload.

### Transaction failure retry

- GIVEN operation persistence fails and the transaction rolls back;
- WHEN the same request is retried after the failure is fixed;
- THEN it can execute normally;
- AND it is not falsely treated as a previously accepted duplicate.

### Feature flag disabled

- GIVEN local-first writes are disabled;
- WHEN the user records a buy-in online;
- THEN the existing online-first path remains functional;
- AND no local pending command is fabricated.

### Feature flag enabled offline

- GIVEN local-first writes are enabled and a valid cached session exists;
- WHEN connectivity is lost and the user records a buy-in;
- THEN the command commits locally and later reconciles using the acknowledgement contract.

## Verification

Backend:

- unit and integration tests for new acknowledgement bodies;
- duplicate same-payload test;
- duplicate changed-payload test;
- transaction rollback test;
- reverse acknowledgement test;
- Swagger generation/validation.

Frontend/end-to-end:

- provisional-to-server id mapping test;
- accepted response with reconciliation failure and safe retry;
- feature-flag disabled fallback;
- multiple offline commands followed by reconnect;
- page reload before and during replay;
- throttled network and timeout simulation.

Repository checks:

- `go test ./...`;
- existing web checks;
- focused offline-runtime test suite;
- manual production-like smoke test recorded in the PR.

## Completion output

The implementing agent SHALL report:

- exact success response schema;
- idempotency storage/fingerprint design;
- transaction boundary;
- compatibility analysis;
- feature-flag default and rollout steps;
- automated and manual verification results;
- known rollback procedure.
