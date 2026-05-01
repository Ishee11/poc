package usecase

import (
	"sort"
	"time"

	"github.com/ishee11/poc/internal/entity"
)

const (
	PlayerRankNewcomer = "newcomer"
	PlayerRankFish     = "fish"
	PlayerRankPositive = "positive"
	PlayerRankRegular  = "regular"
	PlayerRankGrinder  = "grinder"
	PlayerRankShark    = "shark"
	PlayerRankMainFish = "main_fish"
)

var playerRankLabels = map[string]string{
	PlayerRankNewcomer: "Новичок",
	PlayerRankFish:     "Рыба",
	PlayerRankPositive: "В плюсе",
	PlayerRankRegular:  "Регуляр",
	PlayerRankGrinder:  "Гриндер",
	PlayerRankShark:    "Акула",
	PlayerRankMainFish: "Главная рыба",
}

func assignPlayerRanks(players []PlayerStat) []PlayerStat {
	if len(players) == 0 {
		return players
	}

	qualificationSessions := playerRankQualificationSessions(players)
	qualified := make([]PlayerStat, 0, len(players))
	for _, player := range players {
		if player.SessionsCount >= qualificationSessions {
			qualified = append(qualified, player)
		}
	}

	sharkID := bestPlayerID(qualified, func(player PlayerStat) bool {
		return player.ProfitMoney > 0
	}, compareShark)
	mainFishID := bestPlayerID(qualified, func(player PlayerStat) bool {
		return player.ProfitMoney < 0
	}, compareMainFish)
	grinderID := bestPlayerID(qualified, func(player PlayerStat) bool {
		return player.PlayerID != sharkID && player.PlayerID != mainFishID
	}, compareGrinder)

	for i := range players {
		rank := PlayerRankNewcomer
		player := players[i]
		switch {
		case player.PlayerID == sharkID:
			rank = PlayerRankShark
		case player.PlayerID == mainFishID:
			rank = PlayerRankMainFish
		case player.PlayerID == grinderID:
			rank = PlayerRankGrinder
		case player.SessionsCount >= qualificationSessions:
			rank = PlayerRankRegular
		case player.ProfitMoney > 0:
			rank = PlayerRankPositive
		case player.ProfitMoney < 0:
			rank = PlayerRankFish
		}
		players[i].Rank = PlayerRank{
			Code:  rank,
			Label: playerRankLabels[rank],
		}
	}

	return players
}

func playerRanksByID(players []PlayerStat) map[entity.PlayerID]PlayerRank {
	ranked := assignPlayerRanks(players)
	result := make(map[entity.PlayerID]PlayerRank, len(ranked))
	for _, player := range ranked {
		result[player.PlayerID] = player.Rank
	}
	return result
}

func playerRankQualificationSessions(players []PlayerStat) int64 {
	counts := make([]int64, 0, len(players))
	for _, player := range players {
		if player.SessionsCount > 0 {
			counts = append(counts, player.SessionsCount)
		}
	}
	if len(counts) == 0 {
		return 3
	}

	sort.Slice(counts, func(i, j int) bool {
		return counts[i] < counts[j]
	})
	median := counts[len(counts)/2]
	if len(counts)%2 == 0 {
		median = (counts[len(counts)/2-1] + counts[len(counts)/2]) / 2
	}
	if median < 3 {
		return 3
	}
	return median
}

func bestPlayerID(players []PlayerStat, eligible func(PlayerStat) bool, better func(PlayerStat, PlayerStat) bool) entity.PlayerID {
	var best PlayerStat
	found := false
	for _, player := range players {
		if !eligible(player) {
			continue
		}
		if !found || better(player, best) {
			best = player
			found = true
		}
	}
	if !found {
		return ""
	}
	return best.PlayerID
}

func compareShark(left PlayerStat, right PlayerStat) bool {
	if left.ProfitMoney != right.ProfitMoney {
		return left.ProfitMoney > right.ProfitMoney
	}
	return compareActivity(left, right)
}

func compareMainFish(left PlayerStat, right PlayerStat) bool {
	if left.ProfitMoney != right.ProfitMoney {
		return left.ProfitMoney < right.ProfitMoney
	}
	return compareActivity(left, right)
}

func compareGrinder(left PlayerStat, right PlayerStat) bool {
	return compareActivity(left, right)
}

func compareActivity(left PlayerStat, right PlayerStat) bool {
	if left.SessionsCount != right.SessionsCount {
		return left.SessionsCount > right.SessionsCount
	}

	leftActivity := parseRankActivity(left.LastActivityAt)
	rightActivity := parseRankActivity(right.LastActivityAt)
	if !leftActivity.Equal(rightActivity) {
		return leftActivity.After(rightActivity)
	}

	return left.PlayerID < right.PlayerID
}

func parseRankActivity(value *string) time.Time {
	if value == nil || *value == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, *value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}
