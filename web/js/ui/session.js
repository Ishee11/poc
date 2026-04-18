import {
  getSession,
  getSessionOperations,
  buyIn,
  cashOut,
} from "../api.js";
import { state } from "../state.js";
import { formatNumber } from "../utils.js";
import { loadPlayers } from "./player.js";

/**
 * открыть сессию
 */
export async function openSession(sessionId) {
  if (!sessionId) {
    console.error("openSession: empty sessionId");
    return;
  }

  // 👉 сохраняем сразу (единый источник правды)
  state.activeSessionId = sessionId;

  // 👉 очищаем старое состояние
  state.session = null;
  state.operations = [];
  state.players = [];

  // 👉 грузим сессию
  const res = await getSession(sessionId);

  if (!res.ok || !res.body) {
    console.error("openSession failed:", res.text);
    return;
  }

  const s = res.body;

  state.session = {
    id: s.session_id,
    status: s.status,
    chipRate: s.chip_rate,
    totalBuyIn: s.total_buy_in,
    totalCashOut: s.total_cash_out,
    totalChips: s.total_chips,
  };

  // 👉 первичный рендер (можно как loading state)
  renderSession();
  renderOperations(); // очистит UI

  // 👉 параллельно грузим данные
  await Promise.all([loadPlayers(sessionId), loadOperations(sessionId)]);

  // 👉 переключаем экран
  setScreen("session");
}

/**
 * загрузка операций
 */
export async function loadOperations(sessionId) {
  if (!sessionId) return;

  const res = await getSessionOperations(sessionId);

  if (!res.ok) {
    console.error("loadOperations failed:", res.text);
    return;
  }

  state.operations = Array.isArray(res.body) ? res.body : [];

  renderOperations();
}

/**
 * рендер сессии (статы)
 */
export function renderSession() {
  const s = state.session;
  if (!s) return;

  const chipRate = document.getElementById("stat-chip-rate");
  const buyIn = document.getElementById("stat-buy-in");
  const cashOut = document.getElementById("stat-cash-out");

  if (chipRate) chipRate.textContent = formatNumber(s.chipRate);
  if (buyIn) buyIn.textContent = formatNumber(s.totalBuyIn);
  if (cashOut) cashOut.textContent = formatNumber(s.totalCashOut);
}

/**
 * рендер операций
 */
export function renderOperations() {
  const wrap = document.getElementById("operations-wrap");
  if (!wrap) return;

  if (!state.operations.length) {
    wrap.innerHTML = "<div>No operations</div>";
    return;
  }

  wrap.innerHTML = state.operations
    .map(
      (op) => `
        <div class="operation-row">
          <span>${op.type}</span>
          <span>${formatNumber(op.chips)}</span>
        </div>
      `,
    )
    .join("");
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

export function initSessionActions() {
  document.addEventListener("click", async (e) => {
    const target = e.target.closest("button");

    if (!target) return;

    if (target.id === "buy-in-btn") {
      await handleBuyIn();
    }

    if (target.id === "cash-out-btn") {
      await handleCashOut();
    }
  });
}

async function handleBuyIn() {
  const playerId = document.getElementById("action-player-id").value;
  const chips = Number(document.getElementById("action-chips").value);

  if (!playerId || !Number.isFinite(chips) || chips <= 0) {
    console.error("invalid input");
    return;
  }

  const res = await buyIn({
    sessionId: state.activeSessionId,
    playerId,
    chips,
  });

  if (!res.ok) {
    console.error("buyIn failed:", res.text);
    return;
  }

  await refreshSession();

  document.getElementById("action-chips").value = "";
}

async function handleCashOut() {
  const playerId = document.getElementById("action-player-id").value;
  const chips = Number(document.getElementById("action-chips").value);

  if (!playerId || !Number.isFinite(chips) || chips <= 0) {
    console.error("invalid input");
    return;
  }

  const res = await cashOut({
    sessionId: state.activeSessionId,
    playerId,
    chips,
  });

  if (!res.ok) {
    console.error("cashOut failed:", res.text);
    return;
  }

  await refreshSession();
}

async function refreshSession() {
  const id = state.activeSessionId;
  if (!id) return;

  const res = await getSession(id);
  if (!res.ok || !res.body) return;

  const s = res.body;

  state.session = {
    id: s.session_id,
    status: s.status,
    chipRate: s.chip_rate,
    totalBuyIn: s.total_buy_in,
    totalCashOut: s.total_cash_out,
    totalChips: s.total_chips,
  };

  renderSession();

  await Promise.all([loadPlayers(id), loadOperations(id)]);
}
