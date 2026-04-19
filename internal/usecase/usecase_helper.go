package usecase

import (
	"strings"
	"time"

	"github.com/ishee11/poc/internal/entity"
)

type Helper struct {
	sessionReader     SessionReader
	sessionWriter     SessionWriter
	playerRepo        PlayerRepository
	opWriter          OperationWriter
	idGen             OperationIDGenerator
	playerIDGenerator PlayerIDGenerator
}

func NewHelper(
	sessionReader SessionReader,
	sessionWriter SessionWriter,
	playerRepo PlayerRepository,
	opWriter OperationWriter,
	idGen OperationIDGenerator,
	playerIDGenerator PlayerIDGenerator,
) *Helper {
	return &Helper{
		sessionReader:     sessionReader,
		sessionWriter:     sessionWriter,
		playerRepo:        playerRepo,
		opWriter:          opWriter,
		idGen:             idGen,
		playerIDGenerator: playerIDGenerator,
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

func (h *Helper) BuildPlayer(name string) (*entity.Player, error) {
	name = strings.TrimSpace(name)

	id := h.playerIDGenerator.New()
	return entity.NewPlayer(id, name)
}

func (h *Helper) SavePlayer(tx Tx, player *entity.Player) (entity.PlayerID, error) {
	if err := h.playerRepo.Create(tx, player); err != nil {
		return "", err
	}
	return player.ID(), nil
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
