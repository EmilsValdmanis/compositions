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
	first.unadjustedTotalPoints = 15
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

func TestGameStatePersistenceSnapshotHandlesNilStateAndMembers(t *testing.T) {
	if got := (*GameState)(nil).PersistenceSnapshot(); !reflect.DeepEqual(got, PersistenceSnapshot{}) {
		t.Fatalf("nil PersistenceSnapshot() = %#v; want empty snapshot", got)
	}

	state := NewGameStateWithDeck(nil)
	state.players = []*Player{nil, {ID: "active", hand: nil}}
	state.activeCompositions = []*Composition{nil, mustSet(t, NewCard(Ace, Hearts), NewCard(Ace, Diamonds), NewCard(Ace, Clubs))}
	state.drawPile = nil
	state.discardPile = nil

	got := state.PersistenceSnapshot()
	if len(got.Players) != 1 || got.Players[0].ID != "active" {
		t.Fatalf("PersistenceSnapshot().Players = %#v; want one active player", got.Players)
	}
	if got.Players[0].Hand == nil {
		t.Fatal("PersistenceSnapshot().Players[0].Hand = nil; want empty slice")
	}
	if len(got.ActiveCompositions) != 1 {
		t.Fatalf("len(PersistenceSnapshot().ActiveCompositions) = %d; want 1", len(got.ActiveCompositions))
	}
	if got.DrawPile == nil || got.DiscardPile == nil {
		t.Fatal("PersistenceSnapshot() piles are nil; want empty slices")
	}
}

func TestRestoreGameStateRejectsInvalidFields(t *testing.T) {
	validCard := CardSnapshot{Rank: Ace, Suit: Hearts}
	validSet := PersistenceCompositionSnapshot{
		Variant: string(set),
		Cards: []CardSnapshot{
			{Rank: Ace, Suit: Hearts},
			{Rank: Ace, Suit: Diamonds},
			{Rank: Ace, Suit: Clubs},
		},
	}
	validPlayer := PersistencePlayerSnapshot{ID: "player-1"}
	validSnapshot := func() PersistenceSnapshot {
		return PersistenceSnapshot{
			Version:          PersistenceSnapshotVersion,
			MaxPlayers:       4,
			Phase:            PhaseLobby,
			Round:            1,
			Players:          []PersistencePlayerSnapshot{validPlayer},
			DrawPile:         []CardSnapshot{validCard},
			DiscardPile:      []CardSnapshot{validCard},
			DealerIndex:      0,
			RoundWinnerIndex: -1,
			Turn:             PersistenceTurnSnapshot{Number: 1, PlayerIndex: 0},
		}
	}

	tests := []struct {
		name   string
		mutate func(*PersistenceSnapshot)
	}{
		{"max players", func(snapshot *PersistenceSnapshot) { snapshot.MaxPlayers = 0 }},
		{"round", func(snapshot *PersistenceSnapshot) { snapshot.Round = 0 }},
		{"phase", func(snapshot *PersistenceSnapshot) { snapshot.Phase = GamePhase(99) }},
		{"player id", func(snapshot *PersistenceSnapshot) { snapshot.Players[0].ID = "" }},
		{"duplicate player id", func(snapshot *PersistenceSnapshot) {
			snapshot.Players = append(snapshot.Players, validPlayer)
		}},
		{"player hand card", func(snapshot *PersistenceSnapshot) {
			snapshot.Players[0].Hand = []CardSnapshot{{Rank: Rank(99), Suit: Hearts}}
		}},
		{"too many players", func(snapshot *PersistenceSnapshot) {
			snapshot.MaxPlayers = 1
			snapshot.Players = []PersistencePlayerSnapshot{{ID: "player-1"}, {ID: "player-2"}}
		}},
		{"composition card", func(snapshot *PersistenceSnapshot) {
			snapshot.ActiveCompositions = []PersistenceCompositionSnapshot{{
				Variant: string(set),
				Cards:   []CardSnapshot{{Rank: Rank(99), Suit: Hearts}},
			}}
		}},
		{"composition validity", func(snapshot *PersistenceSnapshot) {
			snapshot.ActiveCompositions = []PersistenceCompositionSnapshot{{
				Variant: string(set),
				Cards:   []CardSnapshot{{Rank: Ace, Suit: Hearts}, {Rank: Two, Suit: Hearts}, {Rank: Three, Suit: Hearts}},
			}}
		}},
		{"draw pile card", func(snapshot *PersistenceSnapshot) {
			snapshot.DrawPile = []CardSnapshot{{Rank: Rank(99), Suit: Hearts}}
		}},
		{"discard pile card", func(snapshot *PersistenceSnapshot) {
			snapshot.DiscardPile = []CardSnapshot{{Rank: Ace, Suit: Suit(99)}}
		}},
		{"turn number", func(snapshot *PersistenceSnapshot) { snapshot.Turn.Number = 0 }},
		{"turn player index", func(snapshot *PersistenceSnapshot) { snapshot.Turn.PlayerIndex = 9 }},
		{"missing discard draw card", func(snapshot *PersistenceSnapshot) {
			snapshot.Turn.MustUseDiscardDraw = true
		}},
		{"discard draw card", func(snapshot *PersistenceSnapshot) {
			snapshot.Turn.MustUseDiscardDraw = true
			snapshot.Turn.DiscardDrawCard = &CardSnapshot{Rank: Rank(99), Suit: Hearts}
		}},
		{"dealer index", func(snapshot *PersistenceSnapshot) { snapshot.DealerIndex = 9 }},
		{"round winner index", func(snapshot *PersistenceSnapshot) { snapshot.RoundWinnerIndex = 9 }},
		{"valid composition control", func(snapshot *PersistenceSnapshot) {
			snapshot.ActiveCompositions = []PersistenceCompositionSnapshot{validSet}
			snapshot.RoundWinnerIndex = 0
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			snapshot := validSnapshot()
			test.mutate(&snapshot)
			_, err := RestoreGameState(snapshot)
			if test.name == "valid composition control" {
				if err != nil {
					t.Fatalf("RestoreGameState() error = %v; want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatal("RestoreGameState() error = nil; want error")
			}
		})
	}
}

func TestRestoreCardAcceptsJokerAndCloneCardsCopies(t *testing.T) {
	card, err := restoreCard(CardSnapshot{IsJoker: true})
	if err != nil {
		t.Fatalf("restoreCard(joker) error = %v", err)
	}
	if !card.IsJoker() {
		t.Fatalf("restoreCard(joker) = %#v; want joker", card)
	}

	source := []Card{NewCard(Ace, Hearts)}
	cloned := cloneCards(source)
	cloned[0] = NewCard(Two, Clubs)
	if cardsEqual(source[0], cloned[0]) {
		t.Fatal("cloneCards() shared backing array with source")
	}
}
