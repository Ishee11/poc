package session

import (
	"context"
	"errors"
	"testing"

	"github.com/ishee11/poc/internal/entity"
)

// --- mock repo ---

type mockRepo struct {
	getFn    func(ctx context.Context, id string) (*entity.Session, error)
	saveFn   func(ctx context.Context, s *entity.Session) error
	createFn func(ctx context.Context, s *entity.Session) error
}

func (m *mockRepo) GetByID(ctx context.Context, id string) (*entity.Session, error) {
	return m.getFn(ctx, id)
}

func (m *mockRepo) Save(ctx context.Context, s *entity.Session) error {
	return m.saveFn(ctx, s)
}

func (m *mockRepo) Create(ctx context.Context, s *entity.Session) error {
	return m.createFn(ctx, s)
}

func TestUseCase_BuyIn_Success(t *testing.T) {
	s := entity.NewSession("s1", 10)
	_ = s.StartSession()

	calledSave := false

	repo := &mockRepo{
		getFn: func(ctx context.Context, id string) (*entity.Session, error) {
			return s, nil
		},
		saveFn: func(ctx context.Context, s *entity.Session) error {
			calledSave = true
			return nil
		},
		createFn: nil,
	}

	uc := NewUseCase(repo)

	err := uc.BuyIn(context.Background(), BuyInCommand{
		SessionID:   "s1",
		OperationID: "op1",
		PlayerID:    "p1",
		Chips:       100,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !calledSave {
		t.Fatalf("expected Save to be called")
	}
}

func TestUseCase_BuyIn_GetError(t *testing.T) {
	repo := &mockRepo{
		getFn: func(ctx context.Context, id string) (*entity.Session, error) {
			return nil, entity.ErrSessionNotFound
		},
		saveFn:   nil,
		createFn: nil,
	}

	uc := NewUseCase(repo)

	err := uc.BuyIn(context.Background(), BuyInCommand{
		SessionID: "s1",
	})

	if !errors.Is(err, entity.ErrSessionNotFound) &&
		!errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUseCase_BuyIn_DomainError(t *testing.T) {
	s := entity.NewSession("s1", 10)
	// не стартуем → ошибка домена

	repo := &mockRepo{
		getFn: func(ctx context.Context, id string) (*entity.Session, error) {
			return s, nil
		},
		saveFn: func(ctx context.Context, s *entity.Session) error {
			t.Fatalf("Save should not be called")
			return nil
		},
		createFn: nil,
	}

	uc := NewUseCase(repo)

	err := uc.BuyIn(context.Background(), BuyInCommand{
		SessionID:   "s1",
		OperationID: "op1",
		PlayerID:    "p1",
		Chips:       100,
	})

	if !errors.Is(err, entity.ErrSessionNotActive) {
		t.Fatalf("expected ErrSessionNotActive, got %v", err)
	}
}

func TestUseCase_CashOut_SaveCalled(t *testing.T) {
	s := entity.NewSession("s1", 10)
	_ = s.StartSession()
	_ = s.PlayerBuyIn("op1", "p1", 100)

	calledSave := false

	repo := &mockRepo{
		getFn: func(ctx context.Context, id string) (*entity.Session, error) {
			return s, nil
		},
		saveFn: func(ctx context.Context, s *entity.Session) error {
			calledSave = true
			return nil
		},
		createFn: nil,
	}

	uc := NewUseCase(repo)

	err := uc.CashOut(context.Background(), CashOutCommand{
		SessionID:   "s1",
		OperationID: "op2",
		PlayerID:    "p1",
		Chips:       50,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !calledSave {
		t.Fatalf("expected Save to be called")
	}
}

func TestUseCase_CreateSession_Error(t *testing.T) {
	repo := &mockRepo{
		createFn: func(ctx context.Context, s *entity.Session) error {
			return ErrSessionAlreadyExists
		},
	}

	uc := NewUseCase(repo)

	err := uc.CreateSession(context.Background(), CreateSessionCommand{
		SessionID: "s1",
		Rate:      10,
	})

	if !errors.Is(err, ErrSessionAlreadyExists) {
		t.Fatalf("expected ErrSessionAlreadyExists, got %v", err)
	}
}
