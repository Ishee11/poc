package usecase

import (
	"time"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/entity/valueobject"
	"github.com/ishee11/poc/internal/usecase/command"
)

type StartSessionUseCase struct {
	sessionReader SessionReader
	sessionWriter SessionWriter
	txManager     TxManager
	idGenerator   SessionIDGenerator
}

func NewStartSessionUseCase(
	sessionReader SessionReader,
	sessionWriter SessionWriter,
	txManager TxManager,
	idGenerator SessionIDGenerator,
) *StartSessionUseCase {
	return &StartSessionUseCase{
		sessionReader: sessionReader,
		sessionWriter: sessionWriter,
		txManager:     txManager,
		idGenerator:   idGenerator,
	}
}

func (uc *StartSessionUseCase) Execute(cmd command.StartSessionCommand) (entity.SessionID, error) {
	var result entity.SessionID

	err := uc.txManager.RunInTx(func(tx Tx) error {
		id, err := uc.execute(tx, cmd)
		if err != nil {
			return err
		}
		result = id
		return nil
	})

	if err != nil {
		return "", err
	}

	return result, nil
}

func (uc *StartSessionUseCase) execute(tx Tx, cmd command.StartSessionCommand) (entity.SessionID, error) {

	rate, err := valueobject.NewChipRate(cmd.ChipRate)
	if err != nil {
		return "", err
	}
	if cmd.BigBlind <= 0 {
		return "", valueobject.ErrInvalidChips
	}

	id := uc.idGenerator.New()

	session := entity.NewSession(id, rate, cmd.BigBlind, time.Now())

	if err := uc.sessionWriter.Save(tx, session); err != nil {
		return "", err
	}

	return id, nil
}
