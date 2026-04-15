package usecase_test

import (
	"testing"
	"time"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/entity/valueobject"
	"github.com/ishee11/poc/internal/usecase"
)

func TestIntegration_FullFlow(t *testing.T) {
	opRepo := &inMemoryOperationRepo{}
	sessionRepo := newSessionRepo()
	txManager := &txManagerStub{}

	rate, _ := valueobject.NewChipRate(2)

	session := entity.NewSession("s1", rate, time.Now())
	_ = sessionRepo.Save(nil, session)

	idGen := &operationIDGeneratorMock{}
	idempotencyRepo := defaultIdempotencyRepo()

	buyInUC := usecase.NewBuyInUseCase(
		opRepo,
		sessionRepo,
		sessionRepo,
		txManager,
		idGen,
		idempotencyRepo,
		&playerRepoMock{},
	)

	cashOutUC := usecase.NewCashOutUseCase(
		opRepo,
		opRepo,
		opRepo,
		sessionRepo,
		sessionRepo,
		txManager,
		idGen,
		idempotencyRepo,
		&playerRepoMock{},
	)

	reverseUC := usecase.NewReverseOperationUseCase(
		opRepo,
		opRepo,
		opRepo,
		sessionRepo,
		sessionRepo,
		txManager,
		idGen,
		idempotencyRepo,
	)

	finishUC := usecase.NewFinishSessionUseCase(
		opRepo,
		sessionRepo,
		sessionRepo,
		txManager,
		idempotencyRepo,
	)

	// --- 1. BuyIn ---
	idGen.id = "op1"
	if err := buyInUC.Execute(usecase.BuyInCommand{
		RequestID: "req-1",
		SessionID: "s1",
		PlayerID:  "p1",
		Chips:     100,
	}); err != nil {
		t.Fatalf("buyin failed: %v", err)
	}

	// --- 2. CashOut ---
	idGen.id = "op2"
	if err := cashOutUC.Execute(usecase.CashOutCommand{
		RequestID: "req-2",
		SessionID: "s1",
		PlayerID:  "p1",
		Chips:     100,
	}); err != nil {
		t.Fatalf("cashout failed: %v", err)
	}

	// --- 3. Reversal (отменяем cashout) ---
	idGen.id = "op3"

	if err := reverseUC.Execute(usecase.ReverseOperationCommand{
		RequestID:         "req-3",
		TargetOperationID: "op2",
	}); err != nil {
		t.Fatalf("reversal failed: %v", err)
	}

	// --- 4. Finish должен упасть ---
	if err := finishUC.Execute(usecase.FinishSessionCommand{
		RequestID: "req-5",
		SessionID: "s1",
	}); err == nil {
		t.Fatalf("expected finish to fail")
	}

	// --- 5. CashOut again ---
	idGen.id = "op4"
	if err := cashOutUC.Execute(usecase.CashOutCommand{
		RequestID: "req-4",
		SessionID: "s1",
		PlayerID:  "p1",
		Chips:     100,
	}); err != nil {
		t.Fatalf("cashout2 failed: %v", err)
	}

	// --- 6. Finish success ---
	if err := finishUC.Execute(usecase.FinishSessionCommand{
		RequestID: "req-6",
		SessionID: "s1",
	}); err != nil {
		t.Fatalf("finish failed: %v", err)
	}

	// --- 7. Проверка состояния ---
	s, _ := sessionRepo.FindByID(nil, "s1")

	if s.Status() != entity.StatusFinished {
		t.Fatalf("expected finished, got %s", s.Status())
	}
}
