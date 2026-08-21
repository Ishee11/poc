# Auth Design

Status: active.

Email login, account management, and open self-registration are enabled by
default. Registration does not use invite codes. Telegram OIDC is an optional
second login method and is disabled until its server-side credentials are set.

This document defines the auth boundary for Poker Session Control.

## Goals

- Protect production data and write operations.
- Keep the current same-origin web UI simple.
- Separate system users from poker players.
- Make debug/admin operations server-protected, not only UI-hidden.
- Allow one account to use email/password and Telegram without duplicating its player.

## Non-Goals

- No JWT stored in browser storage.
- No password reset email flow yet.
- No MFA in the first implementation.
- No split auth microservice yet.

## Identity Model

Auth users are system accounts. They are not poker players.

Login methods are stored separately in `auth_identities`. A Telegram subject is
globally unique, and an account may have at most one Telegram identity. Linking
from the personal account keeps the existing `user_id` and therefore the same
owned poker player. Telegram-only signup creates a normal unowned account and
then uses the existing player onboarding flow.

Telegram uses the OIDC Authorization Code flow with PKCE. The callback consumes
a short-lived single-use state, validates the nonce, verifies the RS256 ID token
against Telegram JWKS, and checks issuer, audience, and expiry. Configure the bot
for RS256 in BotFather and register both the site origin and exact callback URL.
The client secret must exist only in environment/GitHub Secrets, never in Git.

Poker players remain business entities in the `players` table.

A system user owns at most one poker player through a one-to-one join table:

```text
user_players
- user_id
- player_id
- created_at
```

PostgreSQL unique constraints enforce both sides: one player per account and
one account per player. Registration must claim one unowned existing player or
create a new player in the same transaction as the account. A legacy account
without a player must complete the same onboarding flow. Self-service ownership
is write-once; only an administrator may later replace or clear it.

Session visibility is based on linked players:
- A session is public when none of its participating players is linked to a user.
- A session is user-visible when the current user's owned player participated
  in it.
- A session is hidden from a user or guest when it contains another user's
  linked player and the current user's player did not participate.

## Roles

| Role | Purpose |
| --- | --- |
| `admin` | Full access, including temporary debug/admin endpoints and future user management. |
| `user` | Normal authenticated user. Can operate games but cannot use debug/admin endpoints. Visibility is filtered by its owned player. |
| `guest` | Anonymous user. Can operate public games but cannot use debug/admin endpoints. Visibility is filtered to public sessions. |

Authorization is server-side. UI visibility is only a convenience.

## Visibility Rules

Admins see every session, player, operation, and stat.

Authenticated users see:
- public sessions;
- sessions where their owned player participated.

Authenticated users do not see sessions where another user's linked player
participated unless their owned player also participated.

An authenticated account without ownership receives guest-equivalent domain
visibility until onboarding is complete.

Guests see only public sessions. A guest cannot see a session that contains any
player linked to a system user.

The same visibility filter must be applied consistently to:
- `GET /sessions`;
- `GET /sessions/players`;
- `GET /sessions/operations`;
- `GET /stats/sessions`;
- `GET /stats/players`;
- `GET /stats/player`;
- frontend session and player screens after their SPA shell loads.

`GET /stats/player` intentionally separates aggregate and detailed visibility.
Its `player` financial statistics and `total_sessions_count` cover every real
session contributing to the selected player's statistics. The additive
`visible_sessions_count` and `sessions` array apply the current viewer's normal
session-access predicate. Hidden sessions contribute no identifiers,
participants, dates, statuses, amounts, names, or other row-level fields; the
only hidden-history disclosure is the aggregate count already represented by
the existing player statistics model. The legacy `player.sessions_count` field
remains an alias of `total_sessions_count` for compatibility.

## Route Access Matrix

