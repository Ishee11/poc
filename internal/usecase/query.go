package usecase

import (
	"github.com/ishee11/poc/internal/entity"
)

type GetPlayerStatsQuery struct {
	PlayerID entity.PlayerID
	From     *DateTimeRangeBound
	To       *DateTimeRangeBound
}

type GetPlayersQuery struct {
	Limit  int
	Offset int
}

type GetSessionQuery struct {
	SessionID entity.SessionID
}

type GetSessionOperationsQuery struct {
	SessionID entity.SessionID

	Limit  int
	Offset int
}

type GetSessionPlayersQuery struct {
	SessionID entity.SessionID
}

type GetSessionResultsQuery struct {
	SessionID entity.SessionID
}

type GetStatsPlayersQuery struct {
	Limit int
	From  *DateTimeRangeBound
	To    *DateTimeRangeBound
}

type GetStatsSessionsQuery struct {
	Limit         int
	From          *DateTimeRangeBound
	To            *DateTimeRangeBound
	ViewerUserID  *entity.AuthUserID
	GuestPlayerID entity.PlayerID
}
