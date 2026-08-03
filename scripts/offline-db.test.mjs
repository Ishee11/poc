import assert from "node:assert/strict";
import test from "node:test";

import {
  createLocalDatabaseClient,
  isValidOutboxCommand,
  isValidSessionSnapshot,
} from "../web/js/offline-db.js";
import { FakeIndexedDBFactory } from "./test-support/fake-indexeddb.mjs";

function snapshot(sessionId = "session-1", revision = 1) {
  return {
    session_id: sessionId,
    session: { session_id: sessionId, status: "active" },
    players: [],
    operations: [],
    cached_at: "2026-08-03T00:00:00Z",
    local_revision: revision,
    last_server_refresh_status: "fresh",
  };
}

function command(sequence, overrides = {}) {
  return {
    request_id: `request-${sequence}`,
    session_id: "session-1",
    sequence,
    kind: "buy_in",
    payload: { player_id: "player-1", chips: sequence * 100 },
    created_at: "2026-08-03T00:00:00Z",
    status: "pending",
    attempts: 0,
    last_attempt_at: null,
    next_attempt_at: null,
    last_error_kind: null,
    ...overrides,
  };
}

function client(factory, name, version = 2) {
  return createLocalDatabaseClient({
    indexedDBFactory: factory,
    name,
    version,
    broadcastChannelFactory: null,
  });
}

test("atomically commits a projected snapshot and ordered outbox commands", async () => {
  const factory = new FakeIndexedDBFactory();
  const database = client(factory, "atomic-success");
  const changes = [];
  database.subscribeLocalRuntimeChanges((event) => changes.push(event.detail));

  await database.writeProjectedSnapshotWithCommand(snapshot("session-1", 1), command(2));
  await database.writeProjectedSnapshotWithCommand(snapshot("session-1", 2), command(1));
  await database.writeProjectedSnapshotWithCommand(
    snapshot("session-1", 3),
    command(3, { status: "conflict" }),
  );

  assert.deepEqual(await database.readSessionSnapshot("session-1"), snapshot("session-1", 3));
  assert.deepEqual(
    (await database.listPendingCommands("session-1")).map((item) => item.sequence),
    [1, 2],
  );
  assert.deepEqual(await database.countPendingAndBlockedCommands("session-1"), {
    pending: 2,
    blocked: 1,
  });
  assert.equal(changes.length, 3);
  assert.equal(changes[2].type, "local_projection_committed");
});

test("rolls back the snapshot when the outbox write fails", async () => {
  const factory = new FakeIndexedDBFactory();
  const database = client(factory, "atomic-rollback");
  await database.writeServerSnapshot(snapshot("session-1", 1));
  factory.failNextPut("outbox");

  await assert.rejects(
    database.writeProjectedSnapshotWithCommand(snapshot("session-1", 2), command(1)),
    /Injected outbox put failure/,
  );

  assert.deepEqual(await database.readSessionSnapshot("session-1"), snapshot("session-1", 1));
  assert.deepEqual(await database.listPendingCommands("session-1"), []);
});

test("upgrades from schema version 1 without losing a valid snapshot", async () => {
  const factory = new FakeIndexedDBFactory();
  const versionOne = client(factory, "schema-upgrade", 1);
  await versionOne.writeServerSnapshot(snapshot("session-1", 3));
  assert.equal(factory.hasIndex("schema-upgrade", "outbox", "by_session_sequence"), false);
  versionOne.close();

  const versionTwo = client(factory, "schema-upgrade", 2);
  assert.deepEqual(await versionTwo.readSessionSnapshot("session-1"), snapshot("session-1", 3));
  assert.equal(factory.hasIndex("schema-upgrade", "outbox", "by_session_sequence"), true);
  await versionTwo.writeProjectedSnapshotWithCommand(snapshot("session-1", 4), command(1));
  assert.equal((await versionTwo.listPendingCommands("session-1")).length, 1);
});

test("rejects initialization when IndexedDB is unavailable", async () => {
  const database = createLocalDatabaseClient({
    indexedDBFactory: null,
    name: "unavailable",
    broadcastChannelFactory: null,
  });
  await assert.rejects(database.open(), /IndexedDB is unavailable/);
});

test("ignores structurally invalid persisted records", async () => {
  assert.equal(isValidSessionSnapshot({ session_id: "session-1" }), false);
  assert.equal(isValidOutboxCommand(command(0)), false);

  const factory = new FakeIndexedDBFactory();
  const database = client(factory, "validation");
  await database.open();
  factory.seed("validation", "session_snapshots", {
    session_id: "corrupt-session",
    session: {},
    players: "not-an-array",
    operations: [],
  });
  assert.equal(await database.readSessionSnapshot("corrupt-session"), null);

  await assert.rejects(
    database.writeProjectedSnapshotWithCommand(snapshot("session-1", 1), command(0)),
    /Invalid outbox command record/,
  );
  assert.equal(await database.readSessionSnapshot("session-1"), null);
});

test("claims FIFO commands one at a time and recovers an expired sending lease", async () => {
  const factory = new FakeIndexedDBFactory();
  const database = client(factory, "replay-claim");
  await database.writeProjectedSnapshotWithCommand(snapshot("session-1", 1), command(2));
  await database.writeProjectedSnapshotWithCommand(snapshot("session-1", 2), command(1));

  const first = await database.claimNextReplayCommand({
    now: "2026-08-03T00:00:10Z",
    leaseTimeoutMs: 30_000,
  });
  assert.equal(first.command.request_id, "request-1");
  assert.equal(first.command.status, "sending");

  const concurrent = await database.claimNextReplayCommand({
    now: "2026-08-03T00:00:20Z",
    leaseTimeoutMs: 30_000,
  });
  assert.equal(concurrent.command, null);

  const recovered = await database.claimNextReplayCommand({
    now: "2026-08-03T00:00:41Z",
    leaseTimeoutMs: 30_000,
  });
  assert.equal(recovered.command.request_id, "request-1");
  assert.equal(recovered.command.attempts, 0);
});

