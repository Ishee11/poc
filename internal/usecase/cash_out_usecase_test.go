package usecase

import (
	"errors"
	"testing"
	"time"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/entity/valueobject"
)

func TestCashOutUseCase_Execute(t *testing.T) {
	rate, _ := valueobject.NewChipRate(2)

	t.Run("success", func(t *testing.T) {
		session := entity.NewSession("s1", rate, time.Now())

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
				if op.RequestID() != "req-1" {
					t.Fatalf("unexpected requestID: %s", op.RequestID())
				}
				if op.ID() != "op-1" {
					t.Fatalf("unexpected operationID: %s", op.ID())
				}
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
			aggregateReader:   opRepo,
			sessionReader:     sessionRepo,
			sessionWriter:     sessionRepo,
			txManager:         &txManagerMock{},
			idGen:             &operationIDGeneratorMock{id: "op-1"},
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

		if session.TotalCashOut() != 50 {
			t.Fatalf("expected totalCashOut=50, got %d", session.TotalCashOut())
		}
	})

	t.Run("invalid chips", func(t *testing.T) {
		uc := CashOutUseCase{
			opWriter:          &operationRepoMock{},
			playerStateReader: &operationRepoMock{},
			aggregateReader:   &operationRepoMock{},
			sessionReader:     &sessionRepoMock{},
			sessionWriter:     &sessionRepoMock{},
			txManager:         &txManagerMock{},
			idGen:             &operationIDGeneratorMock{id: "op"},
		}

		err := uc.Execute(CashOutCommand{
			RequestID: "req-1",
			SessionID: "s1",
			PlayerID:  "p1",
			Chips:     0,
		})

		if !errors.Is(err, entity.ErrInvalidChips) {
			t.Fatalf("expected ErrInvalidChips, got %v", err)
		}
	})

	t.Run("session not active", func(t *testing.T) {
		session := entity.NewSession("s1", rate, time.Now())
		_ = session.Finish()

		opRepo := &operationRepoMock{}

		sessionRepo := &sessionRepoMock{
			findFn: func(tx Tx, id entity.SessionID) (*entity.Session, error) {
				return session, nil
			},
		}

		uc := CashOutUseCase{
			opWriter:          opRepo,
			playerStateReader: opRepo,
			aggregateReader:   opRepo,
			sessionReader:     sessionRepo,
			sessionWriter:     sessionRepo,
			txManager:         &txManagerMock{},
			idGen:             &operationIDGeneratorMock{id: "op"},
		}

		err := uc.Execute(CashOutCommand{
			RequestID: "req-1",
			SessionID: "s1",
			PlayerID:  "p1",
			Chips:     10,
		})

		if !errors.Is(err, entity.ErrSessionNotActive) {
			t.Fatalf("expected ErrSessionNotActive, got %v", err)
		}
	})

	t.Run("player not in game", func(t *testing.T) {
		session := entity.NewSession("s1", rate, time.Now())

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
			opWriter:          opRepo,
			playerStateReader: opRepo,
			aggregateReader:   opRepo,
			sessionReader:     sessionRepo,
			sessionWriter:     sessionRepo,
			txManager:         &txManagerMock{},
			idGen:             &operationIDGeneratorMock{id: "op"},
		}

		err := uc.Execute(CashOutCommand{
			RequestID: "req-1",
			SessionID: "s1",
			PlayerID:  "p1",
			Chips:     10,
		})

		if !errors.Is(err, entity.ErrPlayerNotInGame) {
			t.Fatalf("expected ErrPlayerNotInGame, got %v", err)
		}
	})

	t.Run("last operation not buyin", func(t *testing.T) {
		session := entity.NewSession("s1", rate, time.Now())

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
			opWriter:          opRepo,
			playerStateReader: opRepo,
			aggregateReader:   opRepo,
			sessionReader:     sessionRepo,
			sessionWriter:     sessionRepo,
			txManager:         &txManagerMock{},
			idGen:             &operationIDGeneratorMock{id: "op"},
		}

		err := uc.Execute(CashOutCommand{
			RequestID: "req-1",
			SessionID: "s1",
			PlayerID:  "p1",
			Chips:     10,
		})

		if !errors.Is(err, entity.ErrInvalidOperation) {
			t.Fatalf("expected ErrInvalidOperation, got %v", err)
		}
	})

	t.Run("exceeds table chips", func(t *testing.T) {
		session := entity.NewSession("s1", rate, time.Now())

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
			opWriter:          opRepo,
			playerStateReader: opRepo,
			aggregateReader:   opRepo,
			sessionReader:     sessionRepo,
			sessionWriter:     sessionRepo,
			txManager:         &txManagerMock{},
			idGen:             &operationIDGeneratorMock{id: "op"},
		}

		err := uc.Execute(CashOutCommand{
			RequestID: "req-1",
			SessionID: "s1",
			PlayerID:  "p1",
			Chips:     20,
		})

		if !errors.Is(err, entity.ErrInvalidCashOut) {
			t.Fatalf("expected ErrInvalidCashOut, got %v", err)
		}
	})

	t.Run("idempotent via duplicate request error", func(t *testing.T) {
		session := entity.NewSession("s1", rate, time.Now())

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
				return entity.ErrDuplicateRequest
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
			aggregateReader:   opRepo,
			sessionReader:     sessionRepo,
			sessionWriter:     sessionRepo,
			txManager:         &txManagerMock{},
			idGen:             &operationIDGeneratorMock{id: "op"},
		}

		err := uc.Execute(CashOutCommand{
			RequestID: "req-1",
			SessionID: "s1",
			PlayerID:  "p1",
			Chips:     10,
		})

		if err != nil {
			t.Fatalf("expected nil due to idempotency, got %v", err)
		}
	})
}
