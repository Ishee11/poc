## ADDED Requirements

### Requirement: Telegram availability distinguishes unknown from disabled
The application SHALL keep Telegram authentication actions available while public auth configuration is loading or failed, and SHALL hide them only after a successful configuration response explicitly disables Telegram.

#### Scenario: Auth config request fails
- **WHEN** `GET /auth/config` times out, is reset, or cannot resolve
- **THEN** the Telegram login action remains visible and the server start endpoint remains authoritative

#### Scenario: Telegram is explicitly disabled
- **WHEN** a successful auth config response contains `telegram_enabled=false`
- **THEN** Telegram login and linking actions are hidden

### Requirement: Incomplete cross-origin login is recoverable
The application SHALL record a bounded, non-sensitive attempt before Telegram navigation and SHALL show persistent recovery when the user returns without a callback or authenticated session. It MUST NOT claim that a Telegram timeout was proven.

#### Scenario: User returns without callback
- **WHEN** an unauthenticated application resumes with a valid pending Telegram attempt and no callback result
- **THEN** `/profile` opens with an incomplete-login message, retry, dismiss, and soft VPN guidance

#### Scenario: Attempt succeeds
- **WHEN** the callback reports success or session restoration finds an authenticated user
- **THEN** the attempt marker is cleared and no incomplete-login error is shown

#### Scenario: Attempt marker is stale
- **WHEN** an attempt marker is invalid or older than ten minutes
- **THEN** it is removed without navigation or notification

### Requirement: Telegram failures remain on the recovery screen
Telegram start and callback failures SHALL redirect to `/profile` with a controlled category and SHALL remain visible after route rendering.

#### Scenario: Provider is unavailable to the backend
- **WHEN** the Telegram token or JWKS request times out, has a transport/DNS failure, is rate limited, or returns 5xx
- **THEN** the user receives `provider_unavailable` recovery and may start a fresh attempt

#### Scenario: Provider reports cancellation
- **WHEN** Telegram returns an OAuth cancellation response
- **THEN** the user receives a controlled cancelled state without raw provider values

#### Scenario: Integration is disabled
- **WHEN** a stale or unknown-config client calls Telegram start while the server integration is disabled
- **THEN** the server redirects safely to disabled recovery rather than panicking or weakening authorization

### Requirement: Telegram recovery is private and bounded
Attempt storage and diagnostics MUST NOT contain OAuth state, code, verifier, nonce, tokens, cookies, Telegram secrets, or identity claims. Recovery MUST NOT automatically reload or repeat navigation.

#### Scenario: User retries
- **WHEN** the user selects retry
- **THEN** one fresh bounded marker is written and one normal Telegram start navigation occurs

#### Scenario: User dismisses recovery
- **WHEN** the user dismisses the message
- **THEN** the persistent feedback is removed without changing authentication state

### Requirement: Authentication navigations bypass the PWA shell
The service worker SHALL use the cached application document only for explicit Poker UI routes. Navigations to authentication and backend endpoints MUST reach the network and MUST NOT receive the cached `/` document.

#### Scenario: Controlled PWA starts Telegram login
- **WHEN** a service-worker-controlled client navigates to `/auth/telegram/start?mode=login`
- **THEN** the worker does not call `respondWith` and the backend can return the Telegram authorization redirect

#### Scenario: Controlled PWA opens an application route
- **WHEN** a controlled client navigates to `/profile` or `/session/{id}`
- **THEN** the worker may return the active coherent shell generation
