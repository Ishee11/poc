package usecase

import (
	"errors"
	"testing"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/entity/valueobject"
)

func TestFinishSessionUseCase_Execute(t *testing.T) {
	chipRate := valueobject.NewChipRate(2)

	t.Run("success", func(t *testing.T) {
		session := entity.NewSession("s1", chipRate)

		opRepo := &operationRepoMock{
			getAggFn: func(tx Tx, sID entity.SessionID) (SessionAggregates, error) {
				return SessionAggregates{
					TotalBuyIn:   100,
					TotalCashOut: 100,
				}, nil
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
			opRepo:      opRepo,
			sessionRepo: sessionRepo,
			txManager:   &txManagerMock{},
		}

		err := uc.Execute(FinishSessionCommand{
			SessionID: "s1",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if session.Status() != entity.StatusFinished {
			t.Fatalf("expected session finished, got %s", session.Status())
		}
	})

	t.Run("table not settled", func(t *testing.T) {
		session := entity.NewSession("s1", chipRate)

		opRepo := &operationRepoMock{
			getAggFn: func(tx Tx, sID entity.SessionID) (SessionAggregates, error) {
				return SessionAggregates{
					TotalBuyIn:   100,
					TotalCashOut: 50,
				}, nil
			},
		}

		sessionRepo := &sessionRepoMock{
			findFn: func(tx Tx, id entity.SessionID) (*entity.Session, error) {
				return session, nil
			},
		}

		uc := FinishSessionUseCase{
			opRepo:      opRepo,
			sessionRepo: sessionRepo,
			txManager:   &txManagerMock{},
		}

		err := uc.Execute(FinishSessionCommand{
			SessionID: "s1",
		})

		if !errors.Is(err, entity.ErrTableNotSettled) {
			t.Fatalf("expected ErrTableNotSettled, got %v", err)
		}
	})

	t.Run("already finished (idempotent)", func(t *testing.T) {
		session := entity.NewSession("s1", chipRate)
		// сначала завершаем
		_ = session.Finish()

		opRepo := &operationRepoMock{}

		sessionRepo := &sessionRepoMock{
			findFn: func(tx Tx, id entity.SessionID) (*entity.Session, error) {
				return session, nil
			},
		}

		uc := FinishSessionUseCase{
			opRepo:      opRepo,
			sessionRepo: sessionRepo,
			txManager:   &txManagerMock{},
		}

		err := uc.Execute(FinishSessionCommand{
			SessionID: "s1",
		})

		if err != nil {
			t.Fatalf("expected nil for idempotent call, got %v", err)
		}
	})

	t.Run("session not active", func(t *testing.T) {
		session := entity.NewSession("s1", chipRate)

		// эмулируем невалидный статус (например вручную)
		// проще: сначала finish с корректным состоянием
		opRepoFinish := &operationRepoMock{
			getAggFn: func(tx Tx, sID entity.SessionID) (SessionAggregates, error) {
				return SessionAggregates{
					TotalBuyIn:   100,
					TotalCashOut: 100,
				}, nil
			},
		}

		sessionRepoFinish := &sessionRepoMock{
			findFn: func(tx Tx, id entity.SessionID) (*entity.Session, error) {
				return session, nil
			},
			saveFn: func(tx Tx, s *entity.Session) error {
				return nil
			},
		}

		ucFinish := FinishSessionUseCase{
			opRepo:      opRepoFinish,
			sessionRepo: sessionRepoFinish,
			txManager:   &txManagerMock{},
		}

		_ = ucFinish.Execute(FinishSessionCommand{SessionID: "s1"})

		// теперь повторно вызываем → уже finished
		uc := FinishSessionUseCase{
			opRepo:      &operationRepoMock{},
			sessionRepo: sessionRepoFinish,
			txManager:   &txManagerMock{},
		}

		err := uc.Execute(FinishSessionCommand{
			SessionID: "s1",
		})

		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})
}
