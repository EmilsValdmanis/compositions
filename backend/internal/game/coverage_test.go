package game

import (
	"errors"
	"testing"
)

func TestGameStateRemainingControlFlow(t *testing.T) {
	t.Run("atomic play restore and play errors", func(t *testing.T) {
		invalidSnapshot := newTurnTestState()
		invalidSnapshot.maxPlayers = 0
		if err := invalidSnapshot.PlayTableAndDiscard(0, nil, nil); err == nil {
			t.Fatal("PlayTableAndDiscard() error = nil for invalid snapshot")
		}
		state := newTurnTestState()
		state.turn.hasDrawn = true
		if err := state.PlayTableAndDiscard(0, nil, nil); !errors.Is(err, ErrInvalidComposition) {
			t.Fatalf("PlayTableAndDiscard() error = %v; want %v", err, ErrInvalidComposition)
		}
	})

	t.Run("accessors and navigation", func(t *testing.T) {
		if _, err := (*GameState)(nil).ForfeitPlayer("missing"); !errors.Is(err, ErrGameNotInProgress) {
			t.Fatalf("nil ForfeitPlayer() error = %v", err)
		}
		state := NewGameState()
		state.players = []*Player{nil, NewPlayer(), NewPlayer()}
		state.players[2].forfeited = true
		state.turn.playerIndex = 2
		if player, err := state.CurrentPlayer(); player != nil || !errors.Is(err, ErrPlayerNotFound) {
			t.Fatalf("CurrentPlayer() = (%v, %v)", player, err)
		}
		if state.IsPlayerForfeited("missing") || !state.IsPlayerForfeited(state.players[2].ID) {
			t.Fatal("IsPlayerForfeited() returned unexpected values")
		}
		if got := state.ActivePlayerIndexes(); len(got) != 1 || got[0] != 1 {
			t.Fatalf("ActivePlayerIndexes() = %v", got)
		}
		empty := &GameState{}
		if empty.nextActivePlayerIndex(3) != 0 || empty.previousActivePlayerIndex(3) != 0 {
			t.Fatal("empty navigation should return zero")
		}
		allInactive := &GameState{players: []*Player{nil, {forfeited: true}}}
		if allInactive.nextActivePlayerIndex(1) != 1 || allInactive.previousActivePlayerIndex(1) != 1 {
			t.Fatal("navigation with no active players should preserve index")
		}
		if dealChooserIndex(0, 4) != 3 || nextPlayerIndex(3, 4) != 0 {
			t.Fatal("seat index helpers returned unexpected values")
		}
	})

	t.Run("end without winner", func(t *testing.T) {
		if err := (*GameState)(nil).EndWithoutWinner(); !errors.Is(err, ErrGameNotInProgress) {
			t.Fatalf("nil EndWithoutWinner() error = %v", err)
		}
		state := newTurnTestState()
		state.turn.hasDrawn = true
		state.turn.mustUseDiscardDraw = true
		if err := state.EndWithoutWinner(); err != nil {
			t.Fatalf("EndWithoutWinner() error = %v", err)
		}
		if state.phase != PhaseGameOver || state.roundWinnerIndex != -1 || state.turn.hasDrawn || state.turn.mustUseDiscardDraw {
			t.Fatalf("state after EndWithoutWinner() = %+v", state)
		}
	})

	t.Run("next round positions", func(t *testing.T) {
		if _, _, err := (*GameState)(nil).NextRoundDealerAndChooser(); !errors.Is(err, ErrNotEnoughPlayers) {
			t.Fatalf("nil NextRoundDealerAndChooser() error = %v", err)
		}
		state := NewGameState()
		state.players = []*Player{NewPlayer(), NewPlayer(), NewPlayer()}
		state.players[1].forfeited = true
		state.dealerIndex = 0
		dealer, chooser, err := state.NextRoundDealerAndChooser()
		if err != nil || dealer != 2 || chooser != 0 {
			t.Fatalf("NextRoundDealerAndChooser() = (%d, %d, %v)", dealer, chooser, err)
		}
	})

	t.Run("forfeit errors and order", func(t *testing.T) {
		state := newTurnTestState()
		if _, err := state.ForfeitPlayer("missing"); !errors.Is(err, ErrPlayerNotFound) {
			t.Fatalf("ForfeitPlayer(missing) error = %v", err)
		}
		state.players = append(state.players, NewPlayer())
		state.players[1].forfeited = true
		state.players[1].statistics.ForfeitOrder = 1
		if _, err := state.ForfeitPlayer(state.players[1].ID); !errors.Is(err, ErrPlayerAlreadyForfeited) {
			t.Fatalf("ForfeitPlayer(already) error = %v", err)
		}
		state.players[2].statistics.RoundsPlayed = 1
		state.players[2].hand = nil
		if _, err := state.ForfeitPlayer(state.players[2].ID); err != nil {
			t.Fatalf("ForfeitPlayer() error = %v", err)
		}
		if state.players[2].statistics.ForfeitOrder != 2 {
			t.Fatalf("ForfeitOrder = %d; want 2", state.players[2].statistics.ForfeitOrder)
		}
	})
}

func TestValidationRemainingBranches(t *testing.T) {
	if hasOpeningCompositionFromHand(nil, []openingPlayCandidate{{newComposition: false}}) {
		t.Fatal("non-new candidate satisfied opening")
	}
	if validateActiveOrder([]int{0}, []int{0, 1}, 2) {
		t.Fatal("short active order validated")
	}
	if validateActiveOrder([]int{0}, []int{-1}, 2) {
		t.Fatal("invalid active index validated")
	}
	if !validateOrder([]int{1, 0}, 2) {
		t.Fatal("valid complete order rejected")
	}
}

func TestCompletedPlayerStatisticsRemainingBranches(t *testing.T) {
	if (*GameState)(nil).CompletedPlayerStatistics() != nil {
		t.Fatal("nil state returned completed statistics")
	}
	players := []*Player{
		{ID: "winner", hand: NewHand()},
		nil,
		{ID: "tie-a", hand: NewHand(), totalPoints: 20},
		{ID: "tie-b", hand: NewHand(), totalPoints: 20},
		{ID: "forfeit-a", hand: NewHand(), forfeited: true, statistics: PlayerGameStatistics{ForfeitOrder: 1}},
		{ID: "forfeit-b", hand: NewHand(), forfeited: true, statistics: PlayerGameStatistics{ForfeitOrder: 2}},
	}
	state := &GameState{players: players, phase: PhaseGameOver, roundWinnerIndex: 0}
	results := state.CompletedPlayerStatistics()
	if len(results) != 5 || results[1].Placement != 2 || results[2].Placement != 2 || results[3].PlayerID != "forfeit-a" || results[4].PlayerID != "forfeit-b" {
		t.Fatalf("CompletedPlayerStatistics() = %+v", results)
	}
}
