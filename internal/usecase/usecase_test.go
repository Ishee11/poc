package usecase

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/ishee11/poc/internal/entity"
	"github.com/ishee11/poc/internal/entity/valueobject"
	"github.com/ishee11/poc/internal/usecase/command"
)

type testTx struct{}

func (testTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	panic("test tx Exec should not be called")
}

func (testTx) Query(context.Context, string, ...any) (pgx.Rows, error) {
	panic("test tx Query should not be called")
}

func (testTx) QueryRow(context.Context, string, ...any) pgx.Row {
	panic("test tx QueryRow should not be called")
}

type fakeTxManager struct{}

func (fakeTxManager) RunInTx(_ context.Context, fn func(tx Tx) error) error {
	return fn(testTx{})
}

type fakeStatsRepo struct {
	players []PlayerStat
}

func (r fakeStatsRepo) ListSessions(_ Tx, _ SessionStatsFilter) ([]SessionStat, error) {
	return nil, nil
}

func (r fakeStatsRepo) ListPlayers(_ Tx, _ PlayerStatsFilter) ([]PlayerStat, error) {
	return r.players, nil
}

func (r fakeStatsRepo) GetPlayerOverall(_ Tx, _ entity.PlayerID, _ PlayerStatsFilter) (*PlayerOverallStat, error) {
	return nil, nil
}

func (r fakeStatsRepo) ListPlayerSessions(_ Tx, _ entity.PlayerID, _ PlayerStatsFilter) ([]PlayerSessionStat, error) {
	return nil, nil
}

type sequenceSessionIDGen struct{ next entity.SessionID }

func (g sequenceSessionIDGen) New() entity.SessionID {
	if g.next == "" {
		return "s1"
	}
	return g.next
}

type sequencePlayerIDGen struct{ next entity.PlayerID }

func (g sequencePlayerIDGen) New() entity.PlayerID {
	if g.next == "" {
		return "p1"
	}
	return g.next
}

type sequenceOperationIDGen struct {
	next entity.OperationID
	n    int
}

func (g *sequenceOperationIDGen) New() entity.OperationID {
	if g.next != "" && g.n == 0 {
		g.n++
		return g.next
	}
	g.n++
	return entity.OperationID("op-test-" + string(rune('0'+g.n)))
}

type fakeIdempotencyRepo struct {
	seen map[string]bool
	err  error
}

func newFakeIdempotencyRepo() *fakeIdempotencyRepo {
	return &fakeIdempotencyRepo{seen: make(map[string]bool)}
}

func (r *fakeIdempotencyRepo) Save(_ Tx, requestID string) error {
	if r.err != nil {
		return r.err
	}
	if r.seen[requestID] {
		return entity.ErrDuplicateRequest
	}
	r.seen[requestID] = true
	return nil
}

type fakeOutboxRepo struct {
	events []OutboxEvent
}

func (r *fakeOutboxRepo) Save(_ Tx, event OutboxEvent) error {
	r.events = append(r.events, event)
	return nil
}

type fakeStore struct {
	sessions map[entity.SessionID]*entity.Session
	players  map[entity.PlayerID]*entity.Player
	ops      map[entity.OperationID]*entity.Operation
	opOrder  []entity.OperationID
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		sessions: make(map[entity.SessionID]*entity.Session),
		players:  make(map[entity.PlayerID]*entity.Player),
		ops:      make(map[entity.OperationID]*entity.Operation),
	}
}

type fakeSessionRepo struct{ store *fakeStore }

func (r fakeSessionRepo) FindByID(_ Tx, id entity.SessionID) (*entity.Session, error) {
	session, ok := r.store.sessions[id]
	if !ok {
		return nil, entity.ErrSessionNotFound
	}
	return session, nil
}

func (r fakeSessionRepo) FindByIDForUpdate(tx Tx, id entity.SessionID) (*entity.Session, error) {
	return r.FindByID(tx, id)
}

func (r fakeSessionRepo) Save(_ Tx, session *entity.Session) error {
	r.store.sessions[session.ID()] = session
	return nil
}

type fakePlayerRepo struct{ store *fakeStore }

func (r fakePlayerRepo) Create(_ Tx, player *entity.Player) error {
	r.store.players[player.ID()] = player
	return nil
}

