import {
  buyIn,
  cashOut,
  closeExpenses,
  createPlayer,
  createExpense,
  adminDeleteSessionFinish,
  adminDeleteSession,
  adminUpdateSessionConfig,
  deleteExpense,
  finishSession,
  getCurrentUser,
  getExpenses,
  getSettlementTransfers,
  getSession,
  getSessionOperations,
  reverseOperation,
  saveSettlementTransfers,
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
  routeToSessionResults,
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
  state.settlementEditing = false;
  state.expenseFormOpen = false;
  delete state.settlementDrafts[sessionId];

  const res = await getSession(sessionId);
  if (!res.ok || !res.body) {
    showNotice(describeError(res, t("error.failedLoadSession")), "error");
    return;
  }

  hydrateSession(res.body);
  renderSession();
  renderActionPlayerOptions();
  renderExpenseForm();
  renderExpenses();
  renderSettlement();
  setScreen("session");

  await Promise.all([
    loadPlayers(sessionId),
    loadOperations(sessionId),
    loadExpenses(sessionId),
    loadSettlementTransfers(sessionId),
  ]);

  renderSession();
  renderActionPlayerOptions();
  renderExpenseForm();
  renderExpenses();
  renderSettlement();

  if (replace) {
    replaceRoute(routeToSession(sessionId));
  } else {
    pushRoute(routeToSession(sessionId));
  }
}

