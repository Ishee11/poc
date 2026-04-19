import {
  buyIn,
  cashOut,
  createPlayer,
  debugDeleteSessionFinish,
  debugDeleteSession,
  debugUpdateSessionConfig,
  finishSession,
  getSession,
  getSessionOperations,
  reverseOperation,
} from "../api.js";
import { operationLabel, statusLabel, t } from "../i18n.js";
import { state } from "../state.js";
import {
  describeError,
  currencySymbol,
  escapeHtml,
  formatDate,
  formatMoney,
  formatNumber,
  openModal,
  pushRoute,
  replaceRoute,
  routeToHome,
  routeToSession,
  setScreen,
  showNotice,
} from "../utils.js";
import { loadSessions } from "./lobby.js";
import { loadPlayers, loadPlayersOverview } from "./player.js";

export async function openSession(sessionId, { replace = false } = {}) {
  if (!sessionId) return;

  state.activeSessionId = sessionId;
  state.session = null;
  state.operations = [];
  state.players = [];

  const res = await getSession(sessionId);
  if (!res.ok || !res.body) {
    showNotice(describeError(res, t("error.failedLoadSession")), "error");
    return;
  }

  hydrateSession(res.body);
  renderSession();
  renderOperations();
  renderActionPlayerOptions();

  await Promise.all([loadPlayers(sessionId), loadOperations(sessionId)]);
  renderActionPlayerOptions();
  setScreen("session");
  if (replace) {
    replaceRoute(routeToSession(sessionId));
  } else {
    pushRoute(routeToSession(sessionId));
  }
}

export async function loadOperations(sessionId) {
  if (!sessionId) return;

  const res = await getSessionOperations(sessionId);
  if (!res.ok) {
    console.error("loadOperations failed:", res.text);
    state.operations = [];
    renderOperations();
    return;
  }

  state.operations = Array.isArray(res.body) ? res.body : [];
  renderOperations();
  applyDefaultRebuyChips();
}

