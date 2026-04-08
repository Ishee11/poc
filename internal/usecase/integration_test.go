package usecase

import (
	"testing"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/entity/valueobject"
)

func TestIntegration_FullFlow(t *testing.T) {
	opRepo := &inMemoryOperationRepo{}
	sessionRepo := newSessionRepo()
	txManager := &txManagerStub{}

	session := entity.NewSession("s1", valueobject.NewChipRate(2))
	_ = sessionRepo.Save(nil, session)

	buyInUC := BuyInUseCase{opRepo, sessionRepo, txManager}
	cashOutUC := CashOutUseCase{opRepo, sessionRepo, txManager}
	finishUC := FinishSessionUseCase{opRepo, sessionRepo, txManager}

	// buyin
	if err := buyInUC.Execute(BuyInCommand{
		OperationID: "op1",
		SessionID:   "s1",
		PlayerID:    "p1",
		Chips:       100,
	}); err != nil {
		t.Fatalf("buyin failed: %v", err)
	}

	// cashout
	if err := cashOutUC.Execute(CashOutCommand{
		OperationID: "op2",
		SessionID:   "s1",
		PlayerID:    "p1",
		Chips:       100,
	}); err != nil {
		t.Fatalf("cashout failed: %v", err)
	}

	// finish
	if err := finishUC.Execute(FinishSessionCommand{
		SessionID: "s1",
	}); err != nil {
		t.Fatalf("finish failed: %v", err)
	}

	s, _ := sessionRepo.FindByID(nil, "s1")

	if s.Status() != entity.StatusFinished {
		t.Fatalf("expected finished, got %s", s.Status())
	}
}
