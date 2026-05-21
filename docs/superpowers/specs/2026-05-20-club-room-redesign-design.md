# Club Room Redesign Design

Date: 2026-05-20

## Goal

Redesign the existing Poker Session Control web UI into a warmer "private club" interface while preserving the current application behavior.

The implementation must stay in the current embedded vanilla frontend stack:

- `web/index.html`
- `web/css/main.css`
- existing JavaScript modules under `web/js`

Do not introduce React, Tailwind, or shadcn/ui as dependencies. The design may borrow shadcn-style ideas: semantic tokens, consistent component variants, clear focus rings, buttons, badges, dialogs, tables, segmented controls, and collapsible blocks.

## Design Direction

Use the "Club Room Control" direction:

- warm dark background instead of the current cold blue-green theme;
- green felt accents for live table/session state;
- brass/gold primary actions for fast table operations;
- compact cards and panels with 8px-style radii where possible;
- high-contrast text for use during real games;
- dense but organized information layout, not a marketing/landing-page style.

The UI should feel like a focused control panel for a private poker night: atmospheric, but still operational.

## Scope

Full visual redesign of the current frontend without changing backend APIs or domain behavior.

In scope:

- lobby;
- active session workspace;
- player rows and player detail;
- player statistics;
- blind timer and blind structure editor;
- account/auth/guest controls;
- admin/debug controls;
- modals, notices, disclosures, tables, forms, and mobile layouts.

Out of scope:

- replacing the frontend framework;
- changing API contracts;
- removing existing features;
- redesigning backend behavior;
- changing poker settlement rules.

## Session Screen

The session screen is the most important workflow.

### Main Metrics

The top session block must keep all current session metrics visible:

- session status;
- session date;
- chip rate;
- big blind;
- chips on table;
- total buy-in;
- total cash-out;
- total money after session finish;
- finish warning/constraint messaging.

Chip rate and big blind must not disappear. In debug mode, their existing edit behavior must remain available.

### Player Actions

The standalone cash-out block should be removed.

Instead, the player list gets a global action mode control:

- `Rebuy`
- `Cash-out`

The selected mode changes the fast action button shown on each player row.

Behavior:

- in `Rebuy` mode, each player row button opens the existing rebuy modal with the last buy-in amount prefilled;
- in `Cash-out` mode, each player row button opens a cash-out modal for that player;
- cash-out should still require entering/confirming chips before recording;
- clicking the player row outside the action button should still open player detail;
- active/settled player status remains visible.

This keeps fast rebuy behavior while making cash-out available in the same player-centered workflow.

### Add Player

`Add existing player` and `Create new player` stay under the players block.

They should continue using the current modal flows:

- choose existing player and initial buy-in;
- create a new player and immediately add them to the session with initial buy-in.

### Finish Session

On mobile, `Finish session` should be placed under the main metrics block.

On desktop, it may sit in the session header or near the main metrics, but it should not live in a right rail that consumes player-list width.

The current finish constraint remains:

- session can only finish when chips on table is zero;
- the existing error/hint behavior remains.

## Lower Session Blocks

Keep the current blocks and behavior:

- latest operations;
- game expenses;
- settlement transfers;
- debug delete session controls.

### Latest Operations

Preserve:

- operation type display;
- player name;
- chips;
- reverse action for reversible operations;
- reversed operation handling;
- finished-session operation display.

### Game Expenses

Preserve:

- expense title and amount;
- participant selection;
- select all / clear all;
- paid-by inputs;
- split-even action;
- add expense;
- delete expense;
- close bill;
- closed bill locked state;
- admin/debug override behavior.

The visual layout can become cleaner, but the full form behavior must stay.

### Settlement Transfers

Preserve:

- automatic transfer list;
- manual edit mode;
- done;
- reset to automatic;
- add manual transfer;
- delete manual transfer;
- editable from/to/amount controls.

Use compact transfer cards on mobile and a denser list/table-like layout on desktop.

## Mobile Layout

Do not use a right-side Action Rail on mobile.

Mobile order:

1. session title/status/date;
2. chip rate and big blind;
3. chips on table;
4. total buy-in and total cash-out;
5. finish session;
6. players block with global `Rebuy / Cash-out` mode;
7. add existing/create new player;
8. latest operations;
9. expenses;
10. settlement transfers;
11. debug/admin controls when enabled.

The player list must remain usable on narrow screens:

- action buttons do not wrap awkwardly;
- long names truncate or wrap cleanly;
- important numbers remain readable;
- no text overlaps.

## Desktop Layout

Desktop should use wider layouts where helpful, but avoid hiding primary controls.

Recommended structure:

- top session panel for metrics and status;
- players block as the main operational block;
- optional compact desktop-only action grouping for non-player actions, but not a dominant rail;
- lower blocks arranged as one or two columns depending on available width;
- expenses and settlement can use wider layouts for dense forms.

