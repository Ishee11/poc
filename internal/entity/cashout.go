package entity

import "fmt"

type CashOut struct {
	id       string
	playerID string
	chips    int64
	money    int64
}

func NewCashOut(id, playerID string, chips int64, rate ChipRate) (CashOut, error) {
	if id == "" {
		return CashOut{}, fmt.Errorf("empty id")
	}
	if playerID == "" {
		return CashOut{}, fmt.Errorf("empty player id")
	}
	if chips <= 0 {
		return CashOut{}, ErrInvalidChips
	}

	return CashOut{
		id:       id,
		playerID: playerID,
		chips:    chips,
		money:    rate.ToMoney(chips),
	}, nil
}
