# Design: Recoverable Telegram login across VPN transitions

## Diagnosis boundary

`/auth/telegram/start` creates the PKCE/nonce flow locally and returns an HTTP redirect. It does not contact Telegram. Once the browser leaves the Poker origin, same-origin isolation prevents Poker from observing whether the Telegram authorization document loaded, timed out, or was cancelled. The application can therefore report an incomplete attempt after return, but MUST NOT claim that a browser-to-Telegram timeout was proven.

Backend token and JWKS requests do run under a ten-second HTTP client timeout. Those transport, DNS, timeout, rate-limit, and 5xx failures can be classified as provider unavailability. Invalid state, invalid tokens, and other validation failures remain authentication failures.

## Availability and attempt state

Telegram availability has three values. `unknown` is the safe startup default and keeps the button visible; `enabled` keeps it visible; only a successful config response with `telegram_enabled=false` selects `disabled` and hides it. The server remains authoritative at the start endpoint, so showing an unknown action does not enable a disabled integration.

Before following a Telegram start link, the browser stores `poker-telegram-auth-attempt-v1` with only `mode` and `startedAt`. Storage access is guarded, prefers local storage for PWA relaunches, and may fall back to session storage. The marker contains no OAuth state, code, verifier, nonce, token, account identifier, or Telegram identity. Its validity is ten minutes, matching the server flow lifetime.

After session restoration, explicit callback query values take precedence. Success or an authenticated session clears the marker. A controlled error consumes it and selects the corresponding persistent feedback. With no callback and no authenticated session, a valid marker is consumed as an incomplete attempt, `/profile` is opened, and the login panel displays recovery. Invalid or expired markers are removed silently.

## Profile recovery UI

The profile screen owns a persistent Telegram feedback panel outside the transient global notice. Incomplete login copy says that login did not complete, that Telegram may be unavailable, and that enabling VPN before retry may help. It offers retry and dismiss actions. Retry records a fresh marker and follows the normal server start URL; dismiss clears the in-memory feedback. There is no automatic navigation, reload, or retry loop.

Callback redirects use controlled values `cancelled`, `provider_unavailable`, `failed`, and `disabled` on `/profile`. Raw provider errors and query values are never reflected. Success continues to use `/profile?telegram=logged_in|linked`.

## Observability and compatibility

Server logs record started, failed, cancelled, and completed lifecycle categories with request ID and mode where known. They do not record OAuth query strings, state, code, verifier, nonce, tokens, cookies, Telegram secrets, or identity claims. Frontend diagnostics record only the incomplete/callback category, online state, mode, and bounded marker age.

Existing email login, guest mode, sessions, PKCE, nonce validation, one-use flows, opaque HttpOnly sessions, and IP audit behavior are unchanged. The service worker receives a new cache generation and the attempt helper is a required shell asset; auth and API paths continue to bypass the worker.

## Rollback

Reverting the frontend/backend commit and shell version restores the previous flow without data rollback. Existing browser markers are versioned, non-sensitive, expire after ten minutes, and are ignored by older code.
