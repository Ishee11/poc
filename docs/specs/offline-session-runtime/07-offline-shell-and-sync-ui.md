# Task 07 — Offline App Shell and Sync UI

## Objective

Allow the PWA to open after connectivity loss and clearly communicate local persistence and synchronization state without blocking routine session interaction.

## Depends on

- Task 03 cached session hydration;
- Task 04 optimistic mutations;
- Task 05 replay status;
- Task 06 conflict/block state.

## Allowed implementation scope

Primary paths:

- `web/sw.js`;
- `web/index.html`;
- `web/css/main.css`;
- `web/js/app.js`;
- focused UI/status helper module;
- manifest only when required;
- focused tests or scripted service-worker checks.

Do not put domain outbox replay into the service worker.

## Requirements

### Versioned app-shell cache

The service worker SHALL define an explicit cache version and SHALL pre-cache the minimum resources required to render the application shell.

The asset list SHALL include:

- `/` or an equivalent index-shell request;
- current CSS;
- all statically imported JavaScript entry modules required at startup;
- manifest;
- required icons/SVG used before network refresh.

A missing optional decorative asset SHALL not prevent service-worker installation.

### Activation cleanup

On activation, obsolete caches owned by this application SHALL be deleted. Unrelated origin caches SHALL not be removed.

The worker SHOULD claim current clients after safe activation.

### Fetch strategy

Navigation requests SHALL use network-first with a cached shell fallback.

Versioned or immutable static assets SHOULD use cache-first. Other known static assets MAY use stale-while-revalidate.

API requests SHALL pass through to the network and SHALL not be stored as authoritative response cache entries.

The fetch handler SHALL avoid intercepting unsupported methods and cross-origin requests unless explicitly required.

### Update safety

A newly installed worker SHALL not leave the application permanently using a mixed incompatible shell. Cache names and app versioning SHALL make update behavior observable and reversible.

The implementation SHALL document whether it uses immediate activation or waits for the next navigation.

### Sync status model

The UI SHALL derive a compact status from durable/local runtime state:

- `online_fresh`;
- `online_syncing`;
- `offline_clean`;
- `offline_pending`;
- `retry_wait`;
- `authorization_blocked`;
- `domain_blocked`;
- `local_storage_unavailable`.

The status SHALL include pending count when greater than zero.

### Status presentation

The session screen SHALL provide one persistent but unobtrusive status indicator.

Examples of acceptable Russian labels:

- `Синхронизация…`;
- `Офлайн`;
- `Офлайн · 3 изменения`;
- `Ожидает повторной отправки`;
- `Нужно войти снова`;
- `Есть несинхронизированная ошибка`.

Each provisional operation row SHALL have a visual pending state and accessible text. Color alone SHALL not carry status.

### Loading behavior

Routine buy-in and cash-out SHALL not apply section-wide `is-loading` while replay is in progress.

Cached hydration MAY show a stale/offline status but SHALL preserve usable controls for approved local-first actions.

Online-only actions SHALL be disabled or fail calmly when offline, with text that does not imply local saving.

### User recovery actions

For retryable pending state, the UI SHOULD expose a manual `Повторить синхронизацию` action.

For authorization blocked state, it SHALL direct the user to restore authentication.

For domain blocked state, it SHALL show the server error in human-readable form and preserve the affected local data. A generic discard button is not required.

## Non-goals

- no Background Sync API requirement;
- no push-based outbox replay;
- no generic conflict editor;
- no offline blind-clock server control;
- no full design-system rewrite.

## Acceptance scenarios

### Offline application startup

- GIVEN the user previously loaded the application successfully;
- WHEN the browser is offline and the user opens the installed PWA;
- THEN the application shell renders;
- AND a known cached session can hydrate from IndexedDB;
- AND API failures do not replace it with a blank screen.

### Pending status

- GIVEN three local commands are pending;
- WHEN the session screen renders offline;
- THEN the persistent indicator states that the app is offline with three changes;
- AND each provisional operation has accessible pending text.

### Replay status transition

- GIVEN connectivity returns;
- WHEN replay starts and later succeeds;
- THEN status changes from offline pending to syncing and then online fresh;
- AND the session controls remain usable throughout except where command ordering requires a specific guard.

### Online-only action offline

- GIVEN the app is offline;
- WHEN the user attempts to start a new session or finish the current session;
- THEN the application does not claim success;
- AND it explains that this action requires connectivity.

### Cache upgrade

- GIVEN an old app-shell cache exists;
- WHEN the new worker activates;
- THEN the new shell is available;
- AND only old Poker Session Control cache versions are removed.

## Verification

- scripted service-worker install/activate/fetch tests where practical;
- browser DevTools offline reload test;
- installed-PWA navigation test on mobile Safari-compatible behavior where possible;
- status-state mapping tests;
- accessibility check for pending and blocked labels;
- existing frontend checks.

## Completion output

The implementing agent SHALL report:

- cache name/version;
- precached asset set;
- navigation and static fetch strategies;
- worker activation policy;
- sync-state mapping and labels;
- manual offline verification performed.
