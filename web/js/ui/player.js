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
  const detail = state.selectedPlayerDetail;

  if (!detail || !detail.Player) {
    wrap.className = "empty";
    wrap.textContent = "No data";
    return;
  }

  const player = detail.Player;
  const sessions = detail.Sessions || [];

  document.getElementById("player-screen-title").textContent =
    `Player ${player.PlayerID}`;

  document.getElementById("player-screen-id").textContent =
    `player_id: ${player.PlayerID}`;

  const rows = sessions
    .map(
      (s) => `
        <tr>
            <td class="mono">${escapeHtml(s.session_id)}</td>
            <td>${escapeHtml(s.status)}</td>
            <td>${formatNumber(s.BuyInChips)}</td>
            <td>${formatNumber(s.CashOutChips)}</td>
            <td>${formatNumber(s.ProfitMoney)}</td>
            <td>${formatDate(s.LastActivityAt)}</td>
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
            Sessions: ${formatNumber(player.SessionsCount)}
        </div>
        <div>
            Profit: ${formatNumber(player.ProfitMoney)}
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

      // динамический импорт, чтобы не было циклов
      const { openSession } = await import("../session.js");
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
 * helpers (можно потом вынести)
 */
function value(id) {
  return document.getElementById(id).value;
}

function buildQuery(params) {
  const q = new URLSearchParams();

  Object.entries(params).forEach(([k, v]) => {
    if (v) q.set(k, v);
  });

  return q.toString() ? `?${q}` : "";
}
