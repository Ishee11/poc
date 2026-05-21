# Club Room Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the Club Room redesign for the existing embedded Poker Session Control frontend while preserving all current workflows.

**Architecture:** Keep the vanilla embedded frontend. Build a local tokenized CSS component system in `web/css/main.css`, make small HTML structure changes in `web/index.html`, and update existing JS render functions only where behavior or generated markup must change.

**Tech Stack:** Go embedded static frontend, plain HTML/CSS, browser ES modules, existing `net/http` app, no React/Tailwind/shadcn dependency.

---

## Source Spec

Design spec: `docs/superpowers/specs/2026-05-20-club-room-redesign-design.md`

## File Structure

- `web/css/main.css`: primary visual system, responsive layout, all Club Room tokens and component styles.
- `web/index.html`: static page structure, lobby header simplification, session block ordering, cash-out block removal.
- `web/js/state.js`: add session player action mode state.
- `web/js/i18n.js`: add labels for the Rebuy/Cash-out mode control and any renamed helper text.
- `web/js/ui/player.js`: preserve richer lobby player overview rows; render session player row action buttons based on current mode.
- `web/js/ui/session.js`: handle action mode switching and player-row cash-out; remove dependence on the standalone cash-out form.
- `web/js/app.js`: keep lobby controls wired after header/lobby relocation.
- `web/js/utils.js`: inspect route/PWA behavior if needed; do not change routing unless the iPhone home-screen issue is reproduced or an obvious blind-timer default is found.

## Task 1: Add Club Room CSS Tokens And Base Components

**Files:**
- Modify: `web/css/main.css`

- [ ] **Step 1: Inspect current tokens and component selectors**

Run:

```bash
rg -n "^-|:root|button|\\.panel|\\.stat|\\.modal|\\.notice|@media" web/css/main.css
```

Expected: output includes current `:root`, `button`, `.panel`, `.stat`, `.modal`, and mobile media rules.

- [ ] **Step 2: Replace root tokens with Club Room semantic tokens**

In `web/css/main.css`, replace the existing `:root` block with:

```css
:root {
    --bg: #120d0a;
    --bg-elevated: #17100c;
    --panel: rgba(27, 19, 15, 0.94);
    --panel-strong: rgba(33, 23, 17, 0.98);
    --panel-felt: #26351f;
    --line: rgba(216, 168, 79, 0.2);
    --line-strong: rgba(216, 168, 79, 0.34);
    --text: #fff2de;
    --muted: #cbb693;
    --muted-soft: #9f8b70;
    --accent: #d8a84f;
    --accent-strong: #f0c875;
    --accent-ink: #1c1109;
    --felt: #48623b;
    --felt-text: #c8e6a8;
    --danger: #f0a092;
    --danger-bg: #2a1714;
    --warning: #f0c875;
    --warning-bg: #2a1e17;
    --input: #120d0a;
    --ring: rgba(240, 200, 117, 0.58);
    --shadow: 0 16px 38px rgba(0, 0, 0, 0.34);
    --radius: 8px;
}
```

- [ ] **Step 3: Update body background**

In `web/css/main.css`, update `body` to use:

```css
body {
    margin: 0;
    min-height: 100vh;
    overflow-x: hidden;
    color: var(--text);
    font-family: "Segoe UI", "Helvetica Neue", sans-serif;
    background:
        radial-gradient(circle at top left, rgba(72, 98, 59, 0.28), transparent 34rem),
        linear-gradient(160deg, #120d0a 0%, #1b120d 44%, #0f0b08 100%);
}
```

- [ ] **Step 4: Run CSS syntax smoke check**

Run:

```bash
node --check web/js/app.js
```

Expected: `node --check` exits 0. This does not validate CSS, but confirms the first task did not accidentally touch JS syntax.

- [ ] **Step 5: Commit**

```bash
git add web/css/main.css
git commit -m "Redesign base theme tokens"
```

## Task 2: Simplify Lobby Header And Mobile Lobby Structure

