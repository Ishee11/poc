## ADDED Requirements

### Requirement: Native Telegram login avoids browser Telegram HTTP endpoints
The primary Telegram login SHALL create a server challenge and launch the configured installed application through `tg://resolve`; it MUST NOT navigate the browser to `oauth.telegram.org`, `t.me`, or another Telegram HTTP origin. The unchanged OIDC route SHALL remain an explicit browser fallback.

#### Scenario: Installed Telegram is used
- **WHEN** a user starts primary Telegram login and challenge creation succeeds
- **THEN** the client opens `tg://resolve` with the configured bot username and high-entropy challenge and displays the waiting panel

#### Scenario: Telegram is not installed
- **WHEN** the custom scheme does not open an application
- **THEN** the waiting panel retains Open Telegram, Cancel, and Sign in through browser actions and does not automatically open `t.me`

### Requirement: Challenges are strong, expiring, browser-bound, and one-use
The server SHALL generate challenges and independent browser bindings with `crypto/rand` and at least 128 bits of entropy, persist only hashes, expire them after three to five minutes, and enforce the explicit state machine. Status, cancellation, and completion SHALL require the creator's HttpOnly binding cookie.

#### Scenario: Another browser has the challenge
- **WHEN** a browser without the matching binding requests status or completion for a valid challenge
- **THEN** the server returns a generic invalid response and creates no auth session

#### Scenario: Two completions race
- **WHEN** two bound requests concurrently complete one approved challenge
- **THEN** exactly one ordinary auth session is created and the challenge becomes consumed

#### Scenario: Challenge expires
- **WHEN** approval or completion is attempted after the deadline
- **THEN** the challenge is expired and no approval or session is created

### Requirement: Verification requires visible matching context
The server SHALL generate a short human-readable code separate from the challenge secret and show it in Poker and the bot confirmation. The code SHALL NOT be accepted as a credential or require manual entry.

#### Scenario: User compares devices
- **WHEN** the bot receives `/start` for a pending challenge
- **THEN** Poker and the bot display the same code and the bot requires explicit Confirm or Cancel

### Requirement: Bot approval uses the shared Telegram identity
The webhook SHALL use Bot API `user.id` as the same Telegram subject used by OIDC, require callback actor consistency, and make duplicate same-action callbacks idempotent. A missing identity SHALL follow the same first-login account creation behavior as OIDC.

#### Scenario: Existing OIDC user approves
- **WHEN** the Bot API user ID matches an existing Telegram identity
- **THEN** completion selects that same Poker account

#### Scenario: Telegram user signs in first time
- **WHEN** no Telegram identity exists
- **THEN** completion creates the same synthetic-email account and identity shape as OIDC and leaves player onboarding to the existing ownership flow

### Requirement: Approval is not authentication until atomic completion
An approved status SHALL NOT authenticate the browser. A separate bound completion SHALL atomically resolve the account, create the ordinary opaque server-side session, and consume the challenge before returning the configured HttpOnly session cookie.

#### Scenario: Approved challenge completes
- **WHEN** the creating browser completes an approved unconsumed challenge
- **THEN** it receives the existing session response/cookie semantics and subsequent authenticated requests use that session rather than the challenge

#### Scenario: Consumed challenge is replayed
- **WHEN** any client repeats completion for a consumed challenge
- **THEN** no second session or reusable credential is produced

### Requirement: Polling respects PWA lifecycle
The client SHALL poll at a non-aggressive interval only while visible, tolerate background suspension, and check immediately after returning to visible state.

#### Scenario: PWA returns from Telegram
- **WHEN** a suspended PWA becomes visible after bot approval
- **THEN** it immediately checks status, completes the challenge, and restores authenticated application state without reload or code entry

### Requirement: Bot ingress and public endpoints are bounded
The bot webhook SHALL require the configured Telegram secret-token header. Creation, status, completion, and bot actions SHALL have bounded rate limits and SHALL avoid logging challenges, cookies, bot credentials, callback data, or Telegram identity claims.

#### Scenario: Webhook secret is invalid
- **WHEN** a request reaches the bot webhook without the exact configured secret header
- **THEN** it is rejected before processing the update
