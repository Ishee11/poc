package usecase

import (
	"time"

	"github.com/ishee11/poc/internal/entity"
)

type OperationIDGenerator interface {
	New() entity.OperationID
}

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

type BuyInCommand struct {
	RequestID string
	SessionID entity.SessionID
	PlayerID  entity.PlayerID
	Chips     int64
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

func (uc *BuyInUseCase) Execute(cmd BuyInCommand) error {
	if cmd.Chips <= 0 {
		return entity.ErrInvalidChips
	}

	return uc.txManager.RunInTx(func(tx Tx) error {
		return Idempotent(tx, uc.idempotencyRepo, cmd.RequestID, func() error {

			// 2. загружаем session
			session, err := uc.sessionReader.FindByID(tx, cmd.SessionID)
			if err != nil {
				return err
			}

			if session.Status() != entity.StatusActive {
				return entity.ErrSessionNotActive
			}

			// 3. бизнес-логика
			if err := session.BuyIn(cmd.Chips); err != nil {
				return err
			}

			// 4. создаём operation
			opID := uc.idGen.New()

			playerID, err := uc.playerRepo.GetOrCreate(
				tx,
				cmd.SessionID,
				string(cmd.PlayerID), // пока используем как name
			)
			if err != nil {
				return err
			}

			op, err := entity.NewOperation(
				opID,
				cmd.RequestID,
				cmd.SessionID,
				entity.OperationBuyIn,
				playerID,
				cmd.Chips,
				time.Now(),
			)
			if err != nil {
				return err
			}

			// 5. сохраняем operation
			if err := uc.opWriter.Save(tx, op); err != nil {
				return err
			}

			// 6. сохраняем session
			if err := uc.sessionWriter.Save(tx, session); err != nil {
				return err
			}

			return nil
		})
	})
}
