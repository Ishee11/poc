package postgres

import (
	"context"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
)

type IdempotencyRepository struct{}

func NewIdempotencyRepository() *IdempotencyRepository {
	return &IdempotencyRepository{}
}

func (r *IdempotencyRepository) Save(tx usecase.Tx, requestID string) error {

	ctx := context.Background()

	cmdTag, err := tx.Exec(ctx, `
		INSERT INTO idempotency_keys (request_id)
		VALUES ($1)
		ON CONFLICT (request_id) DO NOTHING
	`, requestID)

	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return entity.ErrDuplicateRequest
	}

	return nil
}
