package usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/entity/valueobject"
	"github.com/ishee11/poc/internal/usecase"
)

func TestStartSessionUseCase_Execute(t *testing.T) {

	dbErr := errors.New("db error")

	tt := []struct {
		name    string
		setup   func(repo *sessionRepoMock)
		cmd     usecase.StartSessionCommand
		wantErr error
	}{
		{
			name: "success",
			setup: func(repo *sessionRepoMock) {
				repo.findFn = func(tx usecase.Tx, id entity.SessionID) (*entity.Session, error) {
					return nil, entity.ErrSessionNotFound
				}
				repo.saveFn = func(tx usecase.Tx, s *entity.Session) error {
					return nil
				}
			},
			cmd: usecase.NewStartSessionCommand(
				"s1",
				2,
			),
		},
		{
			name: "idempotent - session already exists",
			setup: func(repo *sessionRepoMock) {
				rate, _ := valueobject.NewChipRate(2)
				existing := entity.NewSession("s1", rate, time.Now())

				repo.findFn = func(tx usecase.Tx, id entity.SessionID) (*entity.Session, error) {
					return existing, nil
				}
			},
			cmd: usecase.NewStartSessionCommand(
				"s1",
				2,
			),
		},
		{
			name: "find error",
			setup: func(repo *sessionRepoMock) {
				repo.findFn = func(tx usecase.Tx, id entity.SessionID) (*entity.Session, error) {
					return nil, dbErr
				}
			},
			cmd: usecase.NewStartSessionCommand(
				"s1",
				2,
			),
			wantErr: dbErr,
		},
		{
			name: "invalid chip rate",
			setup: func(repo *sessionRepoMock) {
				repo.findFn = func(tx usecase.Tx, id entity.SessionID) (*entity.Session, error) {
					return nil, entity.ErrSessionNotFound
				}
			},
			cmd: usecase.NewStartSessionCommand(
				"s1",
				0,
			),
			wantErr: valueobject.ErrInvalidChips,
		},
		{
			name: "save error",
			setup: func(repo *sessionRepoMock) {
				repo.findFn = func(tx usecase.Tx, id entity.SessionID) (*entity.Session, error) {
					return nil, entity.ErrSessionNotFound
				}
				repo.saveFn = func(tx usecase.Tx, s *entity.Session) error {
					return dbErr
				}
			},
			cmd: usecase.NewStartSessionCommand(
				"s1",
				2,
			),
			wantErr: dbErr,
		},
		{
			name: "idempotent - save duplicate",
			setup: func(repo *sessionRepoMock) {
				repo.findFn = func(tx usecase.Tx, id entity.SessionID) (*entity.Session, error) {
					return nil, entity.ErrSessionNotFound
				}
				repo.saveFn = func(tx usecase.Tx, s *entity.Session) error {
					return entity.ErrSessionAlreadyExists
				}
			},
			cmd: usecase.NewStartSessionCommand(
				"s1",
				2,
			),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			sessionRepo := &sessionRepoMock{}
			txManager := &txManagerMock{}

			if tc.setup != nil {
				tc.setup(sessionRepo)
			}

			uc := usecase.NewStartSessionUseCase(
				sessionRepo,
				sessionRepo,
				txManager,
			)

			err := uc.Execute(tc.cmd)

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected error %v, got %v", tc.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
