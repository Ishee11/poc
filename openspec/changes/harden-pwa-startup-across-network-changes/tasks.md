## 1. Diagnosis and contracts

- [x] 1.1 Trace document, module, service-worker, auth, session restore, and initial API startup order.
- [x] 1.2 Compare clean production startup with the VPN-side Caddy failure and identify the first divergence.
- [x] 1.3 Verify session lookup is token-based rather than IP-bound.
- [x] 1.4 Add executable startup and coherent-cache contract tests before implementation.

## 2. Dependency-free startup UI

- [x] 2.1 Add a static loading/error shell that remains visible when CSS or modules fail.
- [x] 2.2 Add capture-phase asset, global error, and unhandled rejection classification.
- [x] 2.3 Add user-driven retry and one-attempt application-update recovery with a session marker.
- [x] 2.4 Add privacy-bounded structured startup logging.

## 3. Recoverable bootstrap and network changes

- [x] 3.1 Split synchronous shell readiness from remote auth and lobby refresh.
- [x] 3.2 Ensure auth and initial API failure leave a usable shell with a recoverable network state.
- [x] 3.3 Retry remote/current-route refresh after online and visible-page resume without overlapping runs.
- [x] 3.4 Preserve Telegram callback cleanup and existing auth/session security.

## 4. Coherent PWA lifecycle

- [x] 4.1 Serve controlled navigations and static modules from one active cache generation.
- [x] 4.2 Keep required cache installation atomic, clean old Poker cache generations, and retain immediate activation/claim behavior.
- [x] 4.3 Keep API/auth endpoints outside service-worker caching.
- [x] 4.4 Bump the shell version and cover stale generation/update behavior.

## 5. Validation

- [x] 5.1 Test startup success, bootstrap failure, initial API failure, asset/module failure, retry, guarded recovery, and reload-loop prevention.
- [x] 5.2 Test offline/online and visible-page recovery behavior.
- [x] 5.3 Run frontend syntax/import, network-contract, service-worker, and full Node suites.
- [x] 5.4 Run Go tests/build, strict change/all OpenSpec validation, and `git diff --check`.
- [x] 5.5 Perform clean production-like browser smoke checks and document device/VPN verification limits.
