## 1. Backend Contract and Counting

- [x] 1.1 Add explicit total and viewer-visible session counts to the player-detail response while retaining the existing total-count compatibility field.
- [x] 1.2 Add a repository visible-count operation using the exact player, period, and access semantics of the detailed session query.
- [x] 1.3 Wire total aggregate, visible count, and visible records through the existing player-stats transaction without deriving counts from list length.

## 2. Backend Privacy and Behavior Tests

- [x] 2.1 Add use-case coverage for 0/0, 10/10, 10/4, 10/0, and non-zero financial result with zero visible sessions.
- [x] 2.2 Add PostgreSQL/HTTP integration coverage proving hidden rows do not appear or leak fields and visible counts match the authorized session set.

## 3. Player Detail UI

- [x] 3.1 Render explicit total and visible count states for complete, partial, zero-visible, truly-empty, and unavailable histories.
- [x] 3.2 Add localized explanatory copy and an informative history empty state while clarifying that financial aggregates may include unavailable sessions.
- [x] 3.3 Add focused frontend tests for every required count/visibility state and advance the service-worker cache version for changed web modules.
- [x] 3.4 Label a partially visible player-session list with both the visible and total counts.
- [x] 3.5 Flatten the player-detail visual hierarchy and remove nested borders from the session history presentation.

## 4. Documentation and Validation

- [x] 4.1 Update Swagger artifacts and access-model documentation for the additive counts and privacy boundary.
- [x] 4.2 Run formatting, focused backend/frontend tests, full Go tests/build, web lint/type checks, and `git diff --check`.
- [x] 4.3 Run strict validation for this OpenSpec change and all repository specs.

## 5. Staged Rollout

- [ ] 5.1 Commit only scoped files, push `dev`, and verify the successful dev deployment and external health/API/UI behavior.
- [ ] 5.2 Promote the validated commit to `master`, verify the successful production deployment, external health/API contract, UI behavior, and deployed service-worker version.
