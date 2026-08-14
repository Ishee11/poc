import assert from "node:assert/strict";
import test from "node:test";

import { ERROR_KINDS } from "../web/js/network-contract.js";
import {
  createOutboxReplay,
  REPLAY_STATUSES,
} from "../web/js/offline-sync.js";

function command(sequence, overrides = {}) {
  const requestId = overrides.request_id || `request-${sequence}`;
  return {
    request_id: requestId,
    session_id: overrides.session_id || "session-1",
    sequence,
    kind: overrides.kind || "buy_in",
    payload: overrides.payload || {
      session_id: overrides.session_id || "session-1",
      player_id: "player-1",
      chips: 100 * sequence,
      request_id: requestId,
    },
    created_at: new Date(sequence * 1_000).toISOString(),
    status: overrides.status || "pending",
    attempts: overrides.attempts || 0,
    last_attempt_at: overrides.last_attempt_at ?? null,
    next_attempt_at: overrides.next_attempt_at ?? null,
    last_error_kind: overrides.last_error_kind ?? null,
  };
}

function createMemoryStore(initialCommands) {
  const commands = new Map(initialCommands.map((item) => [item.request_id, structuredClone(item)]));
  const calls = { claims: 0, retries: [], blocks: [], reconciled: [] };
  return {
    calls,
    commands,
    async claimNextReplayCommand({ now, leaseTimeoutMs, allowEarlyRetry }) {
      calls.claims += 1;
      const nowMs = Date.parse(now);
      const sessions = new Map();
      for (const current of commands.values()) {
        if (
          current.status === "sending" &&
          Date.parse(current.last_attempt_at) <= nowMs - leaseTimeoutMs
        ) {
          current.status = "pending";
        }
        const list = sessions.get(current.session_id) || [];
        list.push(current);
        sessions.set(current.session_id, list);
      }
      const candidates = [];
      let nextAttemptAt = null;
      let blockedErrorKind = null;
      for (const list of sessions.values()) {
        list.sort((left, right) => left.sequence - right.sequence);
        const current = list[0];
        if (current.status === "blocked" || current.status === "conflict") {
          blockedErrorKind ||= current.last_error_kind;
          continue;
        }
        if (current.status !== "pending") continue;
        const retryAt = Date.parse(current.next_attempt_at);
        if (!allowEarlyRetry && Number.isFinite(retryAt) && retryAt > nowMs) {
          nextAttemptAt = nextAttemptAt === null ? retryAt : Math.min(nextAttemptAt, retryAt);
          continue;
        }
        candidates.push(current);
      }
      candidates.sort((left, right) =>
        Date.parse(left.created_at) - Date.parse(right.created_at) ||
        left.sequence - right.sequence,
      );
      const selected = candidates[0];
      if (!selected) return { command: null, nextAttemptAt, blockedErrorKind };
      selected.status = "sending";
      selected.last_attempt_at = now;
      return { command: structuredClone(selected), nextAttemptAt: null, blockedErrorKind: null };
    },
    async retryOutboxCommand(update) {
      calls.retries.push(structuredClone(update));
      Object.assign(commands.get(update.requestId), {
        status: "pending",
        attempts: update.attempts,
        last_attempt_at: update.lastAttemptAt,
        next_attempt_at: update.nextAttemptAt,
        last_error_kind: update.errorKind,
      });
    },
    async blockOutboxCommand(update) {
      calls.blocks.push(structuredClone(update));
      Object.assign(commands.get(update.requestId), {
        status: update.conflict ? "conflict" : "blocked",
        attempts: update.attempts,
        last_error_kind: update.errorKind,
      });
    },
    async countPendingAndBlockedCommands() {
      return [...commands.values()].reduce(
        (counts, current) => {
          if (current.status === "pending" || current.status === "sending") counts.pending += 1;
          if (current.status === "blocked" || current.status === "conflict") counts.blocked += 1;
          return counts;
        },
        { pending: 0, blocked: 0 },
      );
    },
    reconcile(requestId) {
      calls.reconciled.push(requestId);
      commands.delete(requestId);
    },
  };
}

