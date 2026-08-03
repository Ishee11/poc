import { isValidSessionSnapshot } from "./offline-db.js";

function isRecord(value) {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

function hasOwn(record, key) {
  return Object.prototype.hasOwnProperty.call(record, key);
}

export function normalizeSession(raw, sessionId) {
  if (!isRecord(raw)) return null;
  const id = String(raw.session_id || raw.id || sessionId || "").trim();
  if (!id || (sessionId && id !== sessionId)) return null;

  return {
    id,
    status: raw.status,
    chipRate: raw.chip_rate ?? raw.chipRate,
    bigBlind: raw.big_blind ?? raw.bigBlind,
    currency: raw.currency || "RUB",
    createdAt: raw.created_at ?? raw.createdAt,
    finishedAt: raw.finished_at ?? raw.finishedAt,
    expensesClosed: Boolean(raw.expenses_closed ?? raw.expensesClosed),
    totalBuyIn: raw.total_buy_in ?? raw.totalBuyIn,
    totalCashOut: raw.total_cash_out ?? raw.totalCashOut,
    totalChips: raw.total_chips ?? raw.totalChips,
  };
}

export function normalizeSessionPlayers(raw) {
  if (!Array.isArray(raw)) return null;
  return [...raw].sort(
    (left, right) =>
      (Number(right?.profit_money) || 0) - (Number(left?.profit_money) || 0),
  );
}

export function normalizeOperations(raw) {
  return Array.isArray(raw) ? [...raw] : null;
}

export function normalizeSettlementTransfers(raw) {
  if (!Array.isArray(raw)) return null;
  return raw
    .map((transfer, index) => {
      const from = String(transfer?.from || "");
      const to = String(transfer?.to || "");
      const amount = Number(transfer?.amount);
      if (!from || !to || !Number.isFinite(amount) || amount <= 0) return null;
      const id = String(
        transfer?.id || `settlement-${from}-${to}-${amount}-${index}`,
      );
      return { id, from, to, amount };
    })
    .filter(Boolean);
}

export function snapshotFromServerResults({
  sessionId,
  sessionResult,
  playersResult,
  operationsResult,
  expensesResult,
  settlementsResult,
  previousSnapshot,
  cachedAt = new Date().toISOString(),
  localRevision = 0,
}) {
  if (!sessionResult?.ok || !playersResult?.ok || !operationsResult?.ok) return null;

  const session = normalizeSession(sessionResult.body, sessionId);
  const players = normalizeSessionPlayers(playersResult.body);
  const operations = normalizeOperations(operationsResult.body);
  if (!session || !players || !operations) return null;

  const snapshot = {
    session_id: sessionId,
    session,
    players,
    operations,
    cached_at: cachedAt,
    local_revision: localRevision,
    last_server_refresh_status: "fresh",
  };

  const confirmedExpenses = expensesResult?.ok
    ? normalizeOperations(expensesResult.body)
    : null;
  const expenses = confirmedExpenses ?? previousSnapshot?.expenses;
  if (expenses) snapshot.expenses = expenses;

  const confirmedSettlements = settlementsResult?.ok
    ? normalizeSettlementTransfers(settlementsResult.body)
    : null;
  const settlements = confirmedSettlements ?? previousSnapshot?.settlements;
  if (settlements) snapshot.settlements = settlements;

  return isValidSessionSnapshot(snapshot) ? snapshot : null;
}

export function snapshotFromState(state, sessionId = state.activeSessionId) {
  if (!sessionId || !isRecord(state.session) || state.session.id !== sessionId) return null;
  const snapshot = {
    session_id: sessionId,
    session: { ...state.session },
    players: Array.isArray(state.players) ? [...state.players] : [],
    operations: Array.isArray(state.operations) ? [...state.operations] : [],
    cached_at: state.sessionCachedAt || new Date().toISOString(),
    local_revision: Number.isInteger(state.sessionLocalRevision)
      ? state.sessionLocalRevision
      : 0,
    last_server_refresh_status: state.sessionRefreshStatus || "idle",
  };
  if (state.sessionExpensesCached && Array.isArray(state.expenses)) {
    snapshot.expenses = [...state.expenses];
  }
  const settlements = state.settlementDrafts?.[sessionId]?.transfers;
  if (state.sessionSettlementsCached) {
    snapshot.settlements = Array.isArray(settlements) ? [...settlements] : [];
  }
  return snapshot;
}

export function applySessionSnapshot(
  state,
  snapshot,
  source,
  { refreshStatus = source === "server" ? "fresh" : "refreshing" } = {},
) {
  if (!isValidSessionSnapshot(snapshot)) return false;

  state.session = { ...snapshot.session };
  state.players = [...snapshot.players];
  state.operations = [...snapshot.operations];
  state.sessionExpensesCached = hasOwn(snapshot, "expenses");
  if (state.sessionExpensesCached) state.expenses = [...snapshot.expenses];
  state.sessionSettlementsCached = hasOwn(snapshot, "settlements");
  if (hasOwn(snapshot, "settlements")) {
    if (snapshot.settlements.length > 0) {
      state.settlementDrafts[snapshot.session_id] = {
        transfers: [...snapshot.settlements],
      };
    } else {
      delete state.settlementDrafts[snapshot.session_id];
    }
  }
  state.sessionDataSource = source;
  state.sessionCachedAt = snapshot.cached_at;
  state.sessionLocalRevision = snapshot.local_revision;
  state.sessionRefreshStatus = refreshStatus;
  return true;
}

export function isCurrentRefresh(state, { sessionId, localRevision }) {
  return (
    state.activeSessionId === sessionId &&
    state.sessionLocalRevision === localRevision
  );
}

export async function hydrateCachedSession({ sessionId, state, readSnapshot, onHydrated }) {
  const snapshot = await readSnapshot(sessionId);
  if (state.activeSessionId !== sessionId || !snapshot) return false;
  if (!applySessionSnapshot(state, snapshot, "cache")) return false;
  onHydrated?.(snapshot);
  return true;
}

export async function refreshSessionSnapshot({
  sessionId,
  state,
  loadResults,
  writeSnapshot,
  transformSnapshot = (snapshot) => snapshot,
  onApplied,
  now = () => new Date().toISOString(),
}) {
  const token = {
    sessionId,
    localRevision: state.sessionLocalRevision,
  };
  if (state.activeSessionId === sessionId) state.sessionRefreshStatus = "refreshing";
  const previousSnapshot = snapshotFromState(state, sessionId);
  let results;
  try {
    results = await loadResults();
  } catch (error) {
    if (isCurrentRefresh(state, token)) state.sessionRefreshStatus = "failed";
    return { status: "failed", snapshot: null, error };
  }
  let snapshot = snapshotFromServerResults({
    sessionId,
    ...results,
    previousSnapshot,
    cachedAt: now(),
    localRevision: token.localRevision,
  });

  if (!snapshot) {
    if (isCurrentRefresh(state, token)) state.sessionRefreshStatus = "failed";
    return { status: "failed", snapshot: null };
  }
  try {
    snapshot = transformSnapshot(snapshot, results);
  } catch (error) {
    if (isCurrentRefresh(state, token)) state.sessionRefreshStatus = "failed";
    return { status: "failed", snapshot: null, error };
  }
  if (!isCurrentRefresh(state, token)) return { status: "stale", snapshot: null };

  try {
    const written = await writeSnapshot(snapshot, token.localRevision);
    if (written === false) return { status: "stale", snapshot: null };
  } catch (error) {
    if (isCurrentRefresh(state, token)) state.sessionRefreshStatus = "failed";
    return { status: "failed", snapshot: null, error };
  }

  if (!isCurrentRefresh(state, token)) return { status: "stale", snapshot };
  applySessionSnapshot(state, snapshot, "server");
  onApplied?.(snapshot);
  return { status: "fresh", snapshot };
}