| Route | Access |
| --- | --- |
| `GET /` | Public |
| `GET /session/{id}` | Public shell, data still requires auth |
| `GET /player/{id}` | Public shell, data still requires auth |
| `GET /static/*` | Public |
| `GET /health` | Public |
| `GET /swagger/*` | Development only, or `admin` in production |
| `GET /auth/config` | Public; exposes login UI and open-registration flags |
| `POST /auth/register` | Public; requires existing/new player selection, no invite code |
| `POST /auth/login` | Public |
| `POST /auth/logout` | Authenticated |
| `GET /auth/me` | Public; returns anonymous state when no session exists |
| `GET /auth/telegram/start?mode=login` | Public when Telegram OIDC is enabled |
| `GET /auth/telegram/start?mode=link` | Authenticated; binds the flow to the current account |
| `GET /auth/telegram/callback` | Public OIDC callback; validates a single-use flow |
| `GET /account` | Authenticated user, `admin` |
| `DELETE /account/identities/telegram` | Authenticated; forbidden for a Telegram-only account |
| `PUT /account/player` | Authenticated unlinked account; one-time existing/new claim |
| `POST /account/players` | Transitional one-time existing-player claim alias |
| `DELETE /account/players` | Disabled; returns `405 method_not_allowed` |
| `GET /account/players/available` | Authenticated; unowned players with session context |
| `GET /admin/accounts` | `admin` only; paginated ownership listing |
| `PUT /admin/accounts/{user_id}/player` | `admin` only; atomic ownership replace |
| `DELETE /admin/accounts/{user_id}/player` | `admin` only; idempotent ownership clear |
| `GET /sessions` | `guest`, `user`, `admin`; visibility-filtered |
| `GET /sessions/players` | `guest`, `user`, `admin`; visibility-filtered |
| `GET /sessions/operations` | `guest`, `user`, `admin`; visibility-filtered |
| `GET /players` | `guest`, `user`, `admin`; visibility-filtered |
| `GET /players/stats` | `guest`, `user`, `admin`; visibility-filtered |
| `GET /stats/player` | `guest`, `user`, `admin`; visibility-filtered |
| `GET /stats/sessions` | `guest`, `user`, `admin`; visibility-filtered |
| `GET /stats/players` | `guest`, `user`, `admin`; visibility-filtered |
| `POST /sessions/start` | `guest`, `user`, `admin` |
| `POST /sessions/finish` | `guest`, `user`, `admin`; visibility-filtered |
| `POST /operations/buy-in` | `guest`, `user`, `admin`; visibility-filtered |
| `POST /operations/cash-out` | `guest`, `user`, `admin`; visibility-filtered |
| `POST /operations/reverse` | `guest`, `user`, `admin`; visibility-filtered |
| `POST /players` | `guest`, `user`, `admin` |
| `/debug/*` | `admin` only |

Notes:
- `GET /session/{id}` and `GET /player/{id}` are frontend routes. They may load
  the public SPA shell, but API calls from that shell must apply visibility
  filtering.
- Swagger should be disabled or admin-only in production because it exposes the
  write API surface.

## Authentication Mechanism

Use server-side sessions with opaque session tokens stored in secure cookies.

Do not use JWT in `localStorage` for the first implementation.

Session token handling:
- Generate a random opaque token with a cryptographically secure RNG.
- Store only a hash of the token in the database.
- Send the raw token only once in the `sid` cookie.
- Rotate the session token after successful login.
- Revoke the session on logout.

Recommended cookie:

```text
sid=<opaque-token>; Path=/; HttpOnly; Secure; SameSite=Lax; Max-Age=<ttl>
```

Development may allow `Secure=false` only through explicit configuration.
Production must use `Secure=true`.

## Session Lifetime

Initial policy:

| Setting | Production | Development |
| --- | --- | --- |
| Absolute session TTL | `12h` | `24h` |
| Idle timeout | `2h` | `8h` |
| Cookie `Secure` | `true` | `false` only for local HTTP |
| Cookie `SameSite` | `Lax` | `Lax` |
| Cookie `HttpOnly` | `true` | `true` |

When either TTL expires, return `401 unauthorized` and clear the cookie.

`last_seen_at` may be updated at most once per minute per session to avoid a
database write on every request.

## Error Semantics

Use stable API error codes consistent with the existing error response style.

| Case | HTTP | Error Code |
| --- | --- | --- |
| Missing session cookie | `401` | `unauthorized` |
| Unknown, expired, or revoked session | `401` | `unauthorized` |
| Disabled user | `401` | `unauthorized` |
| Authenticated but role is insufficient | `403` | `forbidden` |
| Session hidden by visibility rules | `404` | `session_not_found` |
| Player hidden by visibility rules | `404` | `player_not_found` |
| Invalid login credentials | `401` | `invalid_credentials` |
| Login rate limit exceeded | `429` | `rate_limited` |
| Missing or ambiguous player selection | `400` | `invalid_player_selection` |
| Account already owns a player | `409` | `account_already_linked` |
| Player is already owned | `409` | `player_already_linked` |
| Administrator target account missing | `404` | `account_not_found` |

Authentication errors must not reveal whether an email exists.

## CSRF Policy

Because auth uses cookies, unsafe methods must be protected from CSRF.

First implementation:
- `SameSite=Lax` session cookie.
- For unsafe methods (`POST`, `PUT`, `PATCH`, `DELETE`), require `Origin` or
  `Referer` to match the configured app origin.
- Reject credentialed cross-origin requests.
- Keep CORS restrictive. Do not allow arbitrary origins with credentials.

Future enhancement:
- Add double-submit or server-bound CSRF token if cross-site clients are needed.

## Password Policy

Use Argon2id for password hashing.

Baseline parameters:

```text
memory: 19 MiB
iterations: 2
parallelism: 1
salt: unique random salt per password
```

