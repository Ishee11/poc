import { apiGet } from "../api.js";
import { state } from "../state.js";
import { formatNumber, formatDate, escapeHtml, setValue } from "../utils.js";

/**
 * загрузка игроков
 */
export async function loadPlayers(sessionId) {
  if (!sessionId) return;

  const res = await apiGet(`/session/players?session_id=${sessionId}`);

  if (!res.ok) {
    console.error("loadPlayers failed:", res.text);
    return;
  }

  state.players = res.body || [];

  console.log("players:", state.players); // 👈 важно для дебага

  renderPlayers();
}

/**
 * рендер игроков
 */
function renderPlayers() {
  const wrap = document.getElementById("players-wrap");
  if (!wrap) return;

  if (!state.players.length) {
    wrap.innerHTML = "<div>No players</div>";
    return;
  }

  wrap.innerHTML = state.players
    .map((p) => {
      const id = p.player_id || p.id; // 👈 КЛЮЧЕВОЙ ФИКС

      return `
        <div>
          <button data-open-player="${escapeHtml(id)}">
            Open
          </button>
          ${escapeHtml(id)}
        </div>
      `;
    })
    .join("");

  wrap.querySelectorAll("[data-open-player]").forEach((btn) => {
    btn.addEventListener("click", async () => {
      const playerId = btn.getAttribute("data-open-player");

      if (!playerId) {
        console.error("empty playerId");
        return;
      }

      await loadPlayerDetail(playerId);
    });
  });
}

/**
 * загрузка детальной статистики
 */
export async function loadPlayerDetail(playerId) {
  if (!playerId) {
    console.error("playerId is empty");
    return;
  }

  const res = await apiGet(`/stats/player?player_id=${playerId}`);

  if (!res.ok) {
    console.error("loadPlayerDetail failed:", res.text);
    return;
  }

  state.selectedPlayerId = playerId;
  state.selectedPlayerDetail = res.body || null;

  renderPlayerDetail();
  setScreen("player");
}

/**
 * рендер игрока
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

  const title = document.getElementById("player-screen-title");
  const id = document.getElementById("player-screen-id");

  if (title) title.textContent = `Player ${player.player_id}`;
  if (id) id.textContent = `player_id: ${player.player_id}`;

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
 * переключение экранов
 */
function setScreen(name) {
  document
    .getElementById("screen-lobby")
    ?.classList.toggle("active", name === "lobby");

  document
    .getElementById("screen-session")
    ?.classList.toggle("active", name === "session");

  document
    .getElementById("screen-player")
    ?.classList.toggle("active", name === "player");
}
