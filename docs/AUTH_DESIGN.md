# Auth Design

Status: draft.

This document defines the first auth boundary for Poker Session Control. It is a
specification only; implementation is intentionally out of scope for this step.

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

Poker players remain business entities in the `players` table. A future version
may optionally link a system user to a player, but this must not be required for
basic session control.

## Roles

| Role | Purpose |
| --- | --- |
| `admin` | Full access, including temporary debug/admin endpoints and future user management. |
| `member` | Operates games: starts and finishes sessions, manages buy-ins, cash-outs, reversals, and players. |
| `viewer` | Read-only access to sessions, players, operations, and stats. |

Authorization is server-side. UI visibility is only a convenience.

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
| `GET /auth/me` | Authenticated |
| `GET /sessions` | `viewer`, `member`, `admin` |
| `GET /sessions/players` | `viewer`, `member`, `admin` |
| `GET /sessions/operations` | `viewer`, `member`, `admin` |
| `GET /players` | `viewer`, `member`, `admin` |
| `GET /players/stats` | `viewer`, `member`, `admin` |
| `GET /stats/player` | `viewer`, `member`, `admin` |
| `GET /stats/sessions` | `viewer`, `member`, `admin` |
| `GET /stats/players` | `viewer`, `member`, `admin` |
| `POST /sessions/start` | `member`, `admin` |
| `POST /sessions/finish` | `member`, `admin` |
| `POST /operations/buy-in` | `member`, `admin` |
| `POST /operations/cash-out` | `member`, `admin` |
| `POST /operations/reverse` | `member`, `admin` |
| `POST /players` | `member`, `admin` |
| `/debug/*` | `admin` only |

Notes:
- `GET /session/{id}` and `GET /player/{id}` are frontend routes. They may load
  the public SPA shell, but API calls from that shell must still require auth.
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
4. Add `AuthMiddleware` and `RequireRole`.
5. Protect API routes according to the route access matrix.
6. Add login/logout UI and central `401` handling in the frontend.
7. Add CSRF origin checks for unsafe methods.
8. Add seed admin flow and runbook documentation.

## References

- OWASP Authentication Cheat Sheet:
  https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html
- OWASP Session Management Cheat Sheet:
  https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html
- OWASP Password Storage Cheat Sheet:
  https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html
- NIST SP 800-63B:
  https://pages.nist.gov/800-63-4/sp800-63b.html