func (r fakePlayerRepo) Exists(_ Tx, id entity.PlayerID) (bool, error) {
	_, ok := r.store.players[id]
	return ok, nil
}

func (r fakePlayerRepo) GetByID(_ Tx, id entity.PlayerID) (*entity.Player, error) {
	player, ok := r.store.players[id]
	if !ok {
		return nil, entity.ErrPlayerNotFound
	}
	return player, nil
}

func (r fakePlayerRepo) List(_ Tx, _ int, _ int) ([]PlayerDTO, error) {
	result := make([]PlayerDTO, 0, len(r.store.players))
	for _, player := range r.store.players {
		result = append(result, PlayerDTO{ID: player.ID(), Name: player.Name()})
	}
	return result, nil
}

type fakeOperationRepo struct{ store *fakeStore }

func (r fakeOperationRepo) Save(_ Tx, op *entity.Operation) error {
	if _, exists := r.store.ops[op.ID()]; !exists {
		r.store.opOrder = append(r.store.opOrder, op.ID())
	}
	r.store.ops[op.ID()] = op
	return nil
}

func (r fakeOperationRepo) GetByID(_ Tx, id entity.OperationID) (*entity.Operation, error) {
	op, ok := r.store.ops[id]
	if !ok {
		return nil, entity.ErrOperationNotFound
	}
	return op, nil
}

func (r fakeOperationRepo) GetByRequestID(_ Tx, requestID string) (*entity.Operation, error) {
	for _, op := range r.store.ops {
		if op.RequestID() == requestID {
			return op, nil
		}
	}
	return nil, entity.ErrOperationNotFound
}

func (r fakeOperationRepo) ExistsReversal(_ Tx, targetID entity.OperationID) (bool, error) {
	for _, op := range r.store.ops {
		if op.Type() == entity.OperationReversal && op.ReferenceID() != nil && *op.ReferenceID() == targetID {
			return true, nil
		}
	}
	return false, nil
}

type fakeProjectionRepo struct{ store *fakeStore }

func (r fakeProjectionRepo) GetSessionAggregates(_ Tx, sessionID entity.SessionID) (SessionAggregates, error) {
	aggs := r.store.playerAggregates(sessionID)
	var total SessionAggregates
	for _, agg := range aggs {
		total.TotalBuyIn += agg.BuyIn
		total.TotalCashOut += agg.CashOut
	}
	return total, nil
}

func (r fakeProjectionRepo) GetPlayerAggregates(_ Tx, sessionID entity.SessionID) (map[entity.PlayerID]PlayerAggregates, error) {
	return r.store.playerAggregates(sessionID), nil
}

func (r fakeProjectionRepo) GetLastOperationType(_ Tx, sessionID entity.SessionID, playerID entity.PlayerID) (entity.OperationType, bool, error) {
	reversed := r.store.reversedTargets()
	ops := r.store.orderedOps()
	for i := len(ops) - 1; i >= 0; i-- {
		op := ops[i]
		if op.SessionID() == sessionID && op.PlayerID() == playerID && op.Type() != entity.OperationReversal && !reversed[op.ID()] {
			return op.Type(), true, nil
		}
	}
	return "", false, nil
}

func (r fakeProjectionRepo) ListBySession(_ Tx, sessionID entity.SessionID, limit int, offset int) ([]*entity.Operation, error) {
	ops := r.store.orderedOps()
	filtered := make([]*entity.Operation, 0, len(ops))
	for _, op := range ops {
		if op.SessionID() == sessionID {
			filtered = append(filtered, op)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt().After(filtered[j].CreatedAt())
	})
	if offset >= len(filtered) {
		return nil, nil
	}
	end := offset + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[offset:end], nil
}

func (s *fakeStore) saveOperation(t *testing.T, op *entity.Operation) {
	t.Helper()
	if err := (fakeOperationRepo{store: s}).Save(testTx{}, op); err != nil {
		t.Fatal(err)
	}
}

