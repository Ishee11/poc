# Design: Browser-bound Telegram bot challenge login

## Context

Current Telegram OIDC stores a hash of OAuth state, validates PKCE and nonce, resolves `AuthProviderTelegram` by OIDC `sub`, creates a synthetic-email active account on first login, then calls the normal `AuthService.LoginUser` path to create an opaque server-side session and set the configured HttpOnly cookie. Telegram OIDC `sub` is the decimal Telegram user ID, so Bot API `message.from.id` and `callback_query.from.id` use the same identity subject.

There is no user-facing bot update loop to reuse, but the project already has a connected Poker bot token. This change reuses that bot, adds a webhook to the existing Poker HTTP process, and adds a minimal Bot API sender. Production derives the username through Bot API `getMe`, derives a domain-separated webhook secret from the protected bot token, and registers the production webhook during deployment.

## Goals / Non-Goals

**Goals:**

- Complete Telegram login without any browser request to a Telegram HTTP origin.
- Bind approval to the browser that created it and make the verification context visible in both apps.
- Reuse the exact Telegram identity, first-login account resolution, auth-session persistence, and cookie policy used by OIDC.
- Survive PWA background suspension and foreground return.

**Non-Goals:**

- Removing OIDC, using `t.me` fallback, manual code entry, linking Telegram to an already authenticated account, or operating a general-purpose Poker bot.
- Treating a challenge as an API bearer token or trusting a status response as authenticated state.
- Adding Redis solely for this feature.

## Sequence

```mermaid
sequenceDiagram
    participant Browser
    participant PokerBackend as Poker Backend
    participant Telegram as Telegram App/Bot
    participant BotBackend as Bot Backend
    participant AuthSession as Auth Session

    Browser->>PokerBackend: POST /auth/telegram/challenge
    PokerBackend->>PokerBackend: random challenge + browser binding; store hashes and code
    PokerBackend-->>Browser: challenge, code, expires_at, bot_username + HttpOnly binding cookie
    Browser->>Telegram: open tg://resolve?domain=bot&start=challenge
    Telegram->>BotBackend: webhook /start challenge with Telegram user.id
    BotBackend->>PokerBackend: validate pending, unexpired challenge
    PokerBackend-->>Telegram: confirmation message with same code and inline buttons
    Telegram->>BotBackend: approve callback by same Telegram user.id
    BotBackend->>PokerBackend: atomically mark approved and save Telegram user.id
    Browser->>PokerBackend: GET challenge status with binding cookie
    PokerBackend-->>Browser: approved
    Browser->>PokerBackend: POST challenge complete with binding cookie
    PokerBackend->>PokerBackend: lock challenge; resolve provider=telegram subject=user.id
    PokerBackend->>AuthSession: create ordinary opaque server-side session
    PokerBackend->>PokerBackend: mark challenge consumed in same transaction
    PokerBackend-->>Browser: user + expires_at + ordinary HttpOnly session cookie
```

The Bot Backend participant is the bot-facing adapter inside the Poker backend process. It is shown separately to make the trust boundary and Telegram webhook ingress explicit.

## Persistence

Use PostgreSQL because the application already relies on it for OIDC ephemeral state and auth sessions, while Redis is not present. `telegram_login_challenges` stores only SHA-256 hashes of the 256-bit challenge and independent 256-bit browser-binding token, the verification code, state, expiry, optional Telegram subject/profile fields, and timestamps. The raw challenge exists only in the create response, native deep link, and Telegram update. The raw binding exists only in an HttpOnly `SameSite=Strict` cookie.

Rows are retained briefly for state reporting. Read/update paths turn overdue non-consumed rows into `expired`; an expiry index supports later operational cleanup. Expired rows are never approvable or completable.

## Browser binding and session-swapping prevention

Creation rotates a dedicated short-lived HttpOnly binding cookie scoped to `/auth/telegram/challenge`. Status, cancel, and complete require a constant-time-equivalent database match on both `challenge_hash` and `browser_binding_hash`. Knowing or receiving the Telegram deep link is insufficient to complete login from another browser.

The verification code is shown on both devices so the creating user can detect an unexpected or forwarded request before pressing approve. It is deliberately not accepted by any endpoint. Approval records the Telegram user who pressed the inline button. The callback is accepted only from the same user who initiated `/start` for that challenge.

This prevents silent session swapping: browser A remains the only browser able to consume its challenge, and user B sees a code-bearing explicit confirmation before choosing whether browser A may become B. Forwarding can never be made mathematically impossible while B deliberately approves A's displayed request, but the flow eliminates unnoticed approval and makes the account transition explicit.

