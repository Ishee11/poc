package usecase

import "github.com/ishee11/poc/internal/entity"

type GetPlayerStatsQuery struct {
	PlayerID entity.PlayerID
	From     *DateTimeRangeBound
	To       *DateTimeRangeBound
}

type GetPlayerStatsResponse struct {
	Player   PlayerOverallStat
	Sessions []PlayerSessionStat
}

type GetPlayerStatsUseCase struct {
	statsRepo StatsRepository
	txManager TxManager
}

func NewGetPlayerStatsUseCase(
	statsRepo StatsRepository,
	txManager TxManager,
) *GetPlayerStatsUseCase {
	return &GetPlayerStatsUseCase{
		statsRepo: statsRepo,
		txManager: txManager,
	}
}

func (uc *GetPlayerStatsUseCase) Execute(q GetPlayerStatsQuery) (*GetPlayerStatsResponse, error) {
	var result *GetPlayerStatsResponse

	err := uc.txManager.RunInTx(func(tx Tx) error {
		filter := PlayerStatsFilter{
			Limit: 100,
			From:  q.From,
			To:    q.To,
		}

		player, err := uc.statsRepo.GetPlayerOverall(tx, q.PlayerID, filter)
		if err != nil {
			return err
		}

		sessions, err := uc.statsRepo.ListPlayerSessions(tx, q.PlayerID, filter)
		if err != nil {
			return err
		}

		if player == nil {
			player = &PlayerOverallStat{PlayerID: q.PlayerID}
		}

		result = &GetPlayerStatsResponse{
			Player:   *player,
			Sessions: sessions,
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}
