package usecase

import (
	"errors"
	"testing"
	"time"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/entity/valueobject"
)

func TestFinishSessionUseCase_Execute(t *testing.T) {
	rate, _ := valueobject.NewChipRate(2)

	t.Run("success", func(t *testing.T) {
		session := entity.NewSession("s1", rate, time.Now())

		opRepo := &operationRepoMock{
			getAggFn: func(tx Tx, sID entity.SessionID) (SessionAggregates, error) {
				return SessionAggregates{TotalBuyIn: 100, TotalCashOut: 100}, nil
			},
		}

		sessionRepo := &sessionRepoMock{
			findFn: func(tx Tx, id entity.SessionID) (*entity.Session, error) {
				return session, nil
			},
			saveFn: func(tx Tx, s *entity.Session) error {
				return nil
			},
		}

		uc := FinishSessionUseCase{
			projection:      opRepo,
			sessionReader:   sessionRepo,
			sessionWriter:   sessionRepo,
			txManager:       &txManagerMock{},
			idempotencyRepo: defaultIdempotencyRepo(),
		}

		err := uc.Execute(FinishSessionCommand{
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
			getAggFn: func(tx Tx, sID entity.SessionID) (SessionAggregates, error) {
				return SessionAggregates{TotalBuyIn: 100, TotalCashOut: 50}, nil
			},
		}

		uc := FinishSessionUseCase{
			projection:      opRepo,
			sessionReader:   &sessionRepoMock{findFn: func(tx Tx, id entity.SessionID) (*entity.Session, error) { return session, nil }},
			sessionWriter:   &sessionRepoMock{},
			txManager:       &txManagerMock{},
			idempotencyRepo: defaultIdempotencyRepo(),
		}

		err := uc.Execute(FinishSessionCommand{
			RequestID: "req-1",
			SessionID: "s1",
		})

		if !errors.Is(err, entity.ErrSessionNotBalanced) {
			t.Fatalf("expected ErrSessionNotBalanced, got %v", err)
		}
	})
}
