package usecase

import (
	"errors"
	"testing"
	"time"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/entity/valueobject"
)

// --- mock id generator ---

type idGeneratorMock struct {
	id entity.OperationID
}

func (m *idGeneratorMock) New() entity.OperationID {
	return m.id
}

func TestBuyInUseCase_Execute(t *testing.T) {
	rate, err := valueobject.NewChipRate(2)
	if err != nil {
		t.Fatalf("failed to create chip rate: %v", err)
	}

	t.Run("success", func(t *testing.T) {
		session := entity.NewSession("s1", rate, time.Now())

		opRepo := &operationRepoMock{
			getByRequestIDFn: func(tx Tx, requestID string) (*entity.Operation, error) {
				return nil, nil
			},
			saveFn: func(tx Tx, op *entity.Operation) error {
				if op.RequestID() != "req-1" {
					t.Fatalf("unexpected requestID: %s", op.RequestID())
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

		idGen := &idGeneratorMock{id: "op-internal-1"}

		uc := BuyInUseCase{
			opRepo:      opRepo,
			sessionRepo: sessionRepo,
			txManager:   &txManagerMock{},
			idGen:       idGen,
		}

		cmd := BuyInCommand{
			RequestID: "req-1",
			SessionID: "s1",
			PlayerID:  "p1",
			Chips:     100,
		}

		err := uc.Execute(cmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if session.TotalBuyIn() != 100 {
			t.Fatalf("expected totalBuyIn=100, got %d", session.TotalBuyIn())
		}
	})

	t.Run("invalid chips", func(t *testing.T) {
		uc := BuyInUseCase{
			opRepo:      &operationRepoMock{},
			sessionRepo: &sessionRepoMock{},
			txManager:   &txManagerMock{},
			idGen:       &idGeneratorMock{id: "op"},
		}

		cmd := BuyInCommand{
			RequestID: "req-1",
			SessionID: "s1",
			PlayerID:  "p1",
			Chips:     0,
		}

		err := uc.Execute(cmd)
		if !errors.Is(err, entity.ErrInvalidChips) {
			t.Fatalf("expected ErrInvalidChips, got %v", err)
		}
	})

	t.Run("session not active", func(t *testing.T) {
		session := entity.NewSession("s1", rate, time.Now())

		if err := session.Finish(); err != nil {
			t.Fatalf("failed to finish session: %v", err)
		}

		opRepo := &operationRepoMock{
			getByRequestIDFn: func(tx Tx, requestID string) (*entity.Operation, error) {
				return nil, nil
			},
		}

		sessionRepo := &sessionRepoMock{
			findFn: func(tx Tx, id entity.SessionID) (*entity.Session, error) {
				return session, nil
			},
		}

		uc := BuyInUseCase{
			opRepo:      opRepo,
			sessionRepo: sessionRepo,
			txManager:   &txManagerMock{},
			idGen:       &idGeneratorMock{id: "op"},
		}

		cmd := BuyInCommand{
			RequestID: "req-1",
			SessionID: "s1",
			PlayerID:  "p1",
			Chips:     100,
		}

		err := uc.Execute(cmd)
		if !errors.Is(err, entity.ErrSessionNotActive) {
			t.Fatalf("expected ErrSessionNotActive, got %v", err)
		}
	})

	t.Run("operation repo error", func(t *testing.T) {
		session := entity.NewSession("s1", rate, time.Now())

		opRepo := &operationRepoMock{
			getByRequestIDFn: func(tx Tx, requestID string) (*entity.Operation, error) {
				return nil, nil
			},
			saveFn: func(tx Tx, op *entity.Operation) error {
				return errors.New("db error")
			},
		}

		sessionRepo := &sessionRepoMock{
			findFn: func(tx Tx, id entity.SessionID) (*entity.Session, error) {
				return session, nil
			},
		}

		uc := BuyInUseCase{
			opRepo:      opRepo,
			sessionRepo: sessionRepo,
			txManager:   &txManagerMock{},
			idGen:       &idGeneratorMock{id: "op"},
		}

		cmd := BuyInCommand{
			RequestID: "req-1",
			SessionID: "s1",
			PlayerID:  "p1",
			Chips:     100,
		}

		err := uc.Execute(cmd)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("idempotent by requestID", func(t *testing.T) {

		existingOp, _ := entity.NewOperation(
			"existing",
			"req-1",
			"s1",
			entity.OperationBuyIn,
			"p1",
			100,
			time.Now(),
		)

		opRepo := &operationRepoMock{
			getByRequestIDFn: func(tx Tx, requestID string) (*entity.Operation, error) {
				return existingOp, nil
			},
			saveFn: func(tx Tx, op *entity.Operation) error {
				t.Fatal("save should not be called on idempotent request")
				return nil
			},
		}

		sessionRepo := &sessionRepoMock{
			findFn: func(tx Tx, id entity.SessionID) (*entity.Session, error) {
				t.Fatal("session should not be loaded on idempotent request")
				return nil, nil
			},
		}

		uc := BuyInUseCase{
			opRepo:      opRepo,
			sessionRepo: sessionRepo,
			txManager:   &txManagerMock{},
			idGen:       &idGeneratorMock{id: "op"},
		}

		cmd := BuyInCommand{
			RequestID: "req-1",
			SessionID: "s1",
			PlayerID:  "p1",
			Chips:     100,
		}

		err := uc.Execute(cmd)
		if err != nil {
			t.Fatalf("expected nil due to idempotency, got %v", err)
		}
	})
}