func (s *fakeStore) playerAggregates(sessionID entity.SessionID) map[entity.PlayerID]PlayerAggregates {
	result := make(map[entity.PlayerID]PlayerAggregates)
	for _, op := range s.orderedOps() {
		if op.SessionID() != sessionID {
			continue
		}
		agg := result[op.PlayerID()]
		switch op.Type() {
		case entity.OperationBuyIn:
			agg.BuyIn += op.Chips()
		case entity.OperationCashOut:
			agg.CashOut += op.Chips()
		case entity.OperationReversal:
			if op.ReferenceID() == nil {
				continue
			}
			ref := s.ops[*op.ReferenceID()]
			if ref == nil {
				continue
			}
			switch ref.Type() {
			case entity.OperationBuyIn:
				agg.BuyIn -= op.Chips()
			case entity.OperationCashOut:
				agg.CashOut -= op.Chips()
			}
		}
		result[op.PlayerID()] = agg
	}
	return result
}

func (s *fakeStore) reversedTargets() map[entity.OperationID]bool {
	result := make(map[entity.OperationID]bool)
	for _, op := range s.ops {
		if op.Type() == entity.OperationReversal && op.ReferenceID() != nil {
			result[*op.ReferenceID()] = true
		}
	}
	return result
}

func (s *fakeStore) orderedOps() []*entity.Operation {
	result := make([]*entity.Operation, 0, len(s.opOrder))
	for _, id := range s.opOrder {
		result = append(result, s.ops[id])
	}
	return result
}

func addPlayer(t *testing.T, store *fakeStore, id entity.PlayerID, name string) {
	t.Helper()
	player, err := entity.NewPlayer(id, name)
	if err != nil {
		t.Fatal(err)
	}
	store.players[id] = player
}

func addSession(t *testing.T, store *fakeStore, id entity.SessionID, status entity.Status, buyIn int64, cashOut int64) {
	t.Helper()
	store.sessions[id] = entity.RestoreSession(id, mustChipRate(t, 2), 2, entity.CurrencyRUB, status, time.Now(), nil, buyIn, cashOut)
}

func newHelperForStore(store *fakeStore, opGen OperationIDGenerator, playerGen PlayerIDGenerator) *Helper {
	return NewHelper(
		fakeSessionRepo{store: store},
		fakeSessionRepo{store: store},
		fakePlayerRepo{store: store},
		fakeOperationRepo{store: store},
		opGen,
		playerGen,
	)
}

