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
