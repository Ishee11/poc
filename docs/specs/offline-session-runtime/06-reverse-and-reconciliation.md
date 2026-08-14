# Task 06 — Reverse and Reconciliation

## Objective

Support safe reversal of provisional and server-known operations and define how server state is merged without erasing newer pending local effects.

## Depends on

- Task 03 cached session hydration;
- Task 04 optimistic buy-in and cash-out;
- Task 05 outbox replay.

## Allowed implementation scope

Primary paths:

- `web/js/ui/session.js`;
- the pure projection module;
- the replay/reconciliation module;
- local DB repository functions;
- focused tests.

Backend response improvements belong to Task 08. This task SHALL work with the best acknowledgement available and MAY request a coalesced refresh when needed.

## Requirements

### Reverse target classification

Before reversing, the client SHALL classify the target as one of:

- `pending_unsent`: persisted locally and never sent;
- `possibly_sent`: sending lease active, timed out, or send outcome unknown;
- `server_confirmed`: mapped to a server operation id;
- `already_reversed`;
- `unavailable`.

The classification SHALL be derived from durable records, not only DOM state.

### Cancel unsent provisional command

For `pending_unsent`, reversal MAY be implemented as local cancellation rather than a server reverse command.

Cancellation SHALL atomically:

- remove or mark cancelled the original outbox command;
- remove its provisional operation;
- apply the inverse projection to session and player totals;
- increment local revision;
- persist the resulting snapshot.

The UI SHALL show the reverted state immediately.

### Unknown send outcome

For `possibly_sent`, the client SHALL NOT simply delete the original command.

It SHALL first establish server state by replaying/idempotently retrying the original command or refreshing and matching it through request id/acknowledgement data. Until resolved, the reverse action SHALL become pending behind the original lineage or SHALL be calmly unavailable with an explanation.

### Reverse confirmed operation

For `server_confirmed`, reversal SHALL create a new `reverse_operation` outbox command with its own stable request id and the confirmed target operation id.

The client MAY apply an optimistic inverse projection only when the target and its effective contribution are unambiguous in the local snapshot.

The reverse command SHALL follow normal ordered replay and reconciliation.

### Reversal idempotency

The UI SHALL not create two active reverse commands for the same target lineage.

A repeated request after successful reversal SHALL be treated as already reversed or refreshed from server truth.

### Merge server refresh with pending operations

Reconciliation SHALL use this conceptual order:

1. normalize the latest server-confirmed session slice;
2. identify still-pending local commands in sequence order;
3. replay their pure local projectors over the server base;
4. persist the merged snapshot and refreshed metadata;
5. render from the persisted result.

A server refresh SHALL not directly overwrite a snapshot containing pending effects.

### Revision guard

A refresh or acknowledgement reconciliation SHALL compare the local revision/pending revision captured when network work began with the latest revision before commit.

When newer commands appeared, reconciliation SHALL rebase over them or retry from fresh local data. It SHALL not commit a stale server-only snapshot.

### Reconciliation failure

If the server accepted a command but local reconciliation fails, the command SHALL remain in a recoverable durable state. A later replay MAY resend it with the same request id and reconcile the idempotent result or server refresh.

The implementation SHALL not convert this case into a duplicate local projection.

## Non-goals

- no generic user-facing merge editor;
- no reversal of expenses or settlement transfers;
- no reversal after session finish unless the current backend explicitly supports it;
- no server-side event stream.

## Acceptance scenarios

### Cancel unsent buy-in

- GIVEN an offline buy-in is pending and has never been sent;
- WHEN the user reverses it;
- THEN the original command is cancelled atomically;
- AND totals and player state return to their prior values;
- AND no server reverse command is queued.

### Do not delete uncertain command

- GIVEN a buy-in request timed out after send began;
- WHEN the user asks to reverse it;
- THEN the original command remains durable;
- AND the client does not assume it was absent from the server;
- AND resolution waits for retry/refresh identity matching.

### Reverse confirmed operation

- GIVEN a buy-in has a confirmed server operation id;
- WHEN the user reverses it;
- THEN one reverse command with a new request id targets that server id;
- AND the target cannot receive another concurrent reverse command.

### Refresh preserves later pending command

- GIVEN a server refresh starts;
- AND a new local buy-in commits before the refresh finishes;
- WHEN reconciliation applies;
- THEN the refreshed server base is combined with the newer pending buy-in;
- AND the newer projection remains visible and queued.

### Accepted-but-not-reconciled recovery

- GIVEN the server accepted a command;
- AND IndexedDB reconciliation aborted;
- WHEN replay runs again;
- THEN the same request id is retried or matched;
- AND no duplicate local command or server effect is created.

## Verification

- inverse-projector tests;
- unsent cancellation atomicity test;
- unknown-outcome safety test;
- duplicate reverse prevention test;
- refresh rebase over pending commands test;
- accepted-but-reconciliation-failed recovery test;
- existing frontend checks.

## Completion output

The implementing agent SHALL report:

- target classification rules;
- when inverse projection is allowed;
- command-lineage representation;
- server refresh merge algorithm;
- revision guard;
- tests added.
