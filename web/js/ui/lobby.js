import { getSessions } from "../api.js";
import { statusLabel, t } from "../i18n.js";
import { state } from "../state.js";
import {
  escapeHtml,
  formatDate,
  formatMoney,
  formatNumber,
  currencySymbol,
  setValue,
} from "../utils.js";

export async function loadSessions() {
  const res = await getSessions({ guest_player_id: state.authUser ? "" : state.guestPlayerId });

  if (!res.ok) {
    console.error("loadSessions failed:", res.text);
    state.overviewSessions = [];
    renderSessions();
    syncSelect();
    return;
  }

  if (Array.isArray(res.body)) {
    state.overviewSessions = res.body;
  } else if (Array.isArray(res.body?.sessions)) {
    state.overviewSessions = res.body.sessions;
  } else if (Array.isArray(res.body?.items)) {
    state.overviewSessions = res.body.items;
  } else {
    state.overviewSessions = [];
  }

  renderSessions();
  syncSelect();
}

export function renderSessions() {
  const wrap = document.getElementById("overview-sessions-wrap");
  const count = document.getElementById("overview-sessions-count");
  if (!wrap || !count) return;

  const filtered = state.overviewSessionsFilter === "all"
    ? state.overviewSessions
    : state.overviewSessions.filter((s) => s.status === state.overviewSessionsFilter);

  count.textContent = String(state.overviewSessions.length);

  if (!filtered.length) {
    wrap.innerHTML = `<div class="empty-inline">${escapeHtml(t("common.noSessions"))}</div>`;
    return;
  }

  wrap.innerHTML = filtered
    .map((session) => {
      const id = session.session_id || session.id;

      return `
        <div class="session-row clickable-row" data-open-session="${escapeHtml(id)}" tabindex="0" role="button">
          <div class="row-main">
            <div class="row-title">${escapeHtml(formatDate(session.created_at))}</div>
            <div class="inline-stats">
              <span class="status-pill status-${escapeHtml(session.status || "unknown")}">${escapeHtml(statusLabel(session.status || "-"))}</span>
              <span>${escapeHtml(t("session.bigBlindShort"))}: ${formatNumber(session.big_blind)}</span>
              <span>${escapeHtml(t("common.players"))}: ${formatNumber(session.player_count)}</span>
              <span>${escapeHtml(t("common.totalBuyIn"))}: ${escapeHtml(formatBuyInSummary(session))}</span>
            </div>
          </div>
        </div>
      `;
    })
    .join("");

  wrap.querySelectorAll("[data-open-session]").forEach((row) => {
    row.addEventListener("click", async () => openSessionFromRow(row));
    row.addEventListener("keydown", async (event) => {
      if (event.key !== "Enter" && event.key !== " ") return;
      event.preventDefault();
      await openSessionFromRow(row);
    });
  });
}

async function openSessionFromRow(row) {
  const sessionId = row.getAttribute("data-open-session");
  if (!sessionId) return;

  setValue("active-session-select", sessionId);
  const { openSession } = await import("./session.js");
  await openSession(sessionId);
}

export function syncSelect() {
  const select = document.getElementById("active-session-select");
  const connectPanel = document.querySelector(".lobby-connect-panel");
  const connectForm = document.getElementById("connect-session-form");
  const emptyState = document.getElementById("lobby-latest-empty");
  const liveBadge = document.getElementById("lobby-latest-live-badge");
  if (!select) return;

  const current = select.value;
  const activeSessions = state.overviewSessions.filter(
    (session) => session.status === "active",
  );
  const latestActiveSession = activeSessions[0] || null;
  if (connectPanel) {
    connectPanel.hidden = !latestActiveSession;
  }
  if (connectForm) connectForm.hidden = !latestActiveSession;
  if (emptyState) emptyState.hidden = Boolean(latestActiveSession);
  if (liveBadge) liveBadge.hidden = !latestActiveSession;

  const options = activeSessions.map((session) => {
      const id = session.session_id || session.id;
      const chipsOnTable = (Number(session.total_buy_in) || 0) - (Number(session.total_cash_out) || 0);
      const label = t("lobby.sessionOption", {
        date: formatCompactDateTime(session.created_at),
        currencySymbol: currencySymbol(session.currency),
        chipRate: formatNumber(session.chip_rate),
        bigBlind: formatNumber(session.big_blind),
        chips: formatNumber(chipsOnTable),
      });
      return `<option value="${escapeHtml(id)}">${escapeHtml(label)}</option>`;
    });

  select.innerHTML = options.join("");
  renderLatestActiveSession(latestActiveSession);

  const currentExists = activeSessions.some((session) => {
    const id = session.session_id || session.id;
    return id === current;
  });

  if (current && currentExists) {
    select.value = current;
    return;
  }

  if (activeSessions[0]) {
    select.value = activeSessions[0].session_id || activeSessions[0].id;
  }
}

