package usecase

import (
	"errors"
	"testing"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/entity/valueobject"
)

type operationRepoMock struct {
	saveFn func(tx Tx, op *entity.Operation) error
}

func (m *operationRepoMock) Save(tx Tx, op *entity.Operation) error {
	return m.saveFn(tx, op)
}

func (m *operationRepoMock) GetLastOperationType(
	tx Tx,
	sessionID entity.SessionID,
	playerID entity.PlayerID,
) (entity.OperationType, bool, error) {
	panic("not implemented")
}

func (m *operationRepoMock) GetSessionAggregates(
	tx Tx,
	sessionID entity.SessionID,
) (SessionAggregates, error) {
	panic("not implemented")
}

type sessionRepoMock struct {
	findFn func(tx Tx, id entity.SessionID) (*entity.Session, error)
	saveFn func(tx Tx, s *entity.Session) error
}

func (m *sessionRepoMock) FindByID(tx Tx, id entity.SessionID) (*entity.Session, error) {
	return m.findFn(tx, id)
}

func (m *sessionRepoMock) Save(tx Tx, s *entity.Session) error {
	return m.saveFn(tx, s)
}

type txManagerMock struct{}

func (m *txManagerMock) RunInTx(fn func(tx Tx) error) error {
	return fn(struct{}{})
}

func TestBuyInUseCase_Execute(t *testing.T) {
	chipRate := valueobject.NewChipRate(2)

	t.Run("success", func(t *testing.T) {
		session := entity.NewSession("s1", chipRate)

		opRepo := &operationRepoMock{
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

		uc := BuyInUseCase{
			opRepo:      opRepo,
			sessionRepo: sessionRepo,
			txManager:   &txManagerMock{},
		}

		cmd := BuyInCommand{
			OperationID: "op1",
			SessionID:   "s1",
			PlayerID:    "p1",
			Chips:       100,
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
		opRepo := &operationRepoMock{
			saveFn: func(tx Tx, op *entity.Operation) error {
				return nil
			},
		}

		sessionRepo := &sessionRepoMock{}

		uc := BuyInUseCase{
			opRepo:      opRepo,
			sessionRepo: sessionRepo,
			txManager:   &txManagerMock{},
		}

		cmd := BuyInCommand{
			OperationID: "op1",
			SessionID:   "s1",
			PlayerID:    "p1",
			Chips:       0,
		}

		err := uc.Execute(cmd)
		if !errors.Is(err, entity.ErrInvalidChips) {
			t.Fatalf("expected ErrInvalidChips, got %v", err)
		}
	})

	t.Run("session not active", func(t *testing.T) {
		session := entity.NewSession("s1", chipRate)
		_ = session.Finish() // делаем неактивной

		opRepo := &operationRepoMock{
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

		uc := BuyInUseCase{
			opRepo:      opRepo,
			sessionRepo: sessionRepo,
			txManager:   &txManagerMock{},
		}

		cmd := BuyInCommand{
			OperationID: "op1",
			SessionID:   "s1",
			PlayerID:    "p1",
			Chips:       100,
		}

		err := uc.Execute(cmd)
		if !errors.Is(err, entity.ErrSessionNotActive) {
			t.Fatalf("expected ErrSessionNotActive, got %v", err)
		}
	})

	t.Run("operation repo error", func(t *testing.T) {
		session := entity.NewSession("s1", chipRate)

		opRepo := &operationRepoMock{
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
		}

		cmd := BuyInCommand{
			OperationID: "op1",
			SessionID:   "s1",
			PlayerID:    "p1",
			Chips:       100,
		}

		err := uc.Execute(cmd)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
