import assert from "node:assert/strict";
import test from "node:test";

import {
  LOCAL_FIRST_SESSION_WRITES_KEY,
  resolveLocalFirstSessionWrites,
  shouldUseLocalFirstSessionWrites,
} from "../web/js/rollout.js";

function documentFlag(value) {
  return {
    querySelector() {
      return value === undefined ? null : { getAttribute: () => value };
    },
  };
}

test("rollout defaults closed for unknown configuration", () => {
  assert.deepEqual(resolveLocalFirstSessionWrites({
    storage: { getItem: () => null },
    documentRef: documentFlag(undefined),
  }), { enabled: false, source: "unknown", raw: null });
});

test("disabled or unavailable runtime selects the online-first fallback", () => {
  assert.equal(shouldUseLocalFirstSessionWrites({ enabled: false, runtimeStatus: "available" }), false);
  assert.equal(shouldUseLocalFirstSessionWrites({ enabled: true, runtimeStatus: "unavailable" }), false);
  assert.equal(shouldUseLocalFirstSessionWrites({ enabled: true, runtimeStatus: "available" }), true);
});

test("deployment flag takes precedence over stale local cohort setting", () => {
  assert.deepEqual(resolveLocalFirstSessionWrites({
    storage: { getItem: () => null },
    documentRef: documentFlag("false"),
  }), { enabled: false, source: "deployment", raw: "false" });

  let requestedKey = "";
  assert.deepEqual(resolveLocalFirstSessionWrites({
    storage: { getItem: (key) => { requestedKey = key; return "true"; } },
    documentRef: documentFlag("false"),
  }), { enabled: false, source: "deployment", raw: "false" });
  assert.equal(requestedKey, "");
});

test("invalid local storage never overrides deployment, unavailable storage is ignored", () => {
  assert.equal(resolveLocalFirstSessionWrites({
    storage: { getItem: () => "unexpected" },
    documentRef: documentFlag("false"),
  }).enabled, false);
  assert.deepEqual(resolveLocalFirstSessionWrites({
    storage: { getItem: () => { throw new Error("denied"); } },
    documentRef: documentFlag("true"),
  }), { enabled: true, source: "deployment", raw: "true" });
});
