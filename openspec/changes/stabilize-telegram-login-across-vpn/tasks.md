## 1. Contracts and diagnostics

- [x] 1.1 Add focused contracts for tri-state Telegram availability, bounded attempt markers, persistent recovery, and controlled redirects.
- [x] 1.2 Preserve the evidence boundary between unobservable browser navigation failure and observable backend provider failure.

## 2. Frontend recovery

- [x] 2.1 Keep the Telegram action visible for unknown config state and hide it only for confirmed disablement.
- [x] 2.2 Record and consume a ten-minute non-sensitive Telegram attempt marker across PWA return/resume.
- [x] 2.3 Render persistent profile feedback with retry/dismiss and soft VPN guidance after incomplete or failed login.
- [x] 2.4 Apply success/error feedback after route rendering so navigation cannot erase it.

## 3. Backend classification

- [x] 3.1 Guard disabled Telegram start and redirect controlled failures to `/profile`.
- [x] 3.2 Distinguish provider timeout/network/5xx from invalid OAuth flow or token errors.
- [x] 3.3 Add privacy-bounded lifecycle logs without OAuth or personal values.

## 4. PWA and verification

- [x] 4.1 Add the attempt helper to the coherent shell and bump the cache generation without caching auth/API.
- [x] 4.2 Run focused frontend/backend tests, full web/Go checks, strict OpenSpec validation, and diff checks.
- [ ] 4.3 Deploy through dev and production, then verify button visibility, incomplete return recovery, successful callback, and production health.
