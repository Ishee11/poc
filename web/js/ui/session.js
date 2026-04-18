import {
  buyIn,
  cashOut,
  createPlayer,
  finishSession,
  getSession,
  getSessionOperations,
  reverseOperation,
} from "../api.js";
import { operationLabel, statusLabel, t } from "../i18n.js";
import { state } from "../state.js";
import {
  describeError,
  escapeHtml,
  formatDate,
  formatNumber,
  openModal,
  pushRoute,
  replaceRoute,
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
}

export function renderSession() {
  const session = state.session;
  if (!session) return;

  const subtitle = document.getElementById("workspace-subtitle");
  const chipRate = document.getElementById("stat-chip-rate");
  const buyIn = document.getElementById("stat-buy-in");
  const cashOut = document.getElementById("stat-cash-out");
  const totalChips = document.getElementById("stat-total-chips");
  const totalChipsCard = document.getElementById("stat-total-chips-card");
  const totalMoney = document.getElementById("stat-total-money");
  const moneyPanel = document.getElementById("session-money-panel");
  const status = document.getElementById("workspace-status");
  const finishButton = document.getElementById("finish-session-btn");
  const finishHint = document.getElementById("finish-session-hint");
  const playerActions = document.getElementById("session-player-actions");
  const actionsPanel = document.getElementById("session-actions-panel");
  const finishActions = document.getElementById("session-finish-actions");
  const isActive = session.status === "active";
  const onTable = Number(session.totalChips) || 0;

  if (subtitle) {
    subtitle.textContent = formatDate(session.createdAt);
  }
  if (chipRate) {
    chipRate.textContent = t("session.chipRateValue", {
      chips: formatNumber(session.chipRate),
    });
  }
  if (buyIn) buyIn.textContent = formatNumber(session.totalBuyIn);
  if (cashOut) cashOut.textContent = formatNumber(session.totalCashOut);
  if (totalChips) totalChips.textContent = formatNumber(session.totalChips);
  if (totalMoney) totalMoney.textContent = formatNumber(totalMoneyIn(session));
  if (status) {
    status.textContent = statusLabel(session.status);
    status.className = `session-status ${session.status}`;
  }
  if (totalChipsCard) {
    totalChipsCard.classList.toggle("on-table-warning", isActive && onTable > 0);
    totalChipsCard.classList.toggle("on-table-clear", isActive && onTable === 0);
  }
  if (finishButton) finishButton.disabled = !isActive;
  if (finishActions) finishActions.hidden = !isActive;
  if (finishHint) {
    finishHint.hidden = true;
    finishHint.textContent = "";
  }
  if (playerActions) playerActions.hidden = !isActive;
  if (actionsPanel) actionsPanel.hidden = !isActive;
  if (moneyPanel) moneyPanel.hidden = session.status !== "finished";
}

export function renderOperations() {
  const wrap = document.getElementById("operations-wrap");
  const count = document.getElementById("session-operations-count");
  if (!wrap || !count) return;

  count.textContent = String(state.operations.length);

  if (!state.operations.length) {
    wrap.innerHTML = `<div class="empty-inline">${escapeHtml(t("common.noOperations"))}</div>`;
    return;
  }

  const reversedTargets = new Set(
    state.operations
      .filter((operation) => operation.type === "reversal" && operation.reference_id)
      .map((operation) => operation.reference_id),
  );

  wrap.innerHTML = state.operations
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

  wrap.querySelectorAll("[data-reverse-operation]").forEach((button) => {
    button.addEventListener("click", async () => {
      const operationId = button.getAttribute("data-reverse-operation");
      if (!operationId) return;
      await confirmReverse(operationId);
    });
  });
}

export function renderActionPlayerOptions() {
  const select = document.getElementById("action-player-select");
  if (!select) return;

  const current = select.value;
  const options = [
    `<option value="">${escapeHtml(t("session.selectPlayer"))}</option>`,
    ...state.players.map((player) => {
      const id = player.player_id || player.id;
      const name = player.player_name || player.name || id;
      return `<option value="${escapeHtml(id)}">${escapeHtml(name)}</option>`;
    }),
  ];

  select.innerHTML = options.join("");
  const exists = state.players.some((player) => {
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
      case "session-back-home-btn":
        setScreen("lobby");
        pushRoute("/");
        break;
      case "player-back-home-btn":
        setScreen("lobby");
        pushRoute("/");
        break;
      case "player-back-session-btn":
        if (state.activeSessionId) {
          setScreen("session");
          pushRoute(routeToSession(state.activeSessionId));
        } else {
          setScreen("lobby");
          pushRoute("/");
        }
        break;
      default:
        break;
    }
  });
}

async function confirmBuyIn() {
  const playerId = document.getElementById("action-player-select")?.value;
  const chips = Number(document.getElementById("action-chips")?.value);

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
  document.getElementById("action-chips").value = "";
  showNotice(t("notice.buyInRecorded", { name: playerName }), "success");
}

async function confirmCashOut() {
  const playerId = document.getElementById("action-player-select")?.value;
  const chips = Number(document.getElementById("action-chips")?.value);

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
  document.getElementById("action-chips").value = "";
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
    createdAt: raw.created_at,
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
