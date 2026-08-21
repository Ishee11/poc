# Design: Recoverable PWA startup across network changes

## Evidence and first divergence

The normal production path was checked with a clean headless Chrome profile: HTML, CSS, the ES-module graph, service worker, auth config, initial statistics, and lobby render completed. Production DNS currently exposes one IPv4 address and no IPv6 record.

At `2026-08-21T03:31:37Z`, Caddy logged two aborted responses from the same iPhone Chrome HTTP/2 connection through `AS213220 Delta Ltd`: `/` failed with `connection reset by peer`, and `/sw.js` failed because the client disconnected. Both upstream calls completed in less than 1 ms. This places the first divergence at response delivery across the VPN connection, before React/bootstrap/auth/API work (the project uses vanilla ES modules, not React or dynamic route chunks).

The session repository selects the opaque session by token hash; the stored IP is audit metadata and is not a lookup condition. Changing VPN IP therefore does not invalidate a valid Poker session.

## Startup state machine

`index.html` owns a small inline controller that has no module, CSS, service-worker, or API dependency. Inline styles make its loading/error card legible even if `main.css` fails.

States:

1. `loading`: document exists and the module graph is loading.
2. `shell-ready`: event handlers and route shell exist; remote refresh may continue.
3. `error`: an asset/module/bootstrap failure prevented shell readiness.
4. `network-degraded`: shell is usable but auth or initial data refresh failed.

The helper exposes only bounded state transitions. It listens to capture-phase resource errors, `window.error`, and `unhandledrejection`, and records structured console events with a category, phase, error name, online state, and shell version. It never records request bodies, URLs with query strings, cookies, tokens, or Telegram values.

## Bootstrap separation

Synchronous DOM setup and `setScreen` happen in a guarded bootstrap function. It declares `shell-ready` before remote requests. Auth config, auth restore, and initial lobby refresh then execute as recoverable phases using existing request timeouts. Failures produce a network notice and structured startup log but do not unmount or hide the shell.

An `online` event and a debounced visible-page resume retry auth/lobby/current-route data. Only one refresh runs at a time. This covers a VPN toggle while the PWA stays open and avoids request storms.

## Coherent service-worker generations

The current worker can return a network-fresh HTML document while serving cache-first, unversioned JavaScript from its older cache. The design changes controlled navigations to use the active worker's cached `/` shell first. A worker update still checks on navigation/registration, builds the next required cache generation atomically, calls `skipWaiting`, claims clients, and navigates only safe routes after activation. Therefore one page load uses one generation instead of mixing old modules with new markup.

API and auth endpoints remain outside service-worker interception.

## Recovery and loop guard

`Повторить` performs a normal reload initiated by the user. `Обновить приложение` is intended for module/chunk/stale-asset failures. It:

1. writes a shell-version marker to `sessionStorage` before mutation;
2. asks registrations to update and unregisters only the Poker scope;
3. deletes only caches whose names begin with the Poker shell prefix;
4. reloads once.

If the marker already exists for this shell version, the update action remains on the error card but does not repeat destructive recovery automatically; the user can still retry after the network is restored. Successful shell readiness clears markers from older versions, not the current attempt during the same failing load.

## Rejected approaches

- Periodic reload: creates loops and hides the transport failure.
- Disabling auth/session security: production evidence does not implicate it.
- Caching API/auth responses: risks stale or cross-user state and does not repair response delivery.
- Treating service worker as the sole cause: logs show the navigation response itself was reset by the client/VPN path.
- Forcing HTTP/1.1 globally in shared Caddy: the observed reset coincided with a route transition and is not evidence of a server HTTP/2 defect; this would affect unrelated services without fixing an interrupted network path.
