import { apiGet } from "../api.js";
import { state } from "../state.js";
import { formatNumber } from "../utils.js";

export async function openSession(sessionId) {
  const res = await apiGet(`/session?session_id=${sessionId}`);

  if (!res.ok) {
    console.error(res.text);
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

  renderSession();
  setScreen("session");
}

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
