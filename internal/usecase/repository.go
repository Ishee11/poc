package usecase

import "github.com/ishee11/poc/internal/entity"

type SessionAggregates struct {
	TotalBuyIn   int64
	TotalCashOut int64
}

func NewSessionAggregates(totalBuyIn, totalCashOut int64) SessionAggregates {
	return SessionAggregates{
		TotalBuyIn:   totalBuyIn,
		TotalCashOut: totalCashOut,
	}
}

type PlayerAggregates struct {
	BuyIn   int64
	CashOut int64
}

type PlayerDTO struct {
	ID   entity.PlayerID
	Name string
}

type PlayerRepository interface {
	GetOrCreate(
		tx Tx,
		sessionID entity.SessionID,
		name string,
	) (entity.PlayerID, error)
	ListBySession(
		tx Tx,
		sessionID entity.SessionID,
	) ([]PlayerDTO, error)
}

type OperationWriter interface {
	Save(tx Tx, op *entity.Operation) error
}

type OperationReader interface {
	GetByID(tx Tx, id entity.OperationID) (*entity.Operation, error)
	GetByRequestID(tx Tx, requestID string) (*entity.Operation, error)
}

type OperationAggregateReader interface {
	GetSessionAggregates(tx Tx, sessionID entity.SessionID) (SessionAggregates, error)
}

type OperationPlayerStateReader interface {
	GetLastOperationType(tx Tx, sessionID entity.SessionID, playerID entity.PlayerID) (entity.OperationType, bool, error)
}

type OperationReversalChecker interface {
	ExistsReversal(tx Tx, targetID entity.OperationID) (bool, error)
}

type SessionReader interface {
	FindByID(tx Tx, sessionID entity.SessionID) (*entity.Session, error)
}

type SessionLocker interface {
	FindByIDForUpdate(tx Tx, sessionID entity.SessionID) (*entity.Session, error)
}

type SessionWriter interface {
	Save(tx Tx, session *entity.Session) error
}

type ProjectionRepository interface {
	GetSessionAggregates(tx Tx, sessionID entity.SessionID) (SessionAggregates, error)

	GetPlayerAggregates(tx Tx, sessionID entity.SessionID) (map[entity.PlayerID]PlayerAggregates, error)

	GetLastOperationType(tx Tx, sessionID entity.SessionID, playerID entity.PlayerID) (entity.OperationType, bool, error)

	ListBySession(
		tx Tx,
		sessionID entity.SessionID,
		limit int,
		offset int,
	) ([]*entity.Operation, error)
}

type SessionStatsFilter struct {
	Limit int
	From  *DateTimeRangeBound
	To    *DateTimeRangeBound
}

type PlayerStatsFilter struct {
	Limit int
	From  *DateTimeRangeBound
	To    *DateTimeRangeBound
}

type DateTimeRangeBound struct {
	Value string
}

type SessionStat struct {
	SessionID    entity.SessionID
	Status       entity.Status
	ChipRate     int64
	CreatedAt    string
	TotalBuyIn   int64
	TotalCashOut int64
	PlayerCount  int64
}

type PlayerStat struct {
	PlayerID       entity.PlayerID
	SessionsCount  int64
	TotalBuyIn     int64
	TotalCashOut   int64
	ProfitChips    int64
	ProfitMoney    int64
	LastActivityAt *string
}

type PlayerSessionStat struct {
	SessionID        entity.SessionID
	Status           entity.Status
	ChipRate         int64
	SessionCreatedAt string
	LastActivityAt   *string
	BuyInChips       int64
	CashOutChips     int64
	ProfitChips      int64
	ProfitMoney      int64
}

type PlayerOverallStat struct {
	PlayerID       entity.PlayerID
	SessionsCount  int64
	TotalBuyIn     int64
	TotalCashOut   int64
	ProfitChips    int64
	ProfitMoney    int64
	LastActivityAt *string
}

type StatsRepository interface {
	ListSessions(tx Tx, filter SessionStatsFilter) ([]SessionStat, error)
	ListPlayers(tx Tx, filter PlayerStatsFilter) ([]PlayerStat, error)
	GetPlayerOverall(tx Tx, playerID entity.PlayerID, filter PlayerStatsFilter) (*PlayerOverallStat, error)
	ListPlayerSessions(tx Tx, playerID entity.PlayerID, filter PlayerStatsFilter) ([]PlayerSessionStat, error)
}