export function renderSession() {
  const session = state.session;
  if (!session) return;

  const subtitle = document.getElementById("workspace-subtitle");
  const chipRate = document.getElementById("stat-chip-rate");
  const chipRateCard = document.getElementById("stat-chip-rate-card");
  const bigBlind = document.getElementById("stat-big-blind");
  const bigBlindCard = document.getElementById("stat-big-blind-card");
  const buyIn = document.getElementById("stat-buy-in");
  const cashOut = document.getElementById("stat-cash-out");
  const totalChips = document.getElementById("stat-total-chips");
  const totalChipsCard = document.getElementById("stat-total-chips-card");
  const totalMoney = document.getElementById("stat-total-money");
  const moneyPanel = document.getElementById("session-money-panel");
  const status = document.getElementById("workspace-status");
  const finishButton = document.getElementById("finish-session-btn");
  const finishHint = document.getElementById("finish-session-hint");
  const debugDeletePanel = document.getElementById("session-delete-debug-panel");
  const mobileActions = document.getElementById("mobile-session-actions");
  const playerActions = document.getElementById("session-player-actions");
  const playerActionsHint = document.getElementById("session-player-actions-hint");
  const actionsPanel = document.getElementById("session-actions-panel");
  const finishActions = document.getElementById("session-finish-actions");
  const isActive = session.status === "active";
  const onTable = Number(session.totalChips) || 0;

  if (subtitle) {
    subtitle.textContent = formatDate(session.createdAt);
  }
  if (chipRate) {
    chipRate.textContent = t("session.chipRateValue", {
      currencySymbol: currencySymbol(session.currency),
      chips: formatNumber(session.chipRate),
    });
  }
  if (bigBlind) bigBlind.textContent = formatNumber(session.bigBlind);
  if (buyIn) buyIn.textContent = formatNumber(session.totalBuyIn);
  if (cashOut) cashOut.textContent = formatNumber(session.totalCashOut);
  if (totalChips) totalChips.textContent = formatNumber(session.totalChips);
  if (totalMoney) {
    totalMoney.textContent = formatMoney(totalMoneyIn(session), session.currency);
  }
  if (status) {
    status.innerHTML = `
      <span>${escapeHtml(statusLabel(session.status))}</span>
      ${
        state.debugMode && session.status === "finished"
          ? `<button type="button" class="secondary status-debug-action" id="debug-reopen-session-btn">${escapeHtml(t("debug.deleteFinish"))}</button>`
          : ""
      }
    `;
    status.className = `session-status ${session.status}`;
  }
  if (totalChipsCard) {
    totalChipsCard.classList.add("on-table-emphasis");
    totalChipsCard.classList.toggle("on-table-warning", isActive && onTable > 0);
    totalChipsCard.classList.toggle("on-table-clear", isActive && onTable === 0);
  }
  [chipRateCard, bigBlindCard].forEach((card) => {
    if (!card) return;
    card.classList.toggle("debug-editable-stat", state.debugMode);
    card.setAttribute("tabindex", state.debugMode ? "0" : "-1");
    card.setAttribute("role", state.debugMode ? "button" : "presentation");
    card.setAttribute("title", state.debugMode ? t("debug.editSessionConfig") : "");
  });
  if (finishButton) finishButton.disabled = !isActive;
  if (finishActions) finishActions.hidden = !isActive;
  if (finishHint) {
    finishHint.hidden = true;
    finishHint.textContent = "";
  }
  if (playerActions) playerActions.hidden = !isActive;
  if (playerActionsHint) playerActionsHint.hidden = !isActive;
  if (actionsPanel) actionsPanel.hidden = !isActive;
  if (moneyPanel) moneyPanel.hidden = session.status !== "finished";
  if (debugDeletePanel) debugDeletePanel.hidden = !state.debugMode;
  if (mobileActions) mobileActions.hidden = !isActive;

  document
    .getElementById("debug-reopen-session-btn")
    ?.addEventListener("click", async () => {
      await confirmDebugDeleteSessionFinish();
    });

  bindDebugSessionConfigEditor(chipRateCard);
  bindDebugSessionConfigEditor(bigBlindCard);
}

export function renderOperations() {
  const wrap = document.getElementById("operations-wrap");
  const count = document.getElementById("session-operations-count");
  if (!wrap || !count) return;

  const showFinishOperation = state.session?.status === "finished";
  count.textContent = String(state.operations.length + (showFinishOperation ? 1 : 0));

  if (!state.operations.length && !showFinishOperation) {
    wrap.innerHTML = `<div class="empty-inline">${escapeHtml(t("common.noOperations"))}</div>`;
    return;
  }

  const reversedTargets = new Set(
    state.operations
      .filter((operation) => operation.type === "reversal" && operation.reference_id)
      .map((operation) => operation.reference_id),
  );

  const operationRows = state.operations
    .map((operation) => {
      const playerName = findPlayerName(operation.player_id);
      const reversible =
        state.session?.status === "active" &&
        operation.type !== "reversal" &&
        !reversedTargets.has(operation.id);

      return `
        <div class="operation-row">
          <div class="row-main">
            <div class="row-title">
              <span class="operation-type ${escapeHtml(operation.type)}">${escapeHtml(operationLabel(operation.type))}</span>
              ${escapeHtml(playerName)}
            </div>
            <div class="inline-stats">
              <span>${escapeHtml(t("session.chips"))}: ${formatNumber(operation.chips)}</span>
              <span>${escapeHtml(formatDate(operation.created_at))}</span>
            </div>
          </div>
          ${
            reversible
              ? `<button type="button" class="secondary" data-reverse-operation="${escapeHtml(operation.id)}">${escapeHtml(t("common.reverse"))}</button>`
              : '<span class="muted">-</span>'
          }
        </div>
      `;
    })
    .join("");

  const finishRow = showFinishOperation
    ? `
        <div class="operation-row">
          <div class="row-main">
            <div class="row-title">
              <span class="operation-type finish">${escapeHtml(operationLabel("finish"))}</span>
              ${escapeHtml(statusLabel("finished"))}
            </div>
            <div class="inline-stats">
              <span>${escapeHtml(t("common.status"))}: ${escapeHtml(statusLabel("finished"))}</span>
              <span>${escapeHtml(formatDate(state.session.finishedAt || state.session.createdAt))}</span>
            </div>
          </div>
          <span class="muted">-</span>
        </div>
      `
    : "";

  wrap.innerHTML = operationRows + finishRow;

  wrap.querySelectorAll("[data-reverse-operation]").forEach((button) => {
    button.addEventListener("click", async () => {
      const operationId = button.getAttribute("data-reverse-operation");
      if (!operationId) return;
      await confirmReverse(operationId);
    });
  });
}

