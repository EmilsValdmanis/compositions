package game

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestGameStatePersistenceSnapshotRoundTrips(t *testing.T) {
	state := NewGameStateWithDeck([]Card{NewCard(Ace, Hearts)})
	first := NewPlayer()
	first.ID = "first"
	first.hand.cards = []Card{NewCard(Three, Clubs), NewCard(Four, Clubs)}
	first.totalPoints = 10
	first.pointsGained = 5
	first.hasOpened = true
	second := NewPlayer()
	second.ID = "second"
	second.hand.cards = []Card{NewCard(Five, Diamonds)}

	setComp := mustSet(t, NewCard(King, Hearts), NewCard(King, Diamonds), NewJoker())
	state.players = []*Player{first, second}
	state.activeCompositions = []*Composition{setComp}
	state.drawPile = &CardPile{cards: []Card{NewCard(Six, Spades), NewCard(Seven, Spades)}}
	state.discardPile = &CardPile{cards: []Card{NewCard(Eight, Hearts)}}
	state.maxPlayers = 4
	state.phase = PhaseInProgress
	state.round = 3
	state.dealerIndex = 1
	state.turn = Turn{
		number:             8,
		playerIndex:        0,
		hasDrawn:           true,
		mustUseDiscardDraw: true,
		discardDrawCard:    NewCard(Eight, Hearts),
	}
	state.roundWinnerIndex = -1

	beforeFirst, ok := state.SnapshotForPlayer("first")
	if !ok {
		t.Fatal("SnapshotForPlayer(first) ok = false; want true")
	}
	beforeSecond, ok := state.SnapshotForPlayer("second")
	if !ok {
		t.Fatal("SnapshotForPlayer(second) ok = false; want true")
	}

	data, err := json.Marshal(state.PersistenceSnapshot())
	if err != nil {
		t.Fatalf("Marshal(PersistenceSnapshot) error = %v", err)
	}
	var persisted PersistenceSnapshot
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatalf("Unmarshal(PersistenceSnapshot) error = %v", err)
	}
	restored, err := RestoreGameState(persisted)
	if err != nil {
		t.Fatalf("RestoreGameState() error = %v", err)
	}

	afterFirst, ok := restored.SnapshotForPlayer("first")
	if !ok {
		t.Fatal("restored SnapshotForPlayer(first) ok = false; want true")
	}
	afterSecond, ok := restored.SnapshotForPlayer("second")
	if !ok {
		t.Fatal("restored SnapshotForPlayer(second) ok = false; want true")
	}

	if !reflect.DeepEqual(afterFirst, beforeFirst) {
		t.Fatalf("restored first snapshot = %#v; want %#v", afterFirst, beforeFirst)
	}
	if !reflect.DeepEqual(afterSecond, beforeSecond) {
		t.Fatalf("restored second snapshot = %#v; want %#v", afterSecond, beforeSecond)
	}
	if !cardsEqual(restored.turn.discardDrawCard, state.turn.discardDrawCard) {
		t.Fatalf("restored discardDrawCard = %#v; want %#v", restored.turn.discardDrawCard, state.turn.discardDrawCard)
	}
}

func TestRestoreGameStateRejectsInvalidSnapshot(t *testing.T) {
	if _, err := RestoreGameState(PersistenceSnapshot{}); err == nil {
		t.Fatal("RestoreGameState(empty) error = nil; want error")
	}

	_, err := RestoreGameState(PersistenceSnapshot{
		Version:    PersistenceSnapshotVersion,
		MaxPlayers: 4,
		Phase:      PhaseInProgress,
		Round:      1,
		Turn:       PersistenceTurnSnapshot{Number: 1},
	})
	if err == nil {
		t.Fatal("RestoreGameState(non-lobby without players) error = nil; want error")
	}
}
