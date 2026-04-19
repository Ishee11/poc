package postgres

import (
	"context"
	"time"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StatsRepository struct {
	db *pgxpool.Pool
}

func NewStatsRepository(db *pgxpool.Pool) *StatsRepository {
	return &StatsRepository{
		db: db,
	}
}

func (r *StatsRepository) ListSessions(
	tx usecase.Tx,
	filter usecase.SessionStatsFilter,
) ([]usecase.SessionStat, error) {

	ctx := context.Background()
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}

	rows, err := tx.Query(ctx, `
		WITH effective_operations AS (
			SELECT o.id, o.session_id, o.player_id, o.type, o.chips
			FROM operations o
			LEFT JOIN operations rev
				ON rev.reference_id = o.id
				AND rev.type = 'reversal'
			WHERE o.type <> 'reversal'
			  AND rev.id IS NULL
		)
		SELECT
			s.id,
			s.status,
			s.chip_rate,
			s.big_blind,
			s.created_at,
			s.finished_at,
			COALESCE(SUM(CASE WHEN eo.type = 'buy_in' THEN eo.chips ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN eo.type = 'cash_out' THEN eo.chips ELSE 0 END), 0),
			COUNT(DISTINCT eo.player_id)
		FROM sessions s
		LEFT JOIN effective_operations eo ON eo.session_id = s.id
		WHERE ($1::timestamp IS NULL OR s.created_at >= $1::timestamp)
		  AND ($2::timestamp IS NULL OR s.created_at < $2::timestamp)
		GROUP BY s.id, s.status, s.chip_rate, s.big_blind, s.created_at, s.finished_at
		ORDER BY s.created_at DESC
		LIMIT $3
	`, boundTime(filter.From), boundTime(filter.To), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]usecase.SessionStat, 0)
	for rows.Next() {
		var createdAt time.Time
		var finishedAt *time.Time
		var session usecase.SessionStat
		if err := rows.Scan(
			&session.SessionID,
			&session.Status,
			&session.ChipRate,
			&session.BigBlind,
			&createdAt,
			&finishedAt,
			&session.TotalBuyIn,
			&session.TotalCashOut,
			&session.PlayerCount,
		); err != nil {
			return nil, err
		}
		session.CreatedAt = createdAt.Format(time.RFC3339)
		if finishedAt != nil {
			formatted := finishedAt.Format(time.RFC3339)
			session.FinishedAt = &formatted
		}
		result = append(result, session)
	}

	return result, rows.Err()
}

func (r *StatsRepository) ListPlayers(
	tx usecase.Tx,
	filter usecase.PlayerStatsFilter,
) ([]usecase.PlayerStat, error) {

	ctx := context.Background()
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}

	rows, err := tx.Query(ctx, `
		WITH effective_operations AS (
			SELECT o.id, o.session_id, o.player_id, o.type, o.chips, o.created_at
			FROM operations o
			LEFT JOIN operations rev
				ON rev.reference_id = o.id
				AND rev.type = 'reversal'
			WHERE o.type <> 'reversal'
			  AND rev.id IS NULL
		)
		SELECT
		    eo.player_id,
		    COALESCE(p.name, eo.player_id),
		    COUNT(DISTINCT eo.session_id),
		    COALESCE(SUM(CASE WHEN eo.type = 'buy_in' THEN eo.chips ELSE 0 END), 0),
		    COALESCE(SUM(CASE WHEN eo.type = 'cash_out' THEN eo.chips ELSE 0 END), 0),
		    COALESCE(SUM(CASE WHEN eo.type = 'cash_out' THEN eo.chips / s.chip_rate ELSE 0 END), 0)
		        - COALESCE(SUM(CASE WHEN eo.type = 'buy_in' THEN eo.chips / s.chip_rate ELSE 0 END), 0),
		    MAX(eo.created_at)
		FROM effective_operations eo
		JOIN sessions s ON s.id = eo.session_id
		LEFT JOIN players p ON p.id = eo.player_id
		WHERE ($1::timestamp IS NULL OR eo.created_at >= $1::timestamp)
		  AND ($2::timestamp IS NULL OR eo.created_at < $2::timestamp)
		GROUP BY eo.player_id, p.name
		ORDER BY MAX(eo.created_at) DESC, eo.player_id ASC
		LIMIT $3
`, boundTime(filter.From), boundTime(filter.To), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]usecase.PlayerStat, 0)
	for rows.Next() {
		var stat usecase.PlayerStat
		var lastActivity *time.Time
		if err := rows.Scan(
			&stat.PlayerID,
			&stat.PlayerName,
			&stat.SessionsCount,
			&stat.TotalBuyIn,
			&stat.TotalCashOut,
			&stat.ProfitMoney,
			&lastActivity,
		); err != nil {
			return nil, err
		}
		stat.ProfitChips = stat.TotalCashOut - stat.TotalBuyIn
		if lastActivity != nil {
			formatted := lastActivity.Format(time.RFC3339)
			stat.LastActivityAt = &formatted
		}
		result = append(result, stat)
	}

	return result, rows.Err()
}

