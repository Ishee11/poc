package valueobject

import "fmt"

type Money struct {
	amount int64
}

func NewMoney(amount int64) (Money, error) {
	if amount < 0 {
		return Money{}, fmt.Errorf("money cannot be negative")
	}
	return Money{amount: amount}, nil
}

func (m Money) Amount() int64 {
	return m.amount
}

func (m Money) Add(other Money) Money {
	return Money{amount: m.amount + other.amount}
}

func (m Money) Sub(other Money) (Money, error) {
	if m.amount < other.amount {
		return Money{}, fmt.Errorf("not enough money")
	}
	return Money{amount: m.amount - other.amount}, nil
}

func (m Money) Equal(other Money) bool {
	return m.amount == other.amount
}
