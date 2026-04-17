package entity

type PlayerState struct {
	PlayerID PlayerID
	InGame   bool
}

func NewPlayerState(
	playerID PlayerID,
	lastOp OperationType,
	found bool,
) PlayerState {
	return PlayerState{
		PlayerID: playerID,
		InGame:   found && lastOp == OperationBuyIn,
	}
}

func (p PlayerState) ValidateInGame() error {
	if !p.InGame {
		return ErrPlayerNotInGame
	}
	return nil
}
