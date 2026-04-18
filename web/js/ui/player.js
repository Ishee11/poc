import {
  getPlayerStats,
  getPlayers,
  getPlayersStats,
  getSessionPlayers,
} from "../api.js";
import { state } from "../state.js";
import {
  escapeHtml,
  formatDate,
  formatNumber,
  setScreen,
  setValue,
} from "../utils.js";

export async function loadPlayers(sessionId) {
  if (!sessionId) return;

  const res = await getSessionPlayers(sessionId);
  if (!res.ok) {
    console.error("loadPlayers failed:", res.text);
    state.players = [];
    renderPlayers();
    return;
  }

  state.players = Array.isArray(res.body) ? res.body : [];
  renderPlayers();
}

export async function loadPlayersOverview() {
  const [playersRes, statsRes] = await Promise.all([
    getPlayers({ limit: 500 }),
    getPlayersStats(),
  ]);

  const players = Array.isArray(playersRes.body) ? playersRes.body : [];
  const stats = Array.isArray(statsRes.body) ? statsRes.body : [];
  const statsById = new Map(stats.map((item) => [item.player_id, item]));

  state.overviewPlayers = players.map((player) => {
    const stat = statsById.get(player.player_id);
    return {
      player_id: player.player_id,
      player_name: player.name,
      sessions_count: stat?.sessions_count || 0,
      profit_money: stat?.profit_money || 0,
    };
  });

  renderPlayersOverview();
}

export function renderPlayersOverview() {
  const wrap = document.getElementById("overview-players-wrap");
  const count = document.getElementById("overview-players-count");
  if (!wrap || !count) return;

  count.textContent = String(state.overviewPlayers.length);

  if (!state.overviewPlayers.length) {
    wrap.innerHTML = '<div class="empty-inline">No players</div>';
    return;
  }

  wrap.innerHTML = state.overviewPlayers
    .map((player) => {
      const id = player.player_id;
      return `
        <div class="player-row">
          <div class="row-main">
            <div class="row-title">${escapeHtml(player.player_name || id)}</div>
            <div class="inline-stats">
              <span>Sessions: ${formatNumber(player.sessions_count)}</span>
              <span>Profit: ${formatNumber(player.profit_money)}</span>
            </div>
          </div>
          <button type="button" data-open-player="${escapeHtml(id)}">Open</button>
        </div>
      `;
    })
    .join("");

  bindOpenPlayerButtons(wrap);
}

export async function loadPlayerDetail(playerId) {
  if (!playerId) return;

  const res = await getPlayerStats(playerId);
  if (!res.ok) {
    console.error("loadPlayerDetail failed:", res.text);
    return;
  }

  state.selectedPlayerId = playerId;
  state.selectedPlayerDetail = res.body || null;
  renderPlayerDetail();
  setScreen("player");
}

export function renderPlayers() {
  const wrap = document.getElementById("players-wrap");
  if (!wrap) return;

  if (!state.players.length) {
    wrap.innerHTML = '<div class="empty-inline">No players in this session yet</div>';
    return;
  }

  wrap.innerHTML = state.players
    .map((player) => {
      const id = player.player_id || player.id;
      const name = player.player_name || player.name || id;

      return `
        <div class="player-row">
          <div class="row-main">
            <div class="row-title">${escapeHtml(name)}</div>
            <div class="inline-stats">
              <span>Buy in: ${formatNumber(player.buy_in)}</span>
              <span>Cash out: ${formatNumber(player.cash_out)}</span>
              <span>${player.in_game ? "In game" : "Settled"}</span>
            </div>
          </div>
          <button type="button" data-open-player="${escapeHtml(id)}">Open</button>
        </div>
      `;
    })
    .join("");

  bindOpenPlayerButtons(wrap);
}

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
  const playerName = player.player_name || player.name || player.player_id;

  if (title) title.textContent = playerName;
  if (id) id.textContent = `ID: ${player.player_id}`;

  const rows = sessions
    .map(
      (session) => `
        <tr>
          <td>${escapeHtml(formatDate(session.session_created_at))}</td>
          <td>${escapeHtml(session.status)}</td>
          <td>${formatNumber(session.buy_in_chips)}</td>
          <td>${formatNumber(session.cash_out_chips)}</td>
          <td>${formatNumber(session.profit_money)}</td>
          <td>${formatDate(session.last_activity_at)}</td>
          <td>
            <button type="button" data-open-session="${escapeHtml(session.session_id)}">Open</button>
          </td>
        </tr>
      `,
    )
    .join("");

  wrap.className = "";
  wrap.innerHTML = `
    <div class="panel-stack">
      <div>Sessions: ${formatNumber(player.sessions_count)}</div>
      <div>Profit: ${formatNumber(player.profit_money)}</div>
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
    </div>
  `;

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

function bindOpenPlayerButtons(container) {
  container.querySelectorAll("[data-open-player]").forEach((button) => {
    button.addEventListener("click", async () => {
      const playerId = button.getAttribute("data-open-player");
      if (!playerId) return;
      await loadPlayerDetail(playerId);
    });
  });
}