func (r *StatsRepository) GetPlayerOverall(
	tx usecase.Tx,
	playerID entity.PlayerID,
	filter usecase.PlayerStatsFilter,
) (*usecase.PlayerOverallStat, error) {

	ctx := context.Background()
	row := tx.QueryRow(ctx, `
		WITH effective_operations AS (
			SELECT o.id, o.session_id, o.player_id, o.type, o.chips, o.created_at
			FROM operations o
			LEFT JOIN operations rev
				ON rev.reference_id = o.id
				AND rev.type = 'reversal'
			WHERE o.type <> 'reversal'
			  AND rev.id IS NULL
		)
		SELECT
			COALESCE(MAX(p.name), $1),
			COUNT(DISTINCT eo.session_id),
			COALESCE(SUM(CASE WHEN eo.type = 'buy_in' THEN eo.chips ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN eo.type = 'cash_out' THEN eo.chips ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN eo.type = 'buy_in' THEN eo.chips / s.chip_rate ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN eo.type = 'cash_out' THEN eo.chips / s.chip_rate ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN eo.type = 'cash_out' THEN eo.chips / s.chip_rate ELSE 0 END), 0)
				- COALESCE(SUM(CASE WHEN eo.type = 'buy_in' THEN eo.chips / s.chip_rate ELSE 0 END), 0),
			MAX(eo.created_at)
		FROM effective_operations eo
		JOIN sessions s ON s.id = eo.session_id
		LEFT JOIN players p ON p.id = eo.player_id
		WHERE eo.player_id = $1
		  AND ($2::timestamp IS NULL OR eo.created_at >= $2::timestamp)
		  AND ($3::timestamp IS NULL OR eo.created_at < $3::timestamp)
`, playerID, boundTime(filter.From), boundTime(filter.To))

	var stat usecase.PlayerOverallStat
	var lastActivity *time.Time
	stat.PlayerID = playerID

	if err := row.Scan(
		&stat.PlayerName,
		&stat.SessionsCount,
		&stat.TotalBuyIn,
		&stat.TotalCashOut,
		&stat.TotalBuyInMoney,
		&stat.TotalCashOutMoney,
		&stat.ProfitMoney,
		&lastActivity,
	); err != nil {
		return nil, err
	}

	stat.ProfitChips = stat.TotalCashOut - stat.TotalBuyIn
	if stat.SessionsCount > 0 {
		stat.AvgProfitPerSession = float64(stat.ProfitMoney) / float64(stat.SessionsCount)
		stat.AvgBuyInPerSession = float64(stat.TotalBuyIn) / float64(stat.SessionsCount)
	}
	if stat.TotalBuyInMoney > 0 {
		stat.ROIPercent = float64(stat.ProfitMoney) / float64(stat.TotalBuyInMoney) * 100
	}
	if lastActivity != nil {
		formatted := lastActivity.Format(time.RFC3339)
		stat.LastActivityAt = &formatted
	}

	return &stat, nil
}

func (r *StatsRepository) ListPlayerSessions(
	tx usecase.Tx,
	playerID entity.PlayerID,
	filter usecase.PlayerStatsFilter,
) ([]usecase.PlayerSessionStat, error) {

	ctx := context.Background()
	rows, err := tx.Query(ctx, `
		WITH effective_operations AS (
			SELECT o.id, o.session_id, o.player_id, o.type, o.chips, o.created_at
			FROM operations o
			LEFT JOIN operations rev
				ON rev.reference_id = o.id
				AND rev.type = 'reversal'
			WHERE o.type <> 'reversal'
			  AND rev.id IS NULL
		)
		SELECT
			s.id,
			s.status,
			s.chip_rate,
			s.big_blind,
			s.created_at,
			s.finished_at,
			COALESCE(SUM(CASE WHEN eo.type = 'buy_in' THEN eo.chips ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN eo.type = 'cash_out' THEN eo.chips ELSE 0 END), 0),
			MAX(eo.created_at)
		FROM sessions s
		JOIN effective_operations eo ON eo.session_id = s.id
		WHERE eo.player_id = $1
		  AND ($2::timestamp IS NULL OR eo.created_at >= $2::timestamp)
		  AND ($3::timestamp IS NULL OR eo.created_at < $3::timestamp)
		GROUP BY s.id, s.status, s.chip_rate, s.big_blind, s.created_at, s.finished_at
		ORDER BY MAX(eo.created_at) DESC, s.created_at DESC
		LIMIT $4
		`, playerID, boundTime(filter.From), boundTime(filter.To), filterLimit(filter.Limit, 100))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]usecase.PlayerSessionStat, 0)
	for rows.Next() {
		var stat usecase.PlayerSessionStat
		var sessionCreatedAt time.Time
		var sessionFinishedAt *time.Time
		var lastActivity *time.Time
		if err := rows.Scan(
			&stat.SessionID,
			&stat.Status,
			&stat.ChipRate,
			&stat.BigBlind,
			&sessionCreatedAt,
			&sessionFinishedAt,
			&stat.BuyInChips,
			&stat.CashOutChips,
			&lastActivity,
		); err != nil {
			return nil, err
		}

		stat.SessionCreatedAt = sessionCreatedAt.Format(time.RFC3339)
		if sessionFinishedAt != nil {
			formatted := sessionFinishedAt.Format(time.RFC3339)
			stat.SessionFinishedAt = &formatted
		}
		stat.ProfitChips = stat.CashOutChips - stat.BuyInChips
		if stat.ChipRate > 0 {
			stat.ProfitMoney = stat.ProfitChips / stat.ChipRate
		}
		if lastActivity != nil {
			formatted := lastActivity.Format(time.RFC3339)
			stat.LastActivityAt = &formatted
		}
		result = append(result, stat)
	}

	return result, rows.Err()
}

func boundTime(bound *usecase.DateTimeRangeBound) *time.Time {
	if bound == nil {
		return nil
	}
	t, err := time.Parse(time.RFC3339, bound.Value)
	if err != nil {
		return nil
	}
	return &t
}

func filterLimit(limit int, fallback int) int {
	if limit > 0 {
		return limit
	}
	return fallback
}
