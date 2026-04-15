package usecase

import (
	"errors"

	"github.com/ishee11/poc/internal/entity"
)

type IdempotencyRepository interface {
	Save(tx Tx, requestID string) error
}

func Idempotent(
	tx Tx,
	repo IdempotencyRepository,
	requestID string,
	fn func() error,
) error {
	if requestID == "" {
		return entity.ErrInvalidRequestID
	}

	// 1. пробуем записать request_id
	if err := repo.Save(tx, requestID); err != nil {
		if errors.Is(err, entity.ErrDuplicateRequest) {
			return nil
		}
		return err
	}

	// 2. выполняем бизнес-логику
	return fn()
}
