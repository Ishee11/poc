package usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/entity/valueobject"
	"github.com/ishee11/poc/internal/usecase"
)

func TestGetSessionUseCase_Execute(t *testing.T) {

	now := time.Now()

	rate, _ := valueobject.NewChipRate(2)

	t.Run("success", func(t *testing.T) {
		session := entity.NewSession("s1", rate, now)

		opRepo := &operationRepoMock{
			getAggFn: func(tx usecase.Tx, sID entity.SessionID) (usecase.SessionAggregates, error) {
				return usecase.NewSessionAggregates(
					100,
					40,
				), nil
			},
		}

		sessionRepo := &sessionRepoMock{
			findFn: func(tx usecase.Tx, id entity.SessionID) (*entity.Session, error) {
				return session, nil
			},
		}

		uc := usecase.NewGetSessionUseCase(
			sessionRepo,
			opRepo,
			&txManagerMock{},
		)

		res, err := uc.Execute(*usecase.NewGetSessionQuery("s1"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if res.SessionID != "s1" {
			t.Fatalf("unexpected sessionID: %s", res.SessionID)
		}

		if res.Status != entity.StatusActive {
			t.Fatalf("unexpected status: %s", res.Status)
		}

		if res.ChipRate != 2 {
			t.Fatalf("unexpected chipRate: %d", res.ChipRate)
		}

		if res.TotalBuyIn != 100 {
			t.Fatalf("expected TotalBuyIn=100, got %d", res.TotalBuyIn)
		}

		if res.TotalCashOut != 40 {
			t.Fatalf("expected TotalCashOut=40, got %d", res.TotalCashOut)
		}

		if res.TotalChips != 60 {
			t.Fatalf("expected TotalChips=60, got %d", res.TotalChips)
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

		uc := usecase.NewGetSessionUseCase(
			sessionRepo,
			opRepo,
			&txManagerMock{},
		)

		_, err := uc.Execute(*usecase.NewGetSessionQuery(
			"s1",
		))

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
			getAggFn: func(tx usecase.Tx, sID entity.SessionID) (usecase.SessionAggregates, error) {
				return usecase.NewSessionAggregates(0, 0), expectedErr
			},
		}

		uc := usecase.NewGetSessionUseCase(
			sessionRepo,
			opRepo,
			&txManagerMock{},
		)

		_, err := uc.Execute(*usecase.NewGetSessionQuery(
			"s1",
		))

		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected %v, got %v", expectedErr, err)
		}
	})
}
