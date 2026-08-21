## ADDED Requirements

### Requirement: Startup always has a dependency-free visible state
The application SHALL render a meaningful loading state from the HTML document before external CSS, JavaScript modules, service-worker registration, authentication restore, or initial API calls complete. A startup dependency failure MUST replace loading with a recoverable error rather than leave an empty background.

#### Scenario: JavaScript module graph loads
- **WHEN** the document and startup module graph load successfully
- **THEN** the startup shell transitions from loading to the usable application shell

#### Scenario: Entrypoint or imported module fails
- **WHEN** the entrypoint or one of its static or dynamic imports cannot be loaded or evaluated
- **THEN** the user sees a recoverable startup error with retry and application-update actions

#### Scenario: Main stylesheet fails
- **WHEN** the application stylesheet cannot be loaded
- **THEN** the inline startup shell remains legible and reports an asset loading failure

### Requirement: Remote initialization does not block the application shell
Authentication restore and initial API requests SHALL run after the route shell is usable. A timeout, offline result, connection reset, DNS failure, authorization response, or backend error MUST NOT prevent shell rendering.

#### Scenario: Initial API is unavailable
- **WHEN** auth or lobby requests fail during startup
- **THEN** the route shell remains visible and the user receives a recoverable network state

#### Scenario: Network returns
- **WHEN** the browser becomes online or the PWA resumes visibly after a network/VPN change
- **THEN** the application retries remote state refresh without overlapping refresh loops

### Requirement: One page load uses one shell generation
For a service-worker-controlled client, the HTML document and its application modules SHALL come from one completely installed shell cache generation. API and authentication endpoints MUST NOT be stored in the shell cache.

#### Scenario: New deployment is installing
- **WHEN** an old worker controls a client while a new shell generation is being installed
- **THEN** the current load uses the complete old generation until the complete new generation activates

#### Scenario: New generation fails to install
- **WHEN** a network reset prevents one required asset from entering the new cache
- **THEN** the incomplete generation does not replace the active shell and the previous complete generation remains usable

#### Scenario: Navigation is offline
- **WHEN** a controlled client navigates without network access
- **THEN** the active generation's cached application document is returned

### Requirement: Application-update recovery is bounded and scoped
The application SHALL allow stale asset recovery to update/unregister only its own service-worker scope, delete only Poker shell caches, and reload at most once automatically for a shell version. It MUST NOT enter a reload loop.

#### Scenario: First stale-asset recovery attempt
- **WHEN** application update is requested and no marker exists for the current shell version
- **THEN** the application records the marker before scoped cleanup and performs one reload

#### Scenario: Recovery already attempted
- **WHEN** the same shell version fails again after recovery
- **THEN** the application shows the recoverable error without automatically repeating cleanup or reload

### Requirement: Startup failures are distinguishable without secrets
Startup diagnostics SHALL distinguish asset/module, service-worker, bootstrap, authentication, initial API, timeout, offline, and network failures. Diagnostics MUST NOT contain tokens, cookies, Telegram secrets, request bodies, or URL query values.

#### Scenario: Authentication refresh times out
- **WHEN** the auth initialization request reaches its timeout
- **THEN** diagnostics identify the authentication phase and timeout category without authentication material

#### Scenario: Module import fails
- **WHEN** an imported module fails to load
- **THEN** diagnostics identify a module or chunk loading failure and the startup UI offers bounded recovery