function accepted(body = {}) {
  return { ok: true, status: 200, body, errorKind: ERROR_KINDS.NONE };
}

function timeout() {
  return { ok: false, status: 0, body: null, errorKind: ERROR_KINDS.TIMEOUT };
}

test("replays one session in FIFO order with stable request identity", async () => {
  const store = createMemoryStore([command(3), command(1), command(2)]);
  const sent = [];
  const replay = createOutboxReplay({
    store,
    now: () => 10_000,
    send: async (current, serialized) => {
      sent.push({ requestId: current.request_id, payload: serialized.payload });
      return accepted();
    },
    reconcile: async (current) => store.reconcile(current.request_id),
  });

  await replay.requestReplay();

  assert.deepEqual(sent.map((item) => item.requestId), ["request-1", "request-2", "request-3"]);
  assert.deepEqual(sent.map((item) => item.payload.request_id), [
    "request-1",
    "request-2",
    "request-3",
  ]);
  assert.equal(store.commands.size, 0);
  replay.dispose();
});

test("parallel replay triggers coalesce into one active send", async () => {
  const store = createMemoryStore([command(1)]);
  let releaseSend;
  let markSendStarted;
  let sends = 0;
  const sendGate = new Promise((resolve) => { releaseSend = resolve; });
  const sendStarted = new Promise((resolve) => { markSendStarted = resolve; });
  const replay = createOutboxReplay({
    store,
    now: () => 10_000,
    send: async () => {
      sends += 1;
      markSendStarted();
      await sendGate;
      return accepted();
    },
    reconcile: async (current) => store.reconcile(current.request_id),
  });

  const first = replay.requestReplay();
  const second = replay.requestReplay({ allowEarlyRetry: true });
  const third = replay.requestReplay();
  await sendStarted;
  assert.equal(sends, 1);
  releaseSend();
  await Promise.all([first, second, third]);
  assert.equal(sends, 1);
  replay.dispose();
});

test("timeouts schedule bounded backoff and retry the identical command", async () => {
  const store = createMemoryStore([command(1)]);
  const sent = [];
  const timers = [];
  let currentTime = 10_000;
  let attempt = 0;
  const replay = createOutboxReplay({
    store,
    now: () => currentTime,
    random: () => 0.5,
    setTimeoutImpl: (callback, delay) => {
      timers.push({ callback, delay });
      return timers.length;
    },
    clearTimeoutImpl: () => {},
    send: async (current, serialized) => {
      sent.push({ requestId: current.request_id, payload: structuredClone(serialized.payload) });
      attempt += 1;
      return attempt < 3 ? timeout() : accepted();
    },
    reconcile: async (current) => store.reconcile(current.request_id),
  });

  await replay.requestReplay();
  assert.equal(timers.at(-1).delay, 1_000);
  currentTime += 1_000;
  await replay.requestReplay();
  assert.equal(timers.at(-1).delay, 2_000);
  currentTime += 2_000;
  await replay.requestReplay();

  assert.equal(store.commands.size, 0);
  assert.equal(sent.length, 3);
  assert.ok(sent.every((item) => item.requestId === "request-1"));
  assert.ok(sent.every((item) => item.payload.request_id === "request-1"));
  assert.deepEqual(sent[0].payload, sent[2].payload);
  replay.dispose();
});

