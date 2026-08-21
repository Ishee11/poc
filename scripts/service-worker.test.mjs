import assert from "node:assert/strict";
import test from "node:test";

const listeners = new Map();
const deletedCaches = [];
const cachedRequiredAssets = [];
const cachedOptionalAssets = [];
const shellResponse = { source: "shell-cache" };
const staticResponse = { source: "static-cache" };
let openedCacheName = "";
let clientsClaimed = false;
let waitingSkipped = false;
const navigatedClients = [];
let fetchImpl = async () => ({ ok: true });

const cache = {
  async addAll(assets) {
    cachedRequiredAssets.push(...assets);
  },
  async add(asset) {
    cachedOptionalAssets.push(asset);
    if (asset.endsWith("poker-mark.png")) throw new Error("optional asset missing");
  },
  async put() {},
};

globalThis.self = {
  location: { origin: "https://poc.test" },
  registration: { showNotification: async () => {} },
  async skipWaiting() {
    waitingSkipped = true;
  },
  addEventListener(type, listener) {
    listeners.set(type, listener);
  },
};
globalThis.clients = {
  async claim() {
    clientsClaimed = true;
  },
  async matchAll() {
    return [
      {
        url: "https://poc.test/",
        async navigate(url) {
          navigatedClients.push(url);
        },
      },
      {
        url: "https://poc.test/session/session-1",
        async navigate(url) {
          navigatedClients.push(url);
        },
      },
    ];
  },
  async openWindow() {},
};
globalThis.caches = {
  async open(name) {
    openedCacheName = name;
    return cache;
  },
  async keys() {
    return [
      "poker-session-control-shell-v0",
      "poker-session-control-shell-v1-2026-08-04",
      "unrelated-application-cache",
    ];
  },
  async delete(name) {
    deletedCaches.push(name);
    return true;
  },
  async match(request) {
    if (request === "/") return shellResponse;
    if (request?.url?.endsWith("/static/css/main.css")) return staticResponse;
    return undefined;
  },
};
globalThis.fetch = (...args) => fetchImpl(...args);

await import(`../web/sw.js?service-worker-test=${Date.now()}`);

function runExtendable(type, extra = {}) {
  let lifetime;
  listeners.get(type)({
    ...extra,
    waitUntil(promise) {
      lifetime = promise;
    },
  });
  return lifetime;
}

function runFetch(request) {
  let responsePromise = null;
  listeners.get("fetch")({
    request,
    respondWith(promise) {
      responsePromise = promise;
    },
  });
  return responsePromise;
}

test("installs the versioned minimum shell without optional-asset failure", async () => {
  await runExtendable("install");

  assert.equal(openedCacheName, "poker-session-control-shell-v9-2026-08-21-telegram-identity");
  assert.ok(cachedRequiredAssets.includes("/"));
  assert.ok(cachedRequiredAssets.includes("/static/css/main.css"));
  assert.ok(cachedRequiredAssets.includes("/static/js/app.js"));
  assert.ok(cachedRequiredAssets.includes("/static/js/account-ownership-ui.js"));
  assert.ok(cachedRequiredAssets.includes("/static/js/sync-status.js"));
  assert.ok(cachedRequiredAssets.includes("/static/js/rollout.js"));
  assert.ok(cachedRequiredAssets.includes("/static/svg/13-user-outline-white.svg"));
  assert.ok(cachedRequiredAssets.includes("/manifest.webmanifest"));
  assert.ok(cachedOptionalAssets.includes("/static/assets/poker-mark.png"));
  assert.equal(waitingSkipped, true);
});

test("activation removes obsolete caches and refreshes only safe shell clients", async () => {
  await runExtendable("activate");

  assert.deepEqual(deletedCaches, [
    "poker-session-control-shell-v0",
    "poker-session-control-shell-v1-2026-08-04",
  ]);
  assert.equal(clientsClaimed, true);
  assert.deepEqual(navigatedClients, ["https://poc.test/"]);
});

test("navigation is network-first with shell fallback", async () => {
  fetchImpl = async () => { throw new Error("offline"); };
  const response = await runFetch({
    method: "GET",
    mode: "navigate",
    url: "https://poc.test/session/session-1",
  });

  assert.equal(response, shellResponse);
});

test("static assets are cache-first while API and unsupported requests pass through", async () => {
  let networkCalls = 0;
  fetchImpl = async () => {
    networkCalls += 1;
    return { ok: true };
  };
  const staticResult = await runFetch({
    method: "GET",
    mode: "cors",
    url: "https://poc.test/static/css/main.css",
  });
  assert.equal(staticResult, staticResponse);
  assert.equal(networkCalls, 0);

  assert.equal(runFetch({
    method: "GET",
    mode: "cors",
    url: "https://poc.test/sessions?session_id=session-1",
  }), null);
  assert.equal(runFetch({
    method: "POST",
    mode: "cors",
    url: "https://poc.test/operations/buy-in",
  }), null);
  assert.equal(runFetch({
    method: "GET",
    mode: "cors",
    url: "https://cdn.example.test/app.js",
  }), null);
});
