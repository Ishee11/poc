import { getSessions } from "../api.js";
import { state } from "../state.js";
import { escapeHtml, formatDate, setValue } from "../utils.js";

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
    wrap.innerHTML = '<div class="empty-inline">No sessions</div>';
    return;
  }

  wrap.innerHTML = state.overviewSessions
    .map((session) => {
      const id = session.session_id || session.id;

      return `
        <div class="session-row">
          <div class="row-main">
            <div class="row-title">${escapeHtml(formatDate(session.created_at))}</div>
          </div>
          <button type="button" data-open-session="${escapeHtml(id)}">Open</button>
        </div>
      `;
    })
    .join("");

  wrap.querySelectorAll("[data-open-session]").forEach((button) => {
    button.addEventListener("click", async () => {
      const sessionId = button.getAttribute("data-open-session");
      if (!sessionId) return;

      setValue("active-session-select", sessionId);
      const { openSession } = await import("./session.js");
      await openSession(sessionId);
    });
  });
}

function syncSelect() {
  const select = document.getElementById("active-session-select");
  if (!select) return;

  const current = select.value;
  const options = [
    '<option value="">Latest active session</option>',
    ...state.overviewSessions.map((session) => {
      const id = session.session_id || session.id;
      const label = formatDate(session.created_at);
      return `<option value="${escapeHtml(id)}">${escapeHtml(label)}</option>`;
    }),
  ];

  select.innerHTML = options.join("");

  const currentExists = state.overviewSessions.some((session) => {
    const id = session.session_id || session.id;
    return id === current;
  });

  if (current && currentExists) {
    select.value = current;
    return;
  }

  if (state.overviewSessions[0]) {
    select.value =
      state.overviewSessions[0].session_id || state.overviewSessions[0].id;
  }
}
