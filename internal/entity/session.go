package entity

import (
	"time"

	"github.com/ishee11/poc/internal/entity/valueobject"
)

type Status string
type SessionID string

const (
	StatusActive   Status = "active"
	StatusFinished Status = "finished"
)

type Session struct {
	id         SessionID
	chipRate   valueobject.ChipRate
	bigBlind   int64
	status     Status
	createdAt  time.Time
	finishedAt *time.Time

	// cached aggregates, derived from operations (source of truth)
	totalBuyInCache   int64
	totalCashOutCache int64
}

func NewSession(id SessionID, chipRate valueobject.ChipRate, bigBlind int64, createdAt time.Time) *Session {
	return &Session{
		id:                id,
		chipRate:          chipRate,
		bigBlind:          bigBlind,
		status:            StatusActive,
		createdAt:         createdAt,
		totalBuyInCache:   0,
		totalCashOutCache: 0,
	}
}

func RestoreSession(
	id SessionID,
	chipRate valueobject.ChipRate,
	bigBlind int64,
	status Status,
	createdAt time.Time,
	finishedAt *time.Time,
	totalBuyIn int64,
	totalCashOut int64,
) *Session {
	return &Session{
		id:                id,
		chipRate:          chipRate,
		bigBlind:          bigBlind,
		status:            status,
		createdAt:         createdAt,
		finishedAt:        finishedAt,
		totalBuyInCache:   totalBuyIn,
		totalCashOutCache: totalCashOut,
	}
}

func (s *Session) ID() SessionID                  { return s.id }
func (s *Session) ChipRate() valueobject.ChipRate { return s.chipRate }
func (s *Session) BigBlind() int64                { return s.bigBlind }
func (s *Session) Status() Status                 { return s.status }
func (s *Session) TotalBuyIn() int64              { return s.totalBuyInCache }
func (s *Session) TotalCashOut() int64            { return s.totalCashOutCache }
func (s *Session) TotalChips() int64              { return s.totalBuyInCache - s.totalCashOutCache }

func (s *Session) BuyIn(chips int64) error {
	if s.status != StatusActive {
		return ErrSessionNotActive
	}

	if chips <= 0 {
		return ErrInvalidChips
	}

	s.totalBuyInCache += chips

	return nil
}

func (s *Session) CashOut(chips int64) error {
	if s.status != StatusActive {
		return ErrSessionNotActive
	}

	if chips <= 0 {
		return ErrInvalidChips
	}

	s.totalCashOutCache += chips

	return nil
}

func (s *Session) Finish(finishedAt time.Time) error {
	if s.status != StatusActive {
		return ErrSessionNotActive
	}

	s.status = StatusFinished
	s.finishedAt = &finishedAt

	return nil
}

func (s *Session) CreatedAt() time.Time {
	return s.createdAt
}

func (s *Session) FinishedAt() *time.Time {
	return s.finishedAt
}
