package usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/entity/valueobject"
	"github.com/ishee11/poc/internal/usecase"
)

func TestGetSessionResultsUseCase_Execute(t *testing.T) {

	now := time.Now()
	rate, _ := valueobject.NewChipRate(2)

	t.Run("success", func(t *testing.T) {

		session := entity.NewSession("s1", rate, now)

		sessionRepo := &sessionRepoMock{
			findFn: func(tx usecase.Tx, id entity.SessionID) (*entity.Session, error) {
				return session, nil
			},
		}

		opRepo := &operationRepoMock{
			getPlayerAggFn: func(tx usecase.Tx, sID entity.SessionID) (map[entity.PlayerID]usecase.PlayerAggregates, error) {
				return map[entity.PlayerID]usecase.PlayerAggregates{
					"p1": {BuyIn: 100, CashOut: 60},
					"p2": {BuyIn: 200, CashOut: 300},
				}, nil
			},
		}

		uc := usecase.NewGetSessionResultsUseCase(
			sessionRepo,
			opRepo,
			&txManagerMock{},
		)

		res, err := uc.Execute(usecase.GetSessionResultsQuery{
			SessionID: "s1",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(res.Results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(res.Results))
		}

		// проверяем p1
		var p1 usecase.PlayerResultDTO
		for _, r := range res.Results {
			if r.PlayerID == "p1" {
				p1 = r
			}
		}

		if p1.ProfitChips != -40 {
			t.Fatalf("expected p1 profitChips=-40, got %d", p1.ProfitChips)
		}

		if p1.ProfitMoney != -20 {
			t.Fatalf("expected p1 profitMoney=-20, got %d", p1.ProfitMoney)
		}

		// проверяем p2
		var p2 usecase.PlayerResultDTO
		for _, r := range res.Results {
			if r.PlayerID == "p2" {
				p2 = r
			}
		}

		if p2.ProfitChips != 100 {
			t.Fatalf("expected p2 profitChips=100, got %d", p2.ProfitChips)
		}

		if p2.ProfitMoney != 50 {
			t.Fatalf("expected p2 profitMoney=50, got %d", p2.ProfitMoney)
		}
	})

	t.Run("session not found", func(t *testing.T) {

		expectedErr := entity.ErrSessionNotFound

		sessionRepo := &sessionRepoMock{
			findFn: func(tx usecase.Tx, id entity.SessionID) (*entity.Session, error) {
				return nil, expectedErr
			},
		}

		opRepo := &operationRepoMock{}

		uc := usecase.NewGetSessionResultsUseCase(
			sessionRepo,
			opRepo,
			&txManagerMock{},
		)

		_, err := uc.Execute(usecase.GetSessionResultsQuery{
			SessionID: "s1",
		})

		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected %v, got %v", expectedErr, err)
		}
	})

	t.Run("aggregate error", func(t *testing.T) {

		expectedErr := errors.New("db error")

		session := entity.NewSession("s1", rate, now)

		sessionRepo := &sessionRepoMock{
			findFn: func(tx usecase.Tx, id entity.SessionID) (*entity.Session, error) {
				return session, nil
			},
		}

		opRepo := &operationRepoMock{
			getPlayerAggFn: func(tx usecase.Tx, sID entity.SessionID) (map[entity.PlayerID]usecase.PlayerAggregates, error) {
				return nil, expectedErr
			},
		}

		uc := usecase.NewGetSessionResultsUseCase(
			sessionRepo,
			opRepo,
			&txManagerMock{},
		)

		_, err := uc.Execute(usecase.GetSessionResultsQuery{
			SessionID: "s1",
		})

		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected %v, got %v", expectedErr, err)
		}
	})
}