export async function openSessionResults(sessionId, { replace = false } = {}) {
  if (!sessionId) return;

  await openSession(sessionId, { replace: true });
  setScreen("results");
  const brandTitle = document.querySelector(".app-brand h1");
  if (brandTitle) brandTitle.textContent = t("results.title");
  renderResultsSummary();
  if (replace) {
    replaceRoute(routeToSessionResults(sessionId));
  } else {
    pushRoute(routeToSessionResults(sessionId));
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

async function loadSettlementTransfers(sessionId) {
  if (!sessionId) return;

  const res = await getSettlementTransfers(sessionId);
  if (!res.ok) {
    console.error("loadSettlementTransfers failed:", res.text);
    delete state.settlementDrafts[sessionId];
    renderSettlement();
    return;
  }

  const transfers = Array.isArray(res.body)
    ? res.body.map(normalizeSettlementTransfer).filter(Boolean)
    : [];
  if (transfers.length) {
    state.settlementDrafts[sessionId] = { transfers };
  } else {
    delete state.settlementDrafts[sessionId];
  }
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
  const titleTime = document.getElementById("workspace-title-time");
  const finishedDate = document.getElementById("workspace-finished-date");
  const finishedTime = document.getElementById("workspace-finished-time");
  const finishedRow = document.getElementById("workspace-finished-row");
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
  const adminDeletePanel = document.getElementById("session-delete-admin-panel");
  const resultsButton = document.getElementById("session-open-results-btn");
  const playerActions = document.getElementById("session-player-actions");
  const playerActionsHint = document.getElementById("session-player-actions-hint");
  const playerActionSwitch = document.querySelector(".session-action-switch");
  const finishActions = document.getElementById("session-finish-actions");
  const brandTitle = document.querySelector(".app-brand h1");
  const isActive = session.status === "active";
  const onTable = Number(session.totalChips) || 0;

  if (titleDate) {
    titleDate.textContent = formatSessionDate(session.createdAt);
  }
  if (titleTime) {
    titleTime.textContent = formatSessionTime(session.createdAt);
  }
  if (brandTitle) {
    brandTitle.textContent = session.status === "finished"
      ? t("session.headerFinished")
      : t("session.headerActive");
  }
  const hasFinishedAt = session.status === "finished" && Boolean(session.finishedAt);
  if (finishedDate) {
    finishedDate.textContent = hasFinishedAt ? formatSessionDate(session.finishedAt) : "";
  }
  if (finishedTime) {
    finishedTime.textContent = hasFinishedAt ? formatSessionTime(session.finishedAt) : "";
  }
  if (finishedRow) {
    finishedRow.hidden = !hasFinishedAt;
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
        state.adminMode && session.status === "finished"
          ? `<button type="button" class="secondary status-admin-action" id="admin-reopen-session-btn">${escapeHtml(t("admin.deleteFinish"))}</button>`
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
    card.classList.toggle("admin-editable-stat", state.adminMode);
    card.setAttribute("tabindex", state.adminMode ? "0" : "-1");
    card.setAttribute("role", state.adminMode ? "button" : "presentation");
    card.setAttribute("title", state.adminMode ? t("admin.editSessionConfig") : "");
  });
  if (finishButton) finishButton.disabled = !isActive;
  if (finishActions) finishActions.hidden = !isActive;
  if (finishHint) {
    finishHint.hidden = true;
    finishHint.textContent = "";
  }
  if (playerActions) playerActions.hidden = !isActive;
  if (playerActionsHint) playerActionsHint.hidden = !isActive;
  if (playerActionSwitch) playerActionSwitch.hidden = !isActive;
  if (moneyPanel) moneyPanel.hidden = session.status !== "finished";
  if (adminDeletePanel) adminDeletePanel.hidden = !state.adminMode;
  if (resultsButton) resultsButton.hidden = !state.session;

  document
    .getElementById("admin-reopen-session-btn")
    ?.addEventListener("click", async () => {
      await confirmAdminDeleteSessionFinish();
    });

  bindAdminSessionConfigEditor(chipRateCard);
  bindAdminSessionConfigEditor(bigBlindCard);
  renderSessionActionMode();
  renderResultsSummary();
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

function renderSessionActionMode() {
  const mode = state.sessionPlayerActionMode || "rebuy";
  document.querySelectorAll("[data-session-action-mode]").forEach((button) => {
    const isActive = button.getAttribute("data-session-action-mode") === mode;
    button.classList.toggle("active", isActive);
    button.setAttribute("aria-pressed", isActive ? "true" : "false");
  });
}

export function renderExpenseForm() {
  const participantsWrap = document.getElementById("expense-participants-wrap");
  const payersWrap = document.getElementById("expense-payers-wrap");
  const panel = document.getElementById("session-expenses-panel");
  const form = document.getElementById("session-expense-form");
  const openFormButton = document.getElementById("open-expense-form-btn");
  const closeButton = document.getElementById("close-expenses-btn");
  const lockedHint = document.getElementById("session-expenses-locked-hint");
  const status = document.getElementById("session-expenses-status");
  if (!participantsWrap || !payersWrap) return;

  const isActiveOrFinished = state.session?.status === "active" || state.session?.status === "finished";
  if (panel) panel.hidden = !isActiveOrFinished;
  const expensesClosed = Boolean(state.session?.expensesClosed);
  const canEditExpenses = !expensesClosed || state.adminMode;
  if (form) form.hidden = !canEditExpenses || !state.expenseFormOpen;
  if (openFormButton) openFormButton.hidden = !canEditExpenses || state.expenseFormOpen;
  if (closeButton) closeButton.hidden = !isActiveOrFinished || expensesClosed;
  if (lockedHint) lockedHint.hidden = canEditExpenses || !expensesClosed;
  if (status) {
    status.textContent = expensesClosed ? t("expenses.closed") : "";
    status.hidden = !expensesClosed;
  }

  const players = state.players || [];
  const amount = Number(document.getElementById("expense-amount")?.value);
  const selectedPayerIds = readSelectedExpensePayers(players);

  participantsWrap.innerHTML = renderExpenseParticipantControls(players, amount);
  payersWrap.innerHTML = renderExpensePayerControls(players, selectedPayerIds, amount);
  bindExpenseSplitInputs();
  updateExpenseParticipantShares();
}

function bindExpenseSplitInputs() {
  const amountInput = document.getElementById("expense-amount");
  if (amountInput) {
    amountInput.oninput = () => {
      updateExpenseParticipantShares();
      updateExpensePayerShares();
    };
  }

  document.querySelectorAll("[name='expense-participant']").forEach((input) => {
    input.addEventListener("change", updateExpenseParticipantShares);
  });
}

function renderExpenseParticipantControls(players, amount) {
  if (!players.length) {
    return `<div class="empty-inline">${escapeHtml(t("common.noPlayers"))}</div>`;
  }

  const mode = state.expenseParticipantMode || "all";
  const modeSwitch = `
    <div class="expense-participant-mode" role="group" aria-label="${escapeHtml(t("expenses.splitBetween"))}">
      <button type="button" data-expense-participant-mode="all" class="${mode === "all" ? "active" : ""}" aria-pressed="${mode === "all" ? "true" : "false"}">${escapeHtml(t("expenses.selectAll"))}</button>
      <button type="button" data-expense-participant-mode="custom" class="${mode === "custom" ? "active" : ""}" aria-pressed="${mode === "custom" ? "true" : "false"}">${escapeHtml(t("expenses.custom"))}</button>
    </div>
  `;

  if (mode !== "custom") {
    return modeSwitch;
  }

  const shares = calculateEqualShares(amount, players.map((player) => player.player_id || player.id));
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
          <strong data-expense-share="${escapeHtml(id)}">${formatMoney(shares.get(id) || 0, state.session?.currency)}</strong>
        </label>
      `;
    })
    .join("");

  return `
    ${modeSwitch}
    <div class="expense-section-actions expense-participant-actions">
      <button type="button" class="secondary" id="expense-select-all-btn" data-i18n="expenses.selectAllDetailed">${escapeHtml(t("expenses.selectAllDetailed"))}</button>
      <button type="button" class="secondary" id="expense-clear-all-btn" data-i18n="expenses.clearAllDetailed">${escapeHtml(t("expenses.clearAllDetailed"))}</button>
    </div>
    <div class="expense-participant-list">
      ${participantRows}
    </div>
  `;
}

function renderExpensePayerControls(players, selectedPayerIds, amount) {
  if (!players.length) {
    return `<div class="empty-inline">${escapeHtml(t("common.noPlayers"))}</div>`;
  }

  const mode = state.expensePayerMode || "even";
  const modeSwitch = `
    <div class="expense-payer-mode" role="group" aria-label="${escapeHtml(t("expenses.paidBy"))}">
      <button type="button" data-expense-payer-mode="even" class="${mode === "even" ? "active" : ""}" aria-pressed="${mode === "even" ? "true" : "false"}">${escapeHtml(t("expenses.splitEven"))}</button>
      <button type="button" data-expense-payer-mode="custom" class="${mode === "custom" ? "active" : ""}" aria-pressed="${mode === "custom" ? "true" : "false"}">${escapeHtml(t("expenses.custom"))}</button>
    </div>
  `;

  if (mode === "custom") {
    return `
      ${modeSwitch}
      <div class="expense-payer-custom-grid">
        ${players
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
          .join("")}
      </div>
    `;
  }

  const effectiveSelectedIds = selectedPayerIds;
  const selectedSet = new Set(effectiveSelectedIds);
  const rows = [...effectiveSelectedIds, ""]
    .slice(0, players.length)
    .map((selectedId) => renderExpensePayerSelectRow(players, selectedId, selectedSet, amount))
    .join("");

  return `
    ${modeSwitch}
    <div class="expense-payer-select-list">
      ${rows}
    </div>
  `;
}

function renderExpensePayerSelectRow(players, selectedId, selectedSet, amount) {
  const selectedIds = Array.from(selectedSet).filter(Boolean);
  const shares = calculateEqualShares(Number(amount), selectedIds);
  const options = [
    `<option value="">${escapeHtml(t("session.selectPlayer"))}</option>`,
    ...players
      .filter((player) => {
        const id = player.player_id || player.id;
        return id === selectedId || !selectedSet.has(id);
      })
      .map((player) => {
        const id = player.player_id || player.id;
        const name = player.player_name || player.name || id;
        return `<option value="${escapeHtml(id)}" ${id === selectedId ? "selected" : ""}>${escapeHtml(name)}</option>`;
      }),
  ].join("");

  return `
    <label class="expense-payer-select-row">
      <select class="expense-payer-select" data-expense-payer-select>
        ${options}
      </select>
      <strong data-expense-payer-share="${escapeHtml(selectedId)}">${selectedId ? formatMoney(shares.get(selectedId) || 0, state.session?.currency) : formatMoney(0, state.session?.currency)}</strong>
    </label>
  `;
}

function readSelectedExpensePayers(players) {
  const playerIds = new Set(players.map((player) => player.player_id || player.id));
  const selected = Array.from(document.querySelectorAll("[data-expense-payer-select]"))
    .map((select) => select.value)
    .filter((id, index, ids) => id && playerIds.has(id) && ids.indexOf(id) === index);
  return selected;
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
  if ((state.expenseParticipantMode || "all") === "all") {
    return (state.players || []).map((player) => player.player_id || player.id).filter(Boolean);
  }
  return Array.from(document.querySelectorAll("[name='expense-participant']:checked"))
    .map((input) => input.value)
    .filter(Boolean);
}

function setExpenseParticipantsChecked(checked) {
  document.querySelectorAll("[name='expense-participant']").forEach((input) => {
    input.checked = checked;
  });
  updateExpenseParticipantShares();
}

function updateExpenseParticipantShares() {
  const amount = Number(document.getElementById("expense-amount")?.value);
  const shares = calculateEqualShares(amount, selectedExpenseParticipants());

  document.querySelectorAll("[data-expense-share]").forEach((element) => {
    const playerId = element.getAttribute("data-expense-share");
    element.textContent = formatMoney(shares.get(playerId) || 0, state.session?.currency);
  });
}

function updateExpensePayerShares() {
  const amount = Number(document.getElementById("expense-amount")?.value);
  const payerIds = Array.from(document.querySelectorAll("[data-expense-payer-select]"))
    .map((select) => select.value)
    .filter((id, index, ids) => id && ids.indexOf(id) === index);
  const shares = calculateEqualShares(amount, payerIds);

  document.querySelectorAll("[data-expense-payer-share]").forEach((element) => {
    const playerId = element.getAttribute("data-expense-payer-share");
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

function collectExpensePayments(amount) {
  if ((state.expensePayerMode || "even") === "custom") {
    return Array.from(document.querySelectorAll("[data-expense-payer]"))
      .map((input) => ({
        player_id: input.getAttribute("data-expense-payer"),
        amount: Number(input.value),
      }))
      .filter((payment) => payment.player_id && Number.isFinite(payment.amount) && payment.amount > 0);
  }

  const payerIds = Array.from(document.querySelectorAll("[data-expense-payer-select]"))
    .map((select) => select.value)
    .filter((id, index, ids) => id && ids.indexOf(id) === index);
  return Array.from(calculateEqualShares(amount, payerIds).entries()).map(([player_id, paymentAmount]) => ({
    player_id,
    amount: paymentAmount,
  }));
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

  const canEditExpenses = !state.session?.expensesClosed || state.adminMode;
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
          ${
            canEditExpenses
              ? `<button type="button" class="secondary" data-delete-expense="${escapeHtml(expense.id)}">${escapeHtml(t("common.delete"))}</button>`
              : ""
          }
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
  const autoTransfers = settlementTransfers(balances);
  const draft = currentSettlementDraft();
  const manualTransfers = draft?.transfers || [];
  const hasAdjustments = manualTransfers.length > 0;
  const isEditing = state.settlementEditing;
  const adjustedBalances = hasAdjustments ? balancesAfterSettlementTransfers(balances, manualTransfers) : balances;
  const remainingTransfers = hasAdjustments ? settlementTransfers(adjustedBalances) : autoTransfers;
  const balanceRows = Array.from(balances.entries())
    .sort((a, b) => b[1] - a[1])
    .map(([playerId, amount]) => `
      <div class="settlement-balance-row">
        <span>${escapeHtml(findPlayerName(playerId))}</span>
        <strong class="${amount >= 0 ? "profit-positive" : "profit-negative"}">${formatMoney(amount, state.session?.currency)}</strong>
      </div>
    `)
    .join("");

  const transferRows = renderAutoSettlementTransfers(remainingTransfers);
  const fixedTransferRows = isEditing
    ? renderManualSettlementTransfers(manualTransfers)
    : renderAutoSettlementTransfers(manualTransfers);
  const addTransferRow = isEditing ? renderManualSettlementAddRow() : "";

  wrap.innerHTML = `
    <div class="settlement-grid">
      <div>
        <h4>${escapeHtml(t("settlement.balances"))}</h4>
        ${balanceRows || `<div class="empty-inline">${escapeHtml(t("common.noData"))}</div>`}
      </div>
      <div>
        <div class="settlement-heading-row">
          <h4>${escapeHtml(t("settlement.transfers"))}</h4>
          <span class="settlement-mode-pill" title="${escapeHtml(t(hasAdjustments ? "settlement.adjustedMode" : "settlement.autoMode"))}">
            ${
              hasAdjustments
                ? escapeHtml(t("settlement.adjustedMode"))
                : `<img src="/static/svg/12-users-outline-gold.svg" alt="" aria-hidden="true" /><span>${escapeHtml(t("settlement.autoMode"))}</span>`
            }
          </span>
        </div>
        ${
          isEditing || hasAdjustments
            ? `
              <div class="settlement-subtitle">${escapeHtml(t("settlement.fixedTransfers"))}</div>
              ${fixedTransferRows || `<div class="empty-inline">${escapeHtml(t("settlement.noFixedTransfers"))}</div>`}
              ${addTransferRow}
              <div class="settlement-subtitle settlement-remaining-title">${escapeHtml(t("settlement.remainingTransfers"))}</div>
            `
            : ""
        }
        ${transferRows || `<div class="empty-inline">${escapeHtml(t("settlement.noTransfers"))}</div>`}
        <div class="actions settlement-actions">
          ${
            isEditing
              ? `
                <button type="button" class="secondary" id="settlement-done-btn">${escapeHtml(t("settlement.done"))}</button>
                <button type="button" class="secondary" id="settlement-reset-auto-btn">${escapeHtml(t("settlement.resetAuto"))}</button>
              `
              : `<button type="button" class="secondary" id="settlement-edit-btn">${escapeHtml(t("settlement.edit"))}</button>`
          }
        </div>
        ${isEditing ? `<div class="hint">${escapeHtml(t("settlement.manualHint"))}</div>` : ""}
      </div>
    </div>
  `;

  bindManualSettlementControls();
}

function renderResultsSummary() {
  const chips = document.getElementById("results-total-chips");
  const buyIn = document.getElementById("results-total-buy-in");
  const cashOut = document.getElementById("results-total-cash-out");
  if (!chips && !buyIn && !cashOut) return;

  if (chips) chips.textContent = formatNumber(state.session?.totalChips);
  if (buyIn) buyIn.textContent = formatNumber(state.session?.totalBuyIn);
  if (cashOut) cashOut.textContent = formatNumber(state.session?.totalCashOut);
}

async function persistSettlementDraft() {
  const sessionId = state.activeSessionId;
  if (!sessionId) return;

  const draft = currentSettlementDraft();
  const transfers = draft?.transfers || [];
  const res = await saveSettlementTransfers(sessionId, transfers);
  if (!res.ok) {
    showNotice(describeError(res, t("error.failedSaveSettlementTransfers")), "error");
    return;
  }

  const saved = Array.isArray(res.body)
    ? res.body.map(normalizeSettlementTransfer).filter(Boolean)
    : [];
  if (draft || saved.length) {
    state.settlementDrafts[sessionId] = { transfers: saved };
  }
}

function currentSettlementDraft() {
  return state.activeSessionId ? state.settlementDrafts[state.activeSessionId] : null;
}

function normalizeSettlementTransfer(transfer) {
  const from = String(transfer?.from || "");
  const to = String(transfer?.to || "");
  const amount = Number(transfer?.amount);
  const id = String(transfer?.id || `settlement-${Date.now()}-${Math.random().toString(36).slice(2)}`);
  if (!from || !to || !Number.isFinite(amount) || amount <= 0) return null;
  return { id, from, to, amount };
}

async function setSettlementDraft(transfers) {
  if (!state.activeSessionId) return;
  state.settlementDrafts[state.activeSessionId] = {
    transfers: transfers.map(normalizeSettlementTransfer).filter(Boolean),
  };
  await persistSettlementDraft();
}

function renderAutoSettlementTransfers(transfers) {
  return transfers
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
}

function renderManualSettlementTransfers(transfers) {
  return transfers
    .map((transfer) => `
      <div class="settlement-manual-row" data-settlement-transfer="${escapeHtml(transfer.id)}">
        <select data-settlement-field="from" aria-label="${escapeHtml(t("settlement.from"))}">
          ${renderSettlementPlayerOptions(transfer.from)}
        </select>
        <select data-settlement-field="to" aria-label="${escapeHtml(t("settlement.to"))}">
          ${renderSettlementPlayerOptions(transfer.to)}
        </select>
        <input type="number" min="0" step="1" data-settlement-field="amount" value="${escapeHtml(String(transfer.amount))}" aria-label="${escapeHtml(t("settlement.amount"))}" />
        <button type="button" class="secondary settlement-remove-btn" data-delete-settlement-transfer="${escapeHtml(transfer.id)}" aria-label="${escapeHtml(t("settlement.removeTransfer"))}">&times;</button>
      </div>
    `)
    .join("");
}

function renderManualSettlementAddRow() {
  return `
    <div class="settlement-manual-row settlement-add-row">
      <select id="settlement-add-from" aria-label="${escapeHtml(t("settlement.from"))}">
        ${renderSettlementPlayerOptions("")}
      </select>
      <select id="settlement-add-to" aria-label="${escapeHtml(t("settlement.to"))}">
        ${renderSettlementPlayerOptions("")}
      </select>
      <input type="number" min="0" step="1" id="settlement-add-amount" placeholder="${escapeHtml(t("settlement.amount"))}" aria-label="${escapeHtml(t("settlement.amount"))}" />
      <button type="button" id="settlement-add-transfer-btn">${escapeHtml(t("settlement.addTransfer"))}</button>
    </div>
  `;
}

function renderSettlementPlayerOptions(selectedId) {
  const options = [`<option value="">${escapeHtml(t("session.selectPlayer"))}</option>`];
  for (const player of state.players || []) {
    const id = player.player_id || player.id;
    const name = player.player_name || player.name || id;
    options.push(
      `<option value="${escapeHtml(id)}" ${id === selectedId ? "selected" : ""}>${escapeHtml(name)}</option>`,
    );
  }
  return options.join("");
}

function bindManualSettlementControls() {
  document.querySelectorAll("[data-settlement-transfer] [data-settlement-field]").forEach((control) => {
    control.addEventListener("change", async () => {
      await updateManualSettlementTransfer(control);
    });
  });
}

async function updateManualSettlementTransfer(control) {
  const row = control.closest("[data-settlement-transfer]");
  const transferId = row?.getAttribute("data-settlement-transfer");
  const field = control.getAttribute("data-settlement-field");
  const draft = currentSettlementDraft();
  if (!row || !transferId || !field || !draft) return;

  const transfer = draft.transfers.find((item) => item.id === transferId);
  if (!transfer) return;

  const nextTransfer = { ...transfer };
  if (field === "amount") {
    const amount = Number(control.value);
    if (!Number.isFinite(amount) || amount <= 0) {
      renderSettlement();
      return;
    }
    nextTransfer.amount = amount;
  } else if (field === "from" || field === "to") {
    nextTransfer[field] = control.value;
  }

  if (!nextTransfer.from || !nextTransfer.to || nextTransfer.from === nextTransfer.to) {
    renderSettlement();
    return;
  }

  draft.transfers = draft.transfers
    .map((item) => (item.id === transferId ? nextTransfer : item))
    .map(normalizeSettlementTransfer)
    .filter(Boolean);
  await persistSettlementDraft();
  renderSettlement();
}

async function enableManualSettlement() {
  state.settlementEditing = true;
  if (!currentSettlementDraft()) {
    await setSettlementDraft([]);
  }
  renderSettlement();
}

function closeManualSettlement() {
  state.settlementEditing = false;
  renderSettlement();
}

async function resetSettlementToAuto() {
  if (!state.activeSessionId) return;
  delete state.settlementDrafts[state.activeSessionId];
  await persistSettlementDraft();
  state.settlementEditing = false;
  renderSettlement();
}

async function addManualSettlementTransfer() {
  const from = document.getElementById("settlement-add-from")?.value || "";
  const to = document.getElementById("settlement-add-to")?.value || "";
  const amount = Number(document.getElementById("settlement-add-amount")?.value);
  const transfer = normalizeSettlementTransfer({ from, to, amount });
  if (!transfer || from === to) return;

  const draft = currentSettlementDraft() || { transfers: [] };
  draft.transfers = [...draft.transfers, transfer];
  state.settlementDrafts[state.activeSessionId] = draft;
  await persistSettlementDraft();
  renderSettlement();
}

async function deleteManualSettlementTransfer(transferId) {
  const draft = currentSettlementDraft();
  if (!draft) return;
  draft.transfers = draft.transfers.filter((transfer) => transfer.id !== transferId);
  await persistSettlementDraft();
  renderSettlement();
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
      await withBusyButton(button, () => confirmPlayerRebuy(rebuyPlayerId));
      return;
    }
    const cashOutPlayerId = button.getAttribute("data-session-cash-out-player");
    if (cashOutPlayerId) {
      await withBusyButton(button, () => confirmPlayerCashOut(cashOutPlayerId));
      return;
    }
    const sessionActionMode = button.getAttribute("data-session-action-mode");
    if (sessionActionMode) {
      state.sessionPlayerActionMode = sessionActionMode;
      renderSessionActionMode();
      renderPlayers();
      return;
    }
    const expensePayerMode = button.getAttribute("data-expense-payer-mode");
    if (expensePayerMode) {
      state.expensePayerMode = expensePayerMode;
      renderExpenseForm();
      return;
    }
    const expenseParticipantMode = button.getAttribute("data-expense-participant-mode");
    if (expenseParticipantMode) {
      state.expenseParticipantMode = expenseParticipantMode;
      renderExpenseForm();
      return;
    }

    const deleteSettlementTransferId = button.getAttribute("data-delete-settlement-transfer");
    if (deleteSettlementTransferId) {
      await withBusyButton(button, () => deleteManualSettlementTransfer(deleteSettlementTransferId));
      return;
    }

    switch (button.id) {
      case "session-add-player-btn":
        await withBusyButton(button, () => confirmAddPlayer());
        break;
      case "session-open-results-btn":
        await withBusyButton(button, () => openSessionResults(state.activeSessionId));
        break;
      case "finish-session-btn":
        await withBusyButton(button, () => confirmFinishSession());
        break;
      case "open-expense-form-btn":
        state.expenseFormOpen = true;
        renderExpenseForm();
        break;
      case "add-expense-btn":
        await withBusyButton(button, () => confirmAddExpense());
        break;
      case "close-expenses-btn":
        await withBusyButton(button, () => confirmCloseExpenses());
        break;
      case "expense-select-all-btn":
        setExpenseParticipantsChecked(true);
        break;
      case "expense-clear-all-btn":
        setExpenseParticipantsChecked(false);
        break;
      case "expense-split-even-btn":
        fillEqualExpensePayments();
        break;
      case "settlement-edit-btn":
        await withBusyButton(button, () => enableManualSettlement());
        break;
      case "settlement-done-btn":
        closeManualSettlement();
        break;
      case "settlement-reset-auto-btn":
        await withBusyButton(button, () => resetSettlementToAuto());
        break;
      case "settlement-add-transfer-btn":
        await withBusyButton(button, () => addManualSettlementTransfer());
        break;
      case "admin-delete-session-btn":
        await withBusyButton(button, () => confirmAdminDeleteSession());
        break;
      case "session-back-home-btn":
        setScreen("lobby");
        pushRoute(routeToHome());
        break;
      case "results-back-session-btn":
        if (state.activeSessionId) {
          renderSession();
          setScreen("session");
          pushRoute(routeToSession(state.activeSessionId));
        }
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

  document.addEventListener("change", (event) => {
    if (event.target.closest("[data-expense-payer-select]")) {
      renderExpenseForm();
    }
  });
}


async function withBusyButton(button, action) {
  if (!button) {
    await action();
    return;
  }
  if (button.disabled || button.dataset.loading === "true") return;

  button.disabled = true;
  button.dataset.loading = "true";
  button.setAttribute("aria-busy", "true");

  try {
    await action();
  } finally {
    button.disabled = false;
    button.dataset.loading = "false";
    button.setAttribute("aria-busy", "false");
  }
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
    confirmClass: "rebuy-action",
    fields: [
      {
        name: "chips",
        label: t("session.chips"),
        type: "number",
        min: "1",
        step: 1000,
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

async function confirmPlayerCashOut(playerId) {
  const playerName = findPlayerName(playerId);
  const values = await openModal({
    title: t("modal.confirmCashOutTitle"),
    description: t("modal.confirmCashOutDescription", {
      chips: formatNumber(defaultCashOutChips(playerId)),
      name: playerName,
    }),
    confirmText: t("session.cashOut"),
    confirmClass: "cash-out-action",
    fields: [
      {
        name: "chips",
        label: t("session.chips"),
        type: "number",
        min: "1",
        step: 1000,
        value: defaultCashOutChips(playerId) || "",
        placeholder: t("session.chips"),
      },
    ],
  });
  if (!values) return;

  const chips = Number(values.chips);
  if (!playerId || !Number.isFinite(chips) || chips <= 0) {
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

function defaultCashOutChips(playerId) {
  const player = state.players.find((item) => (item.player_id || item.id) === playerId);
  if (!player) return "";

  const chips = (Number(player.buy_in) || 0) - (Number(player.cash_out) || 0);
  return chips > 0 ? String(chips) : "";
}

async function confirmAddPlayer() {
  await loadPlayersOverview();

  const inGameIds = new Set(
    state.players
      .filter((player) => player.in_game)
      .map((player) => player.player_id || player.id),
  );
  const availablePlayers = sortPlayersByLastActivity(
    state.overviewPlayersAll.filter((player) => !inGameIds.has(player.player_id)),
  );

  const values = await openModal({
    title: t("modal.addPlayerTitle"),
    description: t("modal.addPlayerDescription"),
    confirmText: t("modal.addToSession"),
    confirmClass: "rebuy-action",
    fields: [
      {
        name: "player_id",
        label: t("session.player"),
        type: "select",
        value: availablePlayers[0]?.player_id || "__new__",
        options: [
          { value: "__new__", label: t("session.newPlayerOption") },
          ...availablePlayers.map((player) => ({
            value: player.player_id,
            label: player.player_name || player.player_id,
          })),
        ],
      },
      {
        name: "name",
        label: t("session.newPlayerName"),
        type: "text",
        placeholder: t("lobby.playerNamePlaceholder"),
        showWhen: { name: "player_id", value: "__new__" },
      },
      {
        name: "chips",
        label: t("modal.initialBuyIn"),
        type: "number",
        min: "1",
        step: 1000,
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

  if (values.player_id === "__new__") {
    const name = (values.name || "").trim();
    if (!name) {
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
    confirmClass: "results-close-bill-btn",
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

async function confirmAdminDeleteSession() {
  if (!state.adminMode || !state.activeSessionId) return;

  const confirmed = await openModal({
    title: t("modal.deleteSessionTitle"),
    description: t("modal.deleteSessionDescription"),
    confirmText: t("admin.deleteSession"),
    confirmClass: "danger",
  });
  if (!confirmed) return;

  const res = await adminDeleteSession(state.activeSessionId);
  if (!res.ok) {
    if (res.status === 401 || res.status === 403) {
      const me = await getCurrentUser();
      if (!me.ok || me.body?.user?.role !== "admin") {
        state.authUser = null;
        state.adminMode = false;
        showNotice(t("error.adminRequired"), "error");
        renderSession();
        return;
      }
    }

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

function bindAdminSessionConfigEditor(card) {
  if (!card) return;
  const freshCard = card.cloneNode(true);
  card.replaceWith(freshCard);
  if (!state.adminMode) return;

  const openEditor = async () => {
    await confirmAdminUpdateSessionConfig();
  };
  freshCard.addEventListener("click", openEditor);
  freshCard.addEventListener("keydown", async (event) => {
    if (event.key !== "Enter" && event.key !== " ") return;
    event.preventDefault();
    await openEditor();
  });
}

async function confirmAdminUpdateSessionConfig() {
  if (!state.adminMode || !state.activeSessionId || !state.session) return;

  const values = await openModal({
    title: t("modal.editSessionConfigTitle"),
    confirmText: t("common.save"),
    confirmClass: "rebuy-action",
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

  const res = await adminUpdateSessionConfig(state.activeSessionId, {
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

async function confirmAdminDeleteSessionFinish() {
  if (!state.adminMode || !state.activeSessionId) return;

  const confirmed = await openModal({
    title: t("modal.deleteFinishTitle"),
    description: t("modal.deleteFinishDescription"),
    confirmText: t("admin.deleteFinish"),
    confirmClass: "danger",
  });
  if (!confirmed) return;

  const res = await adminDeleteSessionFinish(state.activeSessionId);
  if (!res.ok) {
    if (res.status === 401 || res.status === 403) {
      const me = await getCurrentUser();
      if (!me.ok || me.body?.user?.role !== "admin") {
        state.authUser = null;
        state.adminMode = false;
        showNotice(t("error.adminRequired"), "error");
        renderSession();
        return;
      }
    }

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
    confirmClass: "secondary",
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
  if (state.session?.expensesClosed && !state.adminMode) {
    showNotice(t("notice.expensesClosed"), "error");
    return;
  }

  const title = (document.getElementById("expense-title")?.value || "").trim();
  const amount = Number(document.getElementById("expense-amount")?.value);
  const participants = selectedExpenseParticipants();
  const payments = collectExpensePayments(amount);
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
  state.expenseFormOpen = false;
  renderExpenseForm();
  await loadExpenses(state.activeSessionId);
  showNotice(t("notice.expenseAdded"), "success");
}

async function confirmCloseExpenses() {
  if (!state.activeSessionId || state.session?.expensesClosed) return;

  const confirmed = await openModal({
    title: t("modal.closeExpensesTitle"),
    description: t("modal.closeExpensesDescription"),
    confirmText: t("expenses.closeBill"),
    confirmClass: "results-close-bill-btn",
  });
  if (!confirmed) return;

  const res = await closeExpenses(state.activeSessionId);
  if (!res.ok) {
    showNotice(describeError(res, t("error.failedCloseExpenses")), "error");
    return;
  }

  await refreshSessionData();
  showNotice(t("notice.expensesClosed"), "success");
}

async function confirmDeleteExpense(expenseId) {
  if (state.session?.expensesClosed && !state.adminMode) {
    showNotice(t("notice.expensesClosed"), "error");
    return;
  }

  const confirmed = await openModal({
    title: t("modal.deleteExpenseTitle"),
    description: t("modal.deleteExpenseDescription"),
    confirmText: t("common.delete"),
    confirmClass: "danger",
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
    loadSettlementTransfers(id),
  ]);
  renderActionPlayerOptions();
  renderExpenseForm();
  renderSettlement();

  Promise.allSettled([loadSessions(), loadPlayersOverview()]);
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
    expensesClosed: Boolean(raw.expenses_closed),
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

function formatSessionDate(value) {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return formatDate(value);

  const day = String(date.getDate()).padStart(2, "0");
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const year = date.getFullYear();
  return `${day}.${month}.${year}`;
}

function formatSessionTime(value) {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "-";

  const hours = String(date.getHours()).padStart(2, "0");
  const minutes = String(date.getMinutes()).padStart(2, "0");
  return `${hours}:${minutes}`;
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

function balancesAfterSettlementTransfers(balances, transfers) {
  const next = new Map(balances);
  for (const transfer of transfers || []) {
    const amount = Number(transfer.amount) || 0;
    if (!transfer.from || !transfer.to || amount <= 0) continue;

    next.set(transfer.from, (next.get(transfer.from) || 0) + amount);
    next.set(transfer.to, (next.get(transfer.to) || 0) - amount);
  }
  return next;
}
