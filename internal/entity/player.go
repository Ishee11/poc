package entity

import "github.com/ishee11/poc/internal/entity/valueobject"

type SessionPlayer struct {
	playerID string

	totalMoneySpent     valueobject.Money
	totalMoneyCashedOut valueobject.Money

	totalChipsCashedOut int64
	totalMoneyCashedOut int64
}

func NewSessionPlayer(playerID string) *SessionPlayer {
	if playerID == "" {
		panic("empty player id")
	}
	return &SessionPlayer{playerID: playerID}
}

func (p *SessionPlayer) CurrentChips() int64 {
	return p.totalChipsBought - p.totalChipsCashedOut
}

func (p *SessionPlayer) ApplyBuyIn(chips int64, money valueobject.Money) error {
	if chips <= 0 {
		return ErrInvalidChips
	}

	p.totalChipsBought += chips
	p.totalMoneySpent = p.totalMoneySpent.Add(money)

	return nil
}

func (p *SessionPlayer) ApplyCashOut(chips int64, money valueobject.Money) error {
	if chips <= 0 {
		return ErrInvalidChips
	}

	if chips > p.CurrentChips() {
		return ErrNotEnoughChips
	}

	p.totalChipsCashedOut += chips
	p.totalMoneyCashedOut = p.totalMoneyCashedOut.Add(money)

	return nil
}
