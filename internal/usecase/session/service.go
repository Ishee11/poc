package session

import (
	"context"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/entity/valueobject"
)

// Repository — абстракция доступа к агрегату.
// Важно: здесь предполагается transactional boundary на уровне реализации.
type Repository interface {
	GetByID(ctx context.Context, id string) (*entity.Session, error)
	Save(ctx context.Context, s *entity.Session) error
}

// SessionUseCase — application layer.
// Оркестрирует сценарии, НЕ содержит бизнес-логики.
type SessionUseCase struct {
	repo Repository
}

func NewUseCase(repo Repository) *SessionUseCase {
	return &SessionUseCase{repo: repo}
}

//
// Commands — явное описание входных данных usecase (DDD style)
//

type BuyInCommand struct {
	SessionID   string
	OperationID string // идемпотентность на уровне домена
	PlayerID    string
	Chips       int64
}

type CashOutCommand struct {
	SessionID   string
	OperationID string
	PlayerID    string
	Chips       int64
}

type StartSessionCommand struct {
	SessionID string
}

type CloseSessionCommand struct {
	SessionID string
}

type GetResultQuery struct {
	SessionID string
	PlayerID  string
}

//
// UseCases
//

// BuyIn — игрок покупает фишки.
// flow: load → domain → save
func (uc *SessionUseCase) BuyIn(ctx context.Context, cmd BuyInCommand) error {
	session, err := uc.repo.GetByID(ctx, cmd.SessionID)
	if err != nil {
		return err
	}

	if err := session.PlayerBuyIn(cmd.OperationID, cmd.PlayerID, cmd.Chips); err != nil {
		return err
	}

	return uc.repo.Save(ctx, session)
}

// CashOut — игрок выводит фишки.
func (uc *SessionUseCase) CashOut(ctx context.Context, cmd CashOutCommand) error {
	session, err := uc.repo.GetByID(ctx, cmd.SessionID)
	if err != nil {
		return err
	}

	if err := session.PlayerCashOut(cmd.OperationID, cmd.PlayerID, cmd.Chips); err != nil {
		return err
	}

	return uc.repo.Save(ctx, session)
}

// StartSession — переводит сессию в активное состояние.
func (uc *SessionUseCase) StartSession(ctx context.Context, cmd StartSessionCommand) error {
	session, err := uc.repo.GetByID(ctx, cmd.SessionID)
	if err != nil {
		return err
	}

	if err := session.StartSession(); err != nil {
		return err
	}

	return uc.repo.Save(ctx, session)
}

// CloseSession — завершает сессию (проверки внутри domain).
func (uc *SessionUseCase) CloseSession(ctx context.Context, cmd CloseSessionCommand) error {
	session, err := uc.repo.GetByID(ctx, cmd.SessionID)
	if err != nil {
		return err
	}

	if err := session.FinishSession(); err != nil {
		return err
	}

	return uc.repo.Save(ctx, session)
}

// GetResult — query (не меняет состояние).
// Важно: здесь нет Save.
func (uc *SessionUseCase) GetResult(ctx context.Context, q GetResultQuery) (valueobject.Money, error) {
	session, err := uc.repo.GetByID(ctx, q.SessionID)
	if err != nil {
		return valueobject.Money{}, err
	}

	return session.PlayerResult(q.PlayerID)
}
