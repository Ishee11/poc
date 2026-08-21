export const TELEGRAM_AUTH_ATTEMPT_KEY = "poker-telegram-auth-attempt-v1";
export const TELEGRAM_AUTH_ATTEMPT_TTL_MS = 10 * 60 * 1000;

function availableStorages(explicitStorage) {
  if (explicitStorage) return [explicitStorage];

  const storages = [];
  for (const name of ["localStorage", "sessionStorage"]) {
    try {
      const storage = globalThis[name];
      if (storage && !storages.includes(storage)) storages.push(storage);
    } catch {
      // Some privacy modes expose storage properties that throw on access.
    }
  }
  return storages;
}

function validMode(mode) {
  return mode === "link" ? "link" : "login";
}

export function beginTelegramAuthAttempt({
  mode = "login",
  now = Date.now(),
  storage,
} = {}) {
  const attempt = { mode: validMode(mode), startedAt: now };
  const serialized = JSON.stringify(attempt);

  for (const candidate of availableStorages(storage)) {
    try {
      candidate.setItem(TELEGRAM_AUTH_ATTEMPT_KEY, serialized);
      return attempt;
    } catch {
      // Try the next same-origin storage without blocking authentication.
    }
  }
  return attempt;
}

export function clearTelegramAuthAttempt({ storage } = {}) {
  for (const candidate of availableStorages(storage)) {
    try {
      candidate.removeItem(TELEGRAM_AUTH_ATTEMPT_KEY);
    } catch {
      // Clearing best-effort recovery state must never break startup.
    }
  }
}

export function consumeTelegramAuthAttempt({
  now = Date.now(),
  ttlMs = TELEGRAM_AUTH_ATTEMPT_TTL_MS,
  storage,
} = {}) {
  for (const candidate of availableStorages(storage)) {
    let raw = null;
    try {
      raw = candidate.getItem(TELEGRAM_AUTH_ATTEMPT_KEY);
      candidate.removeItem(TELEGRAM_AUTH_ATTEMPT_KEY);
    } catch {
      continue;
    }
    if (!raw) continue;

    try {
      const attempt = JSON.parse(raw);
      const ageMs = now - Number(attempt?.startedAt);
      if (
        (attempt?.mode !== "login" && attempt?.mode !== "link") ||
        !Number.isFinite(ageMs) ||
        ageMs < 0 ||
        ageMs > ttlMs
      ) {
        return null;
      }
      return { mode: attempt.mode, startedAt: Number(attempt.startedAt), ageMs };
    } catch {
      return null;
    }
  }
  return null;
}
