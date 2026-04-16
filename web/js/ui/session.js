import { apiGet } from "../api.js";
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

  const res = await apiGet(`/session?session_id=${sessionId}`);

  if (!res.ok || !res.body) {
    console.error("openSession failed:", res.text);
    return;
  }

  const s = res.body;

  // 👉 фикс: сохраняем активную сессию
  state.activeSessionId = sessionId;

  // 👉 фикс: очищаем старое состояние (иначе артефакты)
  state.operations = [];
  state.players = [];

  state.session = {
    id: s.session_id,
    status: s.status,
    chipRate: s.chip_rate,
    totalBuyIn: s.total_buy_in,
    totalCashOut: s.total_cash_out,
    totalChips: s.total_chips,
  };

  renderSession();

  // 👉 грузим всё параллельно
  await Promise.all([loadPlayers(sessionId), loadOperations(sessionId)]);

  setScreen("session");
}

/**
 * загрузка операций
 */
export async function loadOperations(sessionId) {
  if (!sessionId) return;

  const res = await apiGet(`/session/operations?session_id=${sessionId}`);

  if (!res.ok) {
    console.error("loadOperations failed:", res.text);
    return;
  }

  state.operations = res.body || [];

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
function renderOperations() {
  const wrap = document.getElementById("operations-wrap");
  if (!wrap) return;

  if (!state.operations.length) {
    wrap.innerHTML = "<div>No operations</div>";
    return;
  }

  wrap.innerHTML = state.operations
    .map(
      (op) => `
        <div>
          ${op.type} — ${op.chips}
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
