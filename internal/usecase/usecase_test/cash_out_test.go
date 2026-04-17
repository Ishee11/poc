package usecase_test

import (
	"testing"
	"time"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/entity/valueobject"
	"github.com/ishee11/poc/internal/usecase"
	"github.com/ishee11/poc/internal/usecase/command"
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

			// 👇 ОСТАЁТСЯ (для table check)
			getAggFn: func(tx usecase.Tx, sID entity.SessionID) (usecase.SessionAggregates, error) {
				return usecase.SessionAggregates{
					TotalBuyIn:   100,
					TotalCashOut: 20,
				}, nil
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

		idGen := &operationIDGeneratorMock{id: "op-1"}

		helper := usecase.NewHelper(
			sessionRepo,
			sessionRepo,
			&playerRepoMock{},
			opRepo,
			idGen,
		)

		uc := usecase.NewCashOutUseCase(
			helper,
			opRepo, // playerStateReader
			opRepo, // projection
			&txManagerMock{},
			defaultIdempotencyRepo(),
		)

		err := uc.Execute(command.CashOutCommand{
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

		idGen := &operationIDGeneratorMock{id: "op"}

		helper := usecase.NewHelper(
			sessionRepo,
			sessionRepo,
			&playerRepoMock{},
			opRepo,
			idGen,
		)

		uc := usecase.NewCashOutUseCase(
			helper,
			opRepo,
			opRepo,
			&txManagerMock{},
			idem,
		)

		err := uc.Execute(command.CashOutCommand{
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
