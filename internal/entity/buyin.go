package entity

import (
	"fmt"

	"github.com/ishee11/poc/internal/entity/valueobject"
)

type BuyIn struct {
	id       string
	playerID string
	chips    int64
	money    int64
}

func NewBuyIn(id, playerID string, chips int64, rate valueobject.ChipRate) (BuyIn, error) {
	if id == "" {
		return BuyIn{}, fmt.Errorf("empty id")
	}
	if playerID == "" {
		return BuyIn{}, fmt.Errorf("empty player id")
	}
	if chips <= 0 {
		return BuyIn{}, ErrInvalidChips
	}

	return BuyIn{
		id:       id,
		playerID: playerID,
		chips:    chips,
		money:    rate.ToMoney(chips),
	}, nil
}
