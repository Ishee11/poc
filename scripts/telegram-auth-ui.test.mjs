import assert from "node:assert/strict";
import { readFileSync } from "node:fs";

const requests = [];
globalThis.window = { location: { origin: "https://poc.test" } };
globalThis.fetch = async (url, options = {}) => {
  requests.push({ url: String(url), method: options.method || "GET" });
  return new Response("{}", {
    status: 200,
    headers: { "content-type": "application/json" },
  });
};

const api = await import("../web/js/api.js");
await api.unlinkTelegram();
await api.createTelegramChallenge();
await api.getTelegramChallengeStatus("secret");
await api.completeTelegramChallenge("secret");
await api.cancelTelegramChallenge("secret");
assert.deepEqual(requests, [
  { url: "https://poc.test/account/identities/telegram", method: "DELETE" },
  { url: "https://poc.test/auth/telegram/challenge", method: "POST" },
  { url: "https://poc.test/auth/telegram/challenge/secret/status", method: "GET" },
  { url: "https://poc.test/auth/telegram/challenge/secret/complete", method: "POST" },
  { url: "https://poc.test/auth/telegram/challenge/secret/cancel", method: "POST" },
]);

const html = readFileSync(new URL("../web/index.html", import.meta.url), "utf8");
assert.match(html, /id="auth-telegram-login"[\s\S]*id="telegram-bot-waiting"/);
assert.match(html, /id="telegram-bot-open"/);
assert.match(html, /id="telegram-bot-browser-fallback"[\s\S]*\/auth\/telegram\/start\?mode=login/);
assert.match(html, /id="account-telegram-link"[\s\S]*\/auth\/telegram\/start\?mode=link/);
assert.match(html, /id="account-telegram-unlink"/);
assert.match(html, /id="telegram-auth-feedback"/);
assert.match(html, /id="telegram-auth-retry"/);
assert.match(html, /id="telegram-auth-dismiss"/);

const app = readFileSync(new URL("../web/js/app.js", import.meta.url), "utf8");
assert.match(app, /state\.telegramAuthAvailability = res\.body\.telegram_enabled === true/);
assert.match(app, /state\.telegramAuthAvailability !== "disabled"/);
assert.match(app, /consumeTelegramAuthAttempt\(\)/);
assert.match(app, /forceProfile: Boolean\(feedbackKind\)/);
assert.match(app, /renderTelegramAuthFeedback\(\)/);
assert.match(app, /identity\.provider === "telegram"/);
assert.match(app, /await unlinkTelegram\(\)/);
assert.match(app, /createTelegramChallenge\(\)/);
assert.match(app, /telegramAppURI\(/);
assert.match(app, /setTimeout\(\(\) => void pollTelegramChallenge\(false\), 2000\)/);
assert.match(app, /document\.visibilityState === "visible"/);
assert.match(app, /await completeTelegramChallenge\(challenge\.challenge\)/);
assert.match(app, /await loadCurrentUser\(\)/);
assert.match(app, /telegramBot\.denied/);
assert.match(app, /telegramBot\.expired/);
assert.match(app, /cancelActiveTelegramChallenge\(\)/);

console.log("telegram auth UI tests passed");
