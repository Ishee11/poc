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
  if (
    typeof globalThis.crypto !== "undefined" &&
    typeof globalThis.crypto.randomUUID === "function"
  ) {
    return globalThis.crypto.randomUUID();
  }

  if (
    typeof globalThis.crypto !== "undefined" &&
    typeof globalThis.crypto.getRandomValues === "function"
  ) {
    const bytes = new Uint8Array(16);
    globalThis.crypto.getRandomValues(bytes);
    bytes[6] = (bytes[6] & 0x0f) | 0x40;
    bytes[8] = (bytes[8] & 0x3f) | 0x80;
    const hex = [...bytes].map((byte) => byte.toString(16).padStart(2, "0"));
    return [
      hex.slice(0, 4).join(""),
      hex.slice(4, 6).join(""),
      hex.slice(6, 8).join(""),
      hex.slice(8, 10).join(""),
      hex.slice(10, 16).join(""),
    ].join("-");
  }

  return `req-${Date.now().toString(36)}-${Math.random()
    .toString(36)
    .slice(2, 12)}`;
}

// ===== sessions =====

export function startSession({ sessionId, chipRate, bigBlind }) {
  return request("/sessions/start", {
    method: "POST",
    body: JSON.stringify({
      // session_id: sessionId,
      chip_rate: chipRate,
      big_blind: bigBlind,
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

export function getPlayerStats(playerId, { from, to } = {}) {
  const params = new URLSearchParams({ player_id: playerId });
  if (from) params.set("from", from);
  if (to) params.set("to", to);
  return request(`/stats/player?${params.toString()}`);
}

export function debugDeletePlayer(playerId) {
  const params = new URLSearchParams({ player_id: playerId });
  return request(`/debug/player?${params.toString()}`, {
    method: "DELETE",
  });
}

export function debugRenamePlayer(playerId, name) {
  const params = new URLSearchParams({ player_id: playerId });
  return request(`/debug/player/rename?${params.toString()}`, {
    method: "PATCH",
    body: JSON.stringify({ name }),
  });
}

export function debugDeleteSession(sessionId) {
  const params = new URLSearchParams({ session_id: sessionId });
  return request(`/debug/session?${params.toString()}`, {
    method: "DELETE",
  });
}

export function debugDeleteSessionFinish(sessionId) {
  const params = new URLSearchParams({ session_id: sessionId });
  return request(`/debug/session/finish?${params.toString()}`, {
    method: "DELETE",
  });
}
