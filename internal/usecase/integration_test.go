package usecase

import (
	"testing"
	"time"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/entity/valueobject"
)

func TestIntegration_FullFlow(t *testing.T) {
	opRepo := &inMemoryOperationRepo{}
	sessionRepo := newSessionRepo()
	txManager := &txManagerStub{}

	rate, _ := valueobject.NewChipRate(2)

	session := entity.NewSession("s1", rate, time.Now())
	if err := sessionRepo.Save(nil, session); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	buyInUC := BuyInUseCase{opRepo, sessionRepo, txManager}
	cashOutUC := CashOutUseCase{opRepo, sessionRepo, txManager}
	reverseUC := ReverseOperationUseCase{opRepo, sessionRepo, txManager}
	finishUC := FinishSessionUseCase{opRepo, sessionRepo, txManager}

	// --- 1. BuyIn ---
	if err := buyInUC.Execute(BuyInCommand{
		OperationID: "op1",
		SessionID:   "s1",
		PlayerID:    "p1",
		Chips:       100,
	}); err != nil {
		t.Fatalf("buyin failed: %v", err)
	}

	// --- 2. CashOut ---
	if err := cashOutUC.Execute(CashOutCommand{
		OperationID: "op2",
		SessionID:   "s1",
		PlayerID:    "p1",
		Chips:       100,
	}); err != nil {
		t.Fatalf("cashout failed: %v", err)
	}

	// --- 3. Reversal (отменяем cashout) ---
	if err := reverseUC.Execute(ReverseOperationCommand{
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
	if err := cashOutUC.Execute(CashOutCommand{
		OperationID: "op4",
		SessionID:   "s1",
		PlayerID:    "p1",
		Chips:       100,
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
