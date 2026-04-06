package entity

import "github.com/ishee11/poc/internal/entity/valueobject"

type Status string
type SessionID string

const (
	StatusActive   Status = "active"
	StatusFinished Status = "finished"
)

type Session struct {
	id           SessionID
	chipRate     valueobject.ChipRate
	status       Status
	totalBuyIn   int64
	totalCashOut int64
}

func NewSession(id SessionID, chipRate valueobject.ChipRate) *Session {
	return &Session{
		id:           id,
		chipRate:     chipRate,
		status:       StatusActive,
		totalBuyIn:   0,
		totalCashOut: 0,
	}
}

func (s *Session) ID() SessionID                  { return s.id }
func (s *Session) ChipRate() valueobject.ChipRate { return s.chipRate }
func (s *Session) Status() Status                 { return s.status }
func (s *Session) TotalBuyIn() int64              { return s.totalBuyIn }
func (s *Session) TotalCashOut() int64            { return s.totalCashOut }
func (s *Session) TotalChips() int64              { return s.totalBuyIn - s.totalCashOut }

func (s *Session) BuyIn(chips int64) error {
	if s.status != StatusActive {
		return ErrSessionNotActive
	}

	if chips <= 0 {
		return ErrInvalidChips
	}

	s.totalBuyIn += chips

	return nil
}

func (s *Session) CashOut(chips int64) error {
	if s.status != StatusActive {
		return ErrSessionNotActive
	}

	if chips <= 0 {
		return ErrInvalidChips
	}

	if chips > s.TotalChips() {
		return ErrInsufficientTableChips
	}

	s.totalCashOut += chips

	return nil
}

func (s *Session) Finish() error {
	if s.status != StatusActive {
		return ErrSessionNotActive
	}

	if s.TotalChips() != 0 {
		return ErrTableNotSettled
	}

	s.status = StatusFinished

	return nil
}