test("reconciliation atomically preserves the command on failure and releases FIFO on success", async () => {
  const factory = new FakeIndexedDBFactory();
  const database = client(factory, "replay-reconcile");
  await database.writeProjectedSnapshotWithCommand(snapshot("session-1", 1), command(1));
  await database.writeProjectedSnapshotWithCommand(snapshot("session-1", 2), command(2));
  await database.claimNextReplayCommand({
    now: "2026-08-03T00:00:10Z",
    leaseTimeoutMs: 30_000,
  });

  factory.failNextPut("session_snapshots");
  await assert.rejects(
    database.reconcileOutboxCommand({
      requestId: "request-1",
      sessionId: "session-1",
      snapshot: snapshot("session-1", 2),
      expectedLocalRevision: 2,
      completedAt: "2026-08-03T00:00:11Z",
    }),
    /Injected session_snapshots put failure/,
  );
  assert.deepEqual(await database.countPendingAndBlockedCommands("session-1"), {
    pending: 2,
    blocked: 0,
  });

  assert.equal(
    await database.reconcileOutboxCommand({
      requestId: "request-1",
      sessionId: "session-1",
      snapshot: snapshot("session-1", 2),
      expectedLocalRevision: 2,
      acknowledgement: { operation_id: "server-operation-1" },
      completedAt: "2026-08-03T00:00:12Z",
    }),
    true,
  );
  assert.deepEqual(
    (await database.listPendingCommands("session-1")).map((item) => item.request_id),
    ["request-2"],
  );
});

test("keeps blocked optimistic commands available for snapshot projection", async () => {
  const factory = new FakeIndexedDBFactory();
  const database = client(factory, "blocked-projection");
  await database.writeProjectedSnapshotWithCommand(
    snapshot("session-1", 1),
    command(1, { status: "conflict", last_error_kind: "domain" }),
  );

  assert.deepEqual(
    (await database.listSessionProjectionCommands("session-1")).map((item) => item.request_id),
    ["request-1"],
  );
  assert.deepEqual(
    await database.listSessionProjectionCommands("session-1", {
      excludeRequestId: "request-1",
    }),
    [],
  );
});

test("cancels only a never-sent provisional command in the snapshot transaction", async () => {
  const factory = new FakeIndexedDBFactory();
  const database = client(factory, "local-cancellation");
  const projected = snapshot("session-1", 2);
  projected.operations = [{
    id: "local-request-1",
    request_id: "request-1",
    type: "buy_in",
    sync_status: "pending",
  }];
  await database.writeProjectedSnapshotWithCommand(projected, command(1, {
    provisional_operation_id: "local-request-1",
  }));

  assert.equal(
    await database.cancelPendingOutboxCommand({
      requestId: "request-1",
      sessionId: "session-1",
      snapshot: snapshot("session-1", 3),
      expectedLocalRevision: 2,
      cancelledAt: "2026-08-04T02:00:00.000Z",
    }),
    true,
  );
  assert.deepEqual(await database.listPendingCommands("session-1"), []);
  assert.equal((await database.readSessionSnapshot("session-1")).local_revision, 3);

  await database.writeProjectedSnapshotWithCommand(
    snapshot("session-1", 4),
    command(2, {
      attempts: 1,
      last_attempt_at: "2026-08-04T02:01:00.000Z",
    }),
  );
  assert.equal(
    await database.cancelPendingOutboxCommand({
      requestId: "request-2",
      sessionId: "session-1",
      snapshot: snapshot("session-1", 5),
      expectedLocalRevision: 4,
    }),
    false,
  );
  assert.equal((await database.listPendingCommands("session-1")).length, 1);
});

test("atomically prevents duplicate reverse lineages", async () => {
  const factory = new FakeIndexedDBFactory();
  const database = client(factory, "reverse-lineage");
  const original = snapshot("session-1", 1);
  original.operations = [{
    id: "server-operation-1",
    type: "buy_in",
    sync_status: "confirmed",
  }];
  await database.writeServerSnapshot(original);
  const reverseCommand = command(1, {
    request_id: "reverse-1",
    kind: "reverse_operation",
    payload: {
      target_operation_id: "server-operation-1",
      request_id: "reverse-1",
    },
    provisional_operation_id: "local-reverse-1",
    target_lineage_id: "server-operation-1",
  });
  const projected = snapshot("session-1", 2);
  projected.operations = [{
    id: "local-reverse-1",
    type: "reversal",
    reference_id: "server-operation-1",
    sync_status: "pending",
  }, ...original.operations];

  assert.equal(
    await database.writeProjectedSnapshotWithReverseCommand(projected, reverseCommand, 1),
    true,
  );
  assert.equal(
    await database.writeProjectedSnapshotWithReverseCommand(
      projected,
      { ...reverseCommand, request_id: "reverse-2", sequence: 2 },
      2,
    ),
    false,
  );
  assert.deepEqual(
    (await database.listSessionProjectionCommands("session-1")).map(
      (item) => item.request_id,
    ),
    ["reverse-1"],
  );
});
