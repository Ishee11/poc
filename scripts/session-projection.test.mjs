import assert from "node:assert/strict";
import test from "node:test";

import { createLocalDatabaseClient } from "../web/js/offline-db.js";
import {
  cancelPendingSessionCommandProjection,
  classifyReverseTarget,
  commitProjectedSessionCommand,
  createPendingCommand,
  createPendingReverseCommand,
  createSingleFlightCommitter,
  LocalProjectionError,
  projectSessionCommand,
  projectReverseSessionCommand,
  reconcileSessionOperationAcknowledgement,
  reapplyPendingSessionCommands,
  REVERSE_TARGET_KINDS,
} from "../web/js/session-projection.js";
import { FakeIndexedDBFactory } from "./test-support/fake-indexeddb.mjs";

function baseSnapshot(overrides = {}) {
  return {
    session_id: "session-1",
    session: {
      id: "session-1",
      status: "active",
      chipRate: 10,
      totalBuyIn: 1000,
      totalCashOut: 0,
      totalChips: 1000,
    },
    players: [
      {
        player_id: "player-1",
        name: "Player One",
        buy_in: 1000,
        cash_out: 0,
        profit_chips: -1000,
        profit_money: -100,
        in_game: true,
      },
    ],
    operations: [],
    cached_at: "2026-08-04T00:00:00.000Z",
    local_revision: 2,
    last_server_refresh_status: "fresh",
    ...overrides,
  };
}

function pending(kind, overrides = {}) {
  const requestId = overrides.requestId || `request-${kind}`;
  const chips = overrides.chips ?? 500;
  return createPendingCommand({
    kind,
    sessionId: "session-1",
    playerId: "player-1",
    chips,
    requestId,
    sequence: overrides.sequence || 1,
    createdAt: "2026-08-04T01:00:00.000Z",
    provisionalOperationId: `local-${requestId}`,
    payload: {
      session_id: "session-1",
      player_id: "player-1",
      chips,
      request_id: requestId,
    },
  });
}

function database(factory, name) {
  return createLocalDatabaseClient({
    indexedDBFactory: factory,
    name,
    broadcastChannelFactory: null,
  });
}

test("buy-in projector updates player, session, and provisional operation", () => {
  const { projectionCommand } = pending("buy_in", { chips: 2500 });
  const next = projectSessionCommand(baseSnapshot(), projectionCommand);

  assert.equal(next.session.totalBuyIn, 3500);
  assert.equal(next.session.totalCashOut, 0);
  assert.equal(next.session.totalChips, 3500);
  assert.equal(next.players[0].buy_in, 3500);
  assert.equal(next.players[0].profit_chips, -3500);
  assert.equal(next.players[0].profit_money, -350);
  assert.equal(next.players[0].in_game, true);
  assert.equal(next.local_revision, 3);
  assert.deepEqual(next.operations[0], {
    id: "local-request-buy_in",
    request_id: "request-buy_in",
    session_id: "session-1",
    player_id: "player-1",
    type: "buy_in",
    chips: 2500,
    created_at: "2026-08-04T01:00:00.000Z",
    sync_status: "pending",
    sequence: 1,
  });
});

test("cash-out projector settles player and keeps arithmetic consistent", () => {
  const { projectionCommand } = pending("cash_out", { chips: 600 });
  const next = projectSessionCommand(baseSnapshot(), projectionCommand);

  assert.equal(next.session.totalBuyIn, 1000);
  assert.equal(next.session.totalCashOut, 600);
  assert.equal(next.session.totalChips, 400);
  assert.equal(next.players[0].cash_out, 600);
  assert.equal(next.players[0].profit_chips, -400);
  assert.equal(next.players[0].profit_money, -40);
  assert.equal(next.players[0].in_game, false);
});

