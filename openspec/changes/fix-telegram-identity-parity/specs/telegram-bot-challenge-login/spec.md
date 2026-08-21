## MODIFIED Requirements

### Requirement: Bot approval uses the shared Telegram identity
The webhook SHALL use Bot API `user.id` as the canonical Telegram identity subject. OIDC SHALL use the signed profile `id` as that same canonical subject and SHALL NOT treat OIDC `sub` as the Bot API user ID. A signed legacy OIDC `sub` MAY be used only to migrate an existing identity without changing its Poker account. Runtime resolution SHALL NOT join accounts by username. A missing canonical or legacy identity SHALL follow the same first-login account creation behavior for both flows.

#### Scenario: Existing legacy OIDC user approves
- **WHEN** an established Poker account has a legacy Telegram identity and a bot challenge supplies that Telegram user's numeric Bot API ID
- **THEN** the canonical numeric identity resolves to the established Poker account after the controlled migration and no duplicate account is selected

#### Scenario: OIDC migrates an unambiguous legacy identity
- **WHEN** a valid signed OIDC token contains canonical profile `id` and legacy `sub`, only the legacy identity exists, and it belongs to one Poker account
- **THEN** the identity subject is atomically replaced with the canonical numeric ID without changing its Poker user or player ownership

#### Scenario: Canonical and legacy identities conflict
- **WHEN** the signed canonical ID and legacy subject resolve to different Poker accounts
- **THEN** OIDC resolution fails closed instead of silently selecting or merging either account

#### Scenario: Telegram user signs in first time
- **WHEN** neither the canonical Telegram user ID nor a signed legacy identity exists
- **THEN** completion creates the same synthetic-email account and canonical identity shape for OIDC and bot login and leaves player onboarding to the existing ownership flow
