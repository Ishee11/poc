# Change: Harden PWA startup across VPN and network changes

## Why

An installed iPhone PWA can show a black screen when the user enables a VPN to complete Telegram authentication and returns to Poker. Production Caddy evidence from 2026-08-21 localizes the first divergence before application/API startup: the VPN-side client (AS213220) reset one HTTP/2 connection while Caddy was writing both `GET /` and `GET /sw.js`; the upstream answered in under 1 ms. A fresh non-VPN browser startup succeeds and renders lobby data. The failure is therefore not a slow database, CORS rejection, IP-bound session, or an established service-worker cache miss.

The network reset is expected to be possible during any route change, including VPN switching. The current application amplifies it in three ways:

- there is no dependency-free startup status or global bootstrap error boundary before the module graph loads;
- initial authentication and lobby requests are awaited in one bootstrap chain, so several 5-second network timeouts can delay the usable shell and an unexpected exception can abandon startup;
- network-first HTML combined with cache-first unversioned modules can mix a newly deployed document with the previous active worker's JavaScript during an update.

## What Changes

- Add a static, dependency-free startup shell with loading, recoverable error, retry, and guarded application-update actions.
- Isolate bootstrap phases and render the route shell before remote authentication/lobby refresh completes.
- Classify and log asset/module, service-worker, bootstrap, authentication, initial API, timeout, offline, and network failures without sensitive values.
- Refresh recoverable remote state after an online event or visible-page resume.
- Make the active service worker serve one coherent cached shell generation for navigations and modules, while a newly installed generation is populated atomically and activates before clients move to it.
- Add one-attempt cache/service-worker recovery guarded by `sessionStorage`, with no automatic reload loop.

## Impact

- Affected specs: `pwa-startup-resilience`
- Affected frontend: `web/index.html`, `web/css/main.css`, `web/js/app.js`, a small startup helper, and `web/sw.js`
- Affected tests: startup helper/bootstrap contracts, network interruption, web import checks, and service-worker lifecycle tests
- Backend authorization/session semantics remain unchanged.
- Production Caddy remains the TLS endpoint. The observed reset was initiated by the client/VPN path, so disabling authentication checks or binding behavior is neither necessary nor permitted.
