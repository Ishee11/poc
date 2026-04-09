package usecase

import (
	"errors"

	"github.com/ishee11/poc/internal/entity"
)

func Idempotent(
	tx Tx,
	repo OperationRepository,
	requestID string,
	fn func() error,
) error {
	if requestID == "" {
		return entity.ErrInvalidRequestID
	}

	err := fn()
	if err != nil {
		if errors.Is(err, entity.ErrDuplicateRequest) {
			return nil
		}
		return err
	}

	return nil
}
