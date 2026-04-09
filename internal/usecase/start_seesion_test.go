package usecase

import (
	"errors"
	"testing"
	"time"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/entity/valueobject"
)

func TestStartSessionUseCase_Execute(t *testing.T) {
	tt := []struct {
		name    string
		setup   func(repo *sessionRepoMock)
		cmd     StartSessionCommand
		wantErr error
	}{
		{
			name: "success",
			setup: func(repo *sessionRepoMock) {
				repo.findFn = func(tx Tx, id entity.SessionID) (*entity.Session, error) {
					return nil, entity.ErrSessionNotFound
				}
				repo.saveFn = func(tx Tx, s *entity.Session) error {
					return nil
				}
			},
			cmd: StartSessionCommand{
				SessionID: "s1",
				ChipRate:  2,
			},
		},
		{
			name: "idempotent - session already exists",
			setup: func(repo *sessionRepoMock) {
				rate, _ := valueobject.NewChipRate(2)
				existing := entity.NewSession("s1", rate, time.Now())

				repo.findFn = func(tx Tx, id entity.SessionID) (*entity.Session, error) {
					return existing, nil
				}
			},
			cmd: StartSessionCommand{
				SessionID: "s1",
				ChipRate:  2,
			},
		},
		{
			name: "find error",
			setup: func(repo *sessionRepoMock) {
				repo.findFn = func(tx Tx, id entity.SessionID) (*entity.Session, error) {
					return nil, errors.New("db error")
				}
			},
			cmd: StartSessionCommand{
				SessionID: "s1",
				ChipRate:  2,
			},
			wantErr: errors.New("db error"),
		},
		{
			name: "invalid chip rate",
			setup: func(repo *sessionRepoMock) {
				repo.findFn = func(tx Tx, id entity.SessionID) (*entity.Session, error) {
					return nil, entity.ErrSessionNotFound
				}
			},
			cmd: StartSessionCommand{
				SessionID: "s1",
				ChipRate:  0,
			},
			wantErr: entity.ErrInvalidChips,
		},
		{
			name: "save error",
			setup: func(repo *sessionRepoMock) {
				repo.findFn = func(tx Tx, id entity.SessionID) (*entity.Session, error) {
					return nil, entity.ErrSessionNotFound
				}
				repo.saveFn = func(tx Tx, s *entity.Session) error {
					return errors.New("db error")
				}
			},
			cmd: StartSessionCommand{
				SessionID: "s1",
				ChipRate:  2,
			},
			wantErr: errors.New("db error"),
		},
		{
			name: "idempotent - save duplicate",
			setup: func(repo *sessionRepoMock) {
				repo.findFn = func(tx Tx, id entity.SessionID) (*entity.Session, error) {
					return nil, entity.ErrSessionNotFound
				}
				repo.saveFn = func(tx Tx, s *entity.Session) error {
					return entity.ErrSessionAlreadyExists
				}
			},
			cmd: StartSessionCommand{
				SessionID: "s1",
				ChipRate:  2,
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			sessionRepo := &sessionRepoMock{}
			txManager := &txManagerMock{}

			if tc.setup != nil {
				tc.setup(sessionRepo)
			}

			uc := StartSessionUseCase{
				sessionRepo: sessionRepo,
				txManager:   txManager,
			}

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