export function renderActionPlayerOptions() {
  renderPlayerSelect("rebuy-player-select", state.players);
  renderPlayerSelect(
    "cash-out-player-select",
    state.players.filter((player) => player.in_game),
  );
  applyDefaultRebuyChips();
}

function renderPlayerSelect(selectId, players) {
  const select = document.getElementById(selectId);
  if (!select) return;

  const current = select.value;
  const options = [
    `<option value="">${escapeHtml(t("session.selectPlayer"))}</option>`,
    ...players.map((player) => {
      const id = player.player_id || player.id;
      const name = player.player_name || player.name || id;
      return `<option value="${escapeHtml(id)}">${escapeHtml(name)}</option>`;
    }),
  ];

  select.innerHTML = options.join("");
  const exists = players.some((player) => {
    const id = player.player_id || player.id;
    return id === current;
  });
  select.value = exists ? current : "";
}

export function initSessionActions() {
  document.addEventListener("click", async (event) => {
    const button = event.target.closest("button");
    if (!button) return;

    switch (button.id) {
      case "buy-in-btn":
        await confirmBuyIn();
        break;
      case "cash-out-btn":
        await confirmCashOut();
        break;
      case "session-add-existing-player-btn":
        await confirmAddExistingPlayer();
        break;
      case "session-add-new-player-btn":
        await confirmAddNewPlayer();
        break;
      case "finish-session-btn":
        await confirmFinishSession();
        break;
      case "mobile-buy-in-shortcut":
        focusSessionAction("rebuy-player-select");
        break;
      case "mobile-cash-out-shortcut":
        focusSessionAction("cash-out-player-select");
        break;
      case "mobile-finish-session-btn":
        await confirmFinishSession();
        break;
      case "debug-delete-session-btn":
        await confirmDebugDeleteSession();
        break;
      case "session-back-home-btn":
        setScreen("lobby");
        pushRoute(routeToHome());
        break;
      case "player-back-home-btn":
        setScreen("lobby");
        pushRoute(routeToHome());
        break;
      case "player-back-session-btn":
        if (state.activeSessionId) {
          setScreen("session");
          pushRoute(routeToSession(state.activeSessionId));
        } else {
          setScreen("lobby");
          pushRoute(routeToHome());
        }
        break;
      default:
        break;
    }
  });

  document.addEventListener("change", (event) => {
    if (event.target?.id === "rebuy-player-select") {
      applyDefaultRebuyChips({ overwrite: true });
    }
  });
}

function focusSessionAction(controlId) {
  document.getElementById("session-actions-panel")?.scrollIntoView({
    behavior: "smooth",
    block: "start",
  });
  window.setTimeout(() => {
    document.getElementById(controlId)?.focus();
  }, 220);
}

function applyDefaultRebuyChips({ overwrite = false } = {}) {
  const input = document.getElementById("rebuy-chips");
  if (!input) return;
  if (!overwrite && input.value !== "") return;

  const chips = lastBuyInChipsForRebuy(
    document.getElementById("rebuy-player-select")?.value || "",
  );
  if (chips > 0) {
    input.value = String(chips);
  }
}

