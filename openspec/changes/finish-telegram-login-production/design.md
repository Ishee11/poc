# Design: Evidence-backed Telegram rollout completion

The production diagnostic runs manually through the existing protected SSH secret and executes a read-only PostgreSQL query. Inputs are base64-transported, validated, and passed to `psql` as quoted variables. Output omits user IDs, identity subjects, tokens, cookies, and credentials; it reports only account/identity kinds, username, player links, and active-session counts.

The diagnostic confirmed two numeric subjects for `@seemenovv`: the established email account owns player `Стёпа` but has the historical numeric OIDC `sub`, while an empty synthetic account owns the Bot API user ID and current bot-login session. A one-time fail-closed migration requires that exact email, username, one synthetic duplicate, no duplicate player ownership, and no additional duplicate identity. It revokes the duplicate session, moves the Bot API subject to the established account, and removes the empty synthetic user.

The login form keeps email/password available but orders the native Telegram action first, followed by its waiting state and a quiet “or sign in with email” divider. Opening the account/login screen does not programmatically focus any credential field. Registration-mode focus remains user-initiated. The service-worker generation changes with the HTML, CSS, JavaScript, and translations.

Guest player selection is not authentication. It changes which sessions an unauthenticated participant may view when their player is still unowned, so it belongs in the expanded session-list controls rather than above the login methods.
