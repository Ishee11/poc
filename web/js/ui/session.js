import { getSession, getSessionOperations } from "../api.js";

import { state } from "../state.js";
import { formatNumber } from "../utils.js";
import { loadPlayers } from "./player.js";

export async function openSession(sessionId) {
  if (!sessionId) {
    console.error("openSession: empty sessionId");
    return;
  }

  const res = await getSession(sessionId);

  if (!res.ok || !res.body) {
    console.error("openSession failed:", res.text);
    return;
  }

  const s = res.body;

  state.activeSessionId = sessionId;

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

  await Promise.all([loadPlayers(sessionId), loadOperations(sessionId)]);

  setScreen("session");
}

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

export function renderSession() {
  const s = state.session;
  if (!s) return;

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
