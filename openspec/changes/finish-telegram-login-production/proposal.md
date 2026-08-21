# Change: Finish Telegram login production rollout

## Why

Production acceptance showed that the first identity repair did not match the actual rows, and the login screen still emphasized email/password by focusing the email field immediately. The rollout needs evidence from production data and a Telegram-first mobile experience.

## What Changes

- Add a credential-safe, read-only production identity diagnostic before any further data repair.
- Reconcile the confirmed Telegram identity only after inspecting its actual account and player relationships.
- Place Telegram above email/password as the primary login action.
- Stop focusing credential fields when the login screen opens so mobile keyboards and password prompts do not appear automatically.
- Advance the PWA shell cache generation for the changed login assets.
