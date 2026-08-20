package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ishee11/poc/internal/entity"
)

type fakeAdminOwnershipRepo struct {
	accounts map[entity.AuthUserID]AccountOwnershipDTO
	players  map[entity.PlayerID]string
	links    map[entity.AuthUserID]entity.PlayerID
}

func newFakeAdminOwnershipRepo() *fakeAdminOwnershipRepo {
	return &fakeAdminOwnershipRepo{
		accounts: make(map[entity.AuthUserID]AccountOwnershipDTO),
		players:  make(map[entity.PlayerID]string),
		links:    make(map[entity.AuthUserID]entity.PlayerID),
	}
}

func (r *fakeAdminOwnershipRepo) ListAccounts(_ Tx, query string, limit int, offset int) ([]AccountOwnershipDTO, int64, error) {
	matched := make([]AccountOwnershipDTO, 0)
	for id, account := range r.accounts {
		playerID, linked := r.links[id]
		if linked {
			account.Player = &PlayerDTO{ID: playerID, Name: r.players[playerID]}
		}
		if query == "" || strings.Contains(account.Email, query) || account.Player != nil && strings.Contains(account.Player.Name, query) {
			matched = append(matched, account)
		}
	}
	total := int64(len(matched))
	if offset >= len(matched) {
		return []AccountOwnershipDTO{}, total, nil
	}
	matched = matched[offset:]
	if len(matched) > limit {
		matched = matched[:limit]
	}
	return matched, total, nil
}

func (r *fakeAdminOwnershipRepo) LockUser(_ Tx, userID entity.AuthUserID) error {
	if _, ok := r.accounts[userID]; !ok {
		return entity.ErrAuthUserNotFound
	}
	return nil
}

func (r *fakeAdminOwnershipRepo) LockPlayer(_ Tx, playerID entity.PlayerID) error {
	if _, ok := r.players[playerID]; !ok {
		return entity.ErrPlayerNotFound
	}
	return nil
}

func (r *fakeAdminOwnershipRepo) FindUserPlayer(_ Tx, userID entity.AuthUserID) (*PlayerDTO, error) {
	playerID, ok := r.links[userID]
	if !ok {
		return nil, nil
	}
	return &PlayerDTO{ID: playerID, Name: r.players[playerID]}, nil
}

func (r *fakeAdminOwnershipRepo) FindPlayerOwner(_ Tx, playerID entity.PlayerID) (*entity.AuthUserID, error) {
	for userID, linkedPlayerID := range r.links {
		if linkedPlayerID == playerID {
			owner := userID
			return &owner, nil
		}
	}
	return nil, nil
}

func (r *fakeAdminOwnershipRepo) LinkPlayer(_ Tx, userID entity.AuthUserID, playerID entity.PlayerID) error {
	for owner, linkedPlayerID := range r.links {
		if linkedPlayerID == playerID && owner != userID {
			return entity.ErrPlayerAlreadyLinked
		}
	}
	r.links[userID] = playerID
	return nil
}

func (r *fakeAdminOwnershipRepo) UnlinkPlayer(_ Tx, userID entity.AuthUserID, playerID entity.PlayerID) error {
	if r.links[userID] == playerID {
		delete(r.links, userID)
	}
	return nil
}

func TestAdminAccountOwnershipReplaceAndClear(t *testing.T) {
	repo := newFakeAdminOwnershipRepo()
	repo.accounts["user-1"] = AccountOwnershipDTO{ID: "user-1", Email: "one@example.com"}
	repo.players["player-1"] = "Alice"
	repo.players["player-2"] = "Bob"
	repo.links["user-1"] = "player-1"
	service := NewAdminAccountOwnershipService(repo, fakeTxManager{})

	change, err := service.Replace(context.Background(), "user-1", "player-2")
	if err != nil {
		t.Fatalf("Replace returned error: %v", err)
	}
	if repo.links["user-1"] != "player-2" || change.OldPlayerID == nil || *change.OldPlayerID != "player-1" || change.NewPlayerID == nil || *change.NewPlayerID != "player-2" {
		t.Fatalf("unexpected replacement: change=%+v links=%+v", change, repo.links)
	}

	change, err = service.Clear(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("Clear returned error: %v", err)
	}
	if _, linked := repo.links["user-1"]; linked || change.OldPlayerID == nil || *change.OldPlayerID != "player-2" {
		t.Fatalf("unexpected clear: change=%+v links=%+v", change, repo.links)
	}
	if _, err := service.Clear(context.Background(), "user-1"); err != nil {
		t.Fatalf("idempotent Clear returned error: %v", err)
	}
}

func TestAdminAccountOwnershipRejectsOccupiedPlayerWithoutChangingLinks(t *testing.T) {
	repo := newFakeAdminOwnershipRepo()
	repo.accounts["user-1"] = AccountOwnershipDTO{ID: "user-1"}
	repo.accounts["user-2"] = AccountOwnershipDTO{ID: "user-2"}
	repo.players["player-1"] = "Alice"
	repo.players["player-2"] = "Bob"
	repo.links["user-1"] = "player-1"
	repo.links["user-2"] = "player-2"
	service := NewAdminAccountOwnershipService(repo, fakeTxManager{})

	_, err := service.Replace(context.Background(), "user-1", "player-2")
	if !errors.Is(err, entity.ErrPlayerAlreadyLinked) {
		t.Fatalf("expected ErrPlayerAlreadyLinked, got %v", err)
	}
	if repo.links["user-1"] != "player-1" || repo.links["user-2"] != "player-2" {
		t.Fatalf("conflict changed ownership: %+v", repo.links)
	}
}

func TestAdminAccountOwnershipSamePlayerIsIdempotent(t *testing.T) {
	repo := newFakeAdminOwnershipRepo()
	repo.accounts["user-1"] = AccountOwnershipDTO{ID: "user-1"}
	repo.players["player-1"] = "Alice"
	repo.links["user-1"] = "player-1"
	service := NewAdminAccountOwnershipService(repo, fakeTxManager{})

	change, err := service.Replace(context.Background(), "user-1", "player-1")
	if err != nil {
		t.Fatalf("idempotent Replace returned error: %v", err)
	}
	if change.OldPlayerID == nil || change.NewPlayerID == nil || *change.OldPlayerID != *change.NewPlayerID {
		t.Fatalf("unexpected idempotent change: %+v", change)
	}
}
