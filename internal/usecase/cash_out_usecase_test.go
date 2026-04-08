package usecase

import (
	"errors"
	"testing"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/entity/valueobject"
)

func TestCashOutUseCase_Execute(t *testing.T) {
	chipRate := valueobject.NewChipRate(2)

	t.Run("success", func(t *testing.T) {
		session := entity.NewSession("s1", chipRate)

		opRepo := &operationRepoMock{
			getLastOpFn: func(tx Tx, sID entity.SessionID, pID entity.PlayerID) (entity.OperationType, bool, error) {
				return entity.OperationBuyIn, true, nil
			},
			getAggFn: func(tx Tx, sID entity.SessionID) (SessionAggregates, error) {
				return SessionAggregates{
					TotalBuyIn:   100,
					TotalCashOut: 20,
				}, nil
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
			opRepo:      opRepo,
			sessionRepo: sessionRepo,
			txManager:   &txManagerMock{},
		}

		cmd := CashOutCommand{
			OperationID: "op1",
			SessionID:   "s1",
			PlayerID:    "p1",
			Chips:       50,
		}

		err := uc.Execute(cmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if session.TotalCashOut() != 50 {
			t.Fatalf("expected totalCashOut=50, got %d", session.TotalCashOut())
		}
	})

	t.Run("player not in game", func(t *testing.T) {
		session := entity.NewSession("s1", chipRate)

		opRepo := &operationRepoMock{
			getLastOpFn: func(tx Tx, sID entity.SessionID, pID entity.PlayerID) (entity.OperationType, bool, error) {
				return "", false, nil
			},
		}

		sessionRepo := &sessionRepoMock{
			findFn: func(tx Tx, id entity.SessionID) (*entity.Session, error) {
				return session, nil
			},
		}

		uc := CashOutUseCase{
			opRepo:      opRepo,
			sessionRepo: sessionRepo,
			txManager:   &txManagerMock{},
		}

		err := uc.Execute(CashOutCommand{
			SessionID: "s1",
			PlayerID:  "p1",
			Chips:     10,
		})

		if !errors.Is(err, entity.ErrPlayerNotInGame) {
			t.Fatalf("expected ErrPlayerNotInGame, got %v", err)
		}
	})

	t.Run("last operation not buyin", func(t *testing.T) {
		session := entity.NewSession("s1", chipRate)

		opRepo := &operationRepoMock{
			getLastOpFn: func(tx Tx, sID entity.SessionID, pID entity.PlayerID) (entity.OperationType, bool, error) {
				return entity.OperationCashOut, true, nil
			},
		}

		sessionRepo := &sessionRepoMock{
			findFn: func(tx Tx, id entity.SessionID) (*entity.Session, error) {
				return session, nil
			},
		}

		uc := CashOutUseCase{
			opRepo:      opRepo,
			sessionRepo: sessionRepo,
			txManager:   &txManagerMock{},
		}

		err := uc.Execute(CashOutCommand{
			SessionID: "s1",
			PlayerID:  "p1",
			Chips:     10,
		})

		if !errors.Is(err, entity.ErrInvalidOperation) {
			t.Fatalf("expected ErrInvalidOperation, got %v", err)
		}
	})

	t.Run("exceeds table chips", func(t *testing.T) {
		session := entity.NewSession("s1", chipRate)

		opRepo := &operationRepoMock{
			getLastOpFn: func(tx Tx, sID entity.SessionID, pID entity.PlayerID) (entity.OperationType, bool, error) {
				return entity.OperationBuyIn, true, nil
			},
			getAggFn: func(tx Tx, sID entity.SessionID) (SessionAggregates, error) {
				return SessionAggregates{
					TotalBuyIn:   50,
					TotalCashOut: 40,
				}, nil
			},
		}

		sessionRepo := &sessionRepoMock{
			findFn: func(tx Tx, id entity.SessionID) (*entity.Session, error) {
				return session, nil
			},
		}

		uc := CashOutUseCase{
			opRepo:      opRepo,
			sessionRepo: sessionRepo,
			txManager:   &txManagerMock{},
		}

		err := uc.Execute(CashOutCommand{
			SessionID: "s1",
			PlayerID:  "p1",
			Chips:     20,
		})

		if !errors.Is(err, entity.ErrInvalidCashOut) {
			t.Fatalf("expected ErrInvalidCashOut, got %v", err)
		}
	})

	t.Run("idempotent duplicate operation", func(t *testing.T) {
		session := entity.NewSession("s1", chipRate)

		opRepo := &operationRepoMock{
			getLastOpFn: func(tx Tx, sID entity.SessionID, pID entity.PlayerID) (entity.OperationType, bool, error) {
				return entity.OperationBuyIn, true, nil
			},
			getAggFn: func(tx Tx, sID entity.SessionID) (SessionAggregates, error) {
				return SessionAggregates{
					TotalBuyIn:   100,
					TotalCashOut: 0,
				}, nil
			},
			saveFn: func(tx Tx, op *entity.Operation) error {
				return entity.ErrDuplicateOperation
			},
		}

		sessionRepo := &sessionRepoMock{
			findFn: func(tx Tx, id entity.SessionID) (*entity.Session, error) {
				return session, nil
			},
		}

		uc := CashOutUseCase{
			opRepo:      opRepo,
			sessionRepo: sessionRepo,
			txManager:   &txManagerMock{},
		}

		err := uc.Execute(CashOutCommand{
			SessionID: "s1",
			PlayerID:  "p1",
			Chips:     10,
		})

		if err != nil {
			t.Fatalf("expected nil due to idempotency, got %v", err)
		}
	})
}