**Files:**
- Modify: `web/index.html`
- Modify: `web/css/main.css`
- Modify: `web/js/app.js`

- [ ] **Step 1: Confirm current header and lobby controls**

Run:

```bash
sed -n '12,170p' web/index.html
```

Expected: output shows `.app-header`, `.lobby-auth-bar`, lobby connect/start panels, overview sessions, overview players, and language panel.

- [ ] **Step 2: Move non-title controls out of the header**

In `web/index.html`, keep the header to title only:

```html
<header class="app-header">
    <div class="app-header-content">
        <div>
            <h1 data-i18n="app.title">Poker Session Control</h1>
        </div>
        <nav class="top-nav" aria-label="Page navigation" data-i18n-aria-label="nav.page">
            <button type="button" class="secondary nav-session-only" id="session-back-home-btn" data-i18n="nav.backHome">
                Back to Home
            </button>
            <button type="button" class="secondary nav-player-only" id="player-back-session-btn" data-i18n="nav.backSession">
                Back to Session
            </button>
            <button type="button" class="secondary nav-player-only" id="player-back-home-btn" data-i18n="nav.backHome">
                Back to Home
            </button>
            <button type="button" class="secondary nav-players-stats-only" id="players-stats-back-home-btn" data-i18n="nav.backHome">
                Back to Home
            </button>
            <button type="button" class="secondary nav-account-only" id="account-back-home-btn" data-i18n="nav.backHome">
                Back to Home
            </button>
            <button type="button" class="secondary nav-blinds-only" id="blinds-back-home-btn" data-i18n="nav.backHome">
                Back to Home
            </button>
        </nav>
    </div>
</header>
```

Then move the existing `.lobby-auth-bar` markup into `#screen-lobby`, after the players overview and before language/admin controls. Keep all existing IDs unchanged:

```html
<div class="lobby-auth-bar lobby-utility-bar">
    <label class="guest-player-label" id="guest-player-label">
        <span data-i18n="guest.player">Guest player</span>
        <select id="guest-player-select">
            <option value="" data-i18n="guest.noPlayer">No player selected</option>
        </select>
    </label>
    <section class="auth-panel" aria-label="Authentication" data-i18n-aria-label="auth.label">
        ...
    </section>
</div>
```

Use the exact existing auth-panel children, unchanged, so `app.js` IDs continue working.

- [ ] **Step 3: Keep Sessions and Players collapsed by default**

In `web/index.html`, ensure these tags do not have an `open` attribute:

```html
<details class="panel panel-disclosure" id="overview-sessions-panel">
<details class="panel panel-disclosure" id="overview-players-panel">
```

- [ ] **Step 4: Update lobby CSS for minimal header**

In `web/css/main.css`, update header styles:

```css
.app-header {
    padding: 14px 16px;
    border: 1px solid var(--line);
    border-radius: var(--radius);
    background:
        linear-gradient(120deg, rgba(33, 23, 17, 0.98), rgba(38, 53, 31, 0.92)),
        var(--panel-strong);
    box-shadow: var(--shadow);
    overflow: hidden;
    position: relative;
}

.app-header h1 {
    margin: 0;
    font-size: 1.35rem;
    font-weight: 800;
    letter-spacing: 0;
}

.app-header p {
    display: none;
}

.lobby-utility-bar {
    margin-top: 0;
}
```

- [ ] **Step 5: Run syntax checks**

Run:

```bash
node --check web/js/app.js
node --check web/js/ui/lobby.js
```

Expected: both commands exit 0.

- [ ] **Step 6: Commit**

```bash
git add web/index.html web/css/main.css web/js/app.js
git commit -m "Simplify mobile lobby structure"
```

## Task 3: Preserve Rich Lobby Player Overview Rows

**Files:**
- Modify: `web/js/ui/player.js`
- Modify: `web/css/main.css`

- [ ] **Step 1: Inspect current overview player renderer**

Run:

```bash
sed -n '1,150p' web/js/ui/player.js
```