test("authorization and domain failures durably block later session commands", async () => {
  for (const scenario of [
    {
      result: { ok: false, status: 401, body: {}, errorKind: ERROR_KINDS.AUTHORIZATION },
      expectedStatus: REPLAY_STATUSES.AUTHORIZATION_BLOCKED,
      conflict: false,
    },
    {
      result: {
        ok: false,
        status: 409,
        body: { code: "session_not_active" },
        errorKind: ERROR_KINDS.DOMAIN,
      },
      expectedStatus: REPLAY_STATUSES.DOMAIN_BLOCKED,
      conflict: true,
    },
  ]) {
    const store = createMemoryStore([command(1), command(2)]);
    const statuses = [];
    const sent = [];
    const replay = createOutboxReplay({
      store,
      now: () => 10_000,
      send: async (current) => {
        sent.push(current.request_id);
        return scenario.result;
      },
      reconcile: async () => assert.fail("blocked command must not reconcile"),
      onStatus: (status) => statuses.push(status.status),
    });

    await replay.requestReplay();
    assert.deepEqual(sent, ["request-1"]);
    assert.equal(store.calls.blocks[0].conflict, scenario.conflict);
    assert.equal(statuses.at(-1), scenario.expectedStatus);
    assert.equal(store.commands.get("request-2").status, "pending");
    replay.dispose();
  }
});

test("reconciliation failure preserves the command for retry", async () => {
  const store = createMemoryStore([command(1)]);
  const replay = createOutboxReplay({
    store,
    now: () => 10_000,
    random: () => 0.5,
    send: async () => accepted(),
    reconcile: async () => { throw new Error("refresh failed"); },
  });

  await replay.requestReplay();

  assert.equal(store.commands.get("request-1").status, "pending");
  assert.equal(store.commands.get("request-1").last_error_kind, ERROR_KINDS.INVALID_RESPONSE);
  assert.equal(store.calls.reconciled.length, 0);
  replay.dispose();
});

test("stale sending lease is recovered without changing request identity", async () => {
  const stale = command(1, {
    status: "sending",
    last_attempt_at: new Date(1_000).toISOString(),
  });
  const store = createMemoryStore([stale]);
  const sent = [];
  const replay = createOutboxReplay({
    store,
    now: () => 40_000,
    send: async (current) => {
      sent.push(current.request_id);
      return accepted();
    },
    reconcile: async (current) => store.reconcile(current.request_id),
  });

  await replay.requestReplay();

  assert.deepEqual(sent, ["request-1"]);
  assert.equal(store.commands.size, 0);
  replay.dispose();
});

test("reverse replay keeps its own request id and confirmed target lineage", async () => {
  const reverse = command(1, {
    request_id: "reverse-request-1",
    kind: "reverse_operation",
    payload: {
      target_operation_id: "server-operation-1",
      request_id: "reverse-request-1",
    },
  });
  const store = createMemoryStore([reverse]);
  let sent;
  const replay = createOutboxReplay({
    store,
    now: () => 10_000,
    send: async (current, serialized) => {
      sent = { current, serialized };
      return accepted();
    },
    reconcile: async (current) => store.reconcile(current.request_id),
  });

  await replay.requestReplay();

  assert.equal(sent.current.request_id, "reverse-request-1");
  assert.deepEqual(sent.serialized.payload, {
    target_operation_id: "server-operation-1",
    request_id: "reverse-request-1",
  });
  assert.equal(store.commands.size, 0);
  replay.dispose();
});

test("accepted but unreconciled command retries the same lineage without duplication", async () => {
  const store = createMemoryStore([command(1)]);
  let currentTime = 10_000;
  let sends = 0;
  let reconciliations = 0;
  const replay = createOutboxReplay({
    store,
    now: () => currentTime,
    random: () => 0.5,
    setTimeoutImpl: () => 1,
    clearTimeoutImpl: () => {},
    send: async (current) => {
      sends += 1;
      assert.equal(current.request_id, "request-1");
      return accepted();
    },
    reconcile: async (current) => {
      reconciliations += 1;
      if (reconciliations === 1) throw new Error("IndexedDB aborted");
      store.reconcile(current.request_id);
    },
  });

  await replay.requestReplay();
  assert.equal(store.commands.size, 1);
  assert.equal(store.commands.get("request-1").last_error_kind, ERROR_KINDS.INVALID_RESPONSE);
  currentTime += 1_000;
  await replay.requestReplay();

  assert.equal(sends, 2);
  assert.equal(reconciliations, 2);
  assert.equal(store.commands.size, 0);
  replay.dispose();
});
