# Offline session rollout

## Contract

`POST /operations/buy-in` and `POST /operations/cash-out` return `200` with:

```json
{
  "request_id": "request-1",
  "operation_id": "persisted-operation-id",
  "session_id": "session-id",
  "player_id": "player-id",
  "type": "buy_in",
  "chips": 2500,
  "created_at": "2026-08-04T03:00:00Z",
  "idempotent_replay": false
}
```

Reverse returns the same fields with `type: "reversal"` plus
`target_operation_id` and `reversed_operation` containing the persisted target's
`operation_id`, `session_id`, `player_id`, `type`, `chips`, and `created_at`.

The normalized idempotency fingerprint is recovered from the persisted operation:
command type, session, player, chips, and (for reverse) target operation. A duplicate
with the same fingerprint returns the original acknowledgement with
`idempotent_replay: true`. A changed fingerprint returns
`409 idempotency_payload_mismatch` and does not mutate the session.

The idempotency reservation, operation, session totals, outbox event, and resulting
acknowledgement source row are written in one PostgreSQL transaction. Any business
or persistence error rolls the reservation back, so the same request can be retried.

The `200` status is intentionally retained for buy-in and cash-out and adopted for
reverse so a JSON body can be returned. Existing clients that only inspect `res.ok`
remain compatible.

## Rollout guard

Local-first writes are disabled by the deployment meta flag by default:

```html
<meta name="poker-local-first-session-writes" content="false" />
```

An explicit browser cohort can override it:

```js
localStorage.setItem("poker-local-first-session-writes", "true");
location.reload();
```

Only the exact strings `true` and `false` are recognized. Missing, invalid, or
unreadable configuration fails closed. The browser console event
`session_runtime_rollout` exposes the effective value and source. With the flag off,
buy-in, cash-out, and reverse keep the online-first path and no outbox command is
created.

## Verification gate

1. Deploy with local-first writes disabled and verify ordinary online operations.
2. Enable the local override in development for a disposable active session.
3. Record multiple operations offline, reload, reconnect, and verify ordered replay.
4. Repeat one accepted request and verify the same operation ID and no extra totals.
5. Reuse its request ID with changed chips and verify the stable `409` response.
6. Inspect `session_replay_attempt` and backend `operation_acknowledged` logs for
   request ID, command kind, session, attempt/result, and duplicate state.
7. Enable a limited production cohort only after the disposable-session gate passes.
8. Make the deployment default `true` only after blocked commands, retry volume,
   reconciliation failures, and payload mismatches stay within the agreed baseline.

## Rollback

Set the deployment flag to `false` and clear or set the cohort override to `false`.
This immediately restores online-first writes after reload. Do not delete IndexedDB:
already queued commands remain durable and visible, and can be replayed after the
incident is understood or the feature is re-enabled.
