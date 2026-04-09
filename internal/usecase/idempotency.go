package usecase

import "github.com/ishee11/poc/internal/entity"

func Idempotent(
	tx Tx,
	repo OperationRepository,
	requestID string,
	fn func() error,
) error {
	if requestID == "" {
		return entity.ErrInvalidRequestID
	}

	existing, err := repo.GetByRequestID(tx, requestID)
	if err != nil {
		return err
	}

	if existing != nil {
		return nil
	}

	return fn()
}
