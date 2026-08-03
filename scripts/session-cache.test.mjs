import assert from "node:assert/strict";
import test from "node:test";

import {
  hydrateCachedSession,
  refreshSessionSnapshot,
} from "../web/js/session-cache.js";

function snapshot(sessionId = "session-1", overrides = {}) {
  return {
    session_id: sessionId,
    session: {
      id: sessionId,
      status: "active",
      totalBuyIn: 2000,
      totalCashOut: 0,
      totalChips: 2000,
    },
    players: [{ player_id: "player-1", profit_money: 0 }],
    operations: [{ id: "operation-1", type: "buy_in" }],
    expenses: [{ id: "expense-1" }],
    settlements: [],
    cached_at: "2026-08-04T00:00:00.000Z",
    local_revision: 3,
    last_server_refresh_status: "fresh",
    ...overrides,
  };
}

function runtimeState(sessionId = "session-1") {
  return {
    activeSessionId: sessionId,
    session: null,
    players: [],
    operations: [],
    expenses: [],
    settlementDrafts: {},
    sessionDataSource: "none",
    sessionCachedAt: null,
    sessionRefreshStatus: "idle",
    sessionLocalRevision: 0,
    sessionExpensesCached: false,
    sessionSettlementsCached: false,
  };
}

function serverResults(sessionId = "session-1", overrides = {}) {
  return {
    sessionResult: {
      ok: true,
      body: {
        session_id: sessionId,
        status: "active",
        total_buy_in: 2000,
        total_cash_out: 0,
        total_chips: 2000,
      },
    },
    playersResult: { ok: true, body: [{ player_id: "player-1", profit_money: 0 }] },
    operationsResult: { ok: true, body: [] },
    expensesResult: { ok: true, body: [] },
    settlementsResult: { ok: true, body: [] },
    ...overrides,
  };
}

function deferred() {
  let resolve;
  const promise = new Promise((done) => {
    resolve = done;
  });
  return { promise, resolve };
}

test("hydrates cached session before a delayed refresh completes", async () => {
  const state = runtimeState();
  const rendered = [];
  const refreshGate = deferred();

  assert.equal(
    await hydrateCachedSession({
      sessionId: "session-1",
      state,
      readSnapshot: async () => snapshot(),
      onHydrated: () => rendered.push("cache"),
    }),
    true,
  );

  const refresh = refreshSessionSnapshot({
    sessionId: "session-1",
    state,
    loadResults: () => refreshGate.promise,
    writeSnapshot: async () => {},
    onApplied: () => rendered.push("server"),
  });

  assert.deepEqual(rendered, ["cache"]);
  assert.equal(state.sessionDataSource, "cache");
  assert.equal(state.sessionRefreshStatus, "refreshing");

  refreshGate.resolve(serverResults());
  assert.equal((await refresh).status, "fresh");
  assert.deepEqual(rendered, ["cache", "server"]);
  assert.equal(state.sessionDataSource, "server");
});

test("failed required refresh preserves cached players and operations", async () => {
  const state = runtimeState();
  await hydrateCachedSession({
    sessionId: "session-1",
    state,
    readSnapshot: async () => snapshot(),
  });
  const beforePlayers = structuredClone(state.players);
  const beforeOperations = structuredClone(state.operations);
  let writes = 0;

  const result = await refreshSessionSnapshot({
    sessionId: "session-1",
    state,
    loadResults: async () =>
      serverResults("session-1", {
        playersResult: { ok: false, body: null },
        operationsResult: { ok: false, body: null },
      }),
    writeSnapshot: async () => {
      writes += 1;
    },
  });

  assert.equal(result.status, "failed");
  assert.deepEqual(state.players, beforePlayers);
  assert.deepEqual(state.operations, beforeOperations);
  assert.equal(state.sessionRefreshStatus, "failed");
  assert.equal(writes, 0);
});

test("successful empty operations replace cached operations and persist first", async () => {
  const state = runtimeState();
  await hydrateCachedSession({
    sessionId: "session-1",
    state,
    readSnapshot: async () => snapshot(),
  });
  const order = [];

  const result = await refreshSessionSnapshot({
    sessionId: "session-1",
    state,
    loadResults: async () => serverResults(),
    writeSnapshot: async (nextSnapshot) => {
      assert.deepEqual(nextSnapshot.operations, []);
      order.push("persist");
    },
    onApplied: () => order.push("render"),
  });

  assert.equal(result.status, "fresh");
  assert.deepEqual(state.operations, []);
  assert.deepEqual(order, ["persist", "render"]);
});

test("optional request failures retain only previously cached optional data", async () => {
  const state = runtimeState();
  await hydrateCachedSession({
    sessionId: "session-1",
    state,
    readSnapshot: async () => snapshot(),
  });
  let persisted;

  const result = await refreshSessionSnapshot({
    sessionId: "session-1",
    state,
    loadResults: async () =>
      serverResults("session-1", {
        expensesResult: { ok: false, body: null },
        settlementsResult: { ok: false, body: null },
      }),
    writeSnapshot: async (nextSnapshot) => {
      persisted = nextSnapshot;
    },
  });

  assert.equal(result.status, "fresh");
  assert.deepEqual(persisted.expenses, [{ id: "expense-1" }]);
  assert.deepEqual(persisted.settlements, []);
  assert.deepEqual(state.expenses, [{ id: "expense-1" }]);
});

test("refresh for an old route cannot overwrite the active session", async () => {
  const state = runtimeState("session-a");
  state.sessionLocalRevision = 2;
  const gate = deferred();
  let writes = 0;
  const refresh = refreshSessionSnapshot({
    sessionId: "session-a",
    state,
    loadResults: () => gate.promise,
    writeSnapshot: async () => {
      writes += 1;
    },
  });

  state.activeSessionId = "session-b";
  gate.resolve(serverResults("session-a"));
  assert.equal((await refresh).status, "stale");
  assert.equal(writes, 0);
  assert.equal(state.activeSessionId, "session-b");
});

test("newer local revision invalidates an older server refresh", async () => {
  const state = runtimeState();
  state.sessionLocalRevision = 4;
  const gate = deferred();
  let writes = 0;
  const refresh = refreshSessionSnapshot({
    sessionId: "session-1",
    state,
    loadResults: () => gate.promise,
    writeSnapshot: async () => {
      writes += 1;
    },
  });

  state.sessionLocalRevision = 5;
  gate.resolve(serverResults());
  assert.equal((await refresh).status, "stale");
  assert.equal(writes, 0);
});
