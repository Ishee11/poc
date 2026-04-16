import { apiGet } from "./api.js";
import { state } from "./state.js";
import { formatNumber, formatDate } from "./utils.js";

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
}

export function renderSession() {
  const s = state.session;

  document.getElementById("stat-chip-rate").textContent = formatNumber(
    s.chipRate,
  );

  document.getElementById("stat-buy-in").textContent = formatNumber(
    s.totalBuyIn,
  );

  document.getElementById("stat-cash-out").textContent = formatNumber(
    s.totalCashOut,
  );
}
