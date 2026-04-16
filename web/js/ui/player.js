import { apiGet } from "../api.js";
import { state } from "../state.js";
import { formatNumber, formatDate, escapeHtml, setValue } from "../utils.js";

/**
 * Загрузка детальной статистики игрока
 */
export async function loadPlayerDetail(playerId) {
  const query = buildQuery({
    player_id: playerId,
    from: value("player-detail-from"),
    to: value("player-detail-to"),
  });

  const res = await apiGet(`/stats/player${query}`);

  if (!res.ok) {
    console.error(res.text);
    return;
  }

  state.selectedPlayerId = playerId;
  state.selectedPlayerDetail = res.body || null;

  renderPlayerDetail();
  setScreen("player");
}

/**
 * Рендер экрана игрока
 */
export function renderPlayerDetail() {
  const wrap = document.getElementById("player-detail-wrap");
  if (!wrap) return;

  const detail = state.selectedPlayerDetail;

  if (!detail || !detail.player) {
    wrap.className = "empty";
    wrap.textContent = "No data";
    return;
  }

  const player = detail.player;
  const sessions = detail.sessions || [];

  document.getElementById("player-screen-title").textContent =
    `Player ${player.player_id}`;

  document.getElementById("player-screen-id").textContent =
    `player_id: ${player.player_id}`;

  const rows = sessions
    .map(
      (s) => `
        <tr>
            <td class="mono">${escapeHtml(s.session_id)}</td>
            <td>${escapeHtml(s.status)}</td>
            <td>${formatNumber(s.buy_in_chips)}</td>
            <td>${formatNumber(s.cash_out_chips)}</td>
            <td>${formatNumber(s.profit_money)}</td>
            <td>${formatDate(s.last_activity_at)}</td>
            <td>
                <button data-open-session="${escapeHtml(s.session_id)}">
                    Open
                </button>
            </td>
        </tr>
    `,
    )
    .join("");

  wrap.className = "";
  wrap.innerHTML = `
        <div>
            Sessions: ${formatNumber(player.sessions_count)}
        </div>
        <div>
            Profit: ${formatNumber(player.profit_money)}
        </div>

        <div class="table-wrap">
            <table>
                <thead>
                    <tr>
                        <th>Session</th>
                        <th>Status</th>
                        <th>Buy In</th>
                        <th>Cash Out</th>
                        <th>Profit</th>
                        <th>Last Activity</th>
                        <th></th>
                    </tr>
                </thead>
                <tbody>${rows}</tbody>
            </table>
        </div>
    `;

  // обработчик перехода в session
  wrap.querySelectorAll("[data-open-session]").forEach((btn) => {
    btn.addEventListener("click", async () => {
      const sessionId = btn.getAttribute("data-open-session");

      setValue("active-session-select", sessionId);

      const { openSession } = await import("./session.js");
      await openSession(sessionId);
    });
  });
}

/**
 * Переключение экранов
 */
function setScreen(name) {
  document
    .getElementById("screen-lobby")
    .classList.toggle("active", name === "lobby");

  document
    .getElementById("screen-session")
    .classList.toggle("active", name === "session");

  document
    .getElementById("screen-player")
    .classList.toggle("active", name === "player");
}

/**
 * helpers
 */
function value(id) {
  const el = document.getElementById(id);
  return el ? el.value : "";
}

function buildQuery(params) {
  const q = new URLSearchParams();

  Object.entries(params).forEach(([k, v]) => {
    if (v) q.set(k, v);
  });

  return q.toString() ? `?${q}` : "";
}
