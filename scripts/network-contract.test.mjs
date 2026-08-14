import assert from "node:assert/strict";
import test from "node:test";

import {
  ERROR_KINDS,
  READ_TIMEOUT_MS,
  RETRY_ACTIONS,
  WRITE_TIMEOUT_MS,
  classifyHTTPResponse,
  createRequestClient,
  retryPolicy,
  serializeBuyInCommand,
  serializeCashOutCommand,
  serializeReverseOperationCommand,
} from "../web/js/network-contract.js";

function response(status, text = "", ok = status >= 200 && status < 300) {
  return {
    ok,
    status,
    async text() {
      return text;
    },
  };
}

class FakeTimers {
  constructor() {
    this.nextID = 1;
    this.timers = new Map();
  }

  setTimeout(callback, delay) {
    const id = this.nextID;
    this.nextID += 1;
    this.timers.set(id, { callback, delay });
    return id;
  }

  clearTimeout(id) {
    this.timers.delete(id);
  }

  runNext() {
    const entry = this.timers.entries().next().value;
    if (!entry) throw new Error("No fake timer is pending");
    const [id, timer] = entry;
    this.timers.delete(id);
    timer.callback();
    return timer.delay;
  }
}

test("uses explicit read and write timeout defaults", async () => {
  const timers = new FakeTimers();
  const seen = [];
  const request = createRequestClient({
    fetchImpl: () => new Promise(() => {}),
    setTimeoutImpl: timers.setTimeout.bind(timers),
    clearTimeoutImpl: timers.clearTimeout.bind(timers),
  });

  const read = request("/read");
  seen.push(timers.runNext());
  assert.equal((await read).errorKind, ERROR_KINDS.TIMEOUT);

  const write = request("/write", { method: "POST" });
  seen.push(timers.runNext());
  assert.equal((await write).errorKind, ERROR_KINDS.TIMEOUT);
  assert.deepEqual(seen, [READ_TIMEOUT_MS, WRITE_TIMEOUT_MS]);
});

test("aborts a hanging request and classifies it as timeout", async () => {
  const timers = new FakeTimers();
  let aborted = false;
  const request = createRequestClient({
    fetchImpl: (_url, options) =>
      new Promise((_resolve, reject) => {
        options.signal.addEventListener("abort", () => {
          aborted = true;
          reject(new Error("aborted"));
        });
      }),
    setTimeoutImpl: timers.setTimeout.bind(timers),
    clearTimeoutImpl: timers.clearTimeout.bind(timers),
  });

  const pending = request("/slow", { timeoutMs: 25 });
  await Promise.resolve();
  assert.equal(timers.runNext(), 25);
  const result = await pending;

  assert.equal(aborted, true);
  assert.deepEqual(
    {
      ok: result.ok,
      status: result.status,
      body: result.body,
      errorKind: result.errorKind,
    },
    { ok: false, status: 0, body: null, errorKind: ERROR_KINDS.TIMEOUT },
  );
  assert.equal(timers.timers.size, 0);
});

test("clears the timeout after a completed request", async () => {
  const timers = new FakeTimers();
  const request = createRequestClient({
    fetchImpl: async () => response(200, "{}"),
    setTimeoutImpl: timers.setTimeout.bind(timers),
    clearTimeoutImpl: timers.clearTimeout.bind(timers),
  });

  assert.equal((await request("/ready")).errorKind, ERROR_KINDS.NONE);
  assert.equal(timers.timers.size, 0);
});

test("classifies HTTP, malformed JSON, offline and network outcomes", async () => {
  const cases = [
    [200, "{}", ERROR_KINDS.NONE],
    [401, '{"error":"unauthorized"}', ERROR_KINDS.AUTHORIZATION],
    [403, '{"error":"forbidden"}', ERROR_KINDS.AUTHORIZATION],
    [408, '{"error":"timeout"}', ERROR_KINDS.RETRYABLE_HTTP],
    [409, '{"error":"conflict"}', ERROR_KINDS.DOMAIN],
    [429, '{"error":"limited"}', ERROR_KINDS.RETRYABLE_HTTP],
    [500, '{"error":"failed"}', ERROR_KINDS.RETRYABLE_HTTP],
    [200, "not-json", ERROR_KINDS.INVALID_RESPONSE],
  ];

  for (const [status, text, expected] of cases) {
    const request = createRequestClient({ fetchImpl: async () => response(status, text) });
    const result = await request("/classification", { timeoutMs: 0 });
    assert.equal(result.errorKind, expected, `status ${status}, body ${text}`);
    assert.equal(result.status, status);
    assert.equal(typeof result.text, "string");
  }

  const offline = createRequestClient({
    fetchImpl: async () => {
      throw new Error("connection failed");
    },
    isOnline: () => false,
  });
  assert.equal((await offline("/offline", { timeoutMs: 0 })).errorKind, ERROR_KINDS.OFFLINE);

  const network = createRequestClient({
    fetchImpl: async () => {
      throw new Error("connection failed");
    },
    isOnline: () => true,
  });
  assert.equal((await network("/network", { timeoutMs: 0 })).errorKind, ERROR_KINDS.NETWORK);
});

