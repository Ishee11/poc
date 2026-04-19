import { getSessions } from "../api.js";
import { statusLabel, t } from "../i18n.js";
import { state } from "../state.js";
import { escapeHtml, formatDate, formatNumber, setValue } from "../utils.js";

export async function loadSessions() {
  const res = await getSessions();

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

  count.textContent = String(state.overviewSessions.length);

  if (!state.overviewSessions.length) {
    wrap.innerHTML = `<div class="empty-inline">${escapeHtml(t("common.noSessions"))}</div>`;
    return;
  }

  wrap.innerHTML = state.overviewSessions
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
              <span>${escapeHtml(t("common.totalBuyIn"))}: ${formatNumber(session.total_buy_in)}</span>
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
  if (!select) return;

  const current = select.value;
  const activeSessions = state.overviewSessions.filter(
    (session) => session.status === "active",
  );
  const options = [
    `<option value="">${escapeHtml(t("lobby.latestActiveSession"))}</option>`,
    ...activeSessions.map((session) => {
      const id = session.session_id || session.id;
      const label = `${formatDate(session.created_at)} (${statusLabel(session.status || "-")}, BB ${formatNumber(session.big_blind)})`;
      return `<option value="${escapeHtml(id)}">${escapeHtml(label)}</option>`;
    }),
  ];

  select.innerHTML = options.join("");

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

export function firstActiveSessionId() {
  const session = state.overviewSessions.find((item) => item.status === "active");
  return session?.session_id || session?.id || "";
}
