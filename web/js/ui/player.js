import {
  debugDeletePlayer,
  debugRenamePlayer,
  getPlayerStats,
  getPlayers,
  getPlayersStats,
  getSessionPlayers,
} from "../api.js";
import { statusLabel, t } from "../i18n.js";
import { state } from "../state.js";
import {
  describeError,
  escapeHtml,
  formatDate,
  formatNumber,
  openModal,
  pushRoute,
  replaceRoute,
  routeToHome,
  routeToPlayer,
  setScreen,
  setValue,
  showNotice,
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
              <span class="${profitMoney >= 0 ? "profit-positive" : "profit-negative"}">${escapeHtml(t("common.profit"))}: ${formatMoney(profitMoney, state.session?.currency)}</span>
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
    renderPlayerHeaderDebugActions(null);
    renderPlayerDebugActions(null);
    return;
  }

  const player = detail.player;
  const sessions = detail.sessions || [];
  const title = document.getElementById("player-screen-title");
  const id = document.getElementById("player-screen-id");
  const linkedUser = document.getElementById("player-screen-user");
  const playerName = player.player_name || player.name || player.player_id;

  if (title) title.textContent = playerName;
  if (id) id.textContent = `ID: ${player.player_id}`;
  if (linkedUser) {
    linkedUser.textContent = `${t("player.linkedUser")}: ${t("player.noLinkedUser")}`;
  }
  renderPlayerHeaderDebugActions(player);
  renderPlayerDebugActions(player);

  const rows = sessions
    .map(
      (session) => `
        <tr class="clickable-row table-clickable-row" data-open-session="${escapeHtml(session.session_id)}" tabindex="0" role="button">
          <td>${escapeHtml(formatDate(session.session_created_at))}</td>
          <td>${escapeHtml(statusLabel(session.status))}</td>
          <td>${formatNumber(session.buy_in_chips)}</td>
          <td>${formatNumber(session.cash_out_chips)}</td>
          <td>${formatNumber(session.profit_chips)}</td>
          <td>${formatNumber(session.profit_money)}</td>
          <td>${formatDate(session.last_activity_at)}</td>
        </tr>
      `,
    )
    .join("");

  wrap.className = "";
  wrap.innerHTML = `
    <div class="panel-stack">
      <div class="period-toolbar">
        <div>
          <div class="stat-label">${escapeHtml(t("player.period"))}</div>
          <div class="period-summary">${escapeHtml(periodSummary())}</div>
        </div>
        <button type="button" id="player-period-select">${escapeHtml(t("player.selectPeriod"))}</button>
        <button type="button" class="secondary" id="player-period-clear">${escapeHtml(t("player.allTime"))}</button>
      </div>
      <div class="stats player-stats">
        <div class="stat">
          ${statLabel("player.sessions", "player.hint.sessions")}
          <div>${formatNumber(player.sessions_count)}</div>
        </div>
        <div class="stat">
          ${statLabel("player.totalBuyIn", "player.hint.totalBuyIn")}
          <div>${formatNumber(player.total_buy_in)}</div>
        </div>
        <div class="stat">
          ${statLabel("player.totalCashOut", "player.hint.totalCashOut")}
          <div>${formatNumber(player.total_cash_out)}</div>
        </div>
        <div class="stat">
          ${statLabel("player.totalBuyInMoney", "player.hint.totalBuyInMoney")}
          <div>${formatNumber(player.total_buy_in_money)}</div>
        </div>
        <div class="stat">
          ${statLabel("player.totalCashOutMoney", "player.hint.totalCashOutMoney")}
          <div>${formatNumber(player.total_cash_out_money)}</div>
        </div>
        <div class="stat">
          ${statLabel("player.pnl", "player.hint.pnl")}
          <div class="${Number(player.profit_money) >= 0 ? "profit-positive" : "profit-negative"}">${formatNumber(player.profit_money)}</div>
        </div>
        <div class="stat">
          ${statLabel("player.avgProfitPerSession", "player.hint.avgProfitPerSession")}
          <div class="${Number(player.avg_profit_per_session) >= 0 ? "profit-positive" : "profit-negative"}">${formatNumber(roundMetric(player.avg_profit_per_session))}</div>
        </div>
        <div class="stat">
          ${statLabel("player.roi", "player.hint.roi")}
          <div class="${Number(player.roi_percent) >= 0 ? "profit-positive" : "profit-negative"}">${formatPercent(player.roi_percent)}</div>
        </div>
        <div class="stat">
          ${statLabel("player.avgBuyInPerSession", "player.hint.avgBuyInPerSession")}
          <div>${formatNumber(roundMetric(player.avg_buy_in_per_session))}</div>
        </div>
      </div>
      ${state.debugMode ? renderProfitChart(sessions) : ""}
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
            </tr>
          </thead>
          <tbody>${rows}</tbody>
        </table>
      </div>
    </div>
  `;

  wrap.querySelector("#player-period-select")?.addEventListener("click", async () => {
    const values = await openModal({
      title: t("player.selectPeriod"),
      confirmText: t("player.applyPeriod"),
      fields: [
        {
          name: "from",
          label: t("player.from"),
          type: "date",
          value: state.selectedPlayerFilters.from,
        },
        {
          name: "to",
          label: t("player.to"),
          type: "date",
          value: state.selectedPlayerFilters.to,
        },
      ],
    });
    if (!values) return;

    state.selectedPlayerFilters = {
      from: values.from || "",
      to: values.to || "",
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

  bindOpenSessionRows(wrap);
  bindStatHelp(wrap);
}

function renderPlayerHeaderDebugActions(player) {
  const actions = document.getElementById("player-header-debug-actions");
  if (!actions) return;

  actions.hidden = !state.debugMode || !player;
  actions.querySelector("#debug-rename-player-btn")?.replaceWith(
    actions.querySelector("#debug-rename-player-btn").cloneNode(true),
  );
  actions
    .querySelector("#debug-rename-player-btn")
    ?.addEventListener("click", async () => {
      await confirmDebugRenamePlayer(player);
    });
}

function renderPlayerDebugActions(player) {
  const actions = document.getElementById("player-debug-actions");
  if (!actions) return;

  actions.hidden = !state.debugMode || !player;
  actions.querySelector("#debug-delete-player-btn")?.replaceWith(
    actions.querySelector("#debug-delete-player-btn").cloneNode(true),
  );
  actions
    .querySelector("#debug-delete-player-btn")
    ?.addEventListener("click", async () => {
      await confirmDebugDeletePlayer(player);
    });
}

async function confirmDebugRenamePlayer(player) {
  if (!state.debugMode || !player?.player_id) return;

  const playerName = player.player_name || player.name || player.player_id;
  const values = await openModal({
    title: t("modal.renamePlayerTitle"),
    confirmText: t("debug.renamePlayer"),
    fields: [
      {
        name: "name",
        label: t("lobby.playerName"),
        type: "text",
        value: playerName,
      },
    ],
  });
  if (!values) return;

  const name = (values.name || "").trim();
  if (!name) {
    showNotice(t("notice.enterPlayerName"), "error");
    return;
  }

  const res = await debugRenamePlayer(player.player_id, name);
  if (!res.ok) {
    showNotice(describeError(res, t("error.failedRenamePlayer")), "error");
    return;
  }

  await Promise.all([
    loadPlayerDetail(player.player_id, { replace: true, preserveFilters: true }),
    loadPlayersOverview(),
  ]);
  showNotice(t("notice.playerRenamed"), "success");
}

async function confirmDebugDeletePlayer(player) {
  if (!state.debugMode || !player?.player_id) return;

  const playerName = player.player_name || player.name || player.player_id;
  const confirmed = await openModal({
    title: t("modal.deletePlayerTitle"),
    description: t("modal.deletePlayerDescription", { name: playerName }),
    confirmText: t("debug.deletePlayer"),
  });
  if (!confirmed) return;

  const res = await debugDeletePlayer(player.player_id);
  if (!res.ok) {
    showNotice(describeError(res, t("error.failedDeletePlayer")), "error");
    return;
  }

  state.selectedPlayerId = "";
  state.selectedPlayerDetail = null;
  await loadPlayersOverview();
  setScreen("lobby");
  pushRoute(routeToHome());
  showNotice(t("notice.playerDeleted"), "success");
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

function bindOpenSessionRows(container) {
  container.querySelectorAll("[data-open-session]").forEach((row) => {
    row.addEventListener("click", async () => {
      await openSessionFromRow(row);
    });
    row.addEventListener("keydown", async (event) => {
      if (event.key !== "Enter" && event.key !== " ") return;
      event.preventDefault();
      await openSessionFromRow(row);
    });
  });
}

async function openSessionFromRow(row) {
  const sessionId = row.getAttribute("data-open-session");
  if (!sessionId) return;

  setValue("active-session-select", sessionId);
  const { openSession } = await import("./session.js");
  await openSession(sessionId);
}

function bindStatHelp(container) {
  container.querySelectorAll("[data-stat-help]").forEach((button) => {
    button.addEventListener("click", async (event) => {
      event.stopPropagation();
      await openModal({
        title: button.getAttribute("data-stat-title") || "",
        description: button.getAttribute("data-stat-help") || "",
        confirmText: t("common.close"),
        showCancel: false,
      });
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

function statLabel(labelKey, hintKey) {
  const hint = t(hintKey);
  const label = t(labelKey);
  return `
    <div class="stat-label">
      <span>${escapeHtml(label)}</span>
      <button
        type="button"
        class="stat-help"
        title="${escapeHtml(hint)}"
        aria-label="${escapeHtml(hint)}"
        data-stat-title="${escapeHtml(label)}"
        data-stat-help="${escapeHtml(hint)}"
      >?</button>
    </div>
  `;
}

function formatPercent(value) {
  const number = Number(value);
  return Number.isFinite(number) ? `${formatNumber(roundMetric(number))}%` : "-";
}

function formatMoney(value, currency) {
  return `${formatNumber(value)} ${currencySymbol(currency)}`;
}

function currencySymbol(currency) {
  return currency === "USD" ? "$" : "₽";
}

function roundMetric(value) {
  return Math.round(value * 100) / 100;
}

function periodSummary() {
  const { from, to } = state.selectedPlayerFilters;
  if (!from && !to) return t("player.allTime");
  if (from && to) return `${t("player.from")}: ${from} · ${t("player.to")}: ${to}`;
  if (from) return `${t("player.from")}: ${from}`;
  return `${t("player.to")}: ${to}`;
}

function renderProfitChart(sessions) {
  const points = [...sessions]
    .sort((a, b) => String(a.session_created_at).localeCompare(String(b.session_created_at)))
    .map((session) => Number(session.profit_money) || 0)
    .reduce((acc, profit) => {
      const previous = acc.length ? acc[acc.length - 1] : 0;
      acc.push(previous + profit);
      return acc;
    }, []);

  if (points.length < 2) {
    return `
      <div class="player-chart">
        <div class="stat-label">${escapeHtml(t("player.profitChart"))}</div>
        <div class="empty-inline">${escapeHtml(t("player.noChartData"))}</div>
      </div>
    `;
  }

  const width = 640;
  const height = 180;
  const padding = 22;
  const min = Math.min(...points, 0);
  const max = Math.max(...points, 0);
  const range = max - min || 1;
  const step = (width - padding * 2) / (points.length - 1);
  const coordinates = points.map((value, index) => {
    const x = padding + index * step;
    const y = height - padding - ((value - min) / range) * (height - padding * 2);
    return `${x.toFixed(1)},${y.toFixed(1)}`;
  });
  const zeroY = height - padding - ((0 - min) / range) * (height - padding * 2);

  return `
    <div class="player-chart">
      <div class="stat-label">${escapeHtml(t("player.profitChart"))}</div>
      <svg viewBox="0 0 ${width} ${height}" role="img" aria-label="${escapeHtml(t("player.profitChart"))}">
        <line class="axis" x1="${padding}" y1="${zeroY.toFixed(1)}" x2="${width - padding}" y2="${zeroY.toFixed(1)}"></line>
        <polyline points="${escapeHtml(coordinates.join(" "))}"></polyline>
      </svg>
    </div>
  `;
}