test("keeps stable normalized payloads and request ids", () => {
  const first = serializeBuyInCommand({
    sessionId: " session-1 ",
    playerId: " player-1 ",
    chips: "2500",
    requestId: "abc",
  });
  const retry = serializeBuyInCommand({
    sessionId: "session-1",
    playerId: "player-1",
    chips: 2500,
    requestId: "abc",
  });
  assert.deepEqual(first, retry);
  assert.deepEqual(Object.keys(first.payload), [
    "session_id",
    "player_id",
    "chips",
    "request_id",
  ]);
  assert.equal(Object.isFrozen(first.payload), true);

  const corrected = serializeBuyInCommand({
    sessionId: "session-1",
    playerId: "player-1",
    chips: 5000,
  });
  const original = serializeBuyInCommand({
    sessionId: "session-1",
    playerId: "player-1",
    chips: 2500,
  });
  assert.notEqual(corrected.requestId, original.requestId);
  assert.equal(original.payload.chips, 2500);

  assert.deepEqual(
    serializeCashOutCommand({
      sessionId: "session-1",
      playerId: "player-1",
      chips: 1000,
      requestId: "cash-1",
    }).payload,
    {
      session_id: "session-1",
      player_id: "player-1",
      chips: 1000,
      request_id: "cash-1",
    },
  );
  assert.deepEqual(
    serializeReverseOperationCommand({ operationId: "op-1", requestId: "reverse-1" })
      .payload,
    { target_operation_id: "op-1", request_id: "reverse-1" },
  );
});

test("operation helper sends a caller-owned request id unchanged on every retry", async () => {
  const originalWindow = globalThis.window;
  const originalFetch = globalThis.fetch;
  const bodies = [];
  globalThis.window = { location: { origin: "https://poker.test" } };
  globalThis.fetch = async (_url, options) => {
    bodies.push(JSON.parse(options.body));
    return response(204);
  };

  try {
    const { buyIn } = await import("../web/js/api.js?stable-request-id-test");
    for (let attempt = 0; attempt < 3; attempt += 1) {
      const result = await buyIn({
        sessionId: "session-1",
        playerId: "player-1",
        chips: 2500,
        requestId: "abc",
      });
      assert.equal(result.requestId, "abc");
    }
  } finally {
    globalThis.window = originalWindow;
    globalThis.fetch = originalFetch;
  }

  assert.deepEqual(bodies, [
    { session_id: "session-1", player_id: "player-1", chips: 2500, request_id: "abc" },
    { session_id: "session-1", player_id: "player-1", chips: 2500, request_id: "abc" },
    { session_id: "session-1", player_id: "player-1", chips: 2500, request_id: "abc" },
  ]);
});

test("maps classifications and statuses to replay policy", () => {
  const cases = [
    [ERROR_KINDS.NONE, 200, RETRY_ACTIONS.ACCEPTED],
    [ERROR_KINDS.NETWORK, 0, RETRY_ACTIONS.RETRY],
    [ERROR_KINDS.AUTHORIZATION, 401, RETRY_ACTIONS.BLOCK_AUTHORIZATION],
    [ERROR_KINDS.AUTHORIZATION, 403, RETRY_ACTIONS.BLOCK_AUTHORIZATION],
    [ERROR_KINDS.RETRYABLE_HTTP, 408, RETRY_ACTIONS.RETRY],
    [ERROR_KINDS.DOMAIN, 409, RETRY_ACTIONS.BLOCK_DOMAIN],
    [ERROR_KINDS.RETRYABLE_HTTP, 429, RETRY_ACTIONS.RETRY],
    [ERROR_KINDS.RETRYABLE_HTTP, 500, RETRY_ACTIONS.RETRY],
    [ERROR_KINDS.INVALID_RESPONSE, 200, RETRY_ACTIONS.RETRY],
  ];
  for (const [errorKind, status, expected] of cases) {
    assert.equal(retryPolicy(errorKind, status), expected);
  }
  assert.equal(
    classifyHTTPResponse({ status: 409, ok: false }),
    ERROR_KINDS.DOMAIN,
  );
});
