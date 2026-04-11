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

	idGen := &operationIDGeneratorMock{id: "gen-id"}

	tt := []struct {
		name    string
		setup   func(opRepo *operationRepoMock, sessionRepo *sessionRepoMock)
		idem    IdempotencyRepository
		wantErr error
	}{
		{
			name: "success",
			setup: func(opRepo *operationRepoMock, sessionRepo *sessionRepoMock) {
				session := entity.NewSession("s1", rate, time.Now())

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
					if op.ID() != "gen-id" {
						t.Fatalf("unexpected id: %s", op.ID())
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

				opRepo.getByIDFn = func(tx Tx, id entity.OperationID) (*entity.Operation, error) {
					return reversalOp, nil
				}
			},
			wantErr: entity.ErrInvalidOperation,
		},
		{
			name: "already reversed",
			setup: func(opRepo *operationRepoMock, sessionRepo *sessionRepoMock) {
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
			name: "idempotent via duplicate request error",
			setup: func(opRepo *operationRepoMock, sessionRepo *sessionRepoMock) {
				session := entity.NewSession("s1", rate, time.Now())

				opRepo.getByIDFn = func(tx Tx, id entity.OperationID) (*entity.Operation, error) {
					return targetOp, nil
				}
				opRepo.existsReversalFn = func(tx Tx, id entity.OperationID) (bool, error) {
					return false, nil
				}
				opRepo.saveFn = func(tx Tx, op *entity.Operation) error {
					t.Fatal("operation save should not be called for duplicate request")
					return nil
				}

				sessionRepo.findFn = func(tx Tx, id entity.SessionID) (*entity.Session, error) {
					return session, nil
				}
			},
			idem: &idempotencyRepoMock{
				saveFn: func(tx Tx, requestID string) error {
					return entity.ErrDuplicateRequest
				},
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

			var idempotencyRepo IdempotencyRepository = defaultIdempotencyRepo()
			if tc.idem != nil {
				idempotencyRepo = tc.idem
			}

			uc := ReverseOperationUseCase{
				opWriter:        opRepo,
				opReader:        opRepo,
				reversalChecker: opRepo,
				sessionReader:   sessionRepo,
				sessionWriter:   sessionRepo,
				txManager:       txManager,
				idGen:           idGen, // ← ВАЖНО
				idempotencyRepo: idempotencyRepo,
			}

			err := uc.Execute(ReverseOperationCommand{
				RequestID:         "req-1",
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
