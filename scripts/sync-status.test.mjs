import assert from "node:assert/strict";
import test from "node:test";

import {
  deriveSyncUIStatus,
  SYNC_UI_STATUSES,
} from "../web/js/sync-status.js";

function status(overrides = {}) {
  return deriveSyncUIStatus({
    isOnline: true,
    localRuntimeStatus: "available",
    replayStatus: "idle",
    pendingCount: 0,
    blockedCount: 0,
    ...overrides,
  });
}

test("maps online, offline and pending runtime states", () => {
  assert.equal(status().kind, SYNC_UI_STATUSES.ONLINE_FRESH);
  assert.equal(
    status({ replayStatus: "syncing", pendingCount: 3 }).kind,
    SYNC_UI_STATUSES.ONLINE_SYNCING,
  );
  assert.equal(
    status({ isOnline: false }).kind,
    SYNC_UI_STATUSES.OFFLINE_CLEAN,
  );
  const offlinePending = status({ isOnline: false, pendingCount: 3 });
  assert.equal(offlinePending.kind, SYNC_UI_STATUSES.OFFLINE_PENDING);
  assert.equal(offlinePending.pendingCount, 3);
});

test("blocked and storage failures take priority over connectivity", () => {
  assert.equal(
    status({
      isOnline: false,
      replayStatus: "authorization_blocked",
      pendingCount: 2,
    }).kind,
    SYNC_UI_STATUSES.AUTHORIZATION_BLOCKED,
  );
  assert.equal(
    status({ blockedCount: 1 }).kind,
    SYNC_UI_STATUSES.DOMAIN_BLOCKED,
  );
  assert.equal(
    status({ localRuntimeStatus: "unavailable", blockedCount: 1 }).kind,
    SYNC_UI_STATUSES.LOCAL_STORAGE_UNAVAILABLE,
  );
});

test("exposes recovery actions only when they can help", () => {
  assert.equal(status({ replayStatus: "waiting_for_retry" }).action, "retry");
  assert.equal(
    status({ replayStatus: "authorization_blocked" }).action,
    "authenticate",
  );
  assert.equal(status({ isOnline: false, pendingCount: 2 }).action, null);
});