## Challenge state machine and concurrency

```text
pending --approve--> approved --complete--> consumed
pending --cancel---> denied
pending/approved --deadline--> expired
```

Bot approve is idempotent only for the same Telegram subject; duplicate callbacks return the already-approved result without changing identity. Deny is likewise idempotent. Conflicting terminal transitions fail closed. Completion locks the row with `SELECT ... FOR UPDATE`, verifies binding and state, resolves/creates identity, inserts exactly one auth session, and changes state to `consumed` in one transaction. Two concurrent completions serialize; the loser sees consumed and receives no token.

## Shared Telegram account resolution

Extract the existing OIDC identity lookup/create transaction logic into a shared resolver used by both OIDC and bot challenge completion. Both pass `AuthProviderTelegram` and the decimal Telegram user ID as subject. Existing identity rows therefore select the same Poker account. A missing identity creates the same synthetic-email active account and Telegram identity used today. It does not create a player; the existing mandatory account-player onboarding remains authoritative.

The completion service creates the same `AuthSession` entity, random opaque token hash, TTL, `last_login_at`, user-agent/IP metadata, and response cookie used by OIDC. The challenge never authorizes other endpoints.

## Bot transport

`POST /telegram/login-bot/webhook` accepts Telegram updates only when `X-Telegram-Bot-Api-Secret-Token` matches configured `TELEGRAM_LOGIN_BOT_WEBHOOK_SECRET`. `/start <challenge>` validates the token and sends the code plus inline approve/cancel buttons through Bot API. Callback data carries action plus the high-entropy challenge, remains below Telegram's callback-data limit, and is never logged. Callback queries are answered after the state transition.

The production deployment registers this HTTPS webhook with Telegram. Dev MUST NOT register the same bot because Telegram permits one active webhook per bot. Polling is not introduced. Bot token and webhook secret remain backend-only; the public auth config exposes only bot username and feature availability.

## Rate limiting and enumeration resistance

An in-process bounded fixed-window limiter covers challenge creation by client IP, status/complete by IP plus challenge hash, and bot start/callback by Telegram user ID. This matches the current single application process and avoids a new infrastructure dependency. The database remains the final concurrency boundary. If the service scales horizontally, the limiter must move to shared storage before relying on aggregate limits.

Unknown, wrong-binding, and unusable challenges return the same generic not-found/invalid family; no Telegram identity is exposed by status. Raw challenges, cookies, bot tokens, webhook secrets, callback data, and Telegram profile claims are excluded from application logs.

## PWA lifecycle and native launch

The browser constructs `tg://resolve?domain=<single configured username>&start=<challenge>` and assigns it only after a user gesture. It never auto-falls back to `https://t.me`. The waiting panel retains the challenge only in memory/session storage so an iOS or Android PWA can resume after suspension. Polling runs every two seconds only while visible; `visibilitychange` to visible triggers an immediate check. Background suspension is normal and does not surface an error.

The panel shows the verification code, Open Telegram, Cancel, and Sign in through browser. The last action invokes the unchanged `/auth/telegram/start?mode=login` OIDC route. Approved status triggers the separate complete request; only its ordinary session cookie and subsequent `/auth/me` result establish authentication.

Desktop browsers with Telegram Desktop and mobile browsers/PWAs depend on OS custom-scheme registration. If launch is unavailable the waiting panel remains actionable and offers repeat launch or OIDC; it never redirects to Telegram web infrastructure.

## Risks / Trade-offs

- [A forwarded challenge is intentionally approved by another person] -> both apps show the same code and require an explicit account-bearing confirmation; browser binding prevents the recipient from consuming it themselves.
- [Telegram custom schemes are restricted] -> launch occurs from a user gesture, repeat launch remains available, and OIDC is explicit fallback.
- [PWA is frozen in background] -> polling pauses and checks immediately on foreground.
- [Webhook is spoofed] -> require Telegram's secret-token header and validate update actor consistency.
- [Single-process rate limits become insufficient after scaling] -> keep DB correctness constraints and move limiter to shared storage before multi-replica rollout.

## Rollout

1. Apply the additive migration and deploy with login-bot configuration disabled.
2. Resolve the existing Poker bot username through `getMe`, configure the derived webhook secret, and register the production HTTPS webhook.
3. Enable bot login in dev, verify native launch and identity parity with an existing OIDC account, then verify first-login onboarding.
4. Enable production while retaining OIDC fallback and monitor categorized, secret-free lifecycle events.
