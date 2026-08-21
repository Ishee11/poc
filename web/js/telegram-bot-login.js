export const TELEGRAM_BOT_CHALLENGE_KEY = "poker-telegram-bot-challenge-v1";

export function telegramAppURI(botUsername, challenge) {
  const username = String(botUsername || "").replace(/^@/, "");
  return `tg://resolve?${new URLSearchParams({ domain: username, start: challenge }).toString()}`;
}

export function saveTelegramBotChallenge(challenge, storage = globalThis.sessionStorage) {
  try { storage?.setItem(TELEGRAM_BOT_CHALLENGE_KEY, JSON.stringify(challenge)); } catch {}
  return challenge;
}

export function loadTelegramBotChallenge(now = Date.now(), storage = globalThis.sessionStorage) {
  try {
    const raw = storage?.getItem(TELEGRAM_BOT_CHALLENGE_KEY);
    if (!raw) return null;
    const value = JSON.parse(raw);
    if (!value?.challenge || !value?.bot_username || !value?.verification_code || Date.parse(value.expires_at) <= now) {
      storage?.removeItem(TELEGRAM_BOT_CHALLENGE_KEY); return null;
    }
    return value;
  } catch { return null; }
}

export function clearTelegramBotChallenge(storage = globalThis.sessionStorage) {
  try { storage?.removeItem(TELEGRAM_BOT_CHALLENGE_KEY); } catch {}
}

export function telegramChallengeAction(status) {
  if (status === "approved") return "complete";
  if (status === "denied") return "denied";
  if (status === "expired" || status === "consumed") return "expired";
  return "poll";
}
