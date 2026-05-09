import {
  buyIn,
  cashOut,
  createPlayer,
  createExpense,
  debugDeleteSessionFinish,
  debugDeleteSession,
  debugUpdateSessionConfig,
  deleteExpense,
  finishSession,
  getExpenses,
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
import { loadPlayers, loadPlayersOverview, renderPlayers } from "./player.js";

export async function openSession(sessionId, { replace = false } = {}) {
  if (!sessionId) return;

  state.activeSessionId = sessionId;
  state.session = null;
  state.operations = [];
  state.expenses = [];
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

  await Promise.all([loadPlayers(sessionId), loadOperations(sessionId), loadExpenses(sessionId)]);
  renderActionPlayerOptions();
  renderExpenseForm();
  renderExpenses();
  renderSettlement();
  setScreen("session");
  if (replace) {
    replaceRoute(routeToSession(sessionId));
  } else {
    pushRoute(routeToSession(sessionId));
  }
}

export async function loadExpenses(sessionId) {
  if (!sessionId) return;

  const res = await getExpenses(sessionId);
  if (!res.ok) {
    console.error("loadExpenses failed:", res.text);
    state.expenses = [];
    renderExpenses();
    renderSettlement();
    return;
  }

  state.expenses = Array.isArray(res.body) ? res.body : [];
  renderExpenses();
  renderSettlement();
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
  renderPlayers();
}

export function renderSession() {
  const session = state.session;
  if (!session) return;

  const titleDate = document.getElementById("workspace-title-date");
  const finishedAt = document.getElementById("workspace-finished-at");
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
  const playerActions = document.getElementById("session-player-actions");
  const playerActionsHint = document.getElementById("session-player-actions-hint");
  const actionsPanel = document.getElementById("session-actions-panel");
  const finishActions = document.getElementById("session-finish-actions");
  const isActive = session.status === "active";
  const onTable = Number(session.totalChips) || 0;

  if (titleDate) {
    titleDate.textContent = formatDate(session.createdAt);
  }
  if (finishedAt) {
    const hasFinishedAt = session.status === "finished" && Boolean(session.finishedAt);
    finishedAt.hidden = !hasFinishedAt;
    finishedAt.textContent = hasFinishedAt ? formatDate(session.finishedAt) : "";
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
              <span>${state.session.finishedAt ? escapeHtml(formatDate(state.session.finishedAt)) : "-"}</span>
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
  renderPlayerSelect(
    "cash-out-player-select",
    state.players.filter((player) => player.in_game),
  );
}

export function renderExpenseForm() {
  const participantsWrap = document.getElementById("expense-participants-wrap");
  const payersWrap = document.getElementById("expense-payers-wrap");
  const panel = document.getElementById("session-expenses-panel");
  if (!participantsWrap || !payersWrap) return;

  const isActiveOrFinished = state.session?.status === "active" || state.session?.status === "finished";
  if (panel) panel.hidden = !isActiveOrFinished;

  const players = state.players || [];
  const participantRows = players
    .map((player) => {
      const id = player.player_id || player.id;
      const name = player.player_name || player.name || id;
      return `
        <label class="expense-check">
          <span class="expense-check-main">
            <input type="checkbox" name="expense-participant" value="${escapeHtml(id)}" checked />
            <span>${escapeHtml(name)}</span>
          </span>
          <strong data-expense-share="${escapeHtml(id)}">${formatMoney(0, state.session?.currency)}</strong>
        </label>
      `;
    })
    .join("");

  const payerRows = players
    .map((player) => {
      const id = player.player_id || player.id;
      const name = player.player_name || player.name || id;
      return `
        <label class="expense-payer-row">
          <span>${escapeHtml(name)}</span>
          <input type="number" min="0" data-expense-payer="${escapeHtml(id)}" placeholder="0" />
        </label>
      `;
    })
    .join("");

  participantsWrap.innerHTML = participantRows || `<div class="empty-inline">${escapeHtml(t("common.noPlayers"))}</div>`;
  payersWrap.innerHTML = payerRows || `<div class="empty-inline">${escapeHtml(t("common.noPlayers"))}</div>`;
  bindExpenseSplitInputs();
  updateExpenseParticipantShares();
}

function bindExpenseSplitInputs() {
  const amountInput = document.getElementById("expense-amount");
  if (amountInput) {
    amountInput.oninput = updateExpenseParticipantShares;
  }

  document.querySelectorAll("[name='expense-participant']").forEach((input) => {
    input.addEventListener("change", updateExpenseParticipantShares);
  });
}

function calculateEqualShares(amount, participants) {
  if (!Number.isFinite(amount) || amount <= 0 || participants.length === 0) {
    return new Map();
  }

  const baseShare = Math.floor(amount / participants.length);
  let remainder = amount - baseShare * participants.length;
  const shares = new Map();
  for (const playerId of participants) {
    const extra = remainder > 0 ? 1 : 0;
    remainder -= extra;
    shares.set(playerId, baseShare + extra);
  }
  return shares;
}

function selectedExpenseParticipants() {
  return Array.from(document.querySelectorAll("[name='expense-participant']:checked"))
    .map((input) => input.value)
    .filter(Boolean);
}

function updateExpenseParticipantShares() {
  const amount = Number(document.getElementById("expense-amount")?.value);
  const shares = calculateEqualShares(amount, selectedExpenseParticipants());

  document.querySelectorAll("[data-expense-share]").forEach((element) => {
    const playerId = element.getAttribute("data-expense-share");
    element.textContent = formatMoney(shares.get(playerId) || 0, state.session?.currency);
  });
}

function fillEqualExpensePayments() {
  const amount = Number(document.getElementById("expense-amount")?.value);
  const shares = calculateEqualShares(amount, selectedExpenseParticipants());

  if (!shares.size) {
    showNotice(t("notice.invalidExpense"), "error");
    return;
  }

  document.querySelectorAll("[data-expense-payer]").forEach((input) => {
    const playerId = input.getAttribute("data-expense-payer");
    const share = shares.get(playerId) || 0;
    input.value = share > 0 ? String(share) : "";
  });
}

export function renderExpenses() {
  const wrap = document.getElementById("expenses-wrap");
  const count = document.getElementById("session-expenses-count");
  if (!wrap || !count) return;

  count.textContent = String(state.expenses.length);
  if (!state.expenses.length) {
    wrap.innerHTML = `<div class="empty-inline">${escapeHtml(t("expenses.empty"))}</div>`;
    return;
  }

  wrap.innerHTML = state.expenses
    .map((expense) => {
      const participants = (expense.participants || []).map(findPlayerName).join(", ");
      const payments = (expense.payments || [])
        .map((payment) => `${findPlayerName(payment.player_id)}: ${formatMoney(payment.amount, state.session?.currency)}`)
        .join(", ");
      return `
        <div class="operation-row expense-row">
          <div class="row-main">
            <div class="row-title">${escapeHtml(expense.title)}</div>
            <div class="inline-stats">
              <span>${escapeHtml(t("expenses.amount"))}: ${formatMoney(expense.amount, state.session?.currency)}</span>
              <span>${escapeHtml(t("expenses.splitBetween"))}: ${escapeHtml(participants || "-")}</span>
              <span>${escapeHtml(t("expenses.paidBy"))}: ${escapeHtml(payments || "-")}</span>
            </div>
          </div>
          <button type="button" class="secondary" data-delete-expense="${escapeHtml(expense.id)}">${escapeHtml(t("common.delete"))}</button>
        </div>
      `;
    })
    .join("");

  wrap.querySelectorAll("[data-delete-expense]").forEach((button) => {
    button.addEventListener("click", async () => {
      const expenseId = button.getAttribute("data-delete-expense");
      if (!expenseId) return;
      await confirmDeleteExpense(expenseId);
    });
  });
}

export function renderSettlement() {
  const wrap = document.getElementById("settlement-wrap");
  if (!wrap) return;

  const balances = settlementBalances();
  const transfers = settlementTransfers(balances);
  const balanceRows = Array.from(balances.entries())
    .sort((a, b) => b[1] - a[1])
    .map(([playerId, amount]) => `
      <div class="settlement-balance-row">
        <span>${escapeHtml(findPlayerName(playerId))}</span>
        <strong class="${amount >= 0 ? "profit-positive" : "profit-negative"}">${formatMoney(amount, state.session?.currency)}</strong>
      </div>
    `)
    .join("");

  const transferRows = transfers
    .map((transfer) => `
      <div class="operation-row settlement-transfer-row">
        <div class="row-main">
          <div class="row-title">${escapeHtml(findPlayerName(transfer.from))} -> ${escapeHtml(findPlayerName(transfer.to))}</div>
          <div class="inline-stats">
            <span>${formatMoney(transfer.amount, state.session?.currency)}</span>
          </div>
        </div>
      </div>
    `)
    .join("");

  wrap.innerHTML = `
    <div class="settlement-grid">
      <div>
        <h4>${escapeHtml(t("settlement.balances"))}</h4>
        ${balanceRows || `<div class="empty-inline">${escapeHtml(t("common.noData"))}</div>`}
      </div>
      <div>
        <h4>${escapeHtml(t("settlement.transfers"))}</h4>
        ${transferRows || `<div class="empty-inline">${escapeHtml(t("settlement.noTransfers"))}</div>`}
      </div>
    </div>
  `;
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
  if (exists) {
    select.value = current;
  } else if (players.length === 1) {
    select.value = players[0].player_id || players[0].id || "";
  } else {
    select.value = "";
  }
}

export function initSessionActions() {
  document.addEventListener("click", async (event) => {
    const button = event.target.closest("button");
    if (!button) return;

    const rebuyPlayerId = button.getAttribute("data-session-rebuy-player");
    if (rebuyPlayerId) {
      await confirmPlayerRebuy(rebuyPlayerId);
      return;
    }

    switch (button.id) {
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
      case "add-expense-btn":
        await confirmAddExpense();
        break;
      case "expense-split-even-btn":
        fillEqualExpensePayments();
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
      case "account-back-home-btn":
        setScreen("lobby");
        pushRoute(routeToHome());
        break;
      case "players-stats-back-home-btn":
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

function sortPlayersByLastActivity(players) {
  return [...players].sort((left, right) => {
    const leftActivity = Date.parse(left.last_activity_at || "") || 0;
    const rightActivity = Date.parse(right.last_activity_at || "") || 0;
    if (leftActivity !== rightActivity) return rightActivity - leftActivity;

    const nameCompare = String(left.player_name || "").localeCompare(
      String(right.player_name || ""),
      undefined,
      { sensitivity: "base" },
    );
    if (nameCompare !== 0) return nameCompare;

    return String(left.player_id || "").localeCompare(String(right.player_id || ""));
  });
}

async function confirmPlayerRebuy(playerId) {
  const player = state.players.find((item) => (item.player_id || item.id) === playerId);
  if (!playerId || !player) {
    showNotice(t("notice.selectPlayerAndChips"), "error");
    return;
  }

  const playerName = findPlayerName(playerId);
  const chips = lastBuyInChipsForRebuy(playerId);
  const values = await openModal({
    title: t("modal.confirmBuyInTitle"),
    description: t("modal.confirmBuyInDescription", {
      chips: formatNumber(chips),
      name: playerName,
    }),
    confirmText: t("session.buyIn"),
    fields: [
      {
        name: "chips",
        label: t("session.chips"),
        type: "number",
        min: "1",
        value: chips > 0 ? String(chips) : "",
        placeholder: t("session.chips"),
      },
    ],
  });
  if (!values) return;

  const nextChips = Number(values.chips);
  if (!Number.isFinite(nextChips) || nextChips <= 0) {
    showNotice(t("notice.selectPlayerAndChips"), "error");
    return;
  }

  const res = await buyIn({
    sessionId: state.activeSessionId,
    playerId,
    chips: nextChips,
  });
  if (!res.ok) {
    showNotice(describeError(res, t("error.failedBuyIn")), "error");
    return;
  }

  await refreshSessionData();
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

  const inGameIds = new Set(
    state.players
      .filter((player) => player.in_game)
      .map((player) => player.player_id || player.id),
  );
  const availablePlayers = sortPlayersByLastActivity(
    state.overviewPlayersAll.filter((player) => !inGameIds.has(player.player_id)),
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
        value: lastBuyInChipsForRebuy("") || "",
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
  state.expenses = [];
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

async function confirmAddExpense() {
  const title = (document.getElementById("expense-title")?.value || "").trim();
  const amount = Number(document.getElementById("expense-amount")?.value);
  const participants = selectedExpenseParticipants();
  const payments = Array.from(document.querySelectorAll("[data-expense-payer]"))
    .map((input) => ({
      player_id: input.getAttribute("data-expense-payer"),
      amount: Number(input.value),
    }))
    .filter((payment) => payment.player_id && Number.isFinite(payment.amount) && payment.amount > 0);
  const paidTotal = payments.reduce((sum, payment) => sum + payment.amount, 0);

  if (!title || !Number.isFinite(amount) || amount <= 0 || participants.length === 0 || paidTotal !== amount) {
    showNotice(t("notice.invalidExpense"), "error");
    return;
  }

  const res = await createExpense({
    sessionId: state.activeSessionId,
    title,
    amount,
    participants,
    payments,
  });
  if (!res.ok) {
    showNotice(describeError(res, t("error.failedExpense")), "error");
    return;
  }

  document.getElementById("expense-title").value = "";
  document.getElementById("expense-amount").value = "";
  document.querySelectorAll("[data-expense-payer]").forEach((input) => {
    input.value = "";
  });
  updateExpenseParticipantShares();
  await loadExpenses(state.activeSessionId);
  showNotice(t("notice.expenseAdded"), "success");
}

async function confirmDeleteExpense(expenseId) {
  const confirmed = await openModal({
    title: t("modal.deleteExpenseTitle"),
    description: t("modal.deleteExpenseDescription"),
    confirmText: t("common.delete"),
  });
  if (!confirmed) return;

  const res = await deleteExpense(expenseId);
  if (!res.ok) {
    showNotice(describeError(res, t("error.failedDeleteExpense")), "error");
    return;
  }

  await loadExpenses(state.activeSessionId);
  showNotice(t("notice.expenseDeleted"), "success");
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
    loadExpenses(id),
    loadSessions(),
    loadPlayersOverview(),
  ]);
  renderActionPlayerOptions();
  renderExpenseForm();
  renderSettlement();
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

  const overview = state.overviewPlayersAll.find((player) => player.player_id === playerId);
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

function settlementBalances() {
  const balances = new Map();
  for (const player of state.players || []) {
    const id = player.player_id || player.id;
    balances.set(id, Number(player.profit_money) || 0);
  }

  for (const expense of state.expenses || []) {
    const participants = expense.participants || [];
    if (!participants.length) continue;

    const shares = calculateEqualShares(Number(expense.amount) || 0, participants);
    for (const playerId of participants) {
      balances.set(playerId, (balances.get(playerId) || 0) - (shares.get(playerId) || 0));
    }

    for (const payment of expense.payments || []) {
      const paid = Number(payment.amount) || 0;
      balances.set(payment.player_id, (balances.get(payment.player_id) || 0) + paid);
    }
  }

  return balances;
}

function settlementTransfers(balances) {
  const debtors = [];
  const creditors = [];
  for (const [playerId, amount] of balances.entries()) {
    if (amount < 0) debtors.push({ playerId, amount: -amount });
    if (amount > 0) creditors.push({ playerId, amount });
  }

  debtors.sort((a, b) => b.amount - a.amount);
  creditors.sort((a, b) => b.amount - a.amount);

  const transfers = [];
  let debtorIndex = 0;
  let creditorIndex = 0;
  while (debtorIndex < debtors.length && creditorIndex < creditors.length) {
    const debtor = debtors[debtorIndex];
    const creditor = creditors[creditorIndex];
    const amount = Math.min(debtor.amount, creditor.amount);
    if (amount > 0) {
      transfers.push({ from: debtor.playerId, to: creditor.playerId, amount });
    }
    debtor.amount -= amount;
    creditor.amount -= amount;
    if (debtor.amount === 0) debtorIndex += 1;
    if (creditor.amount === 0) creditorIndex += 1;
  }

  return transfers;
}
