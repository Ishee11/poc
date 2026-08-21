import assert from "node:assert/strict";
import {
  clearTelegramBotChallenge,
  loadTelegramBotChallenge,
  saveTelegramBotChallenge,
  telegramChallengeAction,
  telegramAppURI,
} from "../web/js/telegram-bot-login.js";

const uri = telegramAppURI("@PokerLoginBot", "high_entropy_challenge");
assert.equal(uri, "tg://resolve?domain=PokerLoginBot&start=high_entropy_challenge");
assert.ok(!uri.includes("t.me"));
assert.ok(!uri.includes("oauth.telegram.org"));
assert.equal(telegramChallengeAction("pending"), "poll");
assert.equal(telegramChallengeAction("approved"), "complete");
assert.equal(telegramChallengeAction("denied"), "denied");
assert.equal(telegramChallengeAction("expired"), "expired");
assert.equal(telegramChallengeAction("consumed"), "expired");

const values = new Map();
const storage = {
  getItem: (key) => values.get(key) || null,
  setItem: (key, value) => values.set(key, value),
  removeItem: (key) => values.delete(key),
};
const challenge = {
  challenge: "secret",
  bot_username: "PokerLoginBot",
  verification_code: "4831",
  expires_at: "2026-08-21T10:05:00Z",
};
saveTelegramBotChallenge(challenge, storage);
assert.deepEqual(loadTelegramBotChallenge(Date.parse("2026-08-21T10:01:00Z"), storage), challenge);
assert.equal(loadTelegramBotChallenge(Date.parse("2026-08-21T10:06:00Z"), storage), null);
saveTelegramBotChallenge(challenge, storage);
clearTelegramBotChallenge(storage);
assert.equal(values.size, 0);

console.log("telegram bot login tests passed");
