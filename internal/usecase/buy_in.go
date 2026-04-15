package usecase

import (
	"time"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase/command"
)

type BuyInUseCase struct {
	opWriter        OperationWriter
	sessionReader   SessionReader
	sessionWriter   SessionWriter
	txManager       TxManager
	idGen           OperationIDGenerator
	idempotencyRepo IdempotencyRepository
	playerRepo      PlayerRepository
}

func (uc *BuyInUseCase) OperationWriter() OperationWriter {
	return uc.opWriter
}

func NewBuyInUseCase(
	opWriter OperationWriter,
	sessionReader SessionReader,
	sessionWriter SessionWriter,
	txManager TxManager,
	idGen OperationIDGenerator,
	idempotencyRepo IdempotencyRepository,
	playerRepo PlayerRepository,
) *BuyInUseCase {
	return &BuyInUseCase{
		opWriter:        opWriter,
		sessionReader:   sessionReader,
		sessionWriter:   sessionWriter,
		txManager:       txManager,
		idGen:           idGen,
		idempotencyRepo: idempotencyRepo,
		playerRepo:      playerRepo,
	}
}

// func (uc *BuyInUseCase) Execute(cmd command.BuyInCommand) error {
// 	if cmd.Chips <= 0 {
// 		return entity.ErrInvalidChips
// 	}

// 	return uc.txManager.RunInTx(func(tx Tx) error {
// 		return Idempotent(tx, uc.idempotencyRepo, cmd.RequestID, func() error {

// 			// 2. загружаем session
// 			session, err := uc.sessionReader.FindByID(tx, cmd.SessionID)
// 			if err != nil {
// 				return err
// 			}

// 			if session.Status() != entity.StatusActive {
// 				return entity.ErrSessionNotActive
// 			}

// 			// 3. бизнес-логика
// 			if err := session.BuyIn(cmd.Chips); err != nil {
// 				return err
// 			}

// 			// 4. создаём operation
// 			opID := uc.idGen.New()

// 			playerID, err := uc.playerRepo.GetOrCreate(
// 				tx,
// 				cmd.SessionID,
// 				string(cmd.PlayerID), // пока используем как name
// 			)
// 			if err != nil {
// 				return err
// 			}

// 			op, err := entity.NewOperation(
// 				opID,
// 				cmd.RequestID,
// 				cmd.SessionID,
// 				entity.OperationBuyIn,
// 				playerID,
// 				cmd.Chips,
// 				time.Now(),
// 			)
// 			if err != nil {
// 				return err
// 			}

// 			// 5. сохраняем operation
// 			if err := uc.opWriter.Save(tx, op); err != nil {
// 				return err
// 			}

// 			// 6. сохраняем session
// 			if err := uc.sessionWriter.Save(tx, session); err != nil {
// 				return err
// 			}

// 			return nil
// 		})
// 	})
// }

func (uc *BuyInUseCase) Execute(cmd command.BuyInCommand) error {
	if cmd.Chips <= 0 {
		return entity.ErrInvalidChips
	}

	return uc.txManager.RunInTx(func(tx Tx) error {
		return Idempotent(tx, uc.idempotencyRepo, cmd.RequestID, func() error {
			return uc.execute(tx, cmd)
		})
	})
}

func (uc *BuyInUseCase) execute(tx Tx, cmd command.BuyInCommand) error {
	session, err := uc.getActiveSession(tx, cmd.SessionID)
	if err != nil {
		return err
	}

	playerID, err := uc.getOrCreatePlayer(tx, cmd)
	if err != nil {
		return err
	}

	if err := session.BuyIn(cmd.Chips); err != nil {
		return err
	}

	op, err := uc.buildOperation(cmd, playerID)
	if err != nil {
		return err
	}

	return uc.save(tx, op, session)
}

func (uc *BuyInUseCase) getActiveSession(tx Tx, id entity.SessionID) (*entity.Session, error) {
	session, err := uc.sessionReader.FindByID(tx, id)
	if err != nil {
		return nil, err
	}

	if session.Status() != entity.StatusActive {
		return nil, entity.ErrSessionNotActive
	}

	return session, nil
}

func (uc *BuyInUseCase) getOrCreatePlayer(
	tx Tx,
	cmd command.BuyInCommand,
) (entity.PlayerID, error) {
	return uc.playerRepo.GetOrCreate(
		tx,
		cmd.SessionID,
		string(cmd.PlayerID),
	)
}

func (uc *BuyInUseCase) buildOperation(
	cmd command.BuyInCommand,
	playerID entity.PlayerID,
) (*entity.Operation, error) {

	return entity.NewOperation(
		uc.idGen.New(),
		cmd.RequestID,
		cmd.SessionID,
		entity.OperationBuyIn,
		playerID,
		cmd.Chips,
		time.Now(),
	)
}

func (uc *BuyInUseCase) save(
	tx Tx,
	op *entity.Operation,
	session *entity.Session,
) error {

	if err := uc.opWriter.Save(tx, op); err != nil {
		return err
	}

	return uc.sessionWriter.Save(tx, session)
}
