package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
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
		TRUNCATE TABLE idempotency_keys, operations, settlement_transfers, session_expense_payments, session_expense_participants, session_expenses, auth_sessions, login_attempts, user_players, users, sessions, players
		RESTART IDENTITY CASCADE
	`)
	if err != nil {
		t.Fatalf("clean database: %v", err)
	}
}

func ownershipTestHandler(pool *pgxpool.Pool) http.Handler {
	return app.NewContainer(&app.DB{Pool: pool}, &app.Config{
		Auth: app.AuthConfig{
			Enabled:        true,
			CookieName:     "sid",
			CookieSecure:   false,
			CookieSameSite: "Lax",
			SessionTTL:     12 * time.Hour,
			IdleTTL:        2 * time.Hour,
		},
	}).Router
}

func requestJSONWithCookie(t *testing.T, handler http.Handler, method string, path string, body any, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var payload bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&payload).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, &payload)
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestAPIIntegration_RegistrationEstablishesSingularOwnership(t *testing.T) {
	pool := testPool(t)
	cleanDB(t, pool)
	if _, err := pool.Exec(context.Background(), `INSERT INTO players (id, name) VALUES ('player-existing', 'Alice')`); err != nil {
		t.Fatalf("insert player: %v", err)
	}
	handler := ownershipTestHandler(pool)

	register := requestJSON(t, handler, http.MethodPost, "/auth/register", map[string]any{
		"email": "owner@example.com", "password": "long-password",
		"player": map[string]any{"mode": "existing", "player_id": "player-existing"},
	})
	if register.Code != http.StatusOK {
		t.Fatalf("register status=%d body=%s", register.Code, register.Body.String())
	}
	cookies := register.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("registration did not create an auth session cookie")
	}

	account := requestJSONWithCookie(t, handler, http.MethodGet, "/account", nil, cookies[0])
	if account.Code != http.StatusOK {
		t.Fatalf("account status=%d body=%s", account.Code, account.Body.String())
	}
	var body struct {
		Player *struct {
			ID string `json:"player_id"`
		} `json:"player"`
		OnboardingRequired bool  `json:"onboarding_required"`
		Players            []any `json:"players"`
	}
	decodeJSON(t, account, &body)
	if body.Player == nil || body.Player.ID != "player-existing" || body.OnboardingRequired || len(body.Players) != 1 {
		t.Fatalf("unexpected singular account ownership: %+v", body)
	}

	repeat := requestJSONWithCookie(t, handler, http.MethodPut, "/account/player", map[string]any{
		"mode": "new", "name": "Wrong second player",
	}, cookies[0])
	if repeat.Code != http.StatusConflict || !strings.Contains(repeat.Body.String(), "account_already_linked") {
		t.Fatalf("repeat claim status=%d body=%s", repeat.Code, repeat.Body.String())
	}
}

func TestAPIIntegration_RegistrationCreatesPlayerAndRollsBackInvalidSelection(t *testing.T) {
	pool := testPool(t)
	cleanDB(t, pool)
	handler := ownershipTestHandler(pool)

	invalid := requestJSON(t, handler, http.MethodPost, "/auth/register", map[string]any{
		"email": "invalid@example.com", "password": "long-password",
	})
	if invalid.Code != http.StatusBadRequest || !strings.Contains(invalid.Body.String(), "invalid_player_selection") {
		t.Fatalf("invalid registration status=%d body=%s", invalid.Code, invalid.Body.String())
	}
	var invalidUsers int
	if err := pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM users WHERE email = 'invalid@example.com'`).Scan(&invalidUsers); err != nil {
		t.Fatalf("count invalid users: %v", err)
	}
	if invalidUsers != 0 {
		t.Fatalf("invalid registration created %d users", invalidUsers)
	}

	created := requestJSON(t, handler, http.MethodPost, "/auth/register", map[string]any{
		"email": "new@example.com", "password": "long-password",
		"player": map[string]any{"mode": "new", "name": " New player "},
	})
	if created.Code != http.StatusOK {
		t.Fatalf("new-player registration status=%d body=%s", created.Code, created.Body.String())
	}
	var users, players, links int
	if err := pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM users WHERE email = 'new@example.com'`).Scan(&users); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM players WHERE name = 'New player'`).Scan(&players); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM user_players`).Scan(&links); err != nil {
		t.Fatal(err)
	}
	if users != 1 || players != 1 || links != 1 {
		t.Fatalf("registration was not atomic: users=%d players=%d links=%d", users, players, links)
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
	var buyInAck struct {
		RequestID        string `json:"request_id"`
		OperationID      string `json:"operation_id"`
		SessionID        string `json:"session_id"`
		PlayerID         string `json:"player_id"`
		Type             string `json:"type"`
		Chips            int64  `json:"chips"`
		CreatedAt        string `json:"created_at"`
		IdempotentReplay bool   `json:"idempotent_replay"`
	}
	decodeJSON(t, buyIn, &buyInAck)
	if buyInAck.OperationID == "" || buyInAck.RequestID != "req-buy-in-1" || buyInAck.Type != "buy_in" || buyInAck.Chips != 100 || buyInAck.IdempotentReplay {
		t.Fatalf("unexpected buy-in acknowledgement: %+v", buyInAck)
	}

	buyInDuplicate := requestJSON(t, handler, http.MethodPost, "/operations/buy-in", map[string]any{
		"request_id": "req-buy-in-1", "session_id": sessionResp.SessionID,
		"player_id": playerResp.PlayerID, "chips": 100,
	})
	var duplicateAck struct {
		OperationID      string `json:"operation_id"`
		IdempotentReplay bool   `json:"idempotent_replay"`
	}
	decodeJSON(t, buyInDuplicate, &duplicateAck)
	if buyInDuplicate.Code != http.StatusOK || duplicateAck.OperationID != buyInAck.OperationID || !duplicateAck.IdempotentReplay {
		t.Fatalf("unexpected duplicate acknowledgement: status=%d ack=%+v", buyInDuplicate.Code, duplicateAck)
	}

	buyInMismatch := requestJSON(t, handler, http.MethodPost, "/operations/buy-in", map[string]any{
		"request_id": "req-buy-in-1", "session_id": sessionResp.SessionID,
		"player_id": playerResp.PlayerID, "chips": 200,
	})
	var mismatchErr struct {
		Error string `json:"error"`
	}
	decodeJSON(t, buyInMismatch, &mismatchErr)
	if buyInMismatch.Code != http.StatusConflict || mismatchErr.Error != "idempotency_payload_mismatch" {
		t.Fatalf("unexpected payload mismatch: status=%d body=%s", buyInMismatch.Code, buyInMismatch.Body.String())
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
	var reverseAck struct {
		OperationID       string `json:"operation_id"`
		TargetOperationID string `json:"target_operation_id"`
		Type              string `json:"type"`
		ReversedOperation struct {
			OperationID string `json:"operation_id"`
			Type        string `json:"type"`
		} `json:"reversed_operation"`
	}
	decodeJSON(t, reverse, &reverseAck)
	if reverseAck.OperationID == "" || reverseAck.TargetOperationID != ops[0].ID || reverseAck.Type != "reversal" || reverseAck.ReversedOperation.OperationID != ops[0].ID || reverseAck.ReversedOperation.Type != "buy_in" {
		t.Fatalf("unexpected reverse acknowledgement: %+v", reverseAck)
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