Expected: output includes `renderPlayersOverview()`.

- [ ] **Step 2: Update overview row markup to keep metadata**

In `web/js/ui/player.js`, update the player overview row template inside `renderPlayersOverview()` so each row includes name, rank, sessions count, status/activity, profit, and average buy-in. Use existing helpers already in this file:

```js
return `
  <div class="player-row overview-player-row clickable-row" data-open-player="${escapeHtml(id)}" tabindex="0" role="button">
    <div class="row-main">
      <div class="row-title player-name-line">
        <span>${escapeHtml(player.player_name || id)}</span>
        ${renderPlayerRankBadge(player.rank)}
      </div>
      <div class="inline-stats">
        <span>${escapeHtml(t("common.sessions"))}: ${formatNumber(player.sessions_count)}</span>
        <span>${escapeHtml(t("sort.lastActivity"))}: ${escapeHtml(formatDate(player.last_activity_at))}</span>
        <span>${escapeHtml(t("playersStats.avgBuyIn"))}: ${formatNumber(roundMetric(avgBuyInChips(player)))}</span>
        <span class="${profitMoney >= 0 ? "profit-positive" : "profit-negative"}">${escapeHtml(t("common.profit"))}: ${formatMoney(profitMoney, "RUB")}</span>
      </div>
    </div>
  </div>
`;
```

If local variable names differ, adapt only the names, not the displayed data requirements.

- [ ] **Step 3: Add compact mobile metadata CSS**

In `web/css/main.css`, add:

```css
.overview-player-row .inline-stats {
    gap: 6px;
}

.overview-player-row .inline-stats span {
    border: 1px solid rgba(216, 168, 79, 0.16);
    border-radius: 999px;
    padding: 3px 7px;
    background: rgba(18, 13, 10, 0.42);
}
```

- [ ] **Step 4: Run syntax check**

Run:

```bash
node --check web/js/ui/player.js
```

Expected: exits 0.

- [ ] **Step 5: Commit**

```bash
git add web/js/ui/player.js web/css/main.css
git commit -m "Preserve player overview metadata"
```

## Task 4: Add Session Rebuy/Cash-Out Mode

**Files:**
- Modify: `web/js/state.js`
- Modify: `web/js/i18n.js`
- Modify: `web/js/ui/player.js`
- Modify: `web/js/ui/session.js`
- Modify: `web/index.html`
- Modify: `web/css/main.css`

- [ ] **Step 1: Add mode state**

In `web/js/state.js`, add this field near `players`:

```js
  sessionPlayerActionMode: "rebuy",
```

- [ ] **Step 2: Add i18n labels**

In both English and Russian dictionaries in `web/js/i18n.js`, add:

```js
"session.actionMode": "Player action",
"session.actionModeRebuy": "Rebuy",
"session.actionModeCashOut": "Cash-out",
```

Russian:

```js
"session.actionMode": "Действие",
"session.actionModeRebuy": "Ребай",
"session.actionModeCashOut": "Кэшаут",
```

- [ ] **Step 3: Remove the standalone cash-out panel from HTML**

In `web/index.html`, delete the entire block:

```html
<div class="panel action-card" id="session-actions-panel">
    ...
</div>
```

Keep `session-player-actions` under the players block.

- [ ] **Step 4: Render mode control in player block**

In `web/index.html`, add the mode control above `#players-wrap`:

```html
<div class="session-player-mode-row" id="session-player-mode-row">
    <span class="stat-label" data-i18n="session.actionMode">Player action</span>
    <div class="segmented-control" role="group" aria-label="Player action">
        <button type="button" id="session-player-mode-rebuy" data-session-player-mode="rebuy" data-i18n="session.actionModeRebuy">Rebuy</button>
        <button type="button" id="session-player-mode-cash-out" data-session-player-mode="cash_out" data-i18n="session.actionModeCashOut">Cash-out</button>
    </div>
</div>
```

- [ ] **Step 5: Update session player row buttons**

