package usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/entity/valueobject"
	"github.com/ishee11/poc/internal/usecase"
	"github.com/ishee11/poc/internal/usecase/command"
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
			saveFn: func(tx usecase.Tx, op *entity.Operation) error {
				if op.RequestID() != "req-1" {
					t.Fatalf("unexpected requestID: %s", op.RequestID())
				}
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

		idGen := &idGeneratorMock{id: "op-internal-1"}

		helper := usecase.NewHelper(
			sessionRepo,
			sessionRepo,
			&playerRepoMock{},
			opRepo,
			idGen,
		)

		uc := usecase.NewBuyInUseCase(
			helper,
			&txManagerMock{},
			&idempotencyRepoMock{
				saveFn: func(tx usecase.Tx, requestID string) error { return nil },
			},
		)

		cmd := command.BuyInCommand{
			RequestID:  "req-1",
			SessionID:  "s1",
			PlayerName: "p1",
			Chips:      100,
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
		helper := usecase.NewHelper(
			&sessionRepoMock{},
			&sessionRepoMock{},
			&playerRepoMock{},
			&operationRepoMock{},
			&idGeneratorMock{id: "op"},
		)

		uc := usecase.NewBuyInUseCase(
			helper,
			&txManagerMock{},
			&idempotencyRepoMock{
				saveFn: func(tx usecase.Tx, requestID string) error { return nil },
			},
		)

		cmd := command.BuyInCommand{
			RequestID:  "req-1",
			SessionID:  "s1",
			PlayerName: "p1",
			Chips:      0,
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

		sessionRepo := &sessionRepoMock{
			findFn: func(tx usecase.Tx, id entity.SessionID) (*entity.Session, error) {
				return session, nil
			},
		}

		helper := usecase.NewHelper(
			sessionRepo,
			sessionRepo,
			&playerRepoMock{},
			&operationRepoMock{},
			&idGeneratorMock{id: "op"},
		)

		uc := usecase.NewBuyInUseCase(
			helper,
			&txManagerMock{},
			&idempotencyRepoMock{
				saveFn: func(tx usecase.Tx, requestID string) error { return nil },
			},
		)

		cmd := command.BuyInCommand{
			RequestID:  "req-1",
			SessionID:  "s1",
			PlayerName: "p1",
			Chips:      100,
		}

		err := uc.Execute(cmd)
		if !errors.Is(err, entity.ErrSessionNotActive) {
			t.Fatalf("expected ErrSessionNotActive, got %v", err)
		}
	})

	t.Run("operation repo error", func(t *testing.T) {
		session := entity.NewSession("s1", rate, time.Now())

		opRepo := &operationRepoMock{
			saveFn: func(tx usecase.Tx, op *entity.Operation) error {
				return errors.New("db error")
			},
		}

		sessionRepo := &sessionRepoMock{
			findFn: func(tx usecase.Tx, id entity.SessionID) (*entity.Session, error) {
				return session, nil
			},
		}

		helper := usecase.NewHelper(
			sessionRepo,
			sessionRepo,
			&playerRepoMock{},
			opRepo,
			&idGeneratorMock{id: "op"},
		)

		uc := usecase.NewBuyInUseCase(
			helper,
			&txManagerMock{},
			&idempotencyRepoMock{
				saveFn: func(tx usecase.Tx, requestID string) error { return nil },
			},
		)

		cmd := command.BuyInCommand{
			RequestID:  "req-1",
			SessionID:  "s1",
			PlayerName: "p1",
			Chips:      100,
		}

		err := uc.Execute(cmd)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("idempotent via duplicate request error", func(t *testing.T) {
		session := entity.NewSession("s1", rate, time.Now())

		sessionRepo := &sessionRepoMock{
			findFn: func(tx usecase.Tx, id entity.SessionID) (*entity.Session, error) {
				return session, nil
			},
		}

		helper := usecase.NewHelper(
			sessionRepo,
			sessionRepo,
			&playerRepoMock{},
			&operationRepoMock{},
			&idGeneratorMock{id: "op"},
		)

		uc := usecase.NewBuyInUseCase(
			helper,
			&txManagerMock{},
			&idempotencyRepoMock{
				saveFn: func(tx usecase.Tx, requestID string) error {
					return entity.ErrDuplicateRequest
				},
			},
		)

		cmd := command.BuyInCommand{
			RequestID:  "req-1",
			SessionID:  "s1",
			PlayerName: "p1",
			Chips:      100,
		}

		err := uc.Execute(cmd)
		if err != nil {
			t.Fatalf("expected nil due to idempotency, got %v", err)
		}
	})
}
