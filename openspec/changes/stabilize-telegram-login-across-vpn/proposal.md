# Change: Stabilize Telegram login across VPN transitions

## Why

Telegram authentication currently fails in two misleading ways around VPN use. If `GET /auth/config` fails during a VPN-side connection reset or timeout, the frontend leaves `telegramAuthEnabled` at its default `false` and hides the Telegram button even though the server did not disable Telegram. Without VPN, Poker can successfully return the authorization redirect while the browser cannot reach `oauth.telegram.org`; because that navigation happens on another origin, Poker receives no callback or timeout signal and the user returns to an unexplained main screen.

Callback failures have a separate confirmed presentation defect: the backend redirects to `/`, and the frontend's transient notice is cleared by the subsequent route render. The result is again an apparently silent return to the lobby.

Production retesting after the first recovery rollout exposed the first broken request in the installed PWA: the active service worker treats every same-origin navigation as an application route. It therefore answers `/auth/telegram/start?mode=login` with the cached `/` document instead of letting the request reach the auth handler. The immediate lobby render followed by delayed recovery is caused by this interception, not by Telegram or the backend redirect.

## What Changes

- Model Telegram availability as unknown, enabled, or disabled; only a confirmed disabled response hides the action.
- Record a short-lived, non-sensitive browser marker when Telegram login or linking starts.
- On return without a callback, show a persistent, recoverable profile message that accurately says the login did not complete and suggests VPN as a possibility.
- Keep callback/start failures on `/profile`, distinguish cancellation, provider unavailability, invalid flow, and disabled integration without exposing OAuth values.
- Classify backend Telegram token/JWKS network failures and add privacy-bounded auth lifecycle logging.
- Bump the coherent PWA shell generation for the changed frontend module graph; auth/API responses remain outside service-worker caching.
- Restrict cached navigation fallback to explicit application UI routes so `/auth/*` and other backend endpoints always pass through to the network.

## Impact

- Affected spec: `telegram-login-recovery`
- Affected frontend: profile/auth UI, Telegram attempt state, translations, styles, and shell asset manifest
- Affected backend: Telegram start/callback redirects and provider error classification
- No database migration, session binding, cookie relaxation, OAuth validation weakening, or new public JSON API
