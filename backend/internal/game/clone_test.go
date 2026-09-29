package game

import (
	"reflect"
	"testing"
)

func TestCloneIsolatesAllMutableGameData(t *testing.T) {
	if (*GameState)(nil).Clone() != nil {
		t.Fatal("nil clone should be nil")
	}
	state := NewGameState()
	player := NewPlayer()
	player.hand.cards = []Card{NewCard(Ace, Hearts)}
	state.players = []*Player{player, nil, {ID: "no-hand"}}
	state.activeCompositions = []*Composition{nil, {
		variant: set, cards: []Card{NewJoker()},
		jokerRepresentations: map[int][]Card{0: {NewCard(Ace, Clubs)}},
	}}
	state.discardPile.cards = []Card{NewCard(King, Clubs)}
	before := state.PersistenceSnapshot()
	copy := state.Clone()
	if !reflect.DeepEqual(state, copy) {
		t.Fatal("clone changed game data")
	}
	copy.players[0].hand.cards[0] = NewJoker()
	copy.players[0].statistics.TurnsTaken++
	copy.activeCompositions[1].cards[0] = NewCard(Two, Hearts)
	copy.activeCompositions[1].jokerRepresentations[0][0] = NewJoker()
	copy.drawPile.cards[0] = NewJoker()
	copy.discardPile.cards[0] = NewJoker()
	if !reflect.DeepEqual(before, state.PersistenceSnapshot()) || state.activeCompositions[1].jokerRepresentations[0][0].IsJoker() {
		t.Fatal("clone shared mutable data with original")
	}
	state.drawPile, state.discardPile = nil, nil
	copy = state.Clone()
	if copy.drawPile != nil || copy.discardPile != nil {
		t.Fatal("clone changed absent piles")
	}
}
