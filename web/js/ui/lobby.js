import { getSessions } from "../api.js";
import { state } from "../state.js";
import { formatDate, escapeHtml, setValue } from "../utils.js";

export async function loadSessions() {
  const res = await getSessions();

  if (!res.ok) {
    console.error(res.text);
    return;
  }

  state.overviewSessions = Array.isArray(res.body) ? res.body : [];

  renderSessions();
  syncSelect();
}

export function renderSessions() {
  const wrap = document.getElementById("overview-sessions-wrap");
  if (!wrap) return;

  if (!state.overviewSessions.length) {
    wrap.innerHTML = "<div>No sessions</div>";
    return;
  }

  wrap.innerHTML = state.overviewSessions
    .map(
      (s) => `
        <div>
            <button data-open-session="${escapeHtml(s.session_id)}">
                Open
            </button>
            ${escapeHtml(s.session_id)} — ${formatDate(s.created_at)}
        </div>
    `,
    )
    .join("");

  wrap.querySelectorAll("[data-open-session]").forEach((btn) => {
    btn.addEventListener("click", async () => {
      const sessionId = btn.getAttribute("data-open-session");

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
    ...state.overviewSessions.map(
      (s) =>
        `<option value="${escapeHtml(s.session_id)}">${escapeHtml(
          s.session_id,
        )}</option>`,
    ),
  ];

  select.innerHTML = options.join("");

  if (current && state.overviewSessions.some((s) => s.session_id === current)) {
    select.value = current;
    return;
  }

  if (state.overviewSessions.length > 0) {
    select.value = state.overviewSessions[0].session_id;
  }
}