function renderLatestActiveSession(session) {
  const date = document.getElementById("lobby-latest-date");
  const chipRate = document.getElementById("lobby-latest-chip-rate");
  const bigBlind = document.getElementById("lobby-latest-big-blind");
  const chips = document.getElementById("lobby-latest-chips");
  if (!date || !chipRate || !bigBlind || !chips) return;

  if (!session) {
    date.textContent = "";
    chipRate.textContent = "-";
    bigBlind.textContent = "-";
    chips.textContent = "-";
    return;
  }

  const chipsOnTable = (Number(session.total_buy_in) || 0) - (Number(session.total_cash_out) || 0);
  date.textContent = formatLobbyLatestDateTime(session.created_at);
  chipRate.textContent = t("session.chipRateValue", {
    currencySymbol: currencySymbol(session.currency),
    chips: formatNumber(session.chip_rate),
  });
  bigBlind.textContent = formatNumber(session.big_blind);
  chips.textContent = formatNumber(chipsOnTable).replaceAll(",", " ");
}

export function firstActiveSessionId() {
  const session = state.overviewSessions.find((item) => item.status === "active");
  return session?.session_id || session?.id || "";
}

export function initSessionsFilter() {
  const controls = document.getElementById("overview-session-controls");
  if (!controls) return;

  controls.addEventListener("click", (event) => {
    const btn = event.target.closest("[data-session-filter]");
    if (!btn) return;

    controls.querySelectorAll("[data-session-filter]").forEach((el) => {
      el.classList.toggle("is-active", el === btn);
    });

    state.overviewSessionsFilter = btn.getAttribute("data-session-filter");
    renderSessions();
  });
}

export function applyLatestSessionDefaults({ force = false } = {}) {
  const latest = state.overviewSessions[0];
  if (!latest) return;

  setDefaultNumberValue("start-chip-rate", latest.chip_rate, { force });
  setDefaultNumberValue("start-big-blind", latest.big_blind, { force });
}

function setDefaultNumberValue(inputId, value, { force = false } = {}) {
  const input = document.getElementById(inputId);
  if (!input) return;
  if (!force && input.value !== "") return;

  const number = Number(value);
  if (Number.isFinite(number) && number > 0) {
    input.value = String(number);
  }
}

function formatBuyInSummary(session) {
  const chips = Number(session.total_buy_in);
  const chipRate = Number(session.chip_rate);
  const chipsText = `${formatNumber(chips)} ${t("common.chips")}`;

  if (!Number.isFinite(chips) || !Number.isFinite(chipRate) || chipRate <= 0) {
    return chipsText;
  }

  return `${chipsText} · ${formatMoney(chips / chipRate, session.currency)}`;
}

function formatCompactDateTime(value) {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return formatDate(value);

  const day = String(date.getDate()).padStart(2, "0");
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const hours = String(date.getHours()).padStart(2, "0");
  const minutes = String(date.getMinutes()).padStart(2, "0");
  return `${day}.${month} ${hours}:${minutes}`;
}

function formatLobbyLatestDateTime(value) {
  if (!value) return "-";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return formatDate(value);

  const day = String(date.getDate()).padStart(2, "0");
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const year = date.getFullYear();
  const hours = String(date.getHours()).padStart(2, "0");
  const minutes = String(date.getMinutes()).padStart(2, "0");
  return `${day}.${month}.${year}, ${hours}:${minutes}`;
}
