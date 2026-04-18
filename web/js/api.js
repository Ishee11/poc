const API = window.location.origin;

// ===== core =====

async function request(path, options = {}) {
  try {
    const res = await fetch(API + path, {
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json",
      },
      ...options,
    });

    const text = await res.text();

    let body = null;
    if (text) {
      try {
        body = JSON.parse(text);
      } catch {}
    }

    return {
      ok: res.ok,
      status: res.status,
      body,
      text,
    };
  } catch (e) {
    return { ok: false, status: 0, body: null, text: String(e) };
  }
}

export function apiGet(path) {
  return request(path);
}

// ===== utils =====

function rid() {
  return crypto.randomUUID();
}

// ===== sessions =====

export function startSession({ sessionId, chipRate }) {
  return request("/sessions/start", {
    method: "POST",
    body: JSON.stringify({
      // session_id: sessionId,
      chip_rate: chipRate,
    }),
  });
}

export function finishSession({ sessionId }) {
  return request("/sessions/finish", {
    method: "POST",
    body: JSON.stringify({
      session_id: sessionId,
      request_id: rid(),
    }),
  });
}

export function getSession(sessionId) {
  return request(`/sessions?session_id=${sessionId}`);
}

export function getSessions() {
  return request("/stats/sessions");
}

export function getPlayersStats() {
  return request("/stats/players?limit=200");
}

export function getPlayers({ limit, offset } = {}) {
  const params = new URLSearchParams();
  if (Number.isFinite(limit)) params.set("limit", String(limit));
  if (Number.isFinite(offset)) params.set("offset", String(offset));

  const suffix = params.toString() ? `?${params.toString()}` : "";
  return request(`/players${suffix}`);
}

export function getSessionPlayers(sessionId) {
  return request(`/sessions/players?session_id=${sessionId}`);
}

export function getSessionOperations(sessionId) {
  return request(`/sessions/operations?session_id=${sessionId}`);
}

// ===== operations =====

export function buyIn({ sessionId, playerId, chips }) {
  return request("/operations/buy-in", {
    method: "POST",
    body: JSON.stringify({
      session_id: sessionId,
      player_id: playerId,
      chips,
      request_id: rid(),
    }),
  });
}

export function cashOut({ sessionId, playerId, chips }) {
  return request("/operations/cash-out", {
    method: "POST",
    body: JSON.stringify({
      session_id: sessionId,
      player_id: playerId,
      chips,
      request_id: rid(),
    }),
  });
}

export function reverseOperation({ operationId }) {
  return request("/operations/reverse", {
    method: "POST",
    body: JSON.stringify({
      target_operation_id: operationId,
      request_id: rid(),
    }),
  });
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

export function getPlayerStats(playerId) {
  return request(`/stats/player?player_id=${playerId}`);
}
