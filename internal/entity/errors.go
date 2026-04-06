package entity

import "errors"

var (
	ErrSessionNotActive  = errors.New("session not active")
	ErrSessionNotCreated = errors.New("session not created")
	ErrSessionFinished   = errors.New("session finished")

	ErrInvalidChips      = errors.New("invalid chips")
	ErrPlayerNotFound    = errors.New("player not found")
	ErrPlayerStillInGame = errors.New("player still in game")
	ErrNotEnoughChips    = errors.New("not enough chips")
)
