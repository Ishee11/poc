package usecase

import (
	"time"

	"github.com/ishee11/poc/internal/entity"
)

type Helper struct {
	sessionReader SessionReader
	sessionWriter SessionWriter
	playerRepo    PlayerRepository
	opWriter      OperationWriter
	idGen         OperationIDGenerator
}

func NewHelper(
	sessionReader SessionReader,
	sessionWriter SessionWriter,
	playerRepo PlayerRepository,
	opWriter OperationWriter,
	idGen OperationIDGenerator,
) *Helper {
	return &Helper{
		sessionReader: sessionReader,
		sessionWriter: sessionWriter,
		playerRepo:    playerRepo,
		opWriter:      opWriter,
		idGen:         idGen,
	}
}

func (h *Helper) GetActiveSession(tx Tx, id entity.SessionID) (*entity.Session, error) {
	session, err := h.sessionReader.FindByID(tx, id)
	if err != nil {
		return nil, err
	}

	if session.Status() != entity.StatusActive {
		return nil, entity.ErrSessionNotActive
	}

	return session, nil
}

func (h *Helper) GetOrCreatePlayer(
	tx Tx,
	sessionID entity.SessionID,
	playerName string,
) (entity.PlayerID, error) {
	return h.playerRepo.GetOrCreate(tx, sessionID, playerName)
}

func (h *Helper) BuildOperation(
	requestID string,
	sessionID entity.SessionID,
	opType entity.OperationType,
	playerID entity.PlayerID,
	chips int64,
) (*entity.Operation, error) {

	return entity.NewOperation(
		h.idGen.New(),
		requestID,
		sessionID,
		opType,
		playerID,
		chips,
		time.Now(),
	)
}

func (h *Helper) Save(
	tx Tx,
	op *entity.Operation,
	session *entity.Session,
) error {

	if err := h.opWriter.Save(tx, op); err != nil {
		return err
	}

	return h.sessionWriter.Save(tx, session)
}
