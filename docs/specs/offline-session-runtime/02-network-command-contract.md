# Task 02 — Network and Command Contract

## Objective

Make network failures bounded and classifiable, and make write-command identity stable enough for durable replay.

## Depends on

- `README.md`;
- Task 01 public types where command records are shared.

## Allowed implementation scope

Primary paths:

- `web/js/api.js`;
- one focused network/error utility module if needed;
- focused frontend tests;
- API documentation only when a public response contract changes.

Do not introduce optimistic projections or a replay loop in this task.

## Requirements

### Request timeout

The common request function SHALL use `AbortController` when available.

Default timeout classes SHALL be explicit:

- reads: approximately 5 seconds;
- routine writes: approximately 8 seconds;
- authentication MAY use the write timeout;
- callers MAY override the timeout only through a named option.

The timeout SHALL be cleared after every completed request.

### Error classification

Every request result SHALL retain the existing `ok`, `status`, `body` and `text` fields and SHALL add a stable classification usable by replay logic.

At minimum distinguish:

- `none`;
- `offline`;
- `timeout`;
- `network`;
- `authorization`;
- `retryable_http`;
- `domain`;
- `invalid_response`.

Classification SHALL not depend only on `navigator.onLine`; an actual fetch failure remains a network failure even when the browser reports online.

### Stable request id injection

Write helpers SHALL accept an optional caller-provided `requestId`.

When supplied, the exact value SHALL be serialized as `request_id`.

When omitted, the helper SHALL generate one request id exactly once for that call and return or expose it to the caller so the command can be persisted before retry.

The implementation SHALL not generate a new id inside each retry attempt.

### Normalized command serializers

Provide explicit serializers for:

- buy-in;
- cash-out;
- reverse operation.

The serializer SHALL produce deterministic payload keys and values. Replay code SHALL be able to compare the normalized payload with the persisted payload before sending.

### Retry policy helper

Expose a pure policy helper that maps a response classification and HTTP status to one of:

- `accepted`;
- `retry`;
- `block_authorization`;
- `block_domain`.

The helper SHALL treat `401` and `403` as authorization blocking, supported transient statuses and `5xx` as retryable, and other validated `4xx` domain responses as domain blocking.

## Non-goals

- no automatic retry loop;
- no IndexedDB writes from `api.js`;
- no UI notification redesign;
- no service-worker handling;
- no backend idempotency redesign in this task.

## Acceptance scenarios

### Timeout classification

- GIVEN a request that never resolves;
- WHEN its configured timeout elapses;
- THEN the request is aborted;
- AND the returned result is classified as `timeout`;
- AND no unhandled rejection occurs.

### Stable retry identity

- GIVEN a buy-in command with request id `abc`;
- WHEN the same persisted command is serialized and sent three times;
- THEN all three request bodies contain `request_id: "abc"`;
- AND their normalized domain payloads are identical.

### New payload receives new identity

- GIVEN a persisted buy-in command for 2500 chips;
- WHEN the user creates a corrected command for 5000 chips;
- THEN the corrected command uses a different request id;
- AND the original persisted command is not mutated into a different payload under its old id.

### Domain versus network error

- GIVEN a JSON `409` response with an API error code;
- THEN it is classified as `domain` and blocks automatic blind retry.

- GIVEN fetch rejects before an HTTP response exists;
- THEN it is classified as `offline` or `network` and remains retryable.

## Verification

- fake timers for timeout tests;
- deterministic serializer tests;
- classification table tests for status `0`, `401`, `403`, `408`, `409`, `429`, `500` and malformed JSON;
- existing web syntax/import check.

## Completion output

The implementing agent SHALL document:

- timeout defaults;
- result shape;
- request-id ownership;
- retry-policy table;
- compatibility impact on existing callers.
