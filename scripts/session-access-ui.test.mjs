import assert from "node:assert/strict";
import { readFileSync } from "node:fs";

const requests = [];
globalThis.window = { location: { origin: "https://poc.test" } };
globalThis.fetch = async (url) => {
  requests.push(String(url));
  return new Response("{}", {
    status: 200,
    headers: { "content-type": "application/json" },
  });
};

const api = await import("../web/js/api.js");
await api.getSession("session-1", { guestPlayerId: "guest-1" });
await api.getSessionPlayers("session-1", { guestPlayerId: "guest-1" });
await api.getSessionOperations("session-1", { guestPlayerId: "guest-1" });
await api.getExpenses("session-1", { guestPlayerId: "guest-1" });
await api.getSettlementTransfers("session-1", { guestPlayerId: "guest-1" });
await api.getPlayerStats("player-1", { guestPlayerId: "guest-1" });

assert.equal(requests.length, 6);
for (const url of requests) {
  const parsed = new URL(url);
  assert.equal(parsed.searchParams.get("guest_player_id"), "guest-1", url);
}

globalThis.localStorage = {
  getItem(key) {
    return key === "poker-guest-player-id" ? "guest-write" : "";
  },
};
requests.length = 0;
await api.finishSession({ sessionId: "session-1" });
await api.buyIn({ sessionId: "session-1", playerId: "player-1", chips: 10 });
await api.cashOut({ sessionId: "session-1", playerId: "player-1", chips: 10 });
await api.reverseOperation({ operationId: "operation-1", sessionId: "session-1" });
await api.createExpense({ sessionId: "session-1", title: "Tea", amount: 10, participants: [], payments: [] });
await api.closeExpenses("session-1");
await api.deleteExpense("expense-1", "session-1");
await api.saveSettlementTransfers("session-1", []);
assert.equal(requests.length, 8);
for (const url of requests) {
  assert.equal(new URL(url).searchParams.get("guest_player_id"), "guest-write", url);
}

const sessionUI = readFileSync(new URL("../web/js/ui/session.js", import.meta.url), "utf8");
assert.match(sessionUI, /const accessResult = await getSession\(sessionId, sessionAccessOptions\(\)\)/);
assert.match(sessionUI, /ERROR_KINDS\.AUTHORIZATION|!accessResult\.ok/);
assert.match(sessionUI, /if \(!accessResult\.ok\) \{[\s\S]*hydrateCachedSession/);

console.log("session access UI tests passed");