## Lobby

Preserve:

- guest player selector;
- admin login/logout controls;
- connect to latest active session;
- active session select;
- open workspace;
- blind timer entry;
- start new session with chip rate and big blind;
- sessions overview;
- players overview;
- player sort;
- show all players toggle;
- player stats entry;
- language selector;
- admin login footer.

Visual update:

- use the same Club Room tokens;
- keep the lobby header minimal: title only, no auth/language/guest controls in the header;
- make start/connect panels compact and clear;
- preserve collapsible session/player overviews.
- sessions and players overviews should be collapsed by default on the mobile lobby;
- admin login should be a quiet disclosure near the bottom of the lobby, not a primary header control;
- non-admin login/register UI should stay out of the primary mobile lobby until explicitly needed.
- expanded player overview rows must keep useful player metadata, not collapse to only name/profit:
  sessions count, status/activity, rank badge when present, profit, and average buy-in or comparable compact stat.

### PWA / iPhone Home Screen

Investigate and preserve correct behavior when the app is launched from an iPhone home-screen PWA shortcut.

Known concern:

- attempting to create/open the home-screen app on iPhone appeared to always create/open the blind timer screen.

The redesign must not hard-code or accidentally prefer the blind timer route. Initial routing should respect the intended route and should default to the lobby unless a valid route indicates otherwise.

## Blind Timer

Preserve:

- timer display;
- current level;
- current blinds;
- next level;
- start/pause/resume;
- reset timer;
- reset to default structure;
- presentation mode;
- exit presentation;
- previous/next level;
- push alert subscription;
- push warning settings at 60s and 10s;
- test alert;
- blind structure editor;
- selected level;
- small blind, big blind, duration;
- apply duration to following levels;
- add level;
- delete level;
- delete all levels;
- completed/current/future level lock behavior;
- level flash/transition feedback.

Visual update:

- presentation mode can be more dramatic;
- default mode should remain a practical editor/control screen.

## Player Detail And Stats

Preserve:

- opening player detail from rows;
- player rank badge;
- debug player ID;
- linked user display;
- rename player in debug mode;
- delete player in debug mode;
- period filter modal;
- clear period;
- session stats;
- currency stats;
- session history table on desktop;
- mobile session cards;
- opening sessions from player history;
- stat help dialogs;
- players statistics overview.

## Auth, Account, Language, Admin

Preserve:

- auth UI feature flag behavior;
- guest player selection;
- admin login;
- admin logout;
- admin login disclosure;
- admin-only debug mode behavior;
- language switching.

For this redesign, the visible login path is admin-only. Public login/register/account controls should not be presented as primary lobby UI. If `authUiEnabled` is enabled in configuration, existing account/linking behavior should still be styled consistently, but it should remain secondary to the admin-only current product flow.

## Component System

Create a local CSS component system inspired by shadcn-style tokenization, implemented in plain CSS.

Use semantic CSS custom properties for:

- background;
- foreground;
- panel/card;
- primary;
- secondary;
- muted;
- accent;
- destructive;
- warning;
- border;
- input;
- focus ring;
- shadow.

Define consistent variants for:

- primary button;
- secondary button;
- ghost/quiet button;
- warning button;
- destructive button;
- player action button;
- status badge;
- stat card;
- disclosure panel;
- modal;
- table/list row;
- segmented control.

Do not copy shadcn React components into this project.

## Accessibility And Interaction

Preserve keyboard access for clickable rows and buttons.

Requirements:

- visible focus states;
- buttons remain real `<button>` elements;
- clickable rows keep `tabindex` and keyboard activation;
- form labels remain associated with controls;
- disabled states are visually clear;
- modals remain readable on mobile;
- color is not the only status signal.

## Implementation Constraints

Keep changes scoped to the frontend unless a backend issue blocks the redesign.

Expected files:

- `web/index.html`
- `web/css/main.css`
- `web/js/ui/player.js`
- `web/js/ui/session.js`
- possibly `web/js/state.js` for the Rebuy/Cash-out mode state;
- possibly `web/js/i18n.js` for any new labels.

Avoid touching unrelated backend files.

The current dirty worktree contains unrelated backend/config changes. Do not revert them.

## Verification

Before calling the implementation complete, verify:

- the app builds/runs;
- session start still accepts chip rate and big blind;
- active session shows chip rate, big blind, chips on table, buy-in, cash-out;
- rebuy works from player rows;
- cash-out works from player rows after switching mode;
- adding existing player works;
- creating and adding a new player works;
- finish session behavior remains constrained by chips on table;
- reversing operations still works;
- expenses can be added, split, closed, and deleted;
- settlement transfers can be edited/reset/added/deleted;
- player detail and player stats still open;
- blind timer controls still work;
- language switching still updates visible labels;
- mobile viewport does not overlap text or lose actions.
