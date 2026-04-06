package entity

import "github.com/ishee11/poc/internal/entity/valueobject"

type SessionPlayer struct {
	playerID string

	totalChipsBought    int64
	totalChipsCashedOut int64

	totalMoneySpent     valueobject.Money
	totalMoneyCashedOut valueobject.Money
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

func (p *SessionPlayer) PlayerID() string                       { return p.playerID }
func (p *SessionPlayer) TotalChipsBought() int64                { return p.totalChipsBought }
func (p *SessionPlayer) TotalChipsCashedOut() int64             { return p.totalChipsCashedOut }
func (p *SessionPlayer) TotalMoneySpent() valueobject.Money     { return p.totalMoneySpent }
func (p *SessionPlayer) TotalMoneyCashedOut() valueobject.Money { return p.totalMoneyCashedOut }

func (p *SessionPlayer) copy() *SessionPlayer {
	return &SessionPlayer{
		playerID:            p.playerID,
		totalChipsBought:    p.totalChipsBought,
		totalChipsCashedOut: p.totalChipsCashedOut,
		totalMoneySpent:     p.totalMoneySpent,
		totalMoneyCashedOut: p.totalMoneyCashedOut,
	}
}
