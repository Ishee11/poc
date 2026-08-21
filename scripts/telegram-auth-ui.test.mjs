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
assert.deepEqual(requests, [
  { url: "https://poc.test/account/identities/telegram", method: "DELETE" },
]);

const html = readFileSync(new URL("../web/index.html", import.meta.url), "utf8");
assert.match(html, /id="auth-telegram-login"[\s\S]*\/auth\/telegram\/start\?mode=login/);
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

console.log("telegram auth UI tests passed");
