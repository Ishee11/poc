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
const utils = readFileSync(new URL("../web/js/utils.js", import.meta.url), "utf8");
for (const id of [
	"header-account-btn",
	"auth-menu",
	"auth-player-mode",
	"auth-existing-player",
	"auth-new-player-name",
	"account-player-mode",
	"admin-account-search-form",
	"admin-account-replace",
	"admin-account-clear",
	"account-logout-btn",
]) {
	assert.match(html, new RegExp(`id=["']${id}["']`), `missing ownership control ${id}`);
}
for (const removedId of [
	"auth-show-login-btn",
	"auth-account-btn",
	"auth-logout-btn",
	"admin-login-disclosure",
]) {
	assert.doesNotMatch(html, new RegExp(`id=["']${removedId}["']`), `obsolete auth control ${removedId}`);
}
const timerButtonIndex = html.indexOf('id="header-blinds-clock-btn"');
const accountButtonIndex = html.indexOf('id="header-account-btn"');
const accountScreenIndex = html.indexOf('id="screen-account"');
const authMenuIndex = html.indexOf('id="auth-menu"');
const accountPanelIndex = html.indexOf('id="account-panel"');
assert.ok(timerButtonIndex >= 0 && accountButtonIndex > timerButtonIndex, "account icon must follow blind timer");
assert.ok(
	accountScreenIndex >= 0 && authMenuIndex > accountScreenIndex && accountPanelIndex > authMenuIndex,
	"authentication form must be embedded in the account screen",
);
assert.match(app, /state\.accountOnboardingRequired[\s\S]*openAccount\(\{ replace: true \}\)/);
assert.match(app, /header-account-btn[\s\S]*state\.authUser[\s\S]*openAccount\(\)/);
assert.match(app, /account-logout-btn[\s\S]*logout\(\)/);
assert.match(
	app,
	/import\s*\{[\s\S]*?\bplayerId\b[\s\S]*?\}\s*from\s*["']\.\/utils\.js["']/,
	"app bootstrap must import playerId for unauthenticated guest loading",
);
assert.match(api, /\/admin\/accounts\/\$\{encodeURIComponent\(userId\)\}\/player/);
assert.match(utils, /function routeToAccount\(\)[\s\S]*return ["']\/profile["']/);
assert.doesNotMatch(utils, /function routeToAccount\(\)[\s\S]*return ["']\/account["']/);

console.log("account ownership UI tests passed");
