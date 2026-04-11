package usecase

type IdempotencyRepository interface {
	Save(tx Tx, requestID string) error
}
