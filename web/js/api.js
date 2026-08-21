import {
  createRequestClient,
  createRequestId,
  serializeBuyInCommand,
  serializeCashOutCommand,
  serializeReverseOperationCommand,
} from "./network-contract.js";

const API = window.location.origin;

// ===== core =====

const request = createRequestClient({ baseURL: API });

function withRequestId(resultPromise, requestId) {
  return resultPromise.then((result) => ({ ...result, requestId }));
}

export function apiGet(path, { timeoutMs } = {}) {
  return request(path, { timeoutMs });
}

export function apiPost(path, body, { requestId = rid(), timeoutMs } = {}) {
  return withRequestId(
    request(path, {
      method: "POST",
      timeoutMs,
      body: body == null ? undefined : JSON.stringify({ ...body, request_id: requestId }),
    }),
    requestId,
  );
}

export function apiPut(path, body, { requestId = rid(), timeoutMs } = {}) {
  return withRequestId(
    request(path, {
      method: "PUT",
      timeoutMs,
      body: JSON.stringify({ ...(body || {}), request_id: requestId }),
    }),
    requestId,
  );
}

export function apiDelete(path, body, { requestId = rid(), timeoutMs } = {}) {
  return withRequestId(
    request(path, {
      method: "DELETE",
      timeoutMs,
      body: body == null ? undefined : JSON.stringify({ ...body, request_id: requestId }),
    }),
    requestId,
  );
}

// ===== auth =====

export function getAuthConfig() {
  return request("/auth/config");
}

export function login({ email, password }) {
  return request("/auth/login", {
    method: "POST",
    body: JSON.stringify({ email, password, request_id: rid() }),
  });
}

export function register({ email, password, player }) {
	return request("/auth/register", {
		method: "POST",
		body: JSON.stringify({ email, password, player, request_id: rid() }),
  });
}

export function logout() {
  return request("/auth/logout", {
    method: "POST",
    body: JSON.stringify({ request_id: rid() }),
  });
}

export function getCurrentUser() {
  return request("/auth/me");
}

// ===== account =====

export function getAccount() {
  return request("/account");
}

export function getAccountAvailablePlayers({ limit, offset } = {}) {
  const params = new URLSearchParams();
  if (Number.isFinite(limit)) params.set("limit", String(limit));
  if (Number.isFinite(offset)) params.set("offset", String(offset));

  const suffix = params.toString() ? `?${params.toString()}` : "";
  return request(`/account/players/available${suffix}`);
}

export function selectAccountPlayer(player) {
	return request("/account/player", {
		method: "PUT",
		body: JSON.stringify({ ...player, request_id: rid() }),
	});
}

export function getAdminAccounts({ query = "", limit = 50, offset = 0 } = {}) {
	const params = new URLSearchParams({ query, limit: String(limit), offset: String(offset) });
	return request(`/admin/accounts?${params.toString()}`);
}

export function replaceAdminAccountPlayer(userId, playerId) {
	return request(`/admin/accounts/${encodeURIComponent(userId)}/player`, {
		method: "PUT",
		body: JSON.stringify({ player_id: playerId, request_id: rid() }),
	});
}

export function clearAdminAccountPlayer(userId) {
	return request(`/admin/accounts/${encodeURIComponent(userId)}/player`, {
		method: "DELETE",
		body: JSON.stringify({ request_id: rid() }),
	});
}

// ===== utils =====

function rid() {
  return createRequestId();
}

function guestPlayerIdForRequest() {
  try {
    return globalThis.localStorage?.getItem("poker-guest-player-id") || "";
  } catch {
    return "";
  }
}

function guestAccessSuffix() {
  const guestPlayerId = guestPlayerIdForRequest();
  return guestPlayerId
    ? `?guest_player_id=${encodeURIComponent(guestPlayerId)}`
    : "";
}

// ===== sessions =====

export function startSession({ sessionId, chipRate, bigBlind, currency }) {
  return request("/sessions/start", {
    method: "POST",
    body: JSON.stringify({
      // session_id: sessionId,
      chip_rate: chipRate,
      big_blind: bigBlind,
      currency,
      request_id: rid(),
    }),
  });
}

export function finishSession({ sessionId }) {
  return request(`/sessions/finish${guestAccessSuffix()}`, {
    method: "POST",
    body: JSON.stringify({
      session_id: sessionId,
      request_id: rid(),
    }),
  });
}

