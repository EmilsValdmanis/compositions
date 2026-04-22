package game

import (
	"errors"
	"testing"
)

func TestNewGameStateDefaults(t *testing.T) {
	state := NewGameState()

	if state.round != 1 {
		t.Errorf("state.round = %d; want 1", state.round)
	}
	if state.turn.number != 1 {
		t.Errorf("state.turn.number = %d; want 1", state.turn.number)
	}
	if state.turn.playerIndex != 0 {
		t.Errorf("state.turn.playerIndex = %d; want 0", state.turn.playerIndex)
	}
	if state.phase != PhaseLobby {
		t.Errorf("state.phase = %d; want %d", state.phase, PhaseLobby)
	}
	if state.maxPlayers != 4 {
		t.Errorf("state.maxPlayers = %d; want 4", state.maxPlayers)
	}
	if len(state.drawPile.cards) != 108 {
		t.Errorf("len(state.drawPile.cards) = %d; want 108", len(state.drawPile.cards))
	}
	if len(state.discardPile.cards) != 0 {
		t.Errorf("len(state.discardPile.cards) = %d; want 0", len(state.discardPile.cards))
	}
}

func TestGameStateAddPlayerRejectsNilPlayer(t *testing.T) {
	state := NewGameState()

	err := state.AddPlayer(nil)

	if !errors.Is(err, ErrNilPlayer) {
		t.Errorf("AddPlayer(nil) error = %v; want %v", err, ErrNilPlayer)
	}
}

func TestGameStateStartGameDealsHandsAndCreatesDiscardPile(t *testing.T) {
	state := NewGameState()
	first := NewPlayer()
	second := NewPlayer()

	if err := state.AddPlayer(first); err != nil {
		t.Errorf("AddPlayer(first) error = %v", err)
	}
	if err := state.AddPlayer(second); err != nil {
		t.Errorf("AddPlayer(second) error = %v", err)
	}

	err := state.StartGame()

	if err != nil {
		t.Errorf("StartGame() error = %v", err)
	}
	if state.phase != PhaseInProgress {
		t.Errorf("state.phase = %d; want %d", state.phase, PhaseInProgress)
	}
	if state.turn.number != 1 {
		t.Errorf("state.turn.number = %d; want 1", state.turn.number)
	}
	currentPlayer, err := state.CurrentPlayer()
	if err != nil {
		t.Errorf("CurrentPlayer() error = %v", err)
	}
	if currentPlayer == nil {
		t.Error("CurrentPlayer() returned nil")
	}
	if len(first.hand.cards) != InitialHandSize {
		t.Errorf("len(first.hand.cards) = %d; want %d", len(first.hand.cards), InitialHandSize)
	}
	if len(second.hand.cards) != InitialHandSize {
		t.Errorf("len(second.hand.cards) = %d; want %d", len(second.hand.cards), InitialHandSize)
	}
	if len(state.discardPile.cards) != 1 {
		t.Errorf("len(state.discardPile.cards) = %d; want 1", len(state.discardPile.cards))
	}
	if len(state.drawPile.cards) != 108-(2*InitialHandSize)-1 {
		t.Errorf("len(state.drawPile.cards) = %d; want %d", len(state.drawPile.cards), 108-(2*InitialHandSize)-1)
	}
}

func TestGameStateSelectFirstPlayerRequiresPlayers(t *testing.T) {
	state := NewGameState()

	err := state.SelectFirstPlayer()

	if !errors.Is(err, ErrNoPlayers) {
		t.Errorf("SelectFirstPlayer() error = %v; want %v", err, ErrNoPlayers)
	}
}

func TestGameStateCurrentPlayerRequiresPlayers(t *testing.T) {
	state := NewGameState()

	_, err := state.CurrentPlayer()

	if !errors.Is(err, ErrNoPlayers) {
		t.Errorf("CurrentPlayer() error = %v; want %v", err, ErrNoPlayers)
	}
}