In `web/js/ui/player.js`, replace the current `canRebuy` button block in `renderPlayers()` with:

```js
const actionMode = state.sessionPlayerActionMode === "cash_out" ? "cash_out" : "rebuy";
const actionLabel = actionMode === "cash_out" ? t("session.actionModeCashOut") : t("session.actionModeRebuy");
const actionClass = actionMode === "cash_out" ? "cash-out-action" : "rebuy-action";
const actionAttr = actionMode === "cash_out" ? "data-session-cash-out-player" : "data-session-rebuy-player";
```

And render:

```js
canUseSessionActions
  ? `<button type="button" class="${actionClass} row-action" ${actionAttr}="${escapeHtml(id)}">${escapeHtml(actionLabel)}</button>`
  : ""
```

- [ ] **Step 6: Add mode click handling and player cash-out handling**

In `web/js/ui/session.js`, inside `initSessionActions()` after the rebuy handler, add:

```js
const cashOutPlayerId = button.getAttribute("data-session-cash-out-player");
if (cashOutPlayerId) {
  await confirmPlayerCashOut(cashOutPlayerId);
  return;
}

const playerMode = button.getAttribute("data-session-player-mode");
if (playerMode) {
  state.sessionPlayerActionMode = playerMode === "cash_out" ? "cash_out" : "rebuy";
  renderPlayers();
  renderSessionPlayerMode();
  return;
}
```

Create:

```js
function renderSessionPlayerMode() {
  const row = document.getElementById("session-player-mode-row");
  const rebuy = document.getElementById("session-player-mode-rebuy");
  const cashOut = document.getElementById("session-player-mode-cash-out");
  const isActive = state.session?.status === "active";
  if (row) row.hidden = !isActive;
  if (rebuy) rebuy.classList.toggle("is-active", state.sessionPlayerActionMode !== "cash_out");
  if (cashOut) cashOut.classList.toggle("is-active", state.sessionPlayerActionMode === "cash_out");
}
```

Call `renderSessionPlayerMode()` from `renderSession()`.

- [ ] **Step 7: Add player cash-out modal**

In `web/js/ui/session.js`, create:

```js
async function confirmPlayerCashOut(playerId) {
  const player = state.players.find((item) => (item.player_id || item.id) === playerId);
  if (!playerId || !player) {
    showNotice(t("notice.selectPlayerAndChips"), "error");
    return;
  }

  const playerName = findPlayerName(playerId);
  const values = await openModal({
    title: t("modal.confirmCashOutTitle"),
    description: t("modal.confirmCashOutDescription", {
      chips: "",
      name: playerName,
    }),
    confirmText: t("session.cashOut"),
    fields: [
      {
        name: "chips",
        label: t("session.chips"),
        type: "number",
        min: "1",
        step: 1000,
        placeholder: t("session.chips"),
      },
    ],
  });
  if (!values) return;

  const chips = Number(values.chips);
  if (!Number.isFinite(chips) || chips <= 0) {
    showNotice(t("notice.selectPlayerAndChips"), "error");
    return;
  }

  const res = await cashOut({
    sessionId: state.activeSessionId,
    playerId,
    chips,
  });
  if (!res.ok) {
    showNotice(describeError(res, t("error.failedCashOut")), "error");
    return;
  }

  await refreshSessionData();
  showNotice(t("notice.cashOutRecorded", { name: playerName }), "success");
}
```

- [ ] **Step 8: Add segmented control CSS**

In `web/css/main.css`, add:

```css
.session-player-mode-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 10px;
    margin: 8px 0 12px;
}

.segmented-control {
    display: inline-flex;
    gap: 3px;
    padding: 3px;
    border: 1px solid var(--line);
    border-radius: var(--radius);
    background: rgba(18, 13, 10, 0.72);
}

.segmented-control button {
    min-height: 32px;
    padding: 6px 10px;
    border: 0;
    border-radius: 6px;
    color: var(--muted);
    background: transparent;
    box-shadow: none;
}

.segmented-control button.is-active {
    color: var(--accent-ink);
    background: var(--accent);
}
```

