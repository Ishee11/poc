# Design: Canonical Telegram identity parity

Telegram's signed OIDC token exposes `sub` for the OIDC subject and `id` in the `profile` scope for the Telegram user ID. Bot API updates expose the same Telegram user ID as `from.id`. The canonical `auth_identities(provider='telegram', subject)` value is therefore the decimal profile/Bot API user ID.

OIDC token validation now requires a positive `id`, returns it as the canonical subject, and carries `sub` only as a legacy subject. Resolution first looks up the canonical ID. If absent, an identity with the signed legacy `sub` is atomically updated to the canonical ID without changing its Poker `user_id`. No username-based runtime linking is allowed.

The one-time production repair is deliberately narrower than runtime resolution. It requires the exact established email account, its existing non-numeric Telegram identity with username `semenovv`, and exactly one numeric Telegram identity with the same username owned by a synthetic `telegram-...@telegram.invalid` account. It refuses the repair if the duplicate owns a player or has another identity. Duplicate sessions are revoked by deletion, the canonical identity is moved to the established account, and the empty synthetic user is removed. The migration is idempotent and becomes a no-op after success.
