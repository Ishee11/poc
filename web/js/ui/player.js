import {
  getPlayerStats,
  getPlayers,
  getPlayersStats,
  getSessionPlayers,
} from "../api.js";
import { statusLabel, t } from "../i18n.js";
import { state } from "../state.js";
import {
  escapeHtml,
  formatDate,
  formatNumber,
  pushRoute,
  replaceRoute,
  routeToPlayer,
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

  state.players = (Array.isArray(res.body) ? res.body : []).sort(
    (a, b) => (Number(b.profit_money) || 0) - (Number(a.profit_money) || 0),
  );
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
    wrap.innerHTML = `<div class="empty-inline">${escapeHtml(t("common.noPlayers"))}</div>`;
    return;
  }

  wrap.innerHTML = state.overviewPlayers
    .map((player) => {
      const id = player.player_id;
      return `
        <div class="player-row clickable-row" data-open-player="${escapeHtml(id)}" tabindex="0" role="button">
          <div class="row-main">
            <div class="row-title">${escapeHtml(player.player_name || id)}</div>
            <div class="inline-stats">
              <span>${escapeHtml(t("common.sessions"))}: ${formatNumber(player.sessions_count)}</span>
              <span>${escapeHtml(t("common.profit"))}: ${formatNumber(player.profit_money)}</span>
            </div>
          </div>
        </div>
      `;
    })
    .join("");

  bindOpenPlayerButtons(wrap);
}

export async function loadPlayerDetail(
  playerId,
  { replace = false, preserveFilters = false } = {},
) {
  if (!playerId) return;
  if (state.selectedPlayerId !== playerId && !preserveFilters) {
    state.selectedPlayerFilters = { from: "", to: "" };
  }

  const res = await getPlayerStats(playerId, state.selectedPlayerFilters);
  if (!res.ok) {
    console.error("loadPlayerDetail failed:", res.text);
    return;
  }

  state.selectedPlayerId = playerId;
  state.selectedPlayerDetail = res.body || null;
  renderPlayerDetail();
  setScreen("player");
  if (replace) {
    replaceRoute(routeToPlayer(playerId));
  } else {
    pushRoute(routeToPlayer(playerId));
  }
}

export function renderPlayers() {
  const wrap = document.getElementById("players-wrap");
  if (!wrap) return;

  if (!state.players.length) {
    wrap.innerHTML = `<div class="empty-inline">${escapeHtml(t("common.noPlayers"))}</div>`;
    return;
  }

  wrap.innerHTML = state.players
    .map((player) => {
      const id = player.player_id || player.id;
      const name = player.player_name || player.name || id;
      const profitMoney = Number(player.profit_money) || 0;

      return `
        <div class="player-row clickable-row" data-open-player="${escapeHtml(id)}" tabindex="0" role="button">
          <div class="row-main">
            <div class="row-title">${escapeHtml(name)}</div>
            <div class="inline-stats">
              <span>${escapeHtml(t("common.buyIn"))}: ${formatNumber(player.buy_in)}</span>
              <span>${escapeHtml(t("common.cashOut"))}: ${formatNumber(player.cash_out)}</span>
              <span class="${profitMoney >= 0 ? "profit-positive" : "profit-negative"}">${escapeHtml(t("common.profit"))}: ${formatNumber(profitMoney)}</span>
              <span>${player.in_game ? escapeHtml(t("common.inGame")) : escapeHtml(t("common.settled"))}</span>
            </div>
          </div>
        </div>
      `;
    })
    .join("");

  bindOpenPlayerButtons(wrap);
}

