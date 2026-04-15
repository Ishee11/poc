package usecase_test

import (
	"testing"
	"time"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/entity/valueobject"
	"github.com/ishee11/poc/internal/usecase"
)

func defaultIdempotencyRepo() *idempotencyRepoMock {
	return &idempotencyRepoMock{
		saveFn: func(tx usecase.Tx, requestID string) error {
			return nil
		},
	}
}

func TestCashOutUseCase_Execute(t *testing.T) {
	rate, _ := valueobject.NewChipRate(2)

	t.Run("success", func(t *testing.T) {
		session := entity.NewSession("s1", rate, time.Now())

		opRepo := &operationRepoMock{
			getLastOpFn: func(tx usecase.Tx, sID entity.SessionID, pID entity.PlayerID) (entity.OperationType, bool, error) {
				return entity.OperationBuyIn, true, nil
			},
			getAggFn: func(tx usecase.Tx, sID entity.SessionID) (usecase.SessionAggregates, error) {
				return usecase.SessionAggregates{TotalBuyIn: 100, TotalCashOut: 20}, nil
			},
			saveFn: func(tx usecase.Tx, op *entity.Operation) error {
				return nil
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

		uc := usecase.NewCashOutUseCase(
			opRepo,
			opRepo,
			opRepo,
			sessionRepo,
			sessionRepo,
			&txManagerMock{},
			&operationIDGeneratorMock{id: "op-1"},
			defaultIdempotencyRepo(),
			&playerRepoMock{},
		)

		err := uc.Execute(usecase.CashOutCommand{
			RequestID: "req-1",
			SessionID: "s1",
			PlayerID:  "p1",
			Chips:     50,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("idempotent", func(t *testing.T) {
		session := entity.NewSession("s1", rate, time.Now())

		idem := &idempotencyRepoMock{
			saveFn: func(tx usecase.Tx, requestID string) error {
				return entity.ErrDuplicateRequest
			},
		}

		opRepo := &operationRepoMock{
			saveFn: func(tx usecase.Tx, op *entity.Operation) error {
				t.Fatal("should not be called")
				return nil
			},
		}

		sessionRepo := &sessionRepoMock{
			findFn: func(tx usecase.Tx, id entity.SessionID) (*entity.Session, error) {
				return session, nil
			},
		}

		uc := usecase.NewCashOutUseCase(
			opRepo,
			opRepo,
			opRepo,
			sessionRepo,
			sessionRepo,
			&txManagerMock{},
			&operationIDGeneratorMock{id: "op"},
			idem,
			&playerRepoMock{},
		)

		err := uc.Execute(usecase.CashOutCommand{
			RequestID: "req-1",
			SessionID: "s1",
			PlayerID:  "p1",
			Chips:     10,
		})

		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})
}
