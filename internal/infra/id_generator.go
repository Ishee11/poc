package infra

import (
	"github.com/google/uuid"

	"github.com/ishee11/poc/internal/entity"
)

// --- Operation ID ---

type UUIDOperationIDGenerator struct{}

func (g *UUIDOperationIDGenerator) New() entity.OperationID {
	return entity.OperationID(uuid.NewString())
}

// --- Player ID ---

type UUIDPlayerIDGenerator struct{}

func (g *UUIDPlayerIDGenerator) New() entity.PlayerID {
	return entity.PlayerID(uuid.NewString())
}
