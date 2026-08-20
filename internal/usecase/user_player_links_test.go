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

func (r *fakeUserPlayerLinkRepo) FindUserPlayer(_ Tx, userID entity.AuthUserID) (*PlayerDTO, error) {
	for playerID, linkedUserID := range r.links {
		if linkedUserID == userID {
			return &PlayerDTO{ID: playerID, Name: string(playerID)}, nil
		}
	}
	return nil, nil
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

func (r *fakeUserPlayerLinkRepo) ListUnlinkedPlayers(_ Tx, _ int, _ int) ([]AvailablePlayerDTO, error) {
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
	uc := NewUserPlayerLinksUseCase(links, fakePlayerRepo{store: store}, sequencePlayerIDGen{next: "player-new"}, fakeTxManager{})

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

	uc := NewUserPlayerLinksUseCase(links, fakePlayerRepo{store: store}, sequencePlayerIDGen{next: "player-new"}, fakeTxManager{})
	err = uc.LinkPlayer(context.Background(), LinkUserPlayerCommand{
		UserID:   "user-1",
		PlayerID: "player-1",
	})
	if !errors.Is(err, entity.ErrPlayerAlreadyLinked) {
		t.Fatalf("expected ErrPlayerAlreadyLinked, got %v", err)
	}
}

func TestUserPlayerLinksUseCaseRejectsSecondClaimForSameUser(t *testing.T) {
	store := newFakeStore()
	player, err := entity.NewPlayer("player-1", "Alice")
	if err != nil {
		t.Fatal(err)
	}
	store.players[player.ID()] = player

	links := newFakeUserPlayerLinkRepo()
	links.links["player-1"] = "user-1"

	uc := NewUserPlayerLinksUseCase(links, fakePlayerRepo{store: store}, sequencePlayerIDGen{next: "player-new"}, fakeTxManager{})
	err = uc.LinkPlayer(context.Background(), LinkUserPlayerCommand{
		UserID:   "user-1",
		PlayerID: "player-1",
	})
	if !errors.Is(err, entity.ErrAccountAlreadyLinked) {
		t.Fatalf("expected ErrAccountAlreadyLinked, got %v", err)
	}
}

func TestUserPlayerLinksUseCaseUnlinkRejectsForeignLink(t *testing.T) {
	links := newFakeUserPlayerLinkRepo()
	links.links["player-1"] = "user-2"

	uc := NewUserPlayerLinksUseCase(links, fakePlayerRepo{store: newFakeStore()}, sequencePlayerIDGen{next: "player-new"}, fakeTxManager{})
	err := uc.UnlinkPlayer(context.Background(), LinkUserPlayerCommand{
		UserID:   "user-1",
		PlayerID: "player-1",
	})
	if !errors.Is(err, entity.ErrUserPlayerNotLinked) {
		t.Fatalf("expected ErrUserPlayerNotLinked, got %v", err)
	}
}

func TestUserPlayerLinksUseCaseCreatesAndClaimsNewPlayer(t *testing.T) {
	store := newFakeStore()
	links := newFakeUserPlayerLinkRepo()
	uc := NewUserPlayerLinksUseCase(
		links,
		fakePlayerRepo{store: store},
		sequencePlayerIDGen{next: "player-new"},
		fakeTxManager{},
	)

	player, err := uc.ChooseOrCreatePlayer(context.Background(), "user-1", PlayerSelection{
		Mode: PlayerSelectionNew,
		Name: " Alice ",
	})
	if err != nil {
		t.Fatalf("ChooseOrCreatePlayer returned error: %v", err)
	}
	if player.ID != "player-new" || player.Name != "Alice" || links.links[player.ID] != "user-1" {
		t.Fatalf("unexpected claimed player: player=%+v links=%+v", player, links.links)
	}
}

func TestPlayerSelectionValidation(t *testing.T) {
	tests := []struct {
		name      string
		selection PlayerSelection
		wantErr   bool
	}{
		{name: "existing", selection: PlayerSelection{Mode: PlayerSelectionExisting, PlayerID: "player-1"}},
		{name: "new", selection: PlayerSelection{Mode: PlayerSelectionNew, Name: "Alice"}},
		{name: "missing", selection: PlayerSelection{}, wantErr: true},
		{name: "existing with name", selection: PlayerSelection{Mode: PlayerSelectionExisting, PlayerID: "player-1", Name: "Alice"}, wantErr: true},
		{name: "new with id", selection: PlayerSelection{Mode: PlayerSelectionNew, PlayerID: "player-1", Name: "Alice"}, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.selection.Validate()
			if tc.wantErr != errors.Is(err, entity.ErrInvalidPlayerSelection) {
				t.Fatalf("Validate() error=%v wantErr=%v", err, tc.wantErr)
			}
		})
	}
}
