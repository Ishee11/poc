package usecase

import (
	"fmt"
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
	PlayerRankSponsor  = "sponsor"
	PlayerRankLudoman  = "ludoman"
	PlayerRankManiac   = "maniac"
	PlayerRankCaptain  = "captain_run"
)

const (
	playerRankQualifiedMinSessions int64 = 5
	playerRankActiveWindow               = 60 * 24 * time.Hour
)

var playerRankLabels = map[string]string{
	PlayerRankNewcomer: "Новичок",
	PlayerRankFish:     "Рыба",
	PlayerRankPositive: "В плюсе",
	PlayerRankRegular:  "Регуляр",
	PlayerRankGrinder:  "Гриндер",
	PlayerRankShark:    "Акула",
	PlayerRankMainFish: "ГлавРыба",
	PlayerRankSponsor:  "Спонсор",
	PlayerRankLudoman:  "Лудоман",
	PlayerRankManiac:   "Маньяк",
	PlayerRankCaptain:  "Капитан занос",
}

func assignPlayerRanks(players []PlayerStat) []PlayerStat {
	if len(players) == 0 {
		return players
	}

	latestActivity := latestPlayerRankActivity(players)
	contenders := make([]PlayerStat, 0, len(players))
	for _, player := range players {
		if isSpecialRankContender(player, latestActivity) {
			contenders = append(contenders, player)
		}
	}

	sharkID := bestPlayerID(contenders, func(player PlayerStat) bool {
		return player.ProfitMoney > 0
	}, compareShark)
	sponsorID := bestPlayerID(contenders, func(player PlayerStat) bool {
		return player.ProfitMoney < 0
	}, compareSponsor)
	grinderID := bestPlayerID(contenders, func(player PlayerStat) bool {
		return player.PlayerID != sharkID && player.PlayerID != sponsorID
	}, compareGrinder)
	maniacID := bestPlayerID(contenders, func(player PlayerStat) bool {
		return player.ProfitMoney > 0 &&
			player.PlayerID != sharkID &&
			player.PlayerID != grinderID
	}, compareAverageBuyIn)
	ludomanID := bestPlayerID(contenders, func(player PlayerStat) bool {
		return player.ProfitMoney < 0 &&
			player.PlayerID != sponsorID &&
			player.PlayerID != grinderID
	}, compareAverageBuyIn)

	for i := range players {
		rank := PlayerRankNewcomer
		player := players[i]
		switch {
		case player.PositiveStreak >= 5:
			rank = PlayerRankCaptain
		case isNewcomer(player):
			rank = PlayerRankNewcomer
		case player.PlayerID == sharkID:
			rank = PlayerRankShark
		case player.PlayerID == sponsorID:
			rank = PlayerRankSponsor
		case player.PlayerID == grinderID:
			rank = PlayerRankGrinder
		case player.PlayerID == maniacID:
			rank = PlayerRankManiac
		case player.PlayerID == ludomanID:
			rank = PlayerRankLudoman
		case player.SessionsCount >= playerRankQualifiedMinSessions && player.ProfitMoney < 0:
			rank = PlayerRankMainFish
		case player.SessionsCount >= playerRankQualifiedMinSessions:
			rank = PlayerRankRegular
		case player.ProfitMoney > 0:
			rank = PlayerRankPositive
		case player.ProfitMoney < 0:
			rank = PlayerRankFish
		}
		players[i].Rank = PlayerRank{
			Code:  rank,
			Label: playerRankLabel(rank, player),
		}
	}

	return players
}

func isNewcomer(player PlayerStat) bool {
	return player.SessionsCount <= 3
}

func isSpecialRankContender(player PlayerStat, latestActivity time.Time) bool {
	if isNewcomer(player) || player.SessionsCount < playerRankQualifiedMinSessions {
		return false
	}
	activity := parseRankActivity(player.LastActivityAt)
	if activity.IsZero() || latestActivity.IsZero() {
		return false
	}
	return !activity.Before(latestActivity.Add(-playerRankActiveWindow))
}

func playerRankLabel(rank string, player PlayerStat) string {
	if rank == PlayerRankCaptain {
		return fmt.Sprintf("%s x%d", playerRankLabels[rank], player.PositiveStreak)
	}
	return playerRankLabels[rank]
}

func playerRanksByID(players []PlayerStat) map[entity.PlayerID]PlayerRank {
	ranked := assignPlayerRanks(players)
	result := make(map[entity.PlayerID]PlayerRank, len(ranked))
	for _, player := range ranked {
		result[player.PlayerID] = player.Rank
	}
	return result
}

func latestPlayerRankActivity(players []PlayerStat) time.Time {
	var latest time.Time
	for _, player := range players {
		activity := parseRankActivity(player.LastActivityAt)
		if activity.After(latest) {
			latest = activity
		}
	}
	return latest
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

func compareSponsor(left PlayerStat, right PlayerStat) bool {
	if left.ProfitMoney != right.ProfitMoney {
		return left.ProfitMoney < right.ProfitMoney
	}
	return compareActivity(left, right)
}

func compareGrinder(left PlayerStat, right PlayerStat) bool {
	return compareActivity(left, right)
}

func compareAverageBuyIn(left PlayerStat, right PlayerStat) bool {
	leftAvg := averageBuyIn(left)
	rightAvg := averageBuyIn(right)
	if leftAvg != rightAvg {
		return leftAvg > rightAvg
	}
	return compareActivity(left, right)
}

func averageBuyIn(player PlayerStat) float64 {
	if player.SessionsCount <= 0 {
		return 0
	}
	return float64(player.TotalBuyIn) / float64(player.SessionsCount)
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
