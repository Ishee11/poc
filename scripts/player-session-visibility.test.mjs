import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

import {
  financialVisibilityNoticeRequired,
  playerSessionListCount,
  playerSessionVisibility,
  sessionHistoryMessageKey,
} from "../web/js/player-session-visibility.js";

test("distinguishes all required total and visible session states", () => {
  const cases = [
    [{ total_sessions_count: 0, visible_sessions_count: 0 }, "empty", 0, 0],
    [{ total_sessions_count: 10, visible_sessions_count: 10 }, "complete", 10, 10],
    [{ total_sessions_count: 10, visible_sessions_count: 4 }, "partial", 10, 4],
    [{ total_sessions_count: 10, visible_sessions_count: 0 }, "hidden", 10, 0],
  ];

  for (const [response, kind, total, visible] of cases) {
    assert.deepEqual(playerSessionVisibility(response), { kind, total, visible });
  }
});

test("financial result does not infer hidden sessions", () => {
  const withProfit = playerSessionVisibility({
    total_sessions_count: 10,
    visible_sessions_count: 0,
    player: { profit_money: 100000 },
  });
  const withoutProfit = playerSessionVisibility({
    total_sessions_count: 10,
    visible_sessions_count: 0,
    player: { profit_money: 0 },
  });

  assert.deepEqual(withProfit, withoutProfit);
  assert.equal(withProfit.kind, "hidden");
  assert.equal(sessionHistoryMessageKey(withProfit), "player.sessionsUnavailableEmpty");
  assert.equal(financialVisibilityNoticeRequired(withProfit), true);
});

test("missing or invalid explicit counts are unavailable rather than zero", () => {
  for (const response of [
    { player: { sessions_count: 10 } },
    { total_sessions_count: "not-a-number", visible_sessions_count: 0 },
    { total_sessions_count: 10 },
    { total_sessions_count: 4, visible_sessions_count: 10 },
  ]) {
    const visibility = playerSessionVisibility(response);
    assert.equal(visibility.kind, "unavailable");
    assert.equal(sessionHistoryMessageKey(visibility), "player.sessionHistoryUnavailable");
  }
});

test("only restricted or unavailable histories require a financial notice", () => {
  assert.equal(financialVisibilityNoticeRequired(playerSessionVisibility({ total_sessions_count: 0, visible_sessions_count: 0 })), false);
  assert.equal(financialVisibilityNoticeRequired(playerSessionVisibility({ total_sessions_count: 10, visible_sessions_count: 10 })), false);
  assert.equal(financialVisibilityNoticeRequired(playerSessionVisibility({ total_sessions_count: 10, visible_sessions_count: 4 })), true);
});

test("session list count distinguishes visible sessions from the total", () => {
  assert.deepEqual(
    playerSessionListCount(playerSessionVisibility({ total_sessions_count: 53, visible_sessions_count: 6 }), 6),
    { visible: 6, total: 53 },
  );
  assert.deepEqual(
    playerSessionListCount(playerSessionVisibility({ total_sessions_count: 6, visible_sessions_count: 6 }), 6),
    { visible: 6, total: null },
  );
});

test("player session availability has a dedicated second row and safe shell refresh", () => {
  const playerUI = readFileSync(new URL("../web/js/ui/player.js", import.meta.url), "utf8");
  const styles = readFileSync(new URL("../web/css/main.css", import.meta.url), "utf8");
  const serviceWorker = readFileSync(new URL("../web/sw.js", import.meta.url), "utf8");

  assert.match(playerUI, /class="stat player-session-count-stat"/);
  assert.match(playerUI, /player\.sessionsVisibleOfTotal/);
  assert.match(styles, /\.player-session-count-stat \.stat-context\s*\{[^}]*grid-column:\s*1 \/ -1/s);
  assert.match(serviceWorker, /pathname\.startsWith\("\/player\/"\)/);
  assert.match(serviceWorker, /pathname === "\/players\/stats"/);
});

test("player detail uses a flat visual hierarchy", () => {
  const styles = readFileSync(new URL("../web/css/main.css", import.meta.url), "utf8");

  assert.match(styles, /body\[data-screen="player"\] #screen-player > \.panel\s*\{[^}]*border:\s*0;[^}]*background:\s*transparent;/s);
  assert.match(styles, /\.player-sessions-disclosure\s*\{[^}]*border:\s*0;[^}]*background:\s*transparent;/s);
  assert.match(styles, /\.player-sessions-disclosure > \.disclosure-body\s*\{[^}]*padding:\s*0;/s);
  assert.match(styles, /\.player-sessions-disclosure \.mobile-player-session-card \.card-meta-item\s*\{[^}]*border:\s*0;[^}]*background:\s*transparent;/s);
});
