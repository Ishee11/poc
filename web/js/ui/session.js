import {
  buyIn,
  cashOut,
  createPlayer,
  finishSession,
  getSession,
  getSessionOperations,
  reverseOperation,
} from "../api.js";
import { state } from "../state.js";
import {
  describeError,
  escapeHtml,
  formatDate,
  formatNumber,
  openModal,
  setScreen,
  showNotice,
} from "../utils.js";
import { loadSessions } from "./lobby.js";
import { loadPlayers, loadPlayersOverview } from "./player.js";

export async function openSession(sessionId) {
  if (!sessionId) return;

  state.activeSessionId = sessionId;
  state.session = null;
  state.operations = [];
  state.players = [];

  const res = await getSession(sessionId);
  if (!res.ok || !res.body) {
    showNotice(describeError(res, "Failed to load session"), "error");
    return;
  }

  hydrateSession(res.body);
  renderSession();
  renderOperations();
  renderActionPlayerOptions();

  await Promise.all([loadPlayers(sessionId), loadOperations(sessionId)]);
  renderActionPlayerOptions();
  setScreen("session");
}

export async function loadOperations(sessionId) {
  if (!sessionId) return;

  const res = await getSessionOperations(sessionId);
  if (!res.ok) {
    console.error("loadOperations failed:", res.text);
    state.operations = [];
    renderOperations();
    return;
  }

  state.operations = Array.isArray(res.body) ? res.body : [];
  renderOperations();
}

export function renderSession() {
  const session = state.session;
  if (!session) return;

  const subtitle = document.getElementById("workspace-subtitle");
  const chipRate = document.getElementById("stat-chip-rate");
  const buyIn = document.getElementById("stat-buy-in");
  const cashOut = document.getElementById("stat-cash-out");
  const totalChips = document.getElementById("stat-total-chips");
  const finishButton = document.getElementById("finish-session-btn");

  if (subtitle) {
    subtitle.textContent = `${formatDate(session.createdAt)} • ${session.status}`;
  }
  if (chipRate) chipRate.textContent = formatNumber(session.chipRate);
  if (buyIn) buyIn.textContent = formatNumber(session.totalBuyIn);
  if (cashOut) cashOut.textContent = formatNumber(session.totalCashOut);
  if (totalChips) totalChips.textContent = formatNumber(session.totalChips);
  if (finishButton) finishButton.disabled = session.status !== "active";
}

export function renderOperations() {
  const wrap = document.getElementById("operations-wrap");
  const count = document.getElementById("session-operations-count");
  if (!wrap || !count) return;

  count.textContent = String(state.operations.length);

  if (!state.operations.length) {
    wrap.innerHTML = '<div class="empty-inline">No operations yet</div>';
    return;
  }

  const reversedTargets = new Set(
    state.operations
      .filter((operation) => operation.type === "reversal" && operation.reference_id)
      .map((operation) => operation.reference_id),
  );

  wrap.innerHTML = state.operations
    .map((operation) => {
      const playerName = findPlayerName(operation.player_id);
      const reversible =
        state.session?.status === "active" &&
        operation.type !== "reversal" &&
        !reversedTargets.has(operation.id);

      return `
        <div class="operation-row">
          <div class="row-main">
            <div class="row-title">
              <span class="operation-type ${escapeHtml(operation.type)}">${escapeHtml(operation.type)}</span>
              ${escapeHtml(playerName)}
            </div>
            <div class="inline-stats">
              <span>Chips: ${formatNumber(operation.chips)}</span>
              <span>${escapeHtml(formatDate(operation.created_at))}</span>
            </div>
          </div>
          ${
            reversible
              ? `<button type="button" class="secondary" data-reverse-operation="${escapeHtml(operation.id)}">Reverse</button>`
              : '<span class="muted">-</span>'
          }
        </div>
      `;
    })
    .join("");

  wrap.querySelectorAll("[data-reverse-operation]").forEach((button) => {
    button.addEventListener("click", async () => {
      const operationId = button.getAttribute("data-reverse-operation");
      if (!operationId) return;
      await confirmReverse(operationId);
    });
  });
}

