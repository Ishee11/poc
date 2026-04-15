package entity

type PlayerState struct {
	PlayerID PlayerID
	Balance  int64
	InGame   bool
}

func NewPlayerState(
	playerID PlayerID,
	buyIn int64,
	cashOut int64,
	lastOp OperationType,
	found bool,
) PlayerState {
	return PlayerState{
		PlayerID: playerID,
		Balance:  buyIn - cashOut,
		InGame:   found && lastOp == OperationBuyIn,
	}
}

func (p PlayerState) ValidateInGame() error {
	if !p.InGame {
		return ErrPlayerNotInGame
	}
	return nil
}

func (p PlayerState) ValidateCashOut(chips int64) error {
	if err := p.ValidateInGame(); err != nil {
		return err
	}

	if chips <= 0 {
		return ErrInvalidChips
	}

	if chips > p.Balance {
		return ErrInvalidCashOut
	}

	return nil
}
