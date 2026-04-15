package usecase

import (
	"errors"
	"time"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/entity/valueobject"
	"github.com/ishee11/poc/internal/usecase/command"
)

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

func (uc *StartSessionUseCase) Execute(cmd command.StartSessionCommand) error {
	return uc.txManager.RunInTx(func(tx Tx) error {
		return uc.execute(tx, cmd)
	})
}

func (uc *StartSessionUseCase) execute(tx Tx, cmd command.StartSessionCommand) error {

	// 1. идемпотентность
	existing, err := uc.sessionReader.FindByID(tx, cmd.SessionID)
	if err != nil && !errors.Is(err, entity.ErrSessionNotFound) {
		return err
	}

	if existing != nil {
		if existing.Status() == entity.StatusActive {
			return nil
		}
		return entity.ErrSessionAlreadyExists
	}

	// 2. валидация
	rate, err := valueobject.NewChipRate(cmd.ChipRate)
	if err != nil {
		return err
	}

	// 3. создание
	session := entity.NewSession(cmd.SessionID, rate, time.Now())

	// 4. сохранение
	if err := uc.sessionWriter.Save(tx, session); err != nil {
		if errors.Is(err, entity.ErrSessionAlreadyExists) {
			return nil
		}
		return err
	}

	return nil
}