- [ ] **Step 9: Run syntax checks**

Run:

```bash
node --check web/js/state.js
node --check web/js/i18n.js
node --check web/js/ui/player.js
node --check web/js/ui/session.js
```

Expected: all exit 0.

- [ ] **Step 10: Commit**

```bash
git add web/index.html web/css/main.css web/js/state.js web/js/i18n.js web/js/ui/player.js web/js/ui/session.js
git commit -m "Move cash-out into player actions"
```

## Task 5: Restyle Session Metrics, Lower Blocks, And Mobile Order

**Files:**
- Modify: `web/index.html`
- Modify: `web/css/main.css`

- [ ] **Step 1: Confirm session block order**

Run:

```bash
sed -n '307,470p' web/index.html
```

Expected: output shows session toolbar, stat cards, players block, operations, expenses, settlement, debug controls.

- [ ] **Step 2: Place finish button after metrics for mobile-friendly order**

In `web/index.html`, keep `#session-finish-actions` near metrics, after `#finish-session-hint`:

```html
<div id="finish-session-hint" class="finish-hint" hidden></div>

<div class="actions compact-actions session-finish-inline" id="session-finish-actions">
    <button type="button" class="warn" id="finish-session-btn" data-i18n="session.finish">
        Finish Session
    </button>
</div>
```

Remove the old `#session-finish-actions` wrapper from `.session-status-row`.

- [ ] **Step 3: Update stat card CSS**

In `web/css/main.css`, add or update:

```css
.stat {
    border: 1px solid var(--line);
    border-radius: var(--radius);
    background: rgba(33, 23, 17, 0.9);
}

.session-balance-stat {
    background: linear-gradient(135deg, rgba(38, 53, 31, 0.98), rgba(27, 19, 15, 0.96));
    border-color: rgba(72, 98, 59, 0.88);
}

.session-balance-stat #stat-total-chips {
    font-size: clamp(2rem, 8vw, 3.4rem);
    font-weight: 850;
}

.session-finish-inline {
    margin-top: 12px;
}

.session-finish-inline button {
    width: 100%;
}
```

- [ ] **Step 4: Restyle lower blocks without changing IDs**

In `web/css/main.css`, ensure these existing panels use the same surface style:

```css
#session-operations-panel,
#session-expenses-panel,
#session-settlement-panel {
    border-color: var(--line);
    background: var(--panel);
}

.operation-row,
.expense-row,
.settlement-transfer-row,
.settlement-transfer-card {
    border-color: rgba(216, 168, 79, 0.16);
    background: rgba(42, 30, 23, 0.72);
}
```

If `.settlement-transfer-card` does not exist, add the CSS anyway for the mobile transfer-card task.

- [ ] **Step 5: Run syntax checks**

Run:

```bash
node --check web/js/ui/session.js
```

Expected: exits 0.

- [ ] **Step 6: Commit**

```bash
git add web/index.html web/css/main.css
git commit -m "Restyle session workspace"
```

## Task 6: Restyle Player Detail, Player Stats, And Blind Timer

**Files:**
- Modify: `web/css/main.css`
- Modify: `web/js/ui/player.js`
- Modify: `web/js/ui/blinds.js` only if generated markup needs additional classes.

- [ ] **Step 1: Inspect generated player and blind markup**

Run:

```bash
rg -n "player-stats|mobile-session-cards|blinds-|card-item|summary-card" web/js/ui/player.js web/js/ui/blinds.js web/css/main.css
```

Expected: output shows existing selectors and generated class names.

- [ ] **Step 2: Add player detail mobile card styling**

In `web/css/main.css`, add:

```css
.period-toolbar,
.mobile-player-session-card,
.card-item {
    border: 1px solid var(--line);
    border-radius: var(--radius);
    background: rgba(27, 19, 15, 0.92);
}

.player-rank {
    border: 1px solid rgba(216, 168, 79, 0.28);
    background: rgba(216, 168, 79, 0.16);
    color: var(--accent-strong);
}
```

