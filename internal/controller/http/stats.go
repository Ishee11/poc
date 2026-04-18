package http

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
)

// GetStatsSessions godoc
// @Summary Get sessions stats
// @Description Returns statistics for sessions (aggregated)
// @Tags stats
// @Accept json
// @Produce json
// @Param limit query int false "Limit (default 20)"
// @Param from query string false "From date (RFC3339 or YYYY-MM-DD)"
// @Param to query string false "To date (RFC3339 or YYYY-MM-DD)"
// @Success 200 {array} usecase.SessionStat
// @Failure 400 {object} ErrorResponse
// @Router /stats/sessions [get]
func (h *StatsHandler) GetStatsSessions(w http.ResponseWriter, r *http.Request) {
	from, to, err := parseDateRange(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		limit = 0
	}

	res, err := h.getStatsSessionsUC.Execute(usecase.GetStatsSessionsQuery{
		Limit: limit,
		From:  from,
		To:    to,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, res)
}

// GetStatsPlayers godoc
// @Summary Get players stats
// @Description Returns aggregated statistics for players
// @Tags stats
// @Accept json
// @Produce json
// @Param limit query int false "Limit (default 20)"
// @Param from query string false "From date (RFC3339 or YYYY-MM-DD)"
// @Param to query string false "To date (RFC3339 or YYYY-MM-DD)"
// @Success 200 {array} usecase.PlayerStat
// @Failure 400 {object} ErrorResponse
// @Router /stats/players [get]
func (h *StatsHandler) GetStatsPlayers(w http.ResponseWriter, r *http.Request) {
	from, to, err := parseDateRange(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		limit = 0
	}

	res, err := h.getStatsPlayersUC.Execute(usecase.GetStatsPlayersQuery{
		Limit: limit,
		From:  from,
		To:    to,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, res)
}

// GetPlayerStats godoc
// @Summary Get player stats
// @Description Returns overall statistics for a specific player
// @Tags stats
// @Accept json
// @Produce json
// @Param player_id query string true "Player ID"
// @Param from query string false "From date (RFC3339 or YYYY-MM-DD)"
// @Param to query string false "To date (RFC3339 or YYYY-MM-DD)"
// @Success 200 {object} usecase.PlayerOverallStat
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /stats/player [get]
func (h *PlayerHandler) GetPlayerStats(w http.ResponseWriter, r *http.Request) {
	from, to, err := parseDateRange(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	playerID := r.URL.Query().Get("player_id")
	if playerID == "" {
		http.Error(w, "player_id is required", http.StatusBadRequest)
		return
	}

	res, err := h.getPlayerStatsUC.Execute(usecase.GetPlayerStatsQuery{
		PlayerID: entity.PlayerID(playerID),
		From:     from,
		To:       to,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, res)
}

func parseDateRange(r *http.Request) (*usecase.DateTimeRangeBound, *usecase.DateTimeRangeBound, error) {
	from, err := parseDateBound(r.URL.Query().Get("from"), false)
	if err != nil {
		return nil, nil, err
	}

	to, err := parseDateBound(r.URL.Query().Get("to"), true)
	if err != nil {
		return nil, nil, err
	}

	return from, to, nil
}

func parseDateBound(raw string, endExclusive bool) (*usecase.DateTimeRangeBound, error) {
	if raw == "" {
		return nil, nil
	}

	if len(raw) == len("2006-01-02") {
		t, err := time.Parse("2006-01-02", raw)
		if err != nil {
			return nil, err
		}
		if endExclusive {
			t = t.Add(24 * time.Hour)
		}
		return &usecase.DateTimeRangeBound{Value: t.Format(time.RFC3339)}, nil
	}

	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, err
	}

	return &usecase.DateTimeRangeBound{Value: t.Format(time.RFC3339)}, nil
}
