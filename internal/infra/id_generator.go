package infra

import (
	"github.com/google/uuid"

	"github.com/ishee11/poc/internal/entity"
)

type UUIDOperationIDGenerator struct{}

func (g *UUIDOperationIDGenerator) New() entity.OperationID {
	return entity.OperationID(uuid.NewString())
}

type UUIDPlayerIDGenerator struct{}

func (g *UUIDPlayerIDGenerator) New() entity.PlayerID {
	return entity.PlayerID(uuid.New().String())
}

type UUIDSessionIDGenerator struct{}

func (g *UUIDSessionIDGenerator) New() entity.SessionID {
	return entity.SessionID(uuid.New().String())
}

type UUIDBlindClockIDGenerator struct{}

func (g *UUIDBlindClockIDGenerator) New() entity.BlindClockID {
	return entity.BlindClockID(uuid.New().String())
}
