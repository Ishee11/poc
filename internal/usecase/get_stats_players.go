package usecase

import (
	"context"

	"github.com/ishee11/poc/internal/entity"
)

type GetStatsPlayersUseCase struct {
	statsRepo StatsRepository
	txManager TxManager
}

func NewGetStatsPlayersUseCase(
	statsRepo StatsRepository,
	txManager TxManager,
) *GetStatsPlayersUseCase {
	return &GetStatsPlayersUseCase{
		statsRepo: statsRepo,
		txManager: txManager,
	}
}

func (uc *GetStatsPlayersUseCase) Execute(ctx context.Context, q GetStatsPlayersQuery) ([]PlayerStat, error) {
	var result []PlayerStat

	err := uc.txManager.RunInTx(ctx, func(tx Tx) error {
		res, err := uc.execute(tx, q)
		if err != nil {
			return err
		}
		result = res
		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (uc *GetStatsPlayersUseCase) execute(
	tx Tx,
	q GetStatsPlayersQuery,
) ([]PlayerStat, error) {

	limit := q.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	result, err := uc.statsRepo.ListPlayers(tx, PlayerStatsFilter{
		Limit: limit,
		From:  q.From,
		To:    q.To,
	})
	if err != nil {
		return nil, err
	}

	ranks, err := uc.allTimeRanks(tx)
	if err != nil {
		return nil, err
	}
	for i := range result {
		result[i].Rank = ranks[result[i].PlayerID]
	}

	return result, nil
}

func (uc *GetStatsPlayersUseCase) allTimeRanks(tx Tx) (map[entity.PlayerID]PlayerRank, error) {
	allPlayers, err := uc.statsRepo.ListPlayers(tx, PlayerStatsFilter{Limit: 10000})
	if err != nil {
		return nil, err
	}
	return playerRanksByID(allPlayers), nil
}