- [ ] **Step 3: Add blinds styling tokens**

In `web/css/main.css`, update blind hero styling:

```css
.blinds-hero-panel {
    border-color: var(--line-strong);
    background:
        radial-gradient(circle at center, rgba(72, 98, 59, 0.22), transparent 32rem),
        var(--panel-strong);
}

.blinds-timer-display {
    color: var(--accent-strong);
    text-shadow: 0 0 24px rgba(216, 168, 79, 0.24);
}

.blinds-score-item,
.blinds-tool-group,
.blinds-level-form,
.blinds-structure-shell {
    border-color: var(--line);
    background: rgba(18, 13, 10, 0.52);
}
```

- [ ] **Step 4: Run syntax checks**

Run:

```bash
node --check web/js/ui/player.js
node --check web/js/ui/blinds.js
```

Expected: both exit 0.

- [ ] **Step 5: Commit**

```bash
git add web/css/main.css web/js/ui/player.js web/js/ui/blinds.js
git commit -m "Restyle player stats and blind timer"
```

## Task 7: Verify PWA Routing And Full Frontend Smoke

**Files:**
- Modify: `web/js/utils.js` only if the routing bug is found.
- Modify: `web/manifest.webmanifest` only if `start_url` incorrectly points to the blind timer.

- [ ] **Step 1: Inspect manifest and route code**

Run:

```bash
cat web/manifest.webmanifest
rg -n "route|blinds|location|pathname|hash|start_url|setScreen|openInitialRoute" web/js web/manifest.webmanifest
```

Expected: manifest `start_url` should point to `/` or a neutral route. Route code should not default to blinds unless the URL requests it.

- [ ] **Step 2: Fix manifest if it starts at blinds**

If `web/manifest.webmanifest` has a blind timer start URL, change it to:

```json
"start_url": "/",
"scope": "/"
```

If it already has this, do not edit the manifest.

- [ ] **Step 3: Fix route default only if code prefers blinds**

If `openInitialRoute()` or route helpers default to blinds, change the default branch to:

```js
setScreen("lobby");
replaceRoute(routeToHome());
```

If route code already defaults to lobby, do not edit routing.

- [ ] **Step 4: Run syntax checks**

Run:

```bash
node --check web/js/app.js
node --check web/js/utils.js
node --check web/js/ui/session.js
node --check web/js/ui/player.js
node --check web/js/ui/blinds.js
```

Expected: all exit 0.

- [ ] **Step 5: Run Go tests**

Run:

```bash
go test ./...
```

Expected: all packages pass or integration tests skip when local database is unavailable. Investigate any compile failures.

- [ ] **Step 6: Manual browser smoke**

Start the app:

```bash
DATABASE_URL='postgres://poker:poker@127.0.0.1:5432/poker?sslmode=disable' HTTP_PORT=8080 LOG_LEVEL=info HTTP_ACCESS_LOG=errors go run ./cmd/app
```

Open `http://127.0.0.1:8080/` and verify:

- mobile width 390px: header title only;
- lobby sessions and players collapsed by default;
- admin login is a quiet bottom disclosure;
- start session has chip rate and big blind;
- active session shows chip rate, big blind, chips on table, total buy-in, total cash-out;
- Rebuy mode works from player rows;
- Cash-out mode works from player rows;
- add existing/create new player buttons remain under players;
- finish session button is under main metrics on mobile;
- latest operations reverse button remains;
- expenses add/split/close/delete controls remain;
- settlement edit/reset/add/delete controls remain;
- player detail period controls and session history remain;
- blind timer opens only when requested.

- [ ] **Step 7: Commit**

```bash
git add web/js/utils.js web/manifest.webmanifest
git commit -m "Verify PWA route defaults"
```

If neither file changed, skip this commit and note that routing already defaulted to lobby.
