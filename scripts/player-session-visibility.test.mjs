import assert from "node:assert/strict";
import test from "node:test";

import {
  financialVisibilityNoticeRequired,
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