test("inverse projector cancels a never-sent provisional buy-in", () => {
  const original = baseSnapshot();
  const { projectionCommand, outboxCommand } = pending("buy_in", {
    requestId: "cancel-me",
    chips: 500,
  });
  const projected = projectSessionCommand(original, projectionCommand);
  const cancelled = cancelPendingSessionCommandProjection(projected, {
    ...outboxCommand,
    cancelled_at: "2026-08-04T01:01:00.000Z",
  });

  assert.equal(cancelled.session.totalBuyIn, original.session.totalBuyIn);
  assert.equal(cancelled.session.totalCashOut, original.session.totalCashOut);
  assert.equal(cancelled.session.totalChips, original.session.totalChips);
  assert.equal(cancelled.players[0].buy_in, original.players[0].buy_in);
  assert.equal(cancelled.operations.length, 0);
  assert.equal(cancelled.local_revision, 4);
});

test("inverse projector restores an in-game player after cancelling cash-out", () => {
  const original = baseSnapshot();
  const { projectionCommand, outboxCommand } = pending("cash_out", {
    requestId: "cancel-cash-out",
    chips: 600,
  });
  const projected = projectSessionCommand(original, projectionCommand);
  const cancelled = cancelPendingSessionCommandProjection(projected, {
    ...outboxCommand,
    cancelled_at: "2026-08-04T01:01:00.000Z",
  });

  assert.equal(cancelled.session.totalCashOut, 0);
  assert.equal(cancelled.session.totalChips, 1000);
  assert.equal(cancelled.players[0].cash_out, 0);
  assert.equal(cancelled.players[0].in_game, true);
});

test("classifies uncertain lineage without allowing local deletion", () => {
  const { projectionCommand, outboxCommand } = pending("buy_in", {
    requestId: "uncertain",
  });
  const projected = projectSessionCommand(baseSnapshot(), projectionCommand);
  const classification = classifyReverseTarget(
    projected,
    [{
      ...outboxCommand,
      status: "pending",
      attempts: 1,
      last_attempt_at: "2026-08-04T01:00:10.000Z",
      last_error_kind: "timeout",
    }],
    "local-uncertain",
  );

  assert.equal(classification.kind, REVERSE_TARGET_KINDS.POSSIBLY_SENT);
  assert.equal(classification.command.request_id, "uncertain");
});

test("confirmed reverse applies one inverse projection and tracks target lineage", () => {
  const confirmed = baseSnapshot({
    operations: [{
      id: "server-operation-1",
      request_id: "server-buy-in",
      session_id: "session-1",
      player_id: "player-1",
      type: "buy_in",
      chips: 500,
      created_at: "2026-08-04T00:30:00.000Z",
    }],
  });
  const { projectionCommand, outboxCommand } = createPendingReverseCommand({
    sessionId: "session-1",
    targetOperationId: "server-operation-1",
    requestId: "reverse-1",
    sequence: 1,
    createdAt: "2026-08-04T01:00:00.000Z",
    provisionalOperationId: "local-reverse-1",
    payload: {
      target_operation_id: "server-operation-1",
      request_id: "reverse-1",
    },
  });
  const reversed = projectReverseSessionCommand(confirmed, projectionCommand);

  assert.equal(reversed.session.totalBuyIn, 500);
  assert.equal(reversed.session.totalChips, 500);
  assert.equal(reversed.players[0].buy_in, 500);
  assert.equal(reversed.operations[0].type, "reversal");
  assert.equal(reversed.operations[0].reference_id, "server-operation-1");
  assert.equal(outboxCommand.target_lineage_id, "server-operation-1");
  assert.equal(
    classifyReverseTarget(reversed, [outboxCommand], "server-operation-1").kind,
    REVERSE_TARGET_KINDS.ALREADY_REVERSED,
  );
});

test("server refresh reapplies queued commands without changing their revision or identity", () => {
  const { outboxCommand } = pending("buy_in", {
    requestId: "queued-request",
    chips: 2500,
  });
  const refreshed = reapplyPendingSessionCommands(baseSnapshot(), [outboxCommand]);

  assert.equal(refreshed.session.totalBuyIn, 3500);
  assert.equal(refreshed.operations[0].id, "local-queued-request");
  assert.equal(refreshed.operations[0].request_id, "queued-request");
  assert.equal(refreshed.local_revision, 2);
  assert.equal(refreshed.last_server_refresh_status, "fresh_with_pending");
});

