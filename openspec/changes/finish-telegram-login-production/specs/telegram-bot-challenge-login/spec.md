## MODIFIED Requirements

### Requirement: Native Telegram login avoids browser Telegram HTTP endpoints
The primary Telegram login SHALL be the first and visually dominant login action, create a server challenge, and launch the configured installed application through `tg://resolve`; it MUST NOT navigate the browser to `oauth.telegram.org`, `t.me`, or another Telegram HTTP origin. Email/password SHALL remain available as a secondary method below Telegram, and the unchanged OIDC route SHALL remain an explicit browser fallback.

#### Scenario: Login screen opens on a mobile device
- **WHEN** an unauthenticated user opens the login screen
- **THEN** Telegram appears above email/password and no credential field is programmatically focused, so the keyboard and password prompt do not open automatically

#### Scenario: Installed Telegram is used
- **WHEN** a user activates the primary Telegram action and challenge creation succeeds
- **THEN** the client opens `tg://resolve` with the configured bot username and high-entropy challenge and displays the waiting panel

#### Scenario: Telegram is not installed
- **WHEN** the custom scheme does not open an application
- **THEN** the waiting panel retains Open Telegram, Cancel, and Sign in through browser actions and does not automatically open `t.me`
