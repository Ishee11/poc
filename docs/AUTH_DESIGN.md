# Auth Design

Status: draft.

This document defines the auth boundary for Poker Session Control.

## Goals

- Protect production data and write operations.
- Keep the current same-origin web UI simple.
- Separate system users from poker players.
- Make debug/admin operations server-protected, not only UI-hidden.
- Keep the design compatible with a future OAuth/OIDC migration.

## Non-Goals

- No external auth provider in the first implementation.
- No JWT stored in browser storage.
- No password reset email flow yet.
- No MFA in the first implementation.
- No split auth microservice yet.

## Identity Model

Auth users are system accounts. They are not poker players.

Poker players remain business entities in the `players` table.

A system user may be linked to multiple poker players through a join table:

```text
user_players
- user_id
- player_id
- created_at
```

A player may be linked to at most one system user. Users can link players to
their account from a personal account page only when the player is not already
linked to another user.

Session visibility is based on linked players:
- A session is public when none of its participating players is linked to a user.
- A session is user-visible when the current user's linked player participated
  in it.
- A session is hidden from a user or guest when it contains another user's
  linked player and none of the current user's linked players participated.

## Roles

| Role | Purpose |
| --- | --- |
| `admin` | Full access, including temporary debug/admin endpoints and future user management. |
| `user` | Normal authenticated user. Can operate games but cannot use debug/admin endpoints. Visibility is filtered by linked players. |
| `guest` | Anonymous user. Can operate public games but cannot use debug/admin endpoints. Visibility is filtered to public sessions. |

Authorization is server-side. UI visibility is only a convenience.

## Visibility Rules

Admins see every session, player, operation, and stat.

Authenticated users see:
- public sessions;
- sessions where at least one of their linked players participated.

Authenticated users do not see sessions where another user's linked player
participated unless one of their own linked players also participated.

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

## Route Access Matrix

| Route | Access |
| --- | --- |
| `GET /` | Public |
| `GET /session/{id}` | Public shell, data still requires auth |
| `GET /player/{id}` | Public shell, data still requires auth |
| `GET /static/*` | Public |
| `GET /health` | Public |
| `GET /swagger/*` | Development only, or `admin` in production |
| `POST /auth/login` | Public |
| `POST /auth/logout` | Authenticated |
| `GET /auth/me` | Public; returns anonymous state when no session exists |
| `GET /account` | Authenticated user, `admin` |
| `POST /account/players` | Authenticated user, `admin` |
| `DELETE /account/players/{id}` | Authenticated user, `admin` |
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

Do not log raw session tokens or passwords.

Suggested fields:
- `request_id`
- `user_id` when known
- `role` when known
- `ip`
- `user_agent`
- `error_code`

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
APP_ORIGIN=http://193.238.134.58:8080
```

If HTTPS is added, `APP_ORIGIN` must be changed to the HTTPS origin.

## Rollout Plan

1. Add database schema for `users`, `auth_sessions`, and `login_attempts`.
2. Add auth domain/usecase layer and repositories.
3. Add `/auth/login`, `/auth/logout`, and `/auth/me`.
4. Add login/logout UI.
5. Add `user_players` schema and repositories.
6. Add personal account page for linking unlinked players.
7. Add server-side visibility filters for sessions, players, operations, and stats.
8. Add debug/admin route protection.
9. Add CSRF origin checks for unsafe methods.
10. Add runbook documentation.

## References

- OWASP Authentication Cheat Sheet:
  https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html
- OWASP Session Management Cheat Sheet:
  https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html
- OWASP Password Storage Cheat Sheet:
  https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html
- NIST SP 800-63B:
  https://pages.nist.gov/800-63-4/sp800-63b.html