export function renderPlayerDetail() {
  const wrap = document.getElementById("player-detail-wrap");
  if (!wrap) return;

  const detail = normalizePlayerDetail(state.selectedPlayerDetail);
  if (!detail || !detail.player) {
    wrap.className = "empty";
    wrap.textContent = t("common.noData");
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
          <td>${escapeHtml(statusLabel(session.status))}</td>
          <td>${formatNumber(session.buy_in_chips)}</td>
          <td>${formatNumber(session.cash_out_chips)}</td>
          <td>${formatNumber(session.profit_chips)}</td>
          <td>${formatNumber(session.profit_money)}</td>
          <td>${formatDate(session.last_activity_at)}</td>
          <td>
            <button type="button" data-open-session="${escapeHtml(session.session_id)}">${escapeHtml(t("common.open"))}</button>
          </td>
        </tr>
      `,
    )
    .join("");

  wrap.className = "";
  wrap.innerHTML = `
    <div class="panel-stack">
      <form id="player-period-form" class="filters" onsubmit="return false;">
        <label>
          ${escapeHtml(t("player.from"))}
          <input type="date" id="player-period-from" value="${escapeHtml(state.selectedPlayerFilters.from)}" />
        </label>
        <label>
          ${escapeHtml(t("player.to"))}
          <input type="date" id="player-period-to" value="${escapeHtml(state.selectedPlayerFilters.to)}" />
        </label>
        <button type="submit">${escapeHtml(t("player.applyPeriod"))}</button>
        <button type="button" class="secondary" id="player-period-clear">${escapeHtml(t("player.allTime"))}</button>
      </form>
      <div class="stats player-stats">
        <div class="stat">
          <div class="stat-label">${escapeHtml(t("player.sessions"))}</div>
          <div>${formatNumber(player.sessions_count)}</div>
        </div>
        <div class="stat">
          <div class="stat-label">${escapeHtml(t("player.totalBuyIn"))}</div>
          <div>${formatNumber(player.total_buy_in)}</div>
        </div>
        <div class="stat">
          <div class="stat-label">${escapeHtml(t("player.totalCashOut"))}</div>
          <div>${formatNumber(player.total_cash_out)}</div>
        </div>
        <div class="stat">
          <div class="stat-label">${escapeHtml(t("player.profitMoney"))}</div>
          <div class="${Number(player.profit_money) >= 0 ? "profit-positive" : "profit-negative"}">${formatNumber(player.profit_money)}</div>
        </div>
      </div>
      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th>${escapeHtml(t("table.session"))}</th>
              <th>${escapeHtml(t("table.status"))}</th>
              <th>${escapeHtml(t("table.buyIn"))}</th>
              <th>${escapeHtml(t("table.cashOut"))}</th>
              <th>${escapeHtml(t("table.profitChips"))}</th>
              <th>${escapeHtml(t("table.profit"))}</th>
              <th>${escapeHtml(t("table.lastActivity"))}</th>
              <th></th>
            </tr>
          </thead>
          <tbody>${rows}</tbody>
        </table>
      </div>
    </div>
  `;

  wrap.querySelector("#player-period-form")?.addEventListener("submit", async () => {
    state.selectedPlayerFilters = {
      from: document.getElementById("player-period-from")?.value || "",
      to: document.getElementById("player-period-to")?.value || "",
    };
    await loadPlayerDetail(state.selectedPlayerId, {
      replace: true,
      preserveFilters: true,
    });
  });

  wrap.querySelector("#player-period-clear")?.addEventListener("click", async () => {
    state.selectedPlayerFilters = { from: "", to: "" };
    await loadPlayerDetail(state.selectedPlayerId, {
      replace: true,
      preserveFilters: true,
    });
  });

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
  container.querySelectorAll("[data-open-player]").forEach((row) => {
    row.addEventListener("click", async () => {
      const playerId = row.getAttribute("data-open-player");
      if (!playerId) return;
      await loadPlayerDetail(playerId);
    });
    row.addEventListener("keydown", async (event) => {
      if (event.key !== "Enter" && event.key !== " ") return;
      event.preventDefault();
      const playerId = row.getAttribute("data-open-player");
      if (!playerId) return;
      await loadPlayerDetail(playerId);
    });
  });
}

function normalizePlayerDetail(raw) {
  if (!raw) return null;
  return {
    player: raw.player || raw.Player || null,
    sessions: raw.sessions || raw.Sessions || [],
  };
}
