package usecase

import (
	"errors"
	"testing"
	"time"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/entity/valueobject"
)

func TestReverseOperationUseCase(t *testing.T) {

	now := time.Now()

	rate, _ := valueobject.NewChipRate(2)

	targetOp, _ := entity.NewOperation(
		"target",
		"req-target",
		"s1",
		entity.OperationCashOut,
		"p1",
		100,
		now,
	)

	tt := []struct {
		name    string
		setup   func(opRepo *operationRepoMock, sessionRepo *sessionRepoMock)
		wantErr error
	}{
		{
			name: "success",
			setup: func(opRepo *operationRepoMock, sessionRepo *sessionRepoMock) {
				session := entity.NewSession("s1", rate, time.Now())

				opRepo.getByRequestIDFn = func(tx Tx, requestID string) (*entity.Operation, error) {
					return nil, nil
				}
				opRepo.getByIDFn = func(tx Tx, id entity.OperationID) (*entity.Operation, error) {
					return targetOp, nil
				}
				opRepo.existsReversalFn = func(tx Tx, id entity.OperationID) (bool, error) {
					return false, nil
				}
				opRepo.saveFn = func(tx Tx, op *entity.Operation) error {
					if op.RequestID() != "req-1" {
						t.Fatalf("unexpected requestID: %s", op.RequestID())
					}
					return nil
				}

				sessionRepo.findFn = func(tx Tx, id entity.SessionID) (*entity.Session, error) {
					return session, nil
				}
				sessionRepo.saveFn = func(tx Tx, s *entity.Session) error {
					return nil
				}
			},
		},
		{
			name: "target not found",
			setup: func(opRepo *operationRepoMock, sessionRepo *sessionRepoMock) {
				opRepo.getByRequestIDFn = func(tx Tx, requestID string) (*entity.Operation, error) {
					return nil, nil
				}
				opRepo.getByIDFn = func(tx Tx, id entity.OperationID) (*entity.Operation, error) {
					return nil, nil
				}
			},
			wantErr: entity.ErrOperationNotFound,
		},
		{
			name: "target is reversal",
			setup: func(opRepo *operationRepoMock, sessionRepo *sessionRepoMock) {
				reversalOp, _ := entity.NewReversalOperation(
					"rev",
					"req-rev",
					"s1",
					"p1",
					100,
					"ref",
					time.Now(),
				)

				opRepo.getByRequestIDFn = func(tx Tx, requestID string) (*entity.Operation, error) {
					return nil, nil
				}
				opRepo.getByIDFn = func(tx Tx, id entity.OperationID) (*entity.Operation, error) {
					return reversalOp, nil
				}
			},
			wantErr: entity.ErrInvalidOperation,
		},
		{
			name: "already reversed",
			setup: func(opRepo *operationRepoMock, sessionRepo *sessionRepoMock) {
				opRepo.getByRequestIDFn = func(tx Tx, requestID string) (*entity.Operation, error) {
					return nil, nil
				}
				opRepo.getByIDFn = func(tx Tx, id entity.OperationID) (*entity.Operation, error) {
					return targetOp, nil
				}
				opRepo.existsReversalFn = func(tx Tx, id entity.OperationID) (bool, error) {
					return true, nil
				}
			},
			wantErr: entity.ErrOperationAlreadyReversed,
		},
		{
			name: "session not active",
			setup: func(opRepo *operationRepoMock, sessionRepo *sessionRepoMock) {
				finished := entity.NewSession("s1", rate, time.Now())
				_ = finished.Finish()

				opRepo.getByRequestIDFn = func(tx Tx, requestID string) (*entity.Operation, error) {
					return nil, nil
				}
				opRepo.getByIDFn = func(tx Tx, id entity.OperationID) (*entity.Operation, error) {
					return targetOp, nil
				}
				opRepo.existsReversalFn = func(tx Tx, id entity.OperationID) (bool, error) {
					return false, nil
				}

				sessionRepo.findFn = func(tx Tx, id entity.SessionID) (*entity.Session, error) {
					return finished, nil
				}
			},
			wantErr: entity.ErrSessionNotActive,
		},
		{
			name: "idempotent by requestID",
			setup: func(opRepo *operationRepoMock, sessionRepo *sessionRepoMock) {
				existingOp, _ := entity.NewReversalOperation(
					"existing",
					"req-1",
					"s1",
					"p1",
					100,
					"target",
					time.Now(),
				)

				opRepo.getByRequestIDFn = func(tx Tx, requestID string) (*entity.Operation, error) {
					return existingOp, nil
				}
				opRepo.saveFn = func(tx Tx, op *entity.Operation) error {
					t.Fatal("save should not be called on idempotent request")
					return nil
				}

				sessionRepo.findFn = func(tx Tx, id entity.SessionID) (*entity.Session, error) {
					t.Fatal("session should not be loaded on idempotent request")
					return nil, nil
				}
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			opRepo := &operationRepoMock{}
			sessionRepo := &sessionRepoMock{}
			txManager := &txManagerMock{}

			if tc.setup != nil {
				tc.setup(opRepo, sessionRepo)
			}

			uc := ReverseOperationUseCase{
				opRepo:      opRepo,
				sessionRepo: sessionRepo,
				txManager:   txManager,
			}

			err := uc.Execute(ReverseOperationCommand{
				RequestID:         "req-1",
				OperationID:       "op1",
				TargetOperationID: "target",
			})

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected %v, got %v", tc.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
