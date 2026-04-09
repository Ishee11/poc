package usecase

import (
	"errors"
	"time"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/entity/valueobject"
)

type StartSessionCommand struct {
	SessionID entity.SessionID
	ChipRate  int64
}

type StartSessionUseCase struct {
	sessionReader SessionReader
	sessionWriter SessionWriter
	txManager     TxManager
}

func NewStartSessionUseCase(
	sessionReader SessionReader,
	sessionWriter SessionWriter,
	txManager TxManager,
) *StartSessionUseCase {
	return &StartSessionUseCase{
		sessionReader: sessionReader,
		sessionWriter: sessionWriter,
		txManager:     txManager,
	}
}

func (uc *StartSessionUseCase) Execute(cmd StartSessionCommand) error {
	return uc.txManager.RunInTx(func(tx Tx) error {

		// 1. идемпотентность
		existing, err := uc.sessionReader.FindByID(tx, cmd.SessionID)
		if err != nil && !errors.Is(err, entity.ErrSessionNotFound) {
			return err
		}
		if existing != nil {
			return nil
		}

		// 2. безопасная валидация chipRate
		rate, err := valueobject.NewChipRate(cmd.ChipRate)
		if err != nil {
			return err
		}

		// 3. создаём session
		now := time.Now()
		session := entity.NewSession(cmd.SessionID, rate, now)

		// 4. сохраняем
		if err := uc.sessionWriter.Save(tx, session); err != nil {
			if errors.Is(err, entity.ErrSessionAlreadyExists) {
				return nil
			}
			return err
		}

		return nil
	})
}
