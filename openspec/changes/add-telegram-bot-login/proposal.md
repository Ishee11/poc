# Change: Add Telegram app bot-challenge login

## Why

The existing Telegram login redirects the browser from Poker to `oauth.telegram.org`. That domain is unreachable on the target network without a system VPN even though the installed Telegram application works through its own proxy. The login journey therefore requires disruptive VPN switching and is unreliable in the installed PWA.

The repository already has a connected Poker bot token used for Telegram delivery, but it has no user-facing update handler. The application reuses that bot for login, adds a login webhook/client boundary, and does not invent a second Telegram identity model.

## What Changes

- Add a primary Telegram app flow based on a cryptographically random, short-lived, one-use challenge and a `tg://resolve` native deep link.
- Show the same four-digit verification code in Poker and in the bot confirmation message; the code is visual context, not an authentication secret.
- Persist hashed challenges and hashed browser-binding tokens in PostgreSQL with explicit `pending`, `approved`, `denied`, `expired`, and `consumed` states.
- Add status polling and a separate atomic completion step which creates the ordinary opaque HttpOnly Poker session.
- Add a Poker bot webhook handler for `/start <challenge>` and inline approve/cancel callbacks.
- Resolve Bot API `user.id` through the existing `provider=telegram, subject=<id>` identity and preserve the existing first-login account creation behavior.
- Keep the existing Telegram OIDC start/callback flow as the explicit browser fallback.
- Add bounded rate limits for creation, status, completion, and bot actions without introducing Redis.

## Capabilities

### New Capabilities

- `telegram-bot-challenge-login`: native app launch, browser-bound challenge lifecycle, bot confirmation, session completion, foreground recovery, and OIDC fallback.

### Modified Capabilities

- `telegram-login-recovery`: Telegram's primary login action uses the bot challenge while the existing OIDC recovery path remains available as browser fallback.

## Impact

- PostgreSQL gains an ephemeral challenge table and expiry index; no new infrastructure service is required.
- Auth entities, repository ports, use cases, HTTP routes, configuration, and deployment environment forwarding change.
- The Poker backend receives Telegram webhook updates and calls Bot API only from the backend.
- The PWA adds a waiting panel, `tg://` launch, non-aggressive polling, cancellation, and visibility-resume handling.
- Existing OIDC tables, configuration, routes, identity rows, account registration behavior, and session cookies remain compatible.
