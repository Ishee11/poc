package usecase

import (
	"testing"

	"github.com/ishee11/poc/internal/entity"
)

func TestAssignPlayerRanks(t *testing.T) {
	older := "2026-04-01T10:00:00Z"
	newer := "2026-04-02T10:00:00Z"

	players := []PlayerStat{
		{PlayerID: "lucky", SessionsCount: 1, ProfitMoney: 5000, LastActivityAt: &newer},
		{PlayerID: "short_loss", SessionsCount: 2, ProfitMoney: -7000, LastActivityAt: &newer},
		{PlayerID: "regular", SessionsCount: 5, ProfitMoney: 1000, LastActivityAt: &older},
		{PlayerID: "shark", SessionsCount: 6, ProfitMoney: 3000, LastActivityAt: &older},
		{PlayerID: "main_fish", SessionsCount: 6, ProfitMoney: -4000, LastActivityAt: &older},
		{PlayerID: "grinder", SessionsCount: 7, ProfitMoney: 2000, LastActivityAt: &older},
	}

	ranked := assignPlayerRanks(players)
	got := ranksByPlayer(ranked)

	assertRank(t, got, "lucky", PlayerRankPositive)
	assertRank(t, got, "short_loss", PlayerRankFish)
	assertRank(t, got, "regular", PlayerRankRegular)
	assertRank(t, got, "shark", PlayerRankShark)
	assertRank(t, got, "main_fish", PlayerRankMainFish)
	assertRank(t, got, "grinder", PlayerRankGrinder)
}

func TestAssignPlayerRanksUsesMinimumQualification(t *testing.T) {
	players := []PlayerStat{
		{PlayerID: "one", SessionsCount: 1, ProfitMoney: 100},
		{PlayerID: "two", SessionsCount: 2, ProfitMoney: -100},
		{PlayerID: "three", SessionsCount: 3, ProfitMoney: 10},
	}

	ranked := assignPlayerRanks(players)
	got := ranksByPlayer(ranked)

	assertRank(t, got, "one", PlayerRankPositive)
	assertRank(t, got, "two", PlayerRankFish)
	assertRank(t, got, "three", PlayerRankShark)
}

func TestAssignPlayerRanksTieBreaksBySessionsThenActivity(t *testing.T) {
	older := "2026-04-01T10:00:00Z"
	newer := "2026-04-02T10:00:00Z"

	players := []PlayerStat{
		{PlayerID: "short_shark", SessionsCount: 3, ProfitMoney: 1000, LastActivityAt: &newer},
		{PlayerID: "long_shark", SessionsCount: 4, ProfitMoney: 1000, LastActivityAt: &older},
		{PlayerID: "old_fish", SessionsCount: 3, ProfitMoney: -1000, LastActivityAt: &older},
		{PlayerID: "new_fish", SessionsCount: 3, ProfitMoney: -1000, LastActivityAt: &newer},
	}

	ranked := assignPlayerRanks(players)
	got := ranksByPlayer(ranked)

	assertRank(t, got, "long_shark", PlayerRankShark)
	assertRank(t, got, "new_fish", PlayerRankMainFish)
}

func ranksByPlayer(players []PlayerStat) map[entity.PlayerID]string {
	result := make(map[entity.PlayerID]string, len(players))
	for _, player := range players {
		result[player.PlayerID] = player.Rank.Code
	}
	return result
}

func assertRank(t *testing.T, ranks map[entity.PlayerID]string, playerID entity.PlayerID, want string) {
	t.Helper()
	if got := ranks[playerID]; got != want {
		t.Fatalf("rank for %s = %s, want %s", playerID, got, want)
	}
}
