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

type PlayerReader interface {
	ListBySession(tx Tx, sessionID entity.SessionID) ([]PlayerDTO, error)
}

type PlayerRepository interface {
	Create(tx Tx, player *entity.Player) error
	Exists(tx Tx, id entity.PlayerID) (bool, error)
	GetByID(tx Tx, id entity.PlayerID) (*entity.Player, error)
	List(tx Tx, limit int, offset int) ([]PlayerDTO, error)
}

type PlayerIDGenerator interface {
	New() entity.PlayerID
}

type OperationIDGenerator interface {
	New() entity.OperationID
}

type SessionIDGenerator interface {
	New() entity.SessionID
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

type PlayerSessionStat struct {
	SessionID        entity.SessionID `json:"session_id"`
	Status           entity.Status    `json:"status"`
	ChipRate         int64            `json:"chip_rate"`
	SessionCreatedAt string           `json:"session_created_at"`
	LastActivityAt   *string          `json:"last_activity_at"`
	BuyInChips       int64            `json:"buy_in_chips"`
	CashOutChips     int64            `json:"cash_out_chips"`
	ProfitChips      int64            `json:"profit_chips"`
	ProfitMoney      int64            `json:"profit_money"`
}

type StatsRepository interface {
	ListSessions(tx Tx, filter SessionStatsFilter) ([]SessionStat, error)
	ListPlayers(tx Tx, filter PlayerStatsFilter) ([]PlayerStat, error)
	GetPlayerOverall(tx Tx, playerID entity.PlayerID, filter PlayerStatsFilter) (*PlayerOverallStat, error)
	ListPlayerSessions(tx Tx, playerID entity.PlayerID, filter PlayerStatsFilter) ([]PlayerSessionStat, error)
}

type DebugAdminRepository interface {
	DeletePlayer(tx Tx, playerID entity.PlayerID) error
	DeleteSession(tx Tx, sessionID entity.SessionID) error
	DeleteSessionFinish(tx Tx, sessionID entity.SessionID) error
}
