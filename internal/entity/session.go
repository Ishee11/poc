package entity

import (
	"github.com/ishee11/poc/internal/entity/valueobject"
)

const (
	StatusCreated  Status = "created"
	StatusActive   Status = "active"
	StatusFinished Status = "finished"
)

type Status string

type Session struct {
	id     string
	rate   valueobject.ChipRate
	status Status

	players map[string]*SessionPlayer

	operationIDs map[string]struct{}
}

func NewSession(id string, rate int64) *Session {
	if id == "" {
		panic("empty session id")
	}

	return &Session{
		id:     id,
		rate:   valueobject.NewChipRate(rate),
		status: StatusCreated,

		players:      make(map[string]*SessionPlayer),
		operationIDs: make(map[string]struct{}),
	}
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
		p = NewSessionPlayer(playerID)
		s.players[playerID] = p
	}

	buyIn, err := NewBuyIn(opID, playerID, chips, s.rate)
	if err != nil {
		return err
	}

	p.ApplyBuyIn(buyIn.chips, buyIn.money)

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

	cashOut, err := NewCashOut(opID, playerID, chips, s.rate)
	if err != nil {
		return err
	}

	if err := p.ApplyCashOut(cashOut.chips, cashOut.money); err != nil {
		return err
	}

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

	for _, p := range s.players {
		if p.CurrentChips() != 0 {
			return ErrPlayersStillInGame
		}
	}

	if s.totalBought() != s.totalCashedOut() {
		return ErrUnbalancedSession
	}

	s.status = StatusFinished
	return nil
}

func (s *Session) Result(playerID string) (int64, error) {
	if s.status != StatusFinished {
		return 0, ErrSessionNotFinished
	}

	p, ok := s.players[playerID]
	if !ok {
		return 0, ErrPlayerNotFound
	}

	return p.totalMoneyCashedOut - p.totalMoneySpent, nil
}

func (s *Session) totalBought() int64 {
	var total int64
	for _, p := range s.players {
		total += p.totalMoneySpent
	}
	return total
}

func (s *Session) totalCashedOut() int64 {
	var total int64
	for _, p := range s.players {
		total += p.totalMoneyCashedOut
	}
	return total
}
