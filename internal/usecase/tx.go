package usecase

type TxManager interface {
	RunInTx(fn func(tx Tx) error) error
}
type Tx interface {
}
