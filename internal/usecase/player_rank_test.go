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
		{PlayerID: "captain", SessionsCount: 5, ProfitMoney: 2500, PositiveStreak: 5, LastActivityAt: &newer},
		{PlayerID: "sponsor", SessionsCount: 6, ProfitMoney: -4000, TotalBuyIn: 9000, LastActivityAt: &older},
		{PlayerID: "main_fish", SessionsCount: 6, ProfitMoney: -1000, TotalBuyIn: 6000, LastActivityAt: &older},
		{PlayerID: "grinder", SessionsCount: 7, ProfitMoney: 2000, LastActivityAt: &older},
		{PlayerID: "maniac", SessionsCount: 5, ProfitMoney: 2000, TotalBuyIn: 20000, LastActivityAt: &older},
		{PlayerID: "ludoman", SessionsCount: 5, ProfitMoney: -500, TotalBuyIn: 30000, LastActivityAt: &older},
	}

	ranked := assignPlayerRanks(players)
	got := ranksByPlayer(ranked)

	assertRank(t, got, "lucky", PlayerRankNewcomer)
	assertRank(t, got, "short_loss", PlayerRankNewcomer)
	assertRank(t, got, "regular", PlayerRankRegular)
	assertRank(t, got, "shark", PlayerRankShark)
	assertRank(t, got, "captain", PlayerRankCaptain)
	assertRank(t, got, "sponsor", PlayerRankSponsor)
	assertRank(t, got, "main_fish", PlayerRankMainFish)
	assertRank(t, got, "grinder", PlayerRankGrinder)
	assertRank(t, got, "maniac", PlayerRankManiac)
	assertRank(t, got, "ludoman", PlayerRankLudoman)
}

func TestAssignPlayerRanksUsesMinimumQualification(t *testing.T) {
	players := []PlayerStat{
		{PlayerID: "one", SessionsCount: 1, ProfitMoney: 100},
		{PlayerID: "two", SessionsCount: 2, ProfitMoney: -100},
		{PlayerID: "three", SessionsCount: 3, ProfitMoney: 10},
		{PlayerID: "four", SessionsCount: 4, ProfitMoney: 10},
	}

	ranked := assignPlayerRanks(players)
	got := ranksByPlayer(ranked)

	assertRank(t, got, "one", PlayerRankNewcomer)
	assertRank(t, got, "two", PlayerRankNewcomer)
	assertRank(t, got, "three", PlayerRankNewcomer)
	assertRank(t, got, "four", PlayerRankShark)
}

func TestAssignPlayerRanksTieBreaksBySessionsThenActivity(t *testing.T) {
	older := "2026-04-01T10:00:00Z"
	newer := "2026-04-02T10:00:00Z"

	players := []PlayerStat{
		{PlayerID: "short_shark", SessionsCount: 3, ProfitMoney: 1000, LastActivityAt: &newer},
		{PlayerID: "long_shark", SessionsCount: 4, ProfitMoney: 1000, LastActivityAt: &older},
		{PlayerID: "old_fish", SessionsCount: 4, ProfitMoney: -1000, LastActivityAt: &older},
		{PlayerID: "new_fish", SessionsCount: 4, ProfitMoney: -1000, LastActivityAt: &newer},
	}

	ranked := assignPlayerRanks(players)
	got := ranksByPlayer(ranked)

	assertRank(t, got, "long_shark", PlayerRankShark)
	assertRank(t, got, "new_fish", PlayerRankSponsor)
}

func TestAssignPlayerRanksNewcomerHasPriorityForFewSessions(t *testing.T) {
	players := []PlayerStat{
		{PlayerID: "small_winner", SessionsCount: 3, ProfitMoney: 10000, PositiveStreak: 5},
		{PlayerID: "small_loser", SessionsCount: 3, ProfitMoney: -10000},
		{PlayerID: "shark", SessionsCount: 4, ProfitMoney: 100},
		{PlayerID: "sponsor", SessionsCount: 4, ProfitMoney: -100},
	}

	ranked := assignPlayerRanks(players)
	got := ranksByPlayer(ranked)

	assertRank(t, got, "small_winner", PlayerRankNewcomer)
	assertRank(t, got, "small_loser", PlayerRankNewcomer)
	assertRank(t, got, "shark", PlayerRankShark)
	assertRank(t, got, "sponsor", PlayerRankSponsor)
}

func TestAssignPlayerRanksAverageBuyInTieBreaksBySessionsThenActivity(t *testing.T) {
	older := "2026-04-01T10:00:00Z"
	newer := "2026-04-02T10:00:00Z"

	players := []PlayerStat{
		{PlayerID: "shark", SessionsCount: 5, ProfitMoney: 3000, TotalBuyIn: 5000},
		{PlayerID: "grinder", SessionsCount: 7, ProfitMoney: 1000, TotalBuyIn: 7000},
		{PlayerID: "short_maniac", SessionsCount: 3, ProfitMoney: 1000, TotalBuyIn: 6000, LastActivityAt: &newer},
		{PlayerID: "long_maniac", SessionsCount: 4, ProfitMoney: 1000, TotalBuyIn: 8000, LastActivityAt: &older},
		{PlayerID: "sponsor", SessionsCount: 5, ProfitMoney: -3000, TotalBuyIn: 5000},
		{PlayerID: "old_ludoman", SessionsCount: 4, ProfitMoney: -1000, TotalBuyIn: 8000, LastActivityAt: &older},
		{PlayerID: "new_ludoman", SessionsCount: 4, ProfitMoney: -1000, TotalBuyIn: 8000, LastActivityAt: &newer},
	}

	ranked := assignPlayerRanks(players)
	got := ranksByPlayer(ranked)

	assertRank(t, got, "long_maniac", PlayerRankManiac)
	assertRank(t, got, "new_ludoman", PlayerRankLudoman)
}

func TestAssignPlayerRanksCaptainLabelIncludesStreak(t *testing.T) {
	players := []PlayerStat{
		{PlayerID: "shark", SessionsCount: 6, ProfitMoney: 3000},
		{PlayerID: "captain", SessionsCount: 5, ProfitMoney: 1000, PositiveStreak: 7},
	}

	ranked := assignPlayerRanks(players)
	for _, player := range ranked {
		if player.PlayerID != "captain" {
			continue
		}
		if player.Rank.Label != "Капитан занос x7" {
			t.Fatalf("captain label = %s, want Капитан занос x7", player.Rank.Label)
		}
		return
	}
	t.Fatal("captain player not found")
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