func TestIdempotent(t *testing.T) {
	t.Run("empty request id", func(t *testing.T) {
		called := false
		err := Idempotent(testTx{}, newFakeIdempotencyRepo(), "", func() error {
			called = true
			return nil
		})
		if !errors.Is(err, entity.ErrInvalidRequestID) {
			t.Fatalf("expected invalid request id, got %v", err)
		}
		if called {
			t.Fatal("callback should not be called")
		}
	})

	t.Run("duplicate request skips callback", func(t *testing.T) {
		repo := newFakeIdempotencyRepo()
		repo.seen["req1"] = true
		called := false
		err := Idempotent(testTx{}, repo, "req1", func() error {
			called = true
			return nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if called {
			t.Fatal("callback should not be called")
		}
	})

	t.Run("new request calls callback", func(t *testing.T) {
		called := false
		err := Idempotent(testTx{}, newFakeIdempotencyRepo(), "req1", func() error {
			called = true
			return nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !called {
			t.Fatal("callback should be called")
		}
	})
}

func TestStartSessionUseCase(t *testing.T) {
	store := newFakeStore()
	uc := NewStartSessionUseCase(
		fakeSessionRepo{store: store},
		fakeSessionRepo{store: store},
		fakeTxManager{},
		sequenceSessionIDGen{next: "s1"},
	)

	id, err := uc.Execute(context.Background(), command.StartSessionCommand{ChipRate: 2, BigBlind: 2, Currency: entity.CurrencyRUB})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "s1" {
		t.Fatalf("expected session id s1, got %s", id)
	}
	if store.sessions[id].ChipRate().Value() != 2 {
		t.Fatalf("session was not saved with chip rate")
	}

	if _, err := uc.Execute(context.Background(), command.StartSessionCommand{ChipRate: 0, BigBlind: 2, Currency: entity.CurrencyRUB}); !errors.Is(err, valueobject.ErrInvalidChips) {
		t.Fatalf("expected invalid chips, got %v", err)
	}
	if _, err := uc.Execute(context.Background(), command.StartSessionCommand{ChipRate: 2, BigBlind: 0, Currency: entity.CurrencyRUB}); !errors.Is(err, valueobject.ErrInvalidChips) {
		t.Fatalf("expected invalid chips, got %v", err)
	}
}

func TestCreatePlayerUseCase(t *testing.T) {
	store := newFakeStore()
	idem := newFakeIdempotencyRepo()
	helper := newHelperForStore(store, &sequenceOperationIDGen{}, sequencePlayerIDGen{next: "p1"})
	uc := NewCreatePlayerUseCase(helper, fakeTxManager{}, idem)

	id, err := uc.Execute(context.Background(), command.CreatePlayerCommand{RequestID: "req1", Name: " Alice "})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "p1" || store.players[id].Name() != "Alice" {
		t.Fatalf("player was not created with trimmed name")
	}

	duplicateID, err := uc.Execute(context.Background(), command.CreatePlayerCommand{RequestID: "req1", Name: "Bob"})
	if err != nil {
		t.Fatalf("duplicate request should be idempotent, got %v", err)
	}
	if duplicateID != "" {
		t.Fatalf("duplicate request should not return a new id, got %s", duplicateID)
	}

	if _, err := uc.Execute(context.Background(), command.CreatePlayerCommand{RequestID: "req2", Name: " "}); !errors.Is(err, entity.ErrInvalidPlayerName) {
		t.Fatalf("expected invalid player name, got %v", err)
	}
}

func TestBuyInUseCase(t *testing.T) {
	store := newFakeStore()
	addSession(t, store, "s1", entity.StatusActive, 0, 0)
	addPlayer(t, store, "p1", "Alice")

	helper := newHelperForStore(store, &sequenceOperationIDGen{next: "op1"}, sequencePlayerIDGen{})
	outbox := &fakeOutboxRepo{}
	uc := NewBuyInUseCase(helper, fakeTxManager{}, newFakeIdempotencyRepo(), outbox)

	err := uc.Execute(context.Background(), command.BuyInCommand{RequestID: "req1", SessionID: "s1", PlayerID: "p1", Chips: 100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.sessions["s1"].TotalBuyIn() != 100 || len(store.ops) != 1 {
		t.Fatalf("buy in did not update session and save operation")
	}
	if len(outbox.events) != 1 || outbox.events[0].EventType != OutboxEventOperationCreated {
		t.Fatalf("buy in did not save operation.created event")
	}

	err = uc.Execute(context.Background(), command.BuyInCommand{RequestID: "req2", SessionID: "s1", PlayerID: "missing", Chips: 100})
	if !errors.Is(err, entity.ErrPlayerNotFound) {
		t.Fatalf("expected player not found, got %v", err)
	}
}

func TestCashOutUseCase(t *testing.T) {
	store := newFakeStore()
	addSession(t, store, "s1", entity.StatusActive, 100, 0)
	addPlayer(t, store, "p1", "Alice")
	buyInOp, err := entity.NewOperation("op1", "req1", "s1", entity.OperationBuyIn, "p1", 100, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	store.saveOperation(t, buyInOp)

	helper := newHelperForStore(store, &sequenceOperationIDGen{next: "op2"}, sequencePlayerIDGen{})
	outbox := &fakeOutboxRepo{}
	uc := NewCashOutUseCase(
		helper,
		fakeSessionRepo{store: store},
		fakeProjectionRepo{store: store},
		fakeTxManager{},
		newFakeIdempotencyRepo(),
		outbox,
	)

	if err := uc.Execute(context.Background(), command.CashOutCommand{RequestID: "req2", SessionID: "s1", PlayerID: "p1", Chips: 40}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.sessions["s1"].TotalCashOut() != 40 {
		t.Fatalf("cash out did not update session")
	}
	if len(outbox.events) != 1 || outbox.events[0].EventType != OutboxEventOperationCreated {
		t.Fatalf("cash out did not save operation.created event")
	}

	err = uc.Execute(context.Background(), command.CashOutCommand{RequestID: "req3", SessionID: "s1", PlayerID: "p1", Chips: 1000})
	if !errors.Is(err, entity.ErrInvalidCashOut) {
		t.Fatalf("expected invalid cash out, got %v", err)
	}
}

func TestFinishSessionUseCase(t *testing.T) {
	t.Run("balanced session finishes", func(t *testing.T) {
		store := newFakeStore()
		addSession(t, store, "s1", entity.StatusActive, 100, 100)
		outbox := &fakeOutboxRepo{}
		uc := NewFinishSessionUseCase(
			fakeProjectionRepo{store: store},
			fakeSessionRepo{store: store},
			fakeSessionRepo{store: store},
			fakeTxManager{},
			newFakeIdempotencyRepo(),
			outbox,
		)

		if err := uc.Execute(context.Background(), command.FinishSessionCommand{RequestID: "req1", SessionID: "s1"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if store.sessions["s1"].Status() != entity.StatusFinished {
			t.Fatal("session was not finished")
		}
		if len(outbox.events) != 1 || outbox.events[0].EventType != OutboxEventSessionFinished {
			t.Fatalf("finish session did not save session.finished event")
		}
	})

	t.Run("unbalanced session returns remaining chips", func(t *testing.T) {
		store := newFakeStore()
		addSession(t, store, "s1", entity.StatusActive, 100, 40)
		buyInOp, err := entity.NewOperation("op1", "req1", "s1", entity.OperationBuyIn, "p1", 100, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		cashOutOp, err := entity.NewOperation("op2", "req2", "s1", entity.OperationCashOut, "p1", 40, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		store.saveOperation(t, buyInOp)
		store.saveOperation(t, cashOutOp)
		uc := NewFinishSessionUseCase(
			fakeProjectionRepo{store: store},
			fakeSessionRepo{store: store},
			fakeSessionRepo{store: store},
			fakeTxManager{},
			newFakeIdempotencyRepo(),
			&fakeOutboxRepo{},
		)

		err = uc.Execute(context.Background(), command.FinishSessionCommand{RequestID: "req1", SessionID: "s1"})
		var balancedErr *entity.SessionNotBalancedError
		if !errors.As(err, &balancedErr) {
			t.Fatalf("expected SessionNotBalancedError, got %v", err)
		}
		if balancedErr.RemainingChips != 60 {
			t.Fatalf("expected remaining chips 60, got %d", balancedErr.RemainingChips)
		}
	})
}

func TestReverseOperationUseCase(t *testing.T) {
	t.Run("reverses buy in", func(t *testing.T) {
		store := newFakeStore()
		addSession(t, store, "s1", entity.StatusActive, 100, 0)
		addPlayer(t, store, "p1", "Alice")
		target, err := entity.NewOperation("op1", "req1", "s1", entity.OperationBuyIn, "p1", 100, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		store.saveOperation(t, target)

		outbox := &fakeOutboxRepo{}
		uc := NewReverseOperationUseCase(
			fakeOperationRepo{store: store},
			fakeOperationRepo{store: store},
			fakeOperationRepo{store: store},
			fakeSessionRepo{store: store},
			fakeTxManager{},
			&sequenceOperationIDGen{next: "op2"},
			newFakeIdempotencyRepo(),
			fakeSessionRepo{store: store},
			outbox,
		)

		if err := uc.Execute(context.Background(), command.ReverseOperationCommand{RequestID: "req2", TargetOperationID: "op1"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if store.sessions["s1"].TotalChips() != 0 {
			t.Fatalf("expected reversed buy in to clear table chips, got %d", store.sessions["s1"].TotalChips())
		}
		if len(outbox.events) != 1 || outbox.events[0].EventType != OutboxEventOperationReversed {
			t.Fatalf("reverse operation did not save operation.reversed event")
		}

		err = uc.Execute(context.Background(), command.ReverseOperationCommand{RequestID: "req3", TargetOperationID: "op1"})
		if !errors.Is(err, entity.ErrOperationAlreadyReversed) {
			t.Fatalf("expected operation already reversed, got %v", err)
		}
	})

	t.Run("reverses cash out without increasing buy in", func(t *testing.T) {
		store := newFakeStore()
		addSession(t, store, "s1", entity.StatusActive, 100, 40)
		addPlayer(t, store, "p1", "Alice")
		buyInOp, err := entity.NewOperation("op1", "req1", "s1", entity.OperationBuyIn, "p1", 100, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		cashOutOp, err := entity.NewOperation("op2", "req2", "s1", entity.OperationCashOut, "p1", 40, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		store.saveOperation(t, buyInOp)
		store.saveOperation(t, cashOutOp)

		outbox := &fakeOutboxRepo{}
		uc := NewReverseOperationUseCase(
			fakeOperationRepo{store: store},
			fakeOperationRepo{store: store},
			fakeOperationRepo{store: store},
			fakeSessionRepo{store: store},
			fakeTxManager{},
			&sequenceOperationIDGen{next: "op3"},
			newFakeIdempotencyRepo(),
			fakeSessionRepo{store: store},
			outbox,
		)

		if err := uc.Execute(context.Background(), command.ReverseOperationCommand{RequestID: "req3", TargetOperationID: "op2"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if store.sessions["s1"].TotalBuyIn() != 100 {
			t.Fatalf("expected buy in to stay 100, got %d", store.sessions["s1"].TotalBuyIn())
		}
		if store.sessions["s1"].TotalCashOut() != 0 {
			t.Fatalf("expected cash out to become 0, got %d", store.sessions["s1"].TotalCashOut())
		}
		if store.sessions["s1"].TotalChips() != 100 {
			t.Fatalf("expected table chips to become 100, got %d", store.sessions["s1"].TotalChips())
		}
		if len(outbox.events) != 1 || outbox.events[0].EventType != OutboxEventOperationReversed {
			t.Fatalf("reverse operation did not save operation.reversed event")
		}
	})
}

func TestGetSessionPlayersUseCase(t *testing.T) {
	store := newFakeStore()
	store.sessions["s1"] = entity.RestoreSession("s1", mustChipRate(t, 2), 2, entity.CurrencyRUB, entity.StatusFinished, time.Now(), nil, 100, 40)
	addPlayer(t, store, "p1", "Alice")
	buyInOp, err := entity.NewOperation("op1", "req1", "s1", entity.OperationBuyIn, "p1", 100, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	cashOutOp, err := entity.NewOperation("op2", "req2", "s1", entity.OperationCashOut, "p1", 40, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	store.saveOperation(t, buyInOp)
	store.saveOperation(t, cashOutOp)

	uc := NewGetSessionPlayersUseCase(
		fakeProjectionRepo{store: store},
		fakePlayerRepo{store: store},
		fakeStatsRepo{players: []PlayerStat{{PlayerID: "p1", SessionsCount: 3, ProfitMoney: -30}}},
		fakeTxManager{},
		fakeSessionRepo{store: store},
	)
	players, err := uc.Execute(context.Background(), GetSessionPlayersQuery{SessionID: "s1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(players) != 1 {
		t.Fatalf("expected one player, got %d", len(players))
	}
	player := players[0]
	if player.InGame {
		t.Fatal("finished session player should not be in game")
	}
	if player.ProfitChips != -60 || player.ProfitMoney != -30 {
		t.Fatalf("unexpected profit: chips=%d money=%d", player.ProfitChips, player.ProfitMoney)
	}
}

func TestGetSessionPlayersUseCase_CashOutSettlesActivePlayer(t *testing.T) {
	store := newFakeStore()
	store.sessions["s1"] = entity.RestoreSession("s1", mustChipRate(t, 2), 2, entity.CurrencyRUB, entity.StatusActive, time.Now(), nil, 100, 40)
	addPlayer(t, store, "p1", "Alice")
	buyInOp, err := entity.NewOperation("op1", "req1", "s1", entity.OperationBuyIn, "p1", 100, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	cashOutOp, err := entity.NewOperation("op2", "req2", "s1", entity.OperationCashOut, "p1", 40, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	store.saveOperation(t, buyInOp)
	store.saveOperation(t, cashOutOp)

	uc := NewGetSessionPlayersUseCase(
		fakeProjectionRepo{store: store},
		fakePlayerRepo{store: store},
		fakeStatsRepo{players: []PlayerStat{{PlayerID: "p1", SessionsCount: 3, ProfitMoney: -30}}},
		fakeTxManager{},
		fakeSessionRepo{store: store},
	)
	players, err := uc.Execute(context.Background(), GetSessionPlayersQuery{SessionID: "s1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(players) != 1 {
		t.Fatalf("expected one player, got %d", len(players))
	}
	if players[0].InGame {
		t.Fatal("player should be settled after cash-out even when cash-out is lower than buy-in")
	}
}

func mustChipRate(t *testing.T, value int64) valueobject.ChipRate {
	t.Helper()
	rate, err := valueobject.NewChipRate(value)
	if err != nil {
		t.Fatal(err)
	}
	return rate
}