function lastBuyInChipsForRebuy(playerId) {
  const reversedTargets = new Set(
    state.operations
      .filter((operation) => operation.type === "reversal" && operation.reference_id)
      .map((operation) => operation.reference_id),
  );

  const buyIns = state.operations.filter(
    (operation) =>
      operation.type === "buy_in" &&
      !reversedTargets.has(operation.id) &&
      Number(operation.chips) > 0,
  );

  const playerBuyIn = buyIns.find((operation) => operation.player_id === playerId);
  if (playerBuyIn) return Number(playerBuyIn.chips);

  const sessionBuyIn = buyIns[0];
  return sessionBuyIn ? Number(sessionBuyIn.chips) : 0;
}

async function confirmBuyIn() {
  const playerId = document.getElementById("rebuy-player-select")?.value;
  const chips = Number(document.getElementById("rebuy-chips")?.value);

  if (!playerId || !Number.isFinite(chips) || chips <= 0) {
    showNotice(t("notice.selectPlayerAndChips"), "error");
    return;
  }

  const playerName = findPlayerName(playerId);
  const values = await openModal({
    title: t("modal.confirmBuyInTitle"),
    description: t("modal.confirmBuyInDescription", {
      chips: formatNumber(chips),
      name: playerName,
    }),
    confirmText: t("session.buyIn"),
  });
  if (!values) return;

  const res = await buyIn({
    sessionId: state.activeSessionId,
    playerId,
    chips,
  });
  if (!res.ok) {
    showNotice(describeError(res, t("error.failedBuyIn")), "error");
    return;
  }

  await refreshSessionData();
  applyDefaultRebuyChips({ overwrite: true });
  showNotice(t("notice.buyInRecorded", { name: playerName }), "success");
}

async function confirmCashOut() {
  const playerId = document.getElementById("cash-out-player-select")?.value;
  const chips = Number(document.getElementById("cash-out-chips")?.value);

  if (!playerId || !Number.isFinite(chips) || chips <= 0) {
    showNotice(t("notice.selectPlayerAndChips"), "error");
    return;
  }

  const playerName = findPlayerName(playerId);
  const values = await openModal({
    title: t("modal.confirmCashOutTitle"),
    description: t("modal.confirmCashOutDescription", {
      chips: formatNumber(chips),
      name: playerName,
    }),
    confirmText: t("session.cashOut"),
  });
  if (!values) return;

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
  document.getElementById("cash-out-chips").value = "";
  showNotice(t("notice.cashOutRecorded", { name: playerName }), "success");
}

async function confirmAddExistingPlayer() {
  await loadPlayersOverview();

  const currentIds = new Set(
    state.players.map((player) => player.player_id || player.id),
  );
  const availablePlayers = state.overviewPlayers.filter(
    (player) => !currentIds.has(player.player_id),
  );

  if (!availablePlayers.length) {
    showNotice(t("notice.noAvailablePlayers"), "info");
    return;
  }

  const values = await openModal({
    title: t("modal.addPlayerTitle"),
    description: t("modal.addPlayerDescription"),
    confirmText: t("modal.addToSession"),
    fields: [
      {
        name: "player_id",
        label: t("session.player"),
        type: "select",
        options: availablePlayers.map((player) => ({
          value: player.player_id,
          label: player.player_name || player.player_id,
        })),
      },
      {
        name: "chips",
        label: t("modal.initialBuyIn"),
        type: "number",
        min: "1",
        placeholder: t("session.chips"),
      },
    ],
  });
  if (!values) return;

  const chips = Number(values.chips);
  if (!values.player_id || !Number.isFinite(chips) || chips <= 0) {
    showNotice(t("notice.choosePlayerAndBuyIn"), "error");
    return;
  }

  const res = await buyIn({
    sessionId: state.activeSessionId,
    playerId: values.player_id,
    chips,
  });
  if (!res.ok) {
    showNotice(describeError(res, t("error.failedAddPlayer")), "error");
    return;
  }

  await refreshSessionData();
  showNotice(
    t("notice.playerAdded", { name: findPlayerName(values.player_id) }),
    "success",
  );
}

