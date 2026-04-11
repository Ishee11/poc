package http

import (
	"bytes"
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/entity/valueobject"
	"github.com/ishee11/poc/internal/usecase"
)

type finishSessionTxManagerStub struct{}

func (m *finishSessionTxManagerStub) RunInTx(fn func(tx usecase.Tx) error) error {
	return fn(struct{}{})
}

type finishSessionRepoStub struct {
	session *entity.Session
	saveErr error
}

func (r *finishSessionRepoStub) FindByID(tx usecase.Tx, sessionID entity.SessionID) (*entity.Session, error) {
	return r.session, nil
}

func (r *finishSessionRepoStub) Save(tx usecase.Tx, session *entity.Session) error {
	r.session = session
	return r.saveErr
}

type finishProjectionStub struct{}

func (p *finishProjectionStub) GetSessionAggregates(tx usecase.Tx, sessionID entity.SessionID) (usecase.SessionAggregates, error) {
	return usecase.SessionAggregates{TotalBuyIn: 100, TotalCashOut: 100}, nil
}

func (p *finishProjectionStub) GetPlayerAggregates(tx usecase.Tx, sessionID entity.SessionID) (map[entity.PlayerID]usecase.PlayerAggregates, error) {
	return nil, nil
}

func (p *finishProjectionStub) GetLastOperationType(tx usecase.Tx, sessionID entity.SessionID, playerID entity.PlayerID) (entity.OperationType, bool, error) {
	return "", false, nil
}

func (p *finishProjectionStub) ListBySession(tx usecase.Tx, sessionID entity.SessionID, limit int, offset int) ([]*entity.Operation, error) {
	return nil, nil
}

type finishIdempotencyRepoSpy struct {
	requestID string
}

func (r *finishIdempotencyRepoSpy) Save(tx usecase.Tx, requestID string) error {
	r.requestID = requestID
	return nil
}

func TestHandlerFinishSession_PassesRequestID(t *testing.T) {
	rate, err := valueobject.NewChipRate(10)
	if err != nil {
		t.Fatalf("failed to create chip rate: %v", err)
	}

	sessionRepo := &finishSessionRepoStub{
		session: entity.NewSession("s1", rate, time.Now()),
	}
	idempotencyRepo := &finishIdempotencyRepoSpy{}

	uc := usecase.NewFinishSessionUseCase(
		&finishProjectionStub{},
		sessionRepo,
		sessionRepo,
		&finishSessionTxManagerStub{},
		idempotencyRepo,
	)

	handler := &Handler{
		finishSessionUC: uc,
	}

	body, err := json.Marshal(map[string]any{
		"request_id": "req-finish-1",
		"session_id": "s1",
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(stdhttp.MethodPost, "/session/finish", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.FinishSession(rec, req)

	if rec.Code != stdhttp.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	if idempotencyRepo.requestID != "req-finish-1" {
		t.Fatalf("expected request id to be passed to usecase, got %q", idempotencyRepo.requestID)
	}
}
