package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ishee11/poc/internal/app"
	postgres "github.com/ishee11/poc/internal/infra/postgres"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("PG_URL")
	}
	if dsn == "" {
		t.Skip("DATABASE_URL or PG_URL is not set")
	}
	ensureSafeTestDSN(t, dsn)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("postgres is not available: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("postgres is not available: %v", err)
	}
	if err := postgres.RunMigrations(ctx, pool, postgres.MigrationsFS); err != nil {
		pool.Close()
		t.Fatalf("run migrations: %v", err)
	}

	t.Cleanup(pool.Close)
	return pool
}

func ensureSafeTestDSN(t *testing.T, dsn string) {
	t.Helper()
	if os.Getenv("ALLOW_DESTRUCTIVE_INTEGRATION_TESTS") == "true" {
		return
	}
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse database dsn: %v", err)
	}
	switch parsed.Hostname() {
	case "127.0.0.1", "localhost", "::1":
		return
	default:
		t.Skipf("refusing to run destructive integration tests against non-local database host %q", parsed.Hostname())
	}
}

func cleanDB(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		TRUNCATE TABLE idempotency_keys, operations, sessions, players
		RESTART IDENTITY CASCADE
	`)
	if err != nil {
		t.Fatalf("clean database: %v", err)
	}
}

func requestJSON(t *testing.T, handler http.Handler, method string, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var payload bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&payload).Encode(body); err != nil {
			t.Fatal(err)
		}
	}

	req := httptest.NewRequest(method, path, &payload)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), target); err != nil {
		t.Fatalf("decode json body %q: %v", rec.Body.String(), err)
	}
}

func TestAPIIntegration_SessionLifecycle(t *testing.T) {
	pool := testPool(t)
	cleanDB(t, pool)

	handler := app.NewContainer(&app.DB{Pool: pool}).Router

	createPlayer := requestJSON(t, handler, http.MethodPost, "/players", map[string]any{
		"request_id": "req-player-1",
		"name":       "Alice",
	})
	if createPlayer.Code != http.StatusOK {
		t.Fatalf("create player status=%d body=%s", createPlayer.Code, createPlayer.Body.String())
	}
	var playerResp struct {
		PlayerID string `json:"player_id"`
	}
	decodeJSON(t, createPlayer, &playerResp)
	if playerResp.PlayerID == "" {
		t.Fatal("player_id is empty")
	}

	startSession := requestJSON(t, handler, http.MethodPost, "/sessions/start", map[string]any{
		"chip_rate": 2,
		"big_blind": 2,
		"currency":  "RUB",
	})
	if startSession.Code != http.StatusOK {
		t.Fatalf("start session status=%d body=%s", startSession.Code, startSession.Body.String())
	}
	var sessionResp struct {
		SessionID string `json:"session_id"`
	}
	decodeJSON(t, startSession, &sessionResp)
	if sessionResp.SessionID == "" {
		t.Fatal("session_id is empty")
	}

	buyIn := requestJSON(t, handler, http.MethodPost, "/operations/buy-in", map[string]any{
		"request_id": "req-buy-in-1",
		"session_id": sessionResp.SessionID,
		"player_id":  playerResp.PlayerID,
		"chips":      100,
	})
	if buyIn.Code != http.StatusOK {
		t.Fatalf("buy in status=%d body=%s", buyIn.Code, buyIn.Body.String())
	}

	finishUnbalanced := requestJSON(t, handler, http.MethodPost, "/sessions/finish", map[string]any{
		"request_id": "req-finish-1",
		"session_id": sessionResp.SessionID,
	})
	if finishUnbalanced.Code != http.StatusConflict {
		t.Fatalf("unbalanced finish status=%d body=%s", finishUnbalanced.Code, finishUnbalanced.Body.String())
	}
	var finishErr struct {
		Error   string `json:"error"`
		Details struct {
			RemainingChips int64 `json:"remaining_chips"`
		} `json:"details"`
	}
	decodeJSON(t, finishUnbalanced, &finishErr)
	if finishErr.Error != "session_not_balanced" || finishErr.Details.RemainingChips != 100 {
		t.Fatalf("unexpected finish error: %+v", finishErr)
	}

	cashOut := requestJSON(t, handler, http.MethodPost, "/operations/cash-out", map[string]any{
		"request_id": "req-cash-out-1",
		"session_id": sessionResp.SessionID,
		"player_id":  playerResp.PlayerID,
		"chips":      100,
	})
	if cashOut.Code != http.StatusOK {
		t.Fatalf("cash out status=%d body=%s", cashOut.Code, cashOut.Body.String())
	}

	finish := requestJSON(t, handler, http.MethodPost, "/sessions/finish", map[string]any{
		"request_id": "req-finish-2",
		"session_id": sessionResp.SessionID,
	})
	if finish.Code != http.StatusOK {
		t.Fatalf("finish status=%d body=%s", finish.Code, finish.Body.String())
	}

	sessionPlayers := requestJSON(t, handler, http.MethodGet, "/sessions/players?session_id="+sessionResp.SessionID, nil)
	if sessionPlayers.Code != http.StatusOK {
		t.Fatalf("session players status=%d body=%s", sessionPlayers.Code, sessionPlayers.Body.String())
	}
	var players []struct {
		PlayerID    string `json:"player_id"`
		InGame      bool   `json:"in_game"`
		ProfitChips int64  `json:"profit_chips"`
		ProfitMoney int64  `json:"profit_money"`
	}
	decodeJSON(t, sessionPlayers, &players)
	if len(players) != 1 {
		t.Fatalf("expected one player, got %d", len(players))
	}
	if players[0].InGame || players[0].ProfitChips != 0 || players[0].ProfitMoney != 0 {
		t.Fatalf("unexpected player result: %+v", players[0])
	}

	playerStats := requestJSON(t, handler, http.MethodGet, "/stats/player?player_id="+playerResp.PlayerID, nil)
	if playerStats.Code != http.StatusOK {
		t.Fatalf("player stats status=%d body=%s", playerStats.Code, playerStats.Body.String())
	}
	var stats map[string]any
	decodeJSON(t, playerStats, &stats)
	if stats["player"] == nil || stats["sessions"] == nil {
		t.Fatalf("expected lower-case player/sessions JSON keys, got %v", stats)
	}
}

func TestAPIIntegration_ReverseOperation(t *testing.T) {
	pool := testPool(t)
	cleanDB(t, pool)

	handler := app.NewContainer(&app.DB{Pool: pool}).Router

	createPlayer := requestJSON(t, handler, http.MethodPost, "/players", map[string]any{
		"request_id": "req-player-1",
		"name":       "Alice",
	})
	var playerResp struct {
		PlayerID string `json:"player_id"`
	}
	decodeJSON(t, createPlayer, &playerResp)

	startSession := requestJSON(t, handler, http.MethodPost, "/sessions/start", map[string]any{
		"chip_rate": 2,
		"big_blind": 2,
		"currency":  "RUB",
	})
	var sessionResp struct {
		SessionID string `json:"session_id"`
	}
	decodeJSON(t, startSession, &sessionResp)

	buyIn := requestJSON(t, handler, http.MethodPost, "/operations/buy-in", map[string]any{
		"request_id": "req-buy-in-1",
		"session_id": sessionResp.SessionID,
		"player_id":  playerResp.PlayerID,
		"chips":      100,
	})
	if buyIn.Code != http.StatusOK {
		t.Fatalf("buy in status=%d body=%s", buyIn.Code, buyIn.Body.String())
	}

	opsRes := requestJSON(t, handler, http.MethodGet, "/sessions/operations?session_id="+sessionResp.SessionID, nil)
	if opsRes.Code != http.StatusOK {
		t.Fatalf("operations status=%d body=%s", opsRes.Code, opsRes.Body.String())
	}
	var ops []struct {
		ID string `json:"id"`
	}
	decodeJSON(t, opsRes, &ops)
	if len(ops) != 1 || ops[0].ID == "" {
		t.Fatalf("expected one operation with id, got %+v", ops)
	}

	reverse := requestJSON(t, handler, http.MethodPost, "/operations/reverse", map[string]any{
		"request_id":          "req-reverse-1",
		"target_operation_id": ops[0].ID,
	})
	if reverse.Code != http.StatusOK {
		t.Fatalf("reverse status=%d body=%s", reverse.Code, reverse.Body.String())
	}

	sessionRes := requestJSON(t, handler, http.MethodGet, "/sessions?session_id="+sessionResp.SessionID, nil)
	if sessionRes.Code != http.StatusOK {
		t.Fatalf("session status=%d body=%s", sessionRes.Code, sessionRes.Body.String())
	}
	var session struct {
		TotalChips int64 `json:"total_chips"`
	}
	decodeJSON(t, sessionRes, &session)
	if session.TotalChips != 0 {
		t.Fatalf("expected total chips 0 after reversal, got %d", session.TotalChips)
	}

	reverseAgain := requestJSON(t, handler, http.MethodPost, "/operations/reverse", map[string]any{
		"request_id":          "req-reverse-2",
		"target_operation_id": ops[0].ID,
	})
	if reverseAgain.Code != http.StatusConflict {
		t.Fatalf("second reverse status=%d body=%s", reverseAgain.Code, reverseAgain.Body.String())
	}
}
