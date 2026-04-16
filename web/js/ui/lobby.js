import { apiGet } from "../api.js";
import { state } from "../state.js";
import { formatDate, escapeHtml, setValue } from "../utils.js";

export async function loadSessions() {
  const res = await apiGet("/stats/sessions?limit=20");

  if (!res.ok) {
    console.error(res.text);
    return;
  }

  state.overviewSessions = res.body || [];

  renderSessions();
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

  // обработчик открытия session
  wrap.querySelectorAll("[data-open-session]").forEach((btn) => {
    btn.addEventListener("click", async () => {
      const sessionId = btn.getAttribute("data-open-session");

      setValue("active-session-select", sessionId);

      const { openSession } = await import("./session.js");
      await openSession(sessionId);
    });
  });
}
