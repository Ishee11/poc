package usecase

import (
	"errors"
	"testing"
	"time"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/entity/valueobject"
)

func TestFinishSessionUseCase_Execute(t *testing.T) {
	rate, err := valueobject.NewChipRate(2)
	if err != nil {
		t.Fatalf("failed to create chip rate: %v", err)
	}

	t.Run("success", func(t *testing.T) {
		session := entity.NewSession("s1", rate, time.Now())

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
			aggregateReader: opRepo,
			sessionReader:   sessionRepo,
			sessionWriter:   sessionRepo,
			txManager:       &txManagerMock{},
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
		session := entity.NewSession("s1", rate, time.Now())

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
			aggregateReader: opRepo,
			sessionReader:   sessionRepo,
			sessionWriter:   sessionRepo,
			txManager:       &txManagerMock{},
		}

		err := uc.Execute(FinishSessionCommand{
			SessionID: "s1",
		})

		if !errors.Is(err, entity.ErrTableNotSettled) {
			t.Fatalf("expected ErrTableNotSettled, got %v", err)
		}
	})

	t.Run("already finished (idempotent)", func(t *testing.T) {
		session := entity.NewSession("s1", rate, time.Now())

		if err := session.Finish(); err != nil {
			t.Fatalf("failed to finish session: %v", err)
		}

		sessionRepo := &sessionRepoMock{
			findFn: func(tx Tx, id entity.SessionID) (*entity.Session, error) {
				return session, nil
			},
		}

		uc := FinishSessionUseCase{
			aggregateReader: &operationRepoMock{},
			sessionReader:   sessionRepo,
			sessionWriter:   sessionRepo,
			txManager:       &txManagerMock{},
		}

		err := uc.Execute(FinishSessionCommand{
			SessionID: "s1",
		})

		if err != nil {
			t.Fatalf("expected nil for idempotent call, got %v", err)
		}
	})

	t.Run("double finish is idempotent", func(t *testing.T) {
		session := entity.NewSession("s1", rate, time.Now())

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
			aggregateReader: opRepo,
			sessionReader:   sessionRepo,
			sessionWriter:   sessionRepo,
			txManager:       &txManagerMock{},
		}

		// первый finish
		if err := uc.Execute(FinishSessionCommand{SessionID: "s1"}); err != nil {
			t.Fatalf("first finish failed: %v", err)
		}

		// второй finish
		err := uc.Execute(FinishSessionCommand{SessionID: "s1"})
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})
}
