package usecase

import (
	"testing"
	"time"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/entity/valueobject"
)

func TestGetSessionOperationsUseCase_Execute(t *testing.T) {

	now := time.Now()

	rate, _ := valueobject.NewChipRate(2)

	t.Run("success - returns operations in DESC order", func(t *testing.T) {
		// --- setup session ---
		session := entity.NewSession("s1", rate, now)

		sessionRepo := &sessionRepoMock{
			findFn: func(tx Tx, id entity.SessionID) (*entity.Session, error) {
				return session, nil
			},
		}

		// --- setup operations ---
		op1, _ := entity.NewOperation("op1", "req1", "s1", entity.OperationBuyIn, "p1", 100, now.Add(1*time.Minute))
		op2, _ := entity.NewOperation("op2", "req2", "s1", entity.OperationCashOut, "p1", 50, now.Add(2*time.Minute))
		op3, _ := entity.NewOperation("op3", "req3", "s1", entity.OperationBuyIn, "p2", 200, now.Add(3*time.Minute))

		opRepo := &inMemoryOperationRepo{
			operations: []*entity.Operation{op1, op2, op3},
		}

		uc := GetSessionOperationsUseCase{
			sessionReader: sessionRepo,
			projection:    opRepo,
			txManager:     &txManagerMock{},
		}

		// --- execute ---
		res, err := uc.Execute(GetSessionOperationsQuery{
			SessionID: "s1",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(res.Operations) != 3 {
			t.Fatalf("expected 3 operations, got %d", len(res.Operations))
		}

		// DESC порядок: op3, op2, op1
		if res.Operations[0].ID != "op3" {
			t.Fatalf("expected first op=op3, got %s", res.Operations[0].ID)
		}
		if res.Operations[1].ID != "op2" {
			t.Fatalf("expected second op=op2, got %s", res.Operations[1].ID)
		}
		if res.Operations[2].ID != "op1" {
			t.Fatalf("expected third op=op1, got %s", res.Operations[2].ID)
		}
	})

	t.Run("limit works", func(t *testing.T) {
		session := entity.NewSession("s1", rate, now)

		sessionRepo := &sessionRepoMock{
			findFn: func(tx Tx, id entity.SessionID) (*entity.Session, error) {
				return session, nil
			},
		}

		op1, _ := entity.NewOperation("op1", "req1", "s1", entity.OperationBuyIn, "p1", 100, now)
		op2, _ := entity.NewOperation("op2", "req2", "s1", entity.OperationBuyIn, "p1", 200, now)
		op3, _ := entity.NewOperation("op3", "req3", "s1", entity.OperationBuyIn, "p1", 300, now)

		opRepo := &inMemoryOperationRepo{
			operations: []*entity.Operation{op1, op2, op3},
		}

		uc := GetSessionOperationsUseCase{
			sessionReader: sessionRepo,
			projection:    opRepo,
			txManager:     &txManagerMock{},
		}

		res, err := uc.Execute(GetSessionOperationsQuery{
			SessionID: "s1",
			Limit:     2,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(res.Operations) != 2 {
			t.Fatalf("expected 2 operations, got %d", len(res.Operations))
		}
	})

	t.Run("offset works", func(t *testing.T) {
		session := entity.NewSession("s1", rate, now)

		sessionRepo := &sessionRepoMock{
			findFn: func(tx Tx, id entity.SessionID) (*entity.Session, error) {
				return session, nil
			},
		}

		op1, _ := entity.NewOperation("op1", "req1", "s1", entity.OperationBuyIn, "p1", 100, now)
		op2, _ := entity.NewOperation("op2", "req2", "s1", entity.OperationBuyIn, "p1", 200, now)
		op3, _ := entity.NewOperation("op3", "req3", "s1", entity.OperationBuyIn, "p1", 300, now)

		opRepo := &inMemoryOperationRepo{
			operations: []*entity.Operation{op1, op2, op3},
		}

		uc := GetSessionOperationsUseCase{
			sessionReader: sessionRepo,
			projection:    opRepo,
			txManager:     &txManagerMock{},
		}

		res, err := uc.Execute(GetSessionOperationsQuery{
			SessionID: "s1",
			Offset:    1,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// ожидаем: пропустили последнюю → остаётся 2
		if len(res.Operations) != 2 {
			t.Fatalf("expected 2 operations, got %d", len(res.Operations))
		}

		// порядок: op2, op1
		if res.Operations[0].ID != "op2" {
			t.Fatalf("expected first op=op2, got %s", res.Operations[0].ID)
		}
	})

	t.Run("empty result", func(t *testing.T) {
		session := entity.NewSession("s1", rate, now)

		sessionRepo := &sessionRepoMock{
			findFn: func(tx Tx, id entity.SessionID) (*entity.Session, error) {
				return session, nil
			},
		}

		opRepo := &inMemoryOperationRepo{}

		uc := GetSessionOperationsUseCase{
			sessionReader: sessionRepo,
			projection:    opRepo,
			txManager:     &txManagerMock{},
		}

		res, err := uc.Execute(GetSessionOperationsQuery{
			SessionID: "s1",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(res.Operations) != 0 {
			t.Fatalf("expected 0 operations, got %d", len(res.Operations))
		}
	})
}