test("operation acknowledgement replaces the provisional id without a server refresh", () => {
  const { projectionCommand, outboxCommand } = pending("buy_in", {
    requestId: "ack-buy-in",
    chips: 500,
  });
  const projected = projectSessionCommand(baseSnapshot(), projectionCommand);
  const reconciled = reconcileSessionOperationAcknowledgement(projected, outboxCommand, {
    request_id: "ack-buy-in",
    operation_id: "server-operation-9",
    session_id: "session-1",
    player_id: "player-1",
    type: "buy_in",
    chips: 500,
    created_at: "2026-08-04T01:00:01.123Z",
    idempotent_replay: false,
  });

  assert.equal(reconciled.operations[0].id, "server-operation-9");
  assert.equal(reconciled.operations[0].created_at, "2026-08-04T01:00:01.123Z");
  assert.equal("sync_status" in reconciled.operations[0], false);
  assert.equal(reconciled.session.totalBuyIn, projected.session.totalBuyIn);
  assert.equal(reconciled.local_revision, projected.local_revision);
});

test("operation acknowledgement mismatch preserves the provisional projection", () => {
  const { projectionCommand, outboxCommand } = pending("cash_out", {
    requestId: "ack-mismatch",
    chips: 500,
  });
  const projected = projectSessionCommand(baseSnapshot(), projectionCommand);
  assert.throws(
    () => reconcileSessionOperationAcknowledgement(projected, outboxCommand, {
      request_id: "ack-mismatch",
      operation_id: "server-operation-10",
      session_id: "session-1",
      player_id: "player-1",
      type: "cash_out",
      chips: 700,
      created_at: "2026-08-04T01:00:01Z",
    }),
    /does not match/,
  );
  assert.equal(projected.operations[0].id, "local-ack-mismatch");
});

test("reverse acknowledgement maps the provisional reversal to its server id", () => {
  const confirmed = baseSnapshot({
    operations: [{
      id: "server-operation-1", request_id: "buy-1", session_id: "session-1",
      player_id: "player-1", type: "buy_in", chips: 500,
      created_at: "2026-08-04T00:30:00Z",
    }],
  });
  const { projectionCommand, outboxCommand } = createPendingReverseCommand({
    sessionId: "session-1", targetOperationId: "server-operation-1",
    requestId: "reverse-ack", sequence: 1,
    createdAt: "2026-08-04T01:00:00Z",
    provisionalOperationId: "local-reverse-ack",
    payload: { request_id: "reverse-ack", target_operation_id: "server-operation-1" },
  });
  const projected = projectReverseSessionCommand(confirmed, projectionCommand);
  const reconciled = reconcileSessionOperationAcknowledgement(projected, outboxCommand, {
    request_id: "reverse-ack", operation_id: "server-reversal-1",
    session_id: "session-1", player_id: "player-1", type: "reversal", chips: 500,
    created_at: "2026-08-04T01:00:01Z", target_operation_id: "server-operation-1",
    reversed_operation: {
      operation_id: "server-operation-1", session_id: "session-1",
      player_id: "player-1", type: "buy_in", chips: 500,
      created_at: "2026-08-04T00:30:00Z",
    },
  });
  assert.equal(reconciled.operations[0].id, "server-reversal-1");
  assert.equal(reconciled.operations[0].reference_id, "server-operation-1");
});

test("projectors reject locally knowable invalid actions", () => {
  assert.throws(
    () => pending("buy_in", { chips: 0 }),
    (error) => error instanceof LocalProjectionError && error.code === "invalid_chips",
  );
  const buyIn = pending("buy_in").projectionCommand;
  assert.throws(
    () => projectSessionCommand(baseSnapshot({ players: [] }), buyIn),
    (error) => error instanceof LocalProjectionError && error.code === "player_not_found",
  );
  assert.throws(
    () => projectSessionCommand(baseSnapshot({
      session: { ...baseSnapshot().session, status: "finished" },
    }), buyIn),
    (error) => error instanceof LocalProjectionError && error.code === "session_not_active",
  );

  const cashOut = pending("cash_out", { chips: 1500 }).projectionCommand;
  assert.throws(
    () => projectSessionCommand(baseSnapshot(), cashOut),
    (error) => error instanceof LocalProjectionError && error.code === "invalid_cash_out",
  );

  const settled = baseSnapshot({
    players: [{ ...baseSnapshot().players[0], in_game: false }],
  });
  assert.throws(
    () => projectSessionCommand(settled, pending("cash_out").projectionCommand),
    (error) => error instanceof LocalProjectionError && error.code === "player_not_in_game",
  );
});