function sessionAccessParams(sessionId, guestPlayerId) {
  const params = new URLSearchParams({ session_id: sessionId });
  if (guestPlayerId) params.set("guest_player_id", guestPlayerId);
  return params.toString();
}

export function getSession(sessionId, { guestPlayerId } = {}) {
  return request(`/sessions?${sessionAccessParams(sessionId, guestPlayerId)}`);
}

export function getSessions({ guestPlayerId, limit, offset, status } = {}) {
  const params = new URLSearchParams();
  if (guestPlayerId) params.set("guest_player_id", guestPlayerId);
  if (Number.isFinite(limit)) params.set("limit", String(limit));
  if (Number.isFinite(offset)) params.set("offset", String(offset));
  if (status && status !== "all") params.set("status", status);

  const suffix = params.toString() ? `?${params.toString()}` : "";
  return request(`/stats/sessions${suffix}`);
}

export function getPlayersStats({ limit = 200, from, to } = {}) {
  const params = new URLSearchParams();
  if (Number.isFinite(limit)) params.set("limit", String(limit));
  if (from) params.set("from", from);
  if (to) params.set("to", to);

  const suffix = params.toString() ? `?${params.toString()}` : "";
  return request(`/stats/players${suffix}`);
}

export function getPlayers({ limit, offset } = {}) {
  const params = new URLSearchParams();
  if (Number.isFinite(limit)) params.set("limit", String(limit));
  if (Number.isFinite(offset)) params.set("offset", String(offset));

  const suffix = params.toString() ? `?${params.toString()}` : "";
  return request(`/players${suffix}`);
}

export function getUnlinkedPlayers({ limit, offset } = {}) {
  const params = new URLSearchParams();
  if (Number.isFinite(limit)) params.set("limit", String(limit));
  if (Number.isFinite(offset)) params.set("offset", String(offset));

  const suffix = params.toString() ? `?${params.toString()}` : "";
  return request(`/players/unlinked${suffix}`);
}

export function getSessionPlayers(sessionId, { guestPlayerId } = {}) {
  return request(`/sessions/players?${sessionAccessParams(sessionId, guestPlayerId)}`);
}

export function getSessionOperations(sessionId, { guestPlayerId } = {}) {
  return request(`/sessions/operations?${sessionAccessParams(sessionId, guestPlayerId)}`);
}

export function getExpenses(sessionId, { guestPlayerId } = {}) {
  return request(`/expenses?${sessionAccessParams(sessionId, guestPlayerId)}`);
}

export function createExpense({ sessionId, title, amount, participants, payments }) {
  return request(`/expenses${guestAccessSuffix()}`, {
    method: "POST",
    body: JSON.stringify({
      session_id: sessionId,
      title,
      amount,
      participants,
      payments,
      request_id: rid(),
    }),
  });
}

export function closeExpenses(sessionId) {
  return request(`/expenses/close${guestAccessSuffix()}`, {
    method: "POST",
    body: JSON.stringify({ session_id: sessionId, request_id: rid() }),
  });
}

export function deleteExpense(expenseId, sessionId) {
  const params = new URLSearchParams({ expense_id: expenseId, session_id: sessionId });
  const guestPlayerId = guestPlayerIdForRequest();
  if (guestPlayerId) params.set("guest_player_id", guestPlayerId);
  return request(`/expenses?${params.toString()}`, {
    method: "DELETE",
    body: JSON.stringify({ request_id: rid() }),
  });
}

export function getSettlementTransfers(sessionId, { guestPlayerId } = {}) {
  return request(`/settlement-transfers?${sessionAccessParams(sessionId, guestPlayerId)}`);
}

export function saveSettlementTransfers(sessionId, transfers) {
  return request(`/settlement-transfers${guestAccessSuffix()}`, {
    method: "PUT",
    body: JSON.stringify({
      session_id: sessionId,
      transfers,
      request_id: rid(),
    }),
  });
}

// ===== operations =====

export function buyIn({ sessionId, playerId, chips, requestId }) {
  const command = serializeBuyInCommand({ sessionId, playerId, chips, requestId });
  return withRequestId(
    request(`/operations/buy-in${guestAccessSuffix()}`, {
      method: "POST",
      body: JSON.stringify(command.payload),
    }),
    command.requestId,
  );
}

