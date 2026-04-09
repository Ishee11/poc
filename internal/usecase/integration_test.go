package usecase

import (
	"testing"
	"time"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/entity/valueobject"
)

func TestIntegration_FullFlow(t *testing.T) {
	opRepo := &inMemoryOperationRepo{
		operations: []*entity.Operation{},
	}
	sessionRepo := newSessionRepo()
	txManager := &txManagerStub{}

	rate, _ := valueobject.NewChipRate(2)

	session := entity.NewSession("s1", rate, time.Now())
	if err := sessionRepo.Save(nil, session); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	idGen := &operationIDGeneratorMock{}

	buyInUC := BuyInUseCase{opRepo, sessionRepo, txManager, idGen}
	cashOutUC := CashOutUseCase{opRepo, sessionRepo, txManager, idGen}
	reverseUC := ReverseOperationUseCase{opRepo, sessionRepo, txManager}
	finishUC := FinishSessionUseCase{opRepo, sessionRepo, txManager}

	// --- 1. BuyIn ---
	idGen.id = "op1"
	if err := buyInUC.Execute(BuyInCommand{
		RequestID: "req-1",
		SessionID: "s1",
		PlayerID:  "p1",
		Chips:     100,
	}); err != nil {
		t.Fatalf("buyin failed: %v", err)
	}

	// --- 2. CashOut ---
	idGen.id = "op2"
	if err := cashOutUC.Execute(CashOutCommand{
		RequestID: "req-2",
		SessionID: "s1",
		PlayerID:  "p1",
		Chips:     100,
	}); err != nil {
		t.Fatalf("cashout failed: %v", err)
	}

	// --- 3. Reversal (отменяем cashout) ---
	if err := reverseUC.Execute(ReverseOperationCommand{
		RequestID:         "req-3",
		OperationID:       "op3",
		TargetOperationID: "op2",
	}); err != nil {
		t.Fatalf("reversal failed: %v", err)
	}

	// --- 4. Finish ДОЛЖЕН упасть (tableChips = 100) ---
	if err := finishUC.Execute(FinishSessionCommand{
		SessionID: "s1",
	}); err == nil {
		t.Fatalf("expected finish to fail, but got nil")
	}

	// --- 5. Повторный CashOut ---
	idGen.id = "op4"
	if err := cashOutUC.Execute(CashOutCommand{
		RequestID: "req-4",
		SessionID: "s1",
		PlayerID:  "p1",
		Chips:     100,
	}); err != nil {
		t.Fatalf("cashout2 failed: %v", err)
	}

	// --- 6. Теперь Finish ДОЛЖЕН пройти ---
	if err := finishUC.Execute(FinishSessionCommand{
		SessionID: "s1",
	}); err != nil {
		t.Fatalf("finish failed: %v", err)
	}

	// --- 7. Проверка финального состояния ---
	s, err := sessionRepo.FindByID(nil, "s1")
	if err != nil {
		t.Fatalf("failed to find session: %v", err)
	}

	if s.Status() != entity.StatusFinished {
		t.Fatalf("expected finished, got %s", s.Status())
	}
}
