import assert from "node:assert/strict";
import test from "node:test";

import {
  TELEGRAM_AUTH_ATTEMPT_KEY,
  TELEGRAM_AUTH_ATTEMPT_TTL_MS,
  beginTelegramAuthAttempt,
  clearTelegramAuthAttempt,
  consumeTelegramAuthAttempt,
} from "../web/js/telegram-auth-flow.js";

function memoryStorage() {
  const values = new Map();
  return {
    getItem(key) { return values.get(key) ?? null; },
    setItem(key, value) { values.set(key, value); },
    removeItem(key) { values.delete(key); },
  };
}

test("records and consumes only bounded non-sensitive attempt metadata", () => {
  const storage = memoryStorage();
  beginTelegramAuthAttempt({ mode: "login", now: 1_000, storage });

  const raw = storage.getItem(TELEGRAM_AUTH_ATTEMPT_KEY);
  assert.deepEqual(JSON.parse(raw), { mode: "login", startedAt: 1_000 });
  assert.doesNotMatch(raw, /state|code|token|nonce|verifier/i);

  assert.deepEqual(consumeTelegramAuthAttempt({ now: 2_500, storage }), {
    mode: "login",
    startedAt: 1_000,
    ageMs: 1_500,
  });
  assert.equal(storage.getItem(TELEGRAM_AUTH_ATTEMPT_KEY), null);
});

test("normalizes mode and silently discards stale or malformed markers", () => {
  const storage = memoryStorage();
  beginTelegramAuthAttempt({ mode: "unexpected", now: 10, storage });
  assert.equal(consumeTelegramAuthAttempt({ now: 11, storage }).mode, "login");

  storage.setItem(TELEGRAM_AUTH_ATTEMPT_KEY, JSON.stringify({ mode: "link", startedAt: 1 }));
  assert.equal(
    consumeTelegramAuthAttempt({ now: TELEGRAM_AUTH_ATTEMPT_TTL_MS + 2, storage }),
    null,
  );

  storage.setItem(TELEGRAM_AUTH_ATTEMPT_KEY, "not-json");
  assert.equal(consumeTelegramAuthAttempt({ now: 100, storage }), null);
});

test("clear removes recovery state without authentication side effects", () => {
  const storage = memoryStorage();
  beginTelegramAuthAttempt({ mode: "link", storage });
  clearTelegramAuthAttempt({ storage });
  assert.equal(storage.getItem(TELEGRAM_AUTH_ATTEMPT_KEY), null);
});