export function cashOut({ sessionId, playerId, chips, requestId }) {
  const command = serializeCashOutCommand({ sessionId, playerId, chips, requestId });
  return withRequestId(
    request(`/operations/cash-out${guestAccessSuffix()}`, {
      method: "POST",
      body: JSON.stringify(command.payload),
    }),
    command.requestId,
  );
}

export function reverseOperation({ operationId, sessionId, requestId }) {
  const command = serializeReverseOperationCommand({ operationId, requestId });
  return withRequestId(
    request(`/operations/reverse?session_id=${encodeURIComponent(sessionId)}${guestAccessSuffix().replace("?", "&")}`, {
      method: "POST",
      body: JSON.stringify(command.payload),
    }),
    command.requestId,
  );
}

// ===== players =====

export function createPlayer(name) {
  return request("/players", {
    method: "POST",
    body: JSON.stringify({
      name,
      request_id: rid(),
    }),
  });
}

export function getPlayerStats(playerId, { from, to, guestPlayerId } = {}) {
  const params = new URLSearchParams({ player_id: playerId });
  if (from) params.set("from", from);
  if (to) params.set("to", to);
  if (guestPlayerId) params.set("guest_player_id", guestPlayerId);
  return request(`/stats/player?${params.toString()}`);
}

export function adminDeletePlayer(playerId) {
  const params = new URLSearchParams({ player_id: playerId });
  return request(`/admin/player?${params.toString()}`, {
    method: "DELETE",
    body: JSON.stringify({ request_id: rid() }),
  });
}

export function adminUpdateSessionConfig(sessionId, { chipRate, bigBlind, currency }) {
  const params = new URLSearchParams({ session_id: sessionId });
  return request(`/admin/session/config?${params.toString()}`, {
    method: "PATCH",
    body: JSON.stringify({
      chip_rate: chipRate,
      big_blind: bigBlind,
      currency,
      request_id: rid(),
    }),
  });
}

export function adminRenamePlayer(playerId, name) {
  const params = new URLSearchParams({ player_id: playerId });
  return request(`/admin/player/rename?${params.toString()}`, {
    method: "PATCH",
    body: JSON.stringify({ name, request_id: rid() }),
  });
}

export function adminDeleteSession(sessionId) {
  const params = new URLSearchParams({ session_id: sessionId });
  return request(`/admin/session?${params.toString()}`, {
    method: "DELETE",
    body: JSON.stringify({ request_id: rid() }),
  });
}

export function adminDeleteSessionFinish(sessionId) {
  const params = new URLSearchParams({ session_id: sessionId });
  return request(`/admin/session/finish?${params.toString()}`, {
    method: "DELETE",
    body: JSON.stringify({ request_id: rid() }),
  });
}

// ===== blinds clock =====

export function getBlindClock() {
  return request("/blinds-clock");
}

export function startBlindClock() {
  return apiPost("/blinds-clock/start");
}

export function pauseBlindClock() {
  return apiPost("/blinds-clock/pause");
}

export function resumeBlindClock() {
  return apiPost("/blinds-clock/resume");
}

export function resetBlindClock() {
  return apiPost("/blinds-clock/reset");
}

export function resetBlindClockToDefault() {
  return apiPost("/blinds-clock/reset-default");
}

export function previousBlindClockLevel() {
  return apiPost("/blinds-clock/previous");
}

export function nextBlindClockLevel() {
  return apiPost("/blinds-clock/next");
}

export function updateBlindClockLevels(levels) {
  return apiPut("/blinds-clock/levels", { levels });
}

export function getPushConfig() {
  return apiGet("/push/config");
}

export function getBlindClockPushStatus(endpoint) {
  return apiGet(`/push/status?endpoint=${encodeURIComponent(endpoint)}`);
}

export function subscribeBlindClockPush(subscription, userAgent = "", settings = {}) {
  return apiPost("/push/subscriptions", {
    endpoint: subscription.endpoint,
    keys: {
      auth: subscription.keys?.auth || "",
      p256dh: subscription.keys?.p256dh || "",
    },
    user_agent: userAgent,
    notify_warning_60: settings.notifyWarning60 ?? true,
    notify_warning_10: settings.notifyWarning10 ?? true,
  });
}

export function unsubscribeBlindClockPush(endpoint) {
  return apiDelete("/push/subscriptions", { endpoint });
}

export function sendBlindClockPushTest() {
  return apiPost("/push/test");
}
