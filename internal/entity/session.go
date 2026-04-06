package entity

import (
	"fmt"
)

const (
	StatusCreated  Status = "created"
	StatusActive   Status = "active"
	StatusFinished Status = "finished"
)

type Status string

type Session struct {
	id     string
	rate   ChipRate
	status Status

	players  map[string]*SessionPlayer
	buyIns   []BuyIn
	cashOuts []CashOut

	operationIDs map[string]struct{}
}

func NewSession(id string, rate int64) *Session {
	if id == "" {
		panic("empty session id")
	}

	return &Session{
		id:     id,
		rate:   NewChipRate(rate),
		status: StatusCreated,

		players:      make(map[string]*SessionPlayer),
		buyIns:       make([]BuyIn, 0),
		cashOuts:     make([]CashOut, 0),
		operationIDs: make(map[string]struct{}),
	}
}

func (p *SessionPlayer) CurrentChips() int64 {
	return p.totalChipsBought - p.totalChipsCashedOut
}

func (p *SessionPlayer) ApplyBuyIn(chips, money int64) {
	p.totalChipsBought += chips
	p.totalMoneySpent += money
}

func (p *SessionPlayer) ApplyCashOut(chips, money int64) {
	p.totalChipsCashedOut += chips
	p.totalMoneyCashedOut += money
}

func (s *Session) BuyChips(opID, playerID string, chips int64) error {
	if s.status != StatusActive {
		return ErrSessionNotActive
	}

	// 👇 идемпотентность O(1)
	if _, exists := s.operationIDs[opID]; exists {
		return nil
	}

	p, exists := s.players[playerID]
	if !exists {
		p = &SessionPlayer{playerID: playerID}
		s.players[playerID] = p
	}

	buyIn, err := NewBuyIn(opID, playerID, chips, s.rate)
	if err != nil {
		return err
	}

	p.ApplyBuyIn(buyIn.chips, buyIn.money)

	s.buyIns = append(s.buyIns, buyIn)
	s.operationIDs[opID] = struct{}{}

	return nil
}

func (s *Session) CashOut(opID, playerID string, chips int64) error {
	if s.status != StatusActive {
		return ErrSessionNotActive
	}

	// идемпотентность
	if _, exists := s.operationIDs[opID]; exists {
		return nil
	}

	p, ok := s.players[playerID]
	if !ok {
		return ErrPlayerNotFound
	}

	current := p.CurrentChips()

	if chips > current {
		return fmt.Errorf("%w: have %d, want %d", ErrNotEnoughChips, current, chips)
	}

	cashOut, err := NewCashOut(opID, playerID, chips, s.rate)
	if err != nil {
		return err
	}

	p.ApplyCashOut(cashOut.chips, cashOut.money)
	s.cashOuts = append(s.cashOuts, cashOut)

	s.operationIDs[opID] = struct{}{}

	return nil
}

func (s *Session) Start() error {
	if s.status != StatusCreated {
		return ErrSessionNotCreated
	}

	s.status = StatusActive
	return nil
}

func (s *Session) Finish() error {
	if s.status != StatusActive {
		return ErrSessionNotActive
	}

	s.status = StatusFinished
	return nil
}

func (s *Session) Result(playerID string) (int64, error) {
	p, ok := s.players[playerID]
	if !ok {
		return 0, ErrPlayerNotFound
	}

	if p.CurrentChips() > 0 {
		return 0, ErrPlayerStillInGame
	}

	return p.totalMoneyCashedOut - p.totalMoneySpent, nil
}
