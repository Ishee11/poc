package entity

type SessionPlayer struct {
	playerID string

	totalChipsBought int64
	totalMoneySpent  int64

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

func (p *SessionPlayer) ApplyBuyIn(chips, money int64) error {
	if chips <= 0 {
		return ErrInvalidChips
	}

	p.totalChipsBought += chips
	p.totalMoneySpent += money

	return nil
}

func (p *SessionPlayer) ApplyCashOut(chips, money int64) error {
	if chips <= 0 {
		return ErrInvalidChips
	}

	if chips > p.CurrentChips() {
		return ErrNotEnoughChips
	}

	p.totalChipsCashedOut += chips
	p.totalMoneyCashedOut += money

	return nil
}