export function renderActionPlayerOptions() {
  const select = document.getElementById("action-player-select");
  if (!select) return;

  const current = select.value;
  const options = [
    '<option value="">Select player</option>',
    ...state.players.map((player) => {
      const id = player.player_id || player.id;
      const name = player.player_name || player.name || id;
      return `<option value="${escapeHtml(id)}">${escapeHtml(name)}</option>`;
    }),
  ];

  select.innerHTML = options.join("");
  const exists = state.players.some((player) => {
    const id = player.player_id || player.id;
    return id === current;
  });
  select.value = exists ? current : "";
}

export function initSessionActions() {
  document.addEventListener("click", async (event) => {
    const button = event.target.closest("button");
    if (!button) return;

    switch (button.id) {
      case "buy-in-btn":
        await confirmBuyIn();
        break;
      case "cash-out-btn":
        await confirmCashOut();
        break;
      case "session-add-existing-player-btn":
        await confirmAddExistingPlayer();
        break;
      case "session-add-new-player-btn":
        await confirmAddNewPlayer();
        break;
      case "finish-session-btn":
        await confirmFinishSession();
        break;
      case "session-back-home-btn":
        setScreen("lobby");
        break;
      case "player-back-home-btn":
        setScreen("lobby");
        break;
      case "player-back-session-btn":
        if (state.activeSessionId) {
          setScreen("session");
        } else {
          setScreen("lobby");
        }
        break;
      default:
        break;
    }
  });
}

async function confirmBuyIn() {
  const playerId = document.getElementById("action-player-select")?.value;
  const chips = Number(document.getElementById("action-chips")?.value);

  if (!playerId || !Number.isFinite(chips) || chips <= 0) {
    showNotice("Select a player and enter a valid chip amount.", "error");
    return;
  }

  const playerName = findPlayerName(playerId);
  const values = await openModal({
    title: "Confirm Buy In",
    description: `Add ${formatNumber(chips)} chips for ${playerName}?`,
    confirmText: "Confirm Buy In",
  });
  if (!values) return;

  const res = await buyIn({
    sessionId: state.activeSessionId,
    playerId,
    chips,
  });
  if (!res.ok) {
    showNotice(describeError(res, "Failed to apply buy in"), "error");
    return;
  }

  await refreshSessionData();
  document.getElementById("action-chips").value = "";
  showNotice(`Buy in recorded for ${playerName}.`, "success");
}

async function confirmCashOut() {
  const playerId = document.getElementById("action-player-select")?.value;
  const chips = Number(document.getElementById("action-chips")?.value);

  if (!playerId || !Number.isFinite(chips) || chips <= 0) {
    showNotice("Select a player and enter a valid chip amount.", "error");
    return;
  }

  const playerName = findPlayerName(playerId);
  const values = await openModal({
    title: "Confirm Cash Out",
    description: `Cash out ${formatNumber(chips)} chips for ${playerName}?`,
    confirmText: "Confirm Cash Out",
  });
  if (!values) return;

  const res = await cashOut({
    sessionId: state.activeSessionId,
    playerId,
    chips,
  });
  if (!res.ok) {
    showNotice(describeError(res, "Failed to apply cash out"), "error");
    return;
  }

  await refreshSessionData();
  document.getElementById("action-chips").value = "";
  showNotice(`Cash out recorded for ${playerName}.`, "success");
}

async function confirmAddExistingPlayer() {
  await loadPlayersOverview();

  const currentIds = new Set(
    state.players.map((player) => player.player_id || player.id),
  );
  const availablePlayers = state.overviewPlayers.filter(
    (player) => !currentIds.has(player.player_id),
  );

  if (!availablePlayers.length) {
    showNotice("No available players to add. Create a new player instead.", "info");
    return;
  }

  const values = await openModal({
    title: "Add Player to Session",
    description:
      "The player will appear in the session after the first buy in. This matches the current backend flow.",
    confirmText: "Add to Session",
    fields: [
      {
        name: "player_id",
        label: "Player",
        type: "select",
        options: availablePlayers.map((player) => ({
          value: player.player_id,
          label: player.player_name || player.player_id,
        })),
      },
      {
        name: "chips",
        label: "Initial Buy In",
        type: "number",
        min: "1",
        placeholder: "Chips",
      },
    ],
  });
  if (!values) return;

  const chips = Number(values.chips);
  if (!values.player_id || !Number.isFinite(chips) || chips <= 0) {
    showNotice("Choose a player and enter a valid initial buy in.", "error");
    return;
  }

  const res = await buyIn({
    sessionId: state.activeSessionId,
    playerId: values.player_id,
    chips,
  });
  if (!res.ok) {
    showNotice(describeError(res, "Failed to add player to session"), "error");
    return;
  }

  await refreshSessionData();
  showNotice(`Player ${findPlayerName(values.player_id)} added to session.`, "success");
}

