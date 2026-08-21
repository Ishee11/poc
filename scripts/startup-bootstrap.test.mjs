import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const app = await readFile(new URL("../web/js/app.js", import.meta.url), "utf8");

test("route shell becomes ready before remote auth and initial API refresh", () => {
  const bootstrapStart = app.indexOf("async function bootstrapApplication()");
  const bootstrapEnd = app.indexOf("function showInitialRouteShell()", bootstrapStart);
  const bootstrap = app.slice(bootstrapStart, bootstrapEnd);

  const routeShell = bootstrap.indexOf("showInitialRouteShell()");
  const ready = bootstrap.indexOf("pokerStartup?.ready()");
  const remote = bootstrap.indexOf('refreshRemoteState({ reason: "startup"');
  assert.ok(routeShell >= 0);
  assert.ok(ready > routeShell);
  assert.ok(remote > ready);
  assert.doesNotMatch(bootstrap.slice(0, ready), /await loadAuthConfig/);
});

test("bootstrap failure is fatal but remote failures are recoverable", () => {
  assert.match(app, /fail\("bootstrap", error, \{ fatal: true, phase: "bootstrap" \}\)/);
  assert.match(app, /degraded\("initial_api", error, "remote_refresh"\)/);
  assert.match(app, /showNotice\(t\("error\.startupNetwork"\), "error"\)/);
});

test("network changes use one in-flight refresh and visible resume is bounded", () => {
  assert.match(app, /if \(remoteRefreshPromise\) return remoteRefreshPromise/);
  assert.match(app, /addEventListener\("online"/);
  assert.match(app, /addEventListener\("visibilitychange"/);
  assert.match(app, /VISIBLE_REFRESH_INTERVAL_MS = 15_000/);
});

test("startup diagnostics distinguish auth and initial API phases", () => {
  assert.match(app, /reportRemoteFailure\(configResult, "auth", "auth_config"\)/);
  assert.match(app, /reportRemoteFailure\(authResult, "auth", "session_restore"\)/);
  assert.match(app, /reportRemoteFailure\(result, "initial_api", reason\)/);
  assert.match(app, /degraded\("service_worker", error, "service_worker_registration"\)/);
});
