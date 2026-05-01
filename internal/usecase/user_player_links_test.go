package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/ishee11/poc/internal/entity"
)

type fakeUserPlayerLinkRepo struct {
	links map[entity.PlayerID]entity.AuthUserID
}

func newFakeUserPlayerLinkRepo() *fakeUserPlayerLinkRepo {
	return &fakeUserPlayerLinkRepo{links: make(map[entity.PlayerID]entity.AuthUserID)}
}

func (r *fakeUserPlayerLinkRepo) LinkPlayer(_ Tx, userID entity.AuthUserID, playerID entity.PlayerID) error {
	r.links[playerID] = userID
	return nil
}

func (r *fakeUserPlayerLinkRepo) UnlinkPlayer(_ Tx, userID entity.AuthUserID, playerID entity.PlayerID) error {
	if r.links[playerID] == userID {
		delete(r.links, playerID)
	}
	return nil
}

func (r *fakeUserPlayerLinkRepo) ListUserPlayers(_ Tx, userID entity.AuthUserID) ([]PlayerDTO, error) {
	result := make([]PlayerDTO, 0)
	for playerID, linkedUserID := range r.links {
		if linkedUserID == userID {
			result = append(result, PlayerDTO{ID: playerID, Name: string(playerID)})
		}
	}
	return result, nil
}

func (r *fakeUserPlayerLinkRepo) IsPlayerLinked(_ Tx, playerID entity.PlayerID) (bool, error) {
	_, ok := r.links[playerID]
	return ok, nil
}

func (r *fakeUserPlayerLinkRepo) IsPlayerLinkedToUser(
	_ Tx,
	userID entity.AuthUserID,
	playerID entity.PlayerID,
) (bool, error) {
	return r.links[playerID] == userID, nil
}

func (r *fakeUserPlayerLinkRepo) ListUnlinkedPlayers(_ Tx, _ int, _ int) ([]PlayerDTO, error) {
	return nil, nil
}

func TestUserPlayerLinksUseCaseLinkPlayer(t *testing.T) {
	store := newFakeStore()
	player, err := entity.NewPlayer("player-1", "Alice")
	if err != nil {
		t.Fatal(err)
	}
	store.players[player.ID()] = player

	links := newFakeUserPlayerLinkRepo()
	uc := NewUserPlayerLinksUseCase(links, fakePlayerRepo{store: store}, fakeTxManager{})

	err = uc.LinkPlayer(context.Background(), LinkUserPlayerCommand{
		UserID:   "user-1",
		PlayerID: "player-1",
	})
	if err != nil {
		t.Fatalf("LinkPlayer returned error: %v", err)
	}

	if links.links["player-1"] != "user-1" {
		t.Fatalf("player was not linked: %+v", links.links)
	}
}

func TestUserPlayerLinksUseCaseRejectsLinkedPlayerOwnedByAnotherUser(t *testing.T) {
	store := newFakeStore()
	player, err := entity.NewPlayer("player-1", "Alice")
	if err != nil {
		t.Fatal(err)
	}
	store.players[player.ID()] = player

	links := newFakeUserPlayerLinkRepo()
	links.links["player-1"] = "user-2"

	uc := NewUserPlayerLinksUseCase(links, fakePlayerRepo{store: store}, fakeTxManager{})
	err = uc.LinkPlayer(context.Background(), LinkUserPlayerCommand{
		UserID:   "user-1",
		PlayerID: "player-1",
	})
	if !errors.Is(err, entity.ErrPlayerAlreadyLinked) {
		t.Fatalf("expected ErrPlayerAlreadyLinked, got %v", err)
	}
}

func TestUserPlayerLinksUseCaseLinkIsIdempotentForSameUser(t *testing.T) {
	store := newFakeStore()
	player, err := entity.NewPlayer("player-1", "Alice")
	if err != nil {
		t.Fatal(err)
	}
	store.players[player.ID()] = player

	links := newFakeUserPlayerLinkRepo()
	links.links["player-1"] = "user-1"

	uc := NewUserPlayerLinksUseCase(links, fakePlayerRepo{store: store}, fakeTxManager{})
	err = uc.LinkPlayer(context.Background(), LinkUserPlayerCommand{
		UserID:   "user-1",
		PlayerID: "player-1",
	})
	if err != nil {
		t.Fatalf("LinkPlayer returned error: %v", err)
	}
}

func TestUserPlayerLinksUseCaseUnlinkRejectsForeignLink(t *testing.T) {
	links := newFakeUserPlayerLinkRepo()
	links.links["player-1"] = "user-2"

	uc := NewUserPlayerLinksUseCase(links, fakePlayerRepo{store: newFakeStore()}, fakeTxManager{})
	err := uc.UnlinkPlayer(context.Background(), LinkUserPlayerCommand{
		UserID:   "user-1",
		PlayerID: "player-1",
	})
	if !errors.Is(err, entity.ErrUserPlayerNotLinked) {
		t.Fatalf("expected ErrUserPlayerNotLinked, got %v", err)
	}
}