async function confirmAddNewPlayer() {
  const values = await openModal({
    title: t("modal.createNewPlayerTitle"),
    description: t("modal.createNewPlayerDescription"),
    confirmText: t("modal.createAndAdd"),
    fields: [
      {
        name: "name",
        label: t("lobby.playerName"),
        type: "text",
        placeholder: t("lobby.playerNamePlaceholder"),
      },
      {
        name: "chips",
        label: t("modal.initialBuyIn"),
        type: "number",
        min: "1",
        placeholder: t("session.chips"),
      },
    ],
  });
  if (!values) return;

  const name = (values.name || "").trim();
  const chips = Number(values.chips);
  if (!name || !Number.isFinite(chips) || chips <= 0) {
    showNotice(t("notice.enterPlayerAndBuyIn"), "error");
    return;
  }

  const createRes = await createPlayer(name);
  if (!createRes.ok || !createRes.body?.player_id) {
    showNotice(describeError(createRes, t("error.failedCreatePlayer")), "error");
    return;
  }

  const buyInRes = await buyIn({
    sessionId: state.activeSessionId,
    playerId: createRes.body.player_id,
    chips,
  });
  if (!buyInRes.ok) {
    showNotice(describeError(buyInRes, t("error.failedCreateAdd")), "error");
    return;
  }

  await Promise.all([refreshSessionData(), loadPlayersOverview()]);
  showNotice(t("notice.playerCreatedAndAdded", { name }), "success");
}

async function confirmFinishSession() {
  if (state.session?.status !== "active") return;
  if ((Number(state.session.totalChips) || 0) > 0) {
    showNotice(
      t("notice.cannotFinish", {
        chips: formatNumber(state.session.totalChips),
      }),
      "error",
    );
    return;
  }

  const values = await openModal({
    title: t("modal.finishTitle"),
    description: t("modal.finishDescription"),
    confirmText: t("session.finish"),
  });
  if (!values) return;

  const res = await finishSession({ sessionId: state.activeSessionId });
  if (!res.ok) {
    showNotice(describeError(res, t("error.failedFinish")), "error");
    return;
  }

  await refreshSessionData();
  showNotice(t("notice.sessionFinished"), "success");
}

async function confirmDebugDeleteSession() {
  if (!state.debugMode || !state.activeSessionId) return;

  const confirmed = await openModal({
    title: t("modal.deleteSessionTitle"),
    description: t("modal.deleteSessionDescription"),
    confirmText: t("debug.deleteSession"),
  });
  if (!confirmed) return;

  const res = await debugDeleteSession(state.activeSessionId);
  if (!res.ok) {
    showNotice(describeError(res, t("error.failedDeleteSession")), "error");
    return;
  }

  state.activeSessionId = "";
  state.session = null;
  state.players = [];
  state.operations = [];
  await Promise.all([loadSessions(), loadPlayersOverview()]);
  setScreen("lobby");
  pushRoute(routeToHome());
  showNotice(t("notice.sessionDeleted"), "success");
}

function bindDebugSessionConfigEditor(card) {
  if (!card) return;
  const freshCard = card.cloneNode(true);
  card.replaceWith(freshCard);
  if (!state.debugMode) return;

  const openEditor = async () => {
    await confirmDebugUpdateSessionConfig();
  };
  freshCard.addEventListener("click", openEditor);
  freshCard.addEventListener("keydown", async (event) => {
    if (event.key !== "Enter" && event.key !== " ") return;
    event.preventDefault();
    await openEditor();
  });
}