test("atomic persistence survives reload with the same provisional identity", async () => {
  const factory = new FakeIndexedDBFactory();
  const localDB = database(factory, "projection-reload");
  await localDB.writeServerSnapshot(baseSnapshot());
  const { projectionCommand, outboxCommand } = pending("buy_in", {
    requestId: "stable-request",
    sequence: await localDB.nextSessionCommandSequence("session-1"),
  });
  const next = projectSessionCommand(baseSnapshot(), projectionCommand);

  await localDB.writeProjectedSnapshotWithCommand(next, outboxCommand);
  const staleServerSnapshot = {
    ...baseSnapshot(),
    cached_at: "2026-08-04T01:01:00.000Z",
  };
  assert.equal(
    await localDB.writeServerSnapshotIfRevision(staleServerSnapshot, 2),
    false,
  );
  localDB.close();

  const reopened = database(factory, "projection-reload");
  const reloadedSnapshot = await reopened.readSessionSnapshot("session-1");
  const queued = await reopened.listPendingCommands("session-1");
  assert.equal(reloadedSnapshot.operations[0].id, "local-stable-request");
  assert.equal(reloadedSnapshot.operations[0].request_id, "stable-request");
  assert.equal(reloadedSnapshot.local_revision, 3);
  assert.equal(queued.length, 1);
  assert.equal(queued[0].request_id, "stable-request");
  assert.equal(await reopened.nextSessionCommandSequence("session-1"), 2);
});

test("failed local transaction leaves prior snapshot and outbox unchanged", async () => {
  const factory = new FakeIndexedDBFactory();
  const localDB = database(factory, "projection-rollback");
  const original = baseSnapshot();
  await localDB.writeServerSnapshot(original);
  const { projectionCommand, outboxCommand } = pending("cash_out");
  const next = projectSessionCommand(original, projectionCommand);
  factory.failNextPut("outbox");

  await assert.rejects(
    localDB.writeProjectedSnapshotWithCommand(next, outboxCommand),
    /Injected outbox put failure/,
  );
  assert.deepEqual(await localDB.readSessionSnapshot("session-1"), original);
  assert.deepEqual(await localDB.listPendingCommands("session-1"), []);
});

test("single-flight guard coalesces double confirmation during commit", async () => {
  let commits = 0;
  let release;
  const gate = new Promise((resolve) => {
    release = resolve;
  });
  const commitOnce = createSingleFlightCommitter(async (input) => {
    commits += 1;
    await gate;
    return input;
  });

  const first = commitOnce("session-1", { chips: 500 });
  const second = commitOnce("session-1", { chips: 500 });
  assert.strictEqual(first, second);
  await Promise.resolve();
  assert.equal(commits, 1);
  release();
  assert.deepEqual(await first, { chips: 500 });
});

test("visible projection and replay hook run only after durable commit", async () => {
  const { projectionCommand, outboxCommand } = pending("buy_in");
  let release;
  const gate = new Promise((resolve) => {
    release = resolve;
  });
  const events = [];
  const committing = commitProjectedSessionCommand({
    snapshot: baseSnapshot(),
    projectionCommand,
    outboxCommand,
    persist: async () => {
      events.push("persist-start");
      await gate;
      events.push("persist-complete");
    },
    onCommitted: () => events.push("render"),
    requestReplay: () => events.push("replay"),
  });

  await Promise.resolve();
  assert.deepEqual(events, ["persist-start"]);
  release();
  await committing;
  assert.deepEqual(events, ["persist-start", "persist-complete", "render", "replay"]);

  const failedEvents = [];
  await assert.rejects(
    commitProjectedSessionCommand({
      snapshot: baseSnapshot(),
      projectionCommand,
      outboxCommand,
      persist: async () => {
        throw new Error("local storage failed");
      },
      onCommitted: () => failedEvents.push("render"),
      requestReplay: () => failedEvents.push("replay"),
    }),
    /local storage failed/,
  );
  assert.deepEqual(failedEvents, []);
});
