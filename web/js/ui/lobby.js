import { getSessions } from "../api.js";
import { state } from "../state.js";
import { formatDate, escapeHtml, setValue } from "../utils.js";

/**
 * загрузка сессий
 */
export async function loadSessions() {
  const res = await getSessions();

  if (!res.ok) {
    console.error("loadSessions failed:", res.text);
    return;
  }

  console.log("sessions raw:", res.body); // 👈 важно

  // 👉 поддержка разных форматов ответа
  let sessions = [];

  if (Array.isArray(res.body)) {
    sessions = res.body;
  } else if (Array.isArray(res.body?.sessions)) {
    sessions = res.body.sessions;
  } else if (Array.isArray(res.body?.items)) {
    sessions = res.body.items;
  }

  state.overviewSessions = sessions;

  renderSessions();
  syncSelect();
}

/**
 * рендер списка сессий
 */
export function renderSessions() {
  const wrap = document.getElementById("overview-sessions-wrap");
  if (!wrap) return;

  if (!state.overviewSessions.length) {
    wrap.innerHTML = "<div>No sessions</div>";
    return;
  }

  wrap.innerHTML = state.overviewSessions
    .map((s) => {
      const id = s.session_id || s.id;

      return `
        <div class="session-row">
            <button data-open-session="${escapeHtml(id)}">
                Open
            </button>
            ${escapeHtml(id)} — ${formatDate(s.created_at)}
        </div>
      `;
    })
    .join("");

  // 👉 навешиваем события
  wrap.querySelectorAll("[data-open-session]").forEach((btn) => {
    btn.addEventListener("click", async () => {
      const sessionId = btn.getAttribute("data-open-session");

      if (!sessionId) return;

      setValue("active-session-select", sessionId);

      const { openSession } = await import("./session.js");
      await openSession(sessionId);
    });
  });
}

/**
 * синхронизация select
 */
function syncSelect() {
  const select = document.getElementById("active-session-select");
  if (!select) return;

  const current = select.value;

  const options = [
    '<option value="">Latest active session</option>',
    ...state.overviewSessions.map((s) => {
      const id = s.session_id || s.id;

      return `<option value="${escapeHtml(id)}">${escapeHtml(id)}</option>`;
    }),
  ];

  select.innerHTML = options.join("");

  // оставить выбранное если есть
  if (current && state.overviewSessions.some((s) => s.session_id === current)) {
    select.value = current;
    return;
  }

  // иначе выбрать первую
  if (state.overviewSessions.length > 0) {
    select.value =
      state.overviewSessions[0].session_id || state.overviewSessions[0].id;
  }
}
