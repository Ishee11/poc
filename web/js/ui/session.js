import { apiGet } from "./api.js";
import { state } from "./state.js";
import { formatNumber, formatDate } from "./utils.js";

export async function openSession(sessionId) {
  const res = await apiGet(`/session?session_id=${sessionId}`);

  if (!res.ok) {
    console.error(res.text);
    return;
  }

  state.session = res.body;

  renderSession();
}

export function renderSession() {
  const s = state.session;

  document.getElementById("stat-chip-rate").textContent = formatNumber(
    s.chip_rate,
  );

  document.getElementById("stat-buy-in").textContent = formatNumber(
    s.total_buy_in,
  );

  document.getElementById("stat-cash-out").textContent = formatNumber(
    s.total_cash_out,
  );
}
