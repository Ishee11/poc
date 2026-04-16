import { apiGet } from "./api.js";
import { state } from "./state.js";
import { formatDate, escapeHtml } from "./utils.js";

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

  if (!state.overviewSessions.length) {
    wrap.innerHTML = "<div>No sessions</div>";
    return;
  }

  wrap.innerHTML = state.overviewSessions
    .map(
      (s) => `
        <div>
            ${escapeHtml(s.session_id)} — ${formatDate(s.created_at)}
        </div>
    `,
    )
    .join("");
}
