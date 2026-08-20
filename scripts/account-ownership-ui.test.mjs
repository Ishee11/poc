#!/usr/bin/env node

import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import {
	accountRequiresOnboarding,
	buildPlayerSelection,
	ownershipConflictRequiresRefresh,
	playerContext,
} from "../web/js/account-ownership-ui.js";

assert.deepEqual(buildPlayerSelection("existing", "player-1", ""), {
	mode: "existing",
	player_id: "player-1",
});
assert.deepEqual(buildPlayerSelection("new", "", " Alice "), {
	mode: "new",
	name: "Alice",
});
assert.equal(buildPlayerSelection("existing", "", ""), null);
assert.equal(buildPlayerSelection("new", "", "   "), null);
assert.equal(accountRequiresOnboarding({ player: null, onboarding_required: true }), true);
assert.equal(accountRequiresOnboarding({ player: { player_id: "p" }, onboarding_required: false }), false);
assert.equal(ownershipConflictRequiresRefresh({ status: 409, body: { error: "player_already_linked" } }), true);
assert.equal(ownershipConflictRequiresRefresh({ status: 400, body: { error: "invalid_player_id" } }), false);
assert.match(playerContext({ player_id: "123456789", name: "Alice", sessions_count: 3 }), /Alice · 3 · - · 12345678/);

const html = readFileSync(new URL("../web/index.html", import.meta.url), "utf8");
const app = readFileSync(new URL("../web/js/app.js", import.meta.url), "utf8");
const api = readFileSync(new URL("../web/js/api.js", import.meta.url), "utf8");
for (const id of [
	"auth-player-mode",
	"auth-existing-player",
	"auth-new-player-name",
	"account-player-mode",
	"admin-account-search-form",
	"admin-account-replace",
	"admin-account-clear",
]) {
	assert.match(html, new RegExp(`id=["']${id}["']`), `missing ownership control ${id}`);
}
assert.match(app, /state\.accountOnboardingRequired[\s\S]*openAccount\(\{ replace: true \}\)/);
assert.match(api, /\/admin\/accounts\/\$\{encodeURIComponent\(userId\)\}\/player/);

console.log("account ownership UI tests passed");
