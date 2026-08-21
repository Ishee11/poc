# Design: Evidence-backed Telegram rollout completion

The production diagnostic runs manually through the existing protected SSH secret and executes a read-only PostgreSQL query. Inputs are base64-transported, validated, and passed to `psql` as quoted variables. Output omits user IDs, identity subjects, tokens, cookies, and credentials; it reports only account/identity kinds, username, player links, and active-session counts.

The login form keeps email/password available but orders the native Telegram action first, followed by its waiting state and a quiet “or sign in with email” divider. Opening the account/login screen does not programmatically focus any credential field. Registration-mode focus remains user-initiated. The service-worker generation changes with the HTML, CSS, JavaScript, and translations.
