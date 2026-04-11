package usecase

import (
	"testing"
	"time"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/entity/valueobject"
)

func defaultIdempotencyRepo() *idempotencyRepoMock {
	return &idempotencyRepoMock{
		saveFn: func(tx Tx, requestID string) error {
			return nil
		},
	}
}

func TestCashOutUseCase_Execute(t *testing.T) {
	rate, _ := valueobject.NewChipRate(2)

	t.Run("success", func(t *testing.T) {
		session := entity.NewSession("s1", rate, time.Now())

		opRepo := &operationRepoMock{
			getLastOpFn: func(tx Tx, sID entity.SessionID, pID entity.PlayerID) (entity.OperationType, bool, error) {
				return entity.OperationBuyIn, true, nil
			},
			getAggFn: func(tx Tx, sID entity.SessionID) (SessionAggregates, error) {
				return SessionAggregates{TotalBuyIn: 100, TotalCashOut: 20}, nil
			},
			saveFn: func(tx Tx, op *entity.Operation) error {
				return nil
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

		uc := CashOutUseCase{
			opWriter:          opRepo,
			playerStateReader: opRepo,
			projection:        opRepo,
			sessionReader:     sessionRepo,
			sessionWriter:     sessionRepo,
			txManager:         &txManagerMock{},
			idGen:             &operationIDGeneratorMock{id: "op-1"},
			idempotencyRepo:   defaultIdempotencyRepo(),
		}

		err := uc.Execute(CashOutCommand{
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
			saveFn: func(tx Tx, requestID string) error {
				return entity.ErrDuplicateRequest
			},
		}

		opRepo := &operationRepoMock{
			saveFn: func(tx Tx, op *entity.Operation) error {
				t.Fatal("should not be called")
				return nil
			},
		}

		sessionRepo := &sessionRepoMock{
			findFn: func(tx Tx, id entity.SessionID) (*entity.Session, error) {
				return session, nil
			},
		}

		uc := CashOutUseCase{
			opWriter:          opRepo,
			playerStateReader: opRepo,
			projection:        opRepo,
			sessionReader:     sessionRepo,
			sessionWriter:     sessionRepo,
			txManager:         &txManagerMock{},
			idGen:             &operationIDGeneratorMock{id: "op"},
			idempotencyRepo:   idem,
		}

		err := uc.Execute(CashOutCommand{
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
