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
  withLoading,
} from "../utils.js";

let sessionsRequestGeneration = 0;

export async function loadSessions() {
  const requestGeneration = ++sessionsRequestGeneration;
  const pageSize = state.overviewSessionsPageSize;
  const page = state.overviewSessionsPage;
  const res = await withLoading("#overview-sessions-wrap", () =>
    getSessions({
      guestPlayerId: state.authUser ? "" : state.guestPlayerId,
      limit: pageSize + 1,
      offset: page * pageSize,
      status: state.overviewSessionsFilter,
    }));

  if (requestGeneration !== sessionsRequestGeneration) return;

  if (!res.ok) {
    console.error("loadSessions failed:", res.text);
    state.overviewSessionPageItems = [];
    state.overviewSessionsHasNextPage = false;
    renderSessions();
    return;
  }

  let sessions;
  if (Array.isArray(res.body)) {
    sessions = res.body;
  } else if (Array.isArray(res.body?.sessions)) {
    sessions = res.body.sessions;
  } else if (Array.isArray(res.body?.items)) {
    sessions = res.body.items;
  } else {
    sessions = [];
  }

  if (!sessions.length && state.overviewSessionsPage > 0) {
    state.overviewSessionsPage -= 1;
    return loadSessions();
  }

  state.overviewSessionsHasNextPage = sessions.length > pageSize;
  state.overviewSessionPageItems = sessions.slice(0, pageSize);
  if (state.overviewSessionsPage === 0 && state.overviewSessionsFilter === "all") {
    state.overviewSessions = state.overviewSessionPageItems;
    syncSelect();
  }
  renderSessions();
}

export function renderSessions() {
  const wrap = document.getElementById("overview-sessions-wrap");
  const count = document.getElementById("overview-sessions-count");
  if (!wrap || !count) return;

  const sessions = state.overviewSessionPageItems;
  const pageStart = state.overviewSessionsPage * state.overviewSessionsPageSize;

  count.textContent = sessions.length
    ? `${pageStart + 1}–${pageStart + sessions.length}`
    : "0";
  renderSessionsPagination();

  if (!sessions.length) {
    wrap.innerHTML = `<div class="empty-inline">${escapeHtml(t("common.noSessions"))}</div>`;
    return;
  }

  wrap.innerHTML = sessions
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
  const connectForm = document.getElementById("connect-session-form");
  const emptyState = document.getElementById("lobby-latest-empty");
  const liveBadge = document.getElementById("lobby-latest-live-badge");
  if (!select) return;

  const current = select.value;
  const activeSessions = state.overviewSessions.filter(
    (session) => session.status === "active",
  );
  const latestActiveSession = activeSessions[0] || null;
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
  chips.textContent = formatNumber(chipsOnTable);
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
    state.overviewSessionsPage = 0;
    void loadSessions();
  });

  const pageSize = document.getElementById("overview-sessions-page-size");
  const previous = document.getElementById("overview-sessions-prev");
  const next = document.getElementById("overview-sessions-next");

  pageSize?.addEventListener("change", () => {
    const value = Number(pageSize.value);
    if (![20, 50, 100].includes(value)) return;
    state.overviewSessionsPageSize = value;
    state.overviewSessionsPage = 0;
    void loadSessions();
  });
  previous?.addEventListener("click", () => {
    if (state.overviewSessionsPage <= 0) return;
    state.overviewSessionsPage -= 1;
    void loadSessions();
  });
  next?.addEventListener("click", () => {
    if (!state.overviewSessionsHasNextPage) return;
    state.overviewSessionsPage += 1;
    void loadSessions();
  });
}

function renderSessionsPagination() {
  const pageSize = document.getElementById("overview-sessions-page-size");
  const previous = document.getElementById("overview-sessions-prev");
  const next = document.getElementById("overview-sessions-next");
  const label = document.getElementById("overview-sessions-page-label");

  if (pageSize) pageSize.value = String(state.overviewSessionsPageSize);
  if (previous) previous.disabled = state.overviewSessionsPage === 0;
  if (next) next.disabled = !state.overviewSessionsHasNextPage;
  if (label) {
    label.textContent = t("lobby.sessionsPage", {
      page: state.overviewSessionsPage + 1,
    });
  }
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