async function confirmAddNewPlayer() {
  const values = await openModal({
    title: "Create New Player",
    description:
      "A new player is created in the global player list and immediately added to this session through the first buy in.",
    confirmText: "Create and Add",
    fields: [
      {
        name: "name",
        label: "Player Name",
        type: "text",
        placeholder: "Enter player name",
      },
      {
        name: "chips",
        label: "Initial Buy In",
        type: "number",
        min: "1",
        placeholder: "Chips",
      },
    ],
  });
  if (!values) return;

  const name = (values.name || "").trim();
  const chips = Number(values.chips);
  if (!name || !Number.isFinite(chips) || chips <= 0) {
    showNotice("Enter player name and a valid initial buy in.", "error");
    return;
  }

  const createRes = await createPlayer(name);
  if (!createRes.ok || !createRes.body?.player_id) {
    showNotice(describeError(createRes, "Failed to create player"), "error");
    return;
  }

  const buyInRes = await buyIn({
    sessionId: state.activeSessionId,
    playerId: createRes.body.player_id,
    chips,
  });
  if (!buyInRes.ok) {
    showNotice(describeError(buyInRes, "Player created, but add to session failed"), "error");
    return;
  }

  await Promise.all([refreshSessionData(), loadPlayersOverview()]);
  showNotice(`Player ${name} created and added to session.`, "success");
}

async function confirmFinishSession() {
  if (state.session?.status !== "active") return;

  const values = await openModal({
    title: "Finish Session",
    description:
      "Finish the session now? The backend allows this only when total buy in equals total cash out.",
    confirmText: "Finish Session",
  });
  if (!values) return;

  const res = await finishSession({ sessionId: state.activeSessionId });
  if (!res.ok) {
    showNotice(describeError(res, "Failed to finish session"), "error");
    return;
  }

  await refreshSessionData();
  showNotice("Session finished.", "success");
}

async function confirmReverse(operationId) {
  const operation = state.operations.find((item) => item.id === operationId);
  if (!operation) return;

  const values = await openModal({
    title: "Reverse Operation",
    description: `Reverse ${operation.type} for ${findPlayerName(operation.player_id)} with ${formatNumber(operation.chips)} chips?`,
    confirmText: "Reverse",
  });
  if (!values) return;

  const res = await reverseOperation({ operationId });
  if (!res.ok) {
    showNotice(describeError(res, "Failed to reverse operation"), "error");
    return;
  }

  await refreshSessionData();
  showNotice("Operation reversed.", "success");
}

async function refreshSessionData() {
  const id = state.activeSessionId;
  if (!id) return;

  const res = await getSession(id);
  if (!res.ok || !res.body) {
    showNotice(describeError(res, "Failed to refresh session"), "error");
    return;
  }

  hydrateSession(res.body);
  renderSession();

  await Promise.all([
    loadPlayers(id),
    loadOperations(id),
    loadSessions(),
    loadPlayersOverview(),
  ]);
  renderActionPlayerOptions();
}

function hydrateSession(raw) {
  state.session = {
    id: raw.session_id,
    status: raw.status,
    chipRate: raw.chip_rate,
    createdAt: raw.created_at,
    totalBuyIn: raw.total_buy_in,
    totalCashOut: raw.total_cash_out,
    totalChips: raw.total_chips,
  };
}

function findPlayerName(playerId) {
  const inSession = state.players.find((player) => {
    const id = player.player_id || player.id;
    return id === playerId;
  });
  if (inSession) {
    return inSession.player_name || inSession.name || playerId;
  }

  const overview = state.overviewPlayers.find((player) => player.player_id === playerId);
  return overview?.player_name || playerId;
}