Operational rules:
- Enforce a minimum password length of 12 characters for manually created users.
- Allow long passwords.
- Do not require arbitrary character classes.
- Block obviously common passwords when practical.
- Never log passwords or password hashes.

## Audit Logging

Log security events with `slog` and `request_id`.

Events:
- `auth_login_success`
- `auth_login_failed`
- `auth_logout`
- `auth_session_expired`
- `auth_session_revoked`
- `auth_forbidden`
- `account_player_ownership_changed`

Do not log raw session tokens or passwords.

Suggested fields:
- `request_id`
- `user_id` when known
- `role` when known
- `ip`
- `user_agent`
- `error_code`
- `operation`, `actor_user_id`, `target_user_id`, `old_player_id`, and
  `new_player_id` for ownership changes

Ownership audit records must not include email addresses, passwords, cookie
values, or raw session tokens.

## Configuration

Proposed environment variables:

```text
AUTH_ENABLED=true
AUTH_COOKIE_NAME=sid
AUTH_COOKIE_SECURE=true
AUTH_COOKIE_SAMESITE=Lax
AUTH_SESSION_TTL=12h
AUTH_IDLE_TTL=2h
AUTH_LOGIN_RATE_LIMIT=5/min
AUTH_SEED_ADMIN_EMAIL=
AUTH_SEED_ADMIN_PASSWORD=
APP_ORIGIN=
TELEGRAM_OIDC_ENABLED=false
TELEGRAM_OIDC_CLIENT_ID=
TELEGRAM_OIDC_CLIENT_SECRET=
TELEGRAM_LOGIN_BOT_ENABLED=false
TELEGRAM_LOGIN_BOT_USERNAME=
TELEGRAM_LOGIN_BOT_TOKEN=
TELEGRAM_LOGIN_BOT_WEBHOOK_SECRET=
```

Development override:

```text
AUTH_COOKIE_SECURE=false
AUTH_SESSION_TTL=24h
AUTH_IDLE_TTL=8h
APP_ORIGIN=http://193.238.134.58:18080
```

Production example:

```text
AUTH_COOKIE_SECURE=true
AUTH_SESSION_TTL=12h
AUTH_IDLE_TTL=2h
APP_ORIGIN=https://poker.semenovv.space
TELEGRAM_OIDC_ENABLED=true
```

Production callback: `https://poker.semenovv.space/auth/telegram/callback`.
Development callback: `https://dev.semenovv.space/auth/telegram/callback`.
Both exact callbacks and both origins must be included in BotFather Allowed URLs.

### Telegram app login bot

The primary Telegram login creates a four-minute PostgreSQL challenge and opens
`tg://resolve?domain=<TELEGRAM_LOGIN_BOT_USERNAME>&start=<challenge>`. The browser
does not request Telegram HTTP infrastructure. Poker stores only SHA-256 hashes
of the challenge and the independent browser-binding token; the latter is held
in a short-lived HttpOnly, Secure, SameSite=Strict cookie. Approval alone does
not authenticate the browser: the bound complete endpoint atomically creates
the normal opaque auth session and consumes the challenge.

The Poker backend receives updates through
`POST /telegram/login-bot/webhook`. Register that HTTPS URL with Bot API
`setWebhook` and set `secret_token` to the exact
`TELEGRAM_LOGIN_BOT_WEBHOOK_SECRET`; never put the bot token or webhook secret
in frontend configuration. The username is the only bot setting exposed by
`GET /auth/config`. Production reuses the existing `TELEGRAM_BOT_TOKEN`, resolves
its username with `getMe`, derives a domain-separated webhook secret, and
registers the production webhook during deployment. Dev must not register the
same bot because a Telegram bot supports one active webhook.

Bot API `user.id` and Telegram OIDC `sub` are the same decimal Telegram user ID
and resolve through the single `auth_identities(provider='telegram', subject)`
model. A first login preserves current behavior: create an active synthetic-email
account, then require the existing player-ownership onboarding.

## Rollout Plan

1. Add database schema for `users`, `auth_sessions`, and `login_attempts`.
2. Add auth domain/usecase layer and repositories.
3. Add `/auth/login`, `/auth/logout`, and `/auth/me`.
4. Add login/logout UI.
5. Add one-to-one `user_players` ownership constraints and repositories.
6. Require ownership during registration and legacy-account onboarding.
7. Add server-side visibility filters for sessions, players, operations, and stats.
8. Add debug/admin route protection.
9. Add CSRF origin checks for unsafe methods.
10. Add runbook documentation.
11. Add administrator ownership correction and structured ownership logs.

## References

- OWASP Authentication Cheat Sheet:
  https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html
- OWASP Session Management Cheat Sheet:
  https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html
- OWASP Password Storage Cheat Sheet:
  https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html
- NIST SP 800-63B:
  https://pages.nist.gov/800-63-4/sp800-63b.html
