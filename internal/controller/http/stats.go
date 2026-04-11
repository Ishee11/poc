package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/usecase"
)

func (h *Handler) GetStatsSessions(w http.ResponseWriter, r *http.Request) {
	from, to, err := parseDateRange(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	res, err := h.getStatsSessionsUC.Execute(usecase.GetStatsSessionsQuery{
		Limit: limit,
		From:  from,
		To:    to,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	json.NewEncoder(w).Encode(res)
}

func (h *Handler) GetStatsPlayers(w http.ResponseWriter, r *http.Request) {
	from, to, err := parseDateRange(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	res, err := h.getStatsPlayersUC.Execute(usecase.GetStatsPlayersQuery{
		Limit: limit,
		From:  from,
		To:    to,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	json.NewEncoder(w).Encode(res)
}

func (h *Handler) GetPlayerStats(w http.ResponseWriter, r *http.Request) {
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

	json.NewEncoder(w).Encode(res)
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
