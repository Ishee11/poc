package entity

import (
	"errors"
	"testing"
)

func TestNewPlayer(t *testing.T) {
	tests := []struct {
		name    string
		id      PlayerID
		player  string
		wantErr error
	}{
		{name: "valid player", id: "p1", player: "Alice"},
		{name: "empty id", id: "", player: "Alice", wantErr: ErrInvalidPlayerID},
		{name: "empty name", id: "p1", player: "", wantErr: ErrInvalidPlayerName},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			player, err := NewPlayer(tc.id, tc.player)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected error %v, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if player.ID() != tc.id || player.Name() != tc.player {
				t.Fatalf("unexpected player: id=%s name=%s", player.ID(), player.Name())
			}
		})
	}
}

func TestPlayerState_ValidateInGame(t *testing.T) {
	tests := []struct {
		name    string
		lastOp  OperationType
		found   bool
		wantErr error
	}{
		{name: "last buy in means in game", lastOp: OperationBuyIn, found: true},
		{name: "last cash out means not in game", lastOp: OperationCashOut, found: true, wantErr: ErrPlayerNotInGame},
		{name: "not found means not in game", lastOp: "", found: false, wantErr: ErrPlayerNotInGame},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			state := NewPlayerState("p1", tc.lastOp, tc.found)
			err := state.ValidateInGame()
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected error %v, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
