import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import vm from "node:vm";

const html = await readFile(new URL("../web/index.html", import.meta.url), "utf8");
const css = await readFile(new URL("../web/css/main.css", import.meta.url), "utf8");
const source = await readFile(new URL("../web/js/startup.js", import.meta.url), "utf8");

function createHarness({ recoveryMarker = "" } = {}) {
  const listeners = new Map();
  const elements = new Map([
    ["startup-shell", { hidden: false, dataset: {}, className: "", setAttribute() {} }],
    ["startup-message", { textContent: "" }],
    ["startup-detail", { textContent: "", hidden: true }],
    ["startup-retry", { hidden: true, addEventListener(type, listener) { this[type] = listener; } }],
    ["startup-update", { hidden: true, disabled: false, addEventListener(type, listener) { this[type] = listener; } }],
  ]);
  const storage = new Map();
  if (recoveryMarker) storage.set("poker-startup-recovery:test-shell", recoveryMarker);
  const diagnostics = [];
  const deleted = [];
  let reloads = 0;
  let unregistered = 0;
  const context = {
    console: { error(name, detail) { diagnostics.push([name, detail]); }, warn() {} },
    setTimeout() { return 1; },
    clearTimeout() {},
    URL,
    location: {
      origin: "https://poker.test",
      reload() { reloads += 1; },
    },
    navigator: {
      onLine: true,
      serviceWorker: {
        async getRegistrations() {
          return [{ scope: "https://poker.test/", async update() {}, async unregister() { unregistered += 1; } }];
        },
      },
    },
    caches: {
      async keys() { return ["poker-session-control-shell-old", "other-cache"]; },
      async delete(name) { deleted.push(name); },
    },
    sessionStorage: {
      getItem(key) { return storage.get(key) || null; },
      setItem(key, value) { storage.set(key, value); },
      removeItem(key) { storage.delete(key); },
    },
    document: {
      readyState: "loading",
      addEventListener(type, listener) { listeners.set(`document:${type}`, listener); },
      getElementById(id) { return elements.get(id) || null; },
      querySelector(selector) {
        if (selector === 'meta[name="poker-shell-version"]') return { content: "test-shell" };
        return null;
      },
    },
    addEventListener(type, listener) { listeners.set(`window:${type}`, listener); },
  };
  context.window = context;
  context.globalThis = context;
  vm.runInNewContext(source, context, { filename: "startup.js" });
  listeners.get("document:DOMContentLoaded")?.();
  return {
    context,
    diagnostics,
    deleted,
    elements,
    get reloads() { return reloads; },
    get unregistered() { return unregistered; },
  };
}

test("HTML contains a dependency-light loading and recoverable error shell", () => {
  assert.match(html, /meta name="poker-shell-version"/);
  assert.match(html, /id="startup-shell"/);
  assert.match(html, /Загрузка приложения…/);
  assert.match(html, /id="startup-retry"/);
  assert.match(html, /id="startup-update"/);
  assert.match(html, /startup\.js/);
  assert.match(css, /\.startup-shell/);
});

test("startup success hides the loading shell", () => {
  const harness = createHarness();
  harness.context.pokerStartup.ready();
  assert.equal(harness.elements.get("startup-shell").hidden, true);
});

test("bootstrap and module failures render classified recovery", () => {
  const harness = createHarness();
  harness.context.pokerStartup.fail("bootstrap", new Error("boom"));
  assert.equal(harness.elements.get("startup-shell").hidden, false);
  assert.equal(harness.elements.get("startup-shell").dataset.state, "error");
  assert.match(harness.elements.get("startup-message").textContent, /Не удалось загрузить приложение/);
  assert.equal(harness.elements.get("startup-retry").hidden, false);
  assert.equal(harness.diagnostics.at(-1)[1].category, "bootstrap");

  harness.context.pokerStartup.fail("module", new TypeError("Failed to fetch dynamically imported module"));
  assert.equal(harness.diagnostics.at(-1)[1].category, "chunk_loading");
});

test("application update deletes only Poker caches and reloads once per shell", async () => {
  const harness = createHarness();
  await harness.context.pokerStartup.updateApplication();
  assert.deepEqual(harness.deleted, ["poker-session-control-shell-old"]);
  assert.equal(harness.unregistered, 1);
  assert.equal(harness.reloads, 1);

  await harness.context.pokerStartup.updateApplication();
  assert.equal(harness.reloads, 1);
  assert.match(harness.elements.get("startup-detail").textContent, /уже выполнялось/);
});

test("existing recovery marker prevents a reload loop", async () => {
  const harness = createHarness({ recoveryMarker: "attempted" });
  await harness.context.pokerStartup.updateApplication();
  assert.equal(harness.reloads, 0);
  assert.deepEqual(harness.deleted, []);
});
