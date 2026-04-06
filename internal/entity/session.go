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

func (s *Session) PlayerBuyIn(opID, playerID string, money valueobject.Money) error {
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

	chips, err := s.rate.ToChips(money)
	if err != nil {
		return err
	}

	if err := p.ApplyBuyIn(chips, money); err != nil {
		return err
	}

	s.operationIDs[opID] = struct{}{}

	return nil
}

func (s *Session) PlayerCashOut(opID, playerID string, chips int64) error {
	if s.status != StatusActive {
		return ErrSessionNotActive
	}

	if _, exists := s.operationIDs[opID]; exists {
		return nil
	}

	p, ok := s.players[playerID]
	if !ok {
		return ErrPlayerNotFound
	}

	money, err := s.rate.ChipsToMoney(chips)
	if err != nil {
		return err
	}

	if err := p.ApplyCashOut(chips, money); err != nil {
		return err
	}

	s.operationIDs[opID] = struct{}{}
	return nil
}

func (s *Session) ID() string {
	return s.id
}

func (s *Session) StartSession() error {
	if s.status != StatusCreated {
		return ErrSessionNotCreated
	}

	s.status = StatusActive
	return nil
}

func (s *Session) FinishSession() error {
	if s.status != StatusActive {
		return ErrSessionNotActive
	}

	for _, p := range s.players {
		if p.CurrentChips() != 0 {
			return ErrPlayersStillInGame
		}
	}

	if !s.totalBought().Equal(s.totalCashedOut()) {
		return ErrUnbalancedSession
	}

	s.status = StatusFinished
	return nil
}

func (s *Session) PlayerResult(playerID string) (valueobject.Money, error) {
	if s.status != StatusFinished {
		return valueobject.Money{}, ErrSessionNotFinished
	}

	p, ok := s.players[playerID]
	if !ok {
		return valueobject.Money{}, ErrPlayerNotFound
	}

	return p.totalMoneyCashedOut.Sub(p.totalMoneySpent)
}

func (s *Session) totalBought() valueobject.Money {
	total := valueobject.Money{}

	for _, p := range s.players {
		total = total.Add(p.totalMoneySpent)
	}

	return total
}

func (s *Session) totalCashedOut() valueobject.Money {
	total := valueobject.Money{}

	for _, p := range s.players {
		total = total.Add(p.totalMoneyCashedOut)
	}

	return total
}

func (s *Session) Rate() valueobject.ChipRate {
	return s.rate
}

func (s *Session) Status() Status {
	return s.status
}

func (s *Session) Players() map[string]*SessionPlayer {
	return s.players
}

func (s *Session) OperationIDs() map[string]struct{} {
	return s.operationIDs
}

func (s *Session) Copy() *Session {
	clone := &Session{
		id:     s.id,
		rate:   s.rate,
		status: s.status,

		players:      make(map[string]*SessionPlayer),
		operationIDs: make(map[string]struct{}),
	}

	for id, p := range s.players {
		clone.players[id] = p.copy()
	}

	for op := range s.operationIDs {
		clone.operationIDs[op] = struct{}{}
	}

	return clone
}