async function confirmDebugUpdateSessionConfig() {
  if (!state.debugMode || !state.activeSessionId || !state.session) return;

  const values = await openModal({
    title: t("modal.editSessionConfigTitle"),
    confirmText: t("common.save"),
    fields: [
      {
        name: "chip_rate",
        label: t("session.chipRate"),
        type: "number",
        min: "1",
        value: state.session.chipRate,
      },
      {
        name: "big_blind",
        label: t("session.bigBlind"),
        type: "number",
        min: "1",
        value: state.session.bigBlind,
      },
    ],
  });
  if (!values) return;

  const chipRate = Number(values.chip_rate);
  const bigBlind = Number(values.big_blind);
  const currency = "RUB";
  if (!Number.isFinite(chipRate) || chipRate <= 0) {
    showNotice(t("notice.validChipRate"), "error");
    return;
  }
  if (!Number.isFinite(bigBlind) || bigBlind <= 0) {
    showNotice(t("notice.validBigBlind"), "error");
    return;
  }

  const res = await debugUpdateSessionConfig(state.activeSessionId, {
    chipRate,
    bigBlind,
    currency,
  });
  if (!res.ok) {
    showNotice(describeError(res, t("error.failedUpdateSessionConfig")), "error");
    return;
  }

  await refreshSessionData();
  showNotice(t("notice.sessionConfigUpdated"), "success");
}

async function confirmDebugDeleteSessionFinish() {
  if (!state.debugMode || !state.activeSessionId) return;

  const confirmed = await openModal({
    title: t("modal.deleteFinishTitle"),
    description: t("modal.deleteFinishDescription"),
    confirmText: t("debug.deleteFinish"),
  });
  if (!confirmed) return;

  const res = await debugDeleteSessionFinish(state.activeSessionId);
  if (!res.ok) {
    showNotice(describeError(res, t("error.failedDeleteFinish")), "error");
    return;
  }

  await refreshSessionData();
  showNotice(t("notice.finishDeleted"), "success");
}

async function confirmReverse(operationId) {
  const operation = state.operations.find((item) => item.id === operationId);
  if (!operation) return;

  const values = await openModal({
    title: t("modal.reverseTitle"),
    description: t("modal.reverseDescription", {
      type: operationLabel(operation.type),
      name: findPlayerName(operation.player_id),
      chips: formatNumber(operation.chips),
    }),
    confirmText: t("common.reverse"),
  });
  if (!values) return;

  const res = await reverseOperation({ operationId });
  if (!res.ok) {
    showNotice(describeError(res, t("error.failedReverse")), "error");
    return;
  }

  await refreshSessionData();
  showNotice(t("notice.operationReversed"), "success");
}

async function refreshSessionData() {
  const id = state.activeSessionId;
  if (!id) return;

  const res = await getSession(id);
  if (!res.ok || !res.body) {
    showNotice(describeError(res, t("error.failedRefresh")), "error");
    return;
  }

  hydrateSession(res.body);
  renderSession();

  await Promise.all([
    loadPlayers(id),
    loadOperations(id),
    loadSessions(),
    loadPlayersOverview(),
  ]);
  renderActionPlayerOptions();
}

function hydrateSession(raw) {
  state.session = {
    id: raw.session_id,
    status: raw.status,
    chipRate: raw.chip_rate,
    bigBlind: raw.big_blind,
    currency: raw.currency || "RUB",
    createdAt: raw.created_at,
    finishedAt: raw.finished_at,
    totalBuyIn: raw.total_buy_in,
    totalCashOut: raw.total_cash_out,
    totalChips: raw.total_chips,
  };
}

function findPlayerName(playerId) {
  const inSession = state.players.find((player) => {
    const id = player.player_id || player.id;
    return id === playerId;
  });
  if (inSession) {
    return inSession.player_name || inSession.name || playerId;
  }

  const overview = state.overviewPlayers.find((player) => player.player_id === playerId);
  return overview?.player_name || playerId;
}

function totalMoneyIn(session) {
  const chipRate = Number(session.chipRate);
  const totalBuyIn = Number(session.totalBuyIn);
  if (!Number.isFinite(chipRate) || chipRate <= 0 || !Number.isFinite(totalBuyIn)) {
    return 0;
  }
  return totalBuyIn / chipRate;
}
