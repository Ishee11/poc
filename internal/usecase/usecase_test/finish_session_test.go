package usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/entity/valueobject"
	"github.com/ishee11/poc/internal/usecase"
)

func TestFinishSessionUseCase_Execute(t *testing.T) {
	rate, _ := valueobject.NewChipRate(2)

	t.Run("success", func(t *testing.T) {
		session := entity.NewSession("s1", rate, time.Now())

		opRepo := &operationRepoMock{
			getAggFn: func(tx usecase.Tx, sID entity.SessionID) (usecase.SessionAggregates, error) {
				return usecase.SessionAggregates{
					TotalBuyIn:   100,
					TotalCashOut: 100,
				}, nil
			},
		}

		sessionRepo := &sessionRepoMock{
			findFn: func(tx usecase.Tx, id entity.SessionID) (*entity.Session, error) {
				return session, nil
			},
			saveFn: func(tx usecase.Tx, s *entity.Session) error {
				return nil
			},
		}

		uc := usecase.NewFinishSessionUseCase(
			opRepo,
			sessionRepo,
			sessionRepo,
			&txManagerMock{},
			defaultIdempotencyRepo(), // ← вот это ты пропустил
		)

		err := uc.Execute(usecase.FinishSessionCommand{
			RequestID: "req-1",
			SessionID: "s1",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("not balanced", func(t *testing.T) {
		session := entity.NewSession("s1", rate, time.Now())

		opRepo := &operationRepoMock{
			getAggFn: func(tx usecase.Tx, sID entity.SessionID) (usecase.SessionAggregates, error) {
				return usecase.SessionAggregates{
					TotalBuyIn:   100,
					TotalCashOut: 50,
				}, nil
			},
		}

		sessionRepo := &sessionRepoMock{
			findFn: func(tx usecase.Tx, id entity.SessionID) (*entity.Session, error) {
				return session, nil
			},
		}

		uc := usecase.NewFinishSessionUseCase(
			opRepo,
			sessionRepo,
			sessionRepo,
			&txManagerMock{},
			defaultIdempotencyRepo(),
		)

		err := uc.Execute(usecase.FinishSessionCommand{
			RequestID: "req-1",
			SessionID: "s1",
		})

		if !errors.Is(err, entity.ErrSessionNotBalanced) {
			t.Fatalf("expected ErrSessionNotBalanced, got %v", err)
		}
	})
}
