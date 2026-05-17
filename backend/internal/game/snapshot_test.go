package game

import "testing"

func TestCardConstructorsAccessorsAndSnapshots(t *testing.T) {
	queen := NewCard(Queen, Spades)
	if queen.Rank() != Queen {
		t.Fatalf("queen.Rank() = %v; want %v", queen.Rank(), Queen)
	}
	if queen.Suit() != Spades {
		t.Fatalf("queen.Suit() = %v; want %v", queen.Suit(), Spades)
	}
	if queen.IsJoker() {
		t.Fatal("queen.IsJoker() = true; want false")
	}
	queenSnapshot := queen.Snapshot()
	if queenSnapshot.Rank != Queen || queenSnapshot.Suit != Spades || queenSnapshot.IsJoker {
		t.Fatalf("queen.Snapshot() = %#v; want queen of spades", queenSnapshot)
	}

	jokerCard := NewJoker()
	if !jokerCard.IsJoker() {
		t.Fatal("jokerCard.IsJoker() = false; want true")
	}
	jokerSnapshot := jokerCard.Snapshot()
	if !jokerSnapshot.IsJoker || jokerSnapshot.Rank != 0 || jokerSnapshot.Suit != 0 {
		t.Fatalf("jokerCard.Snapshot() = %#v; want joker snapshot", jokerSnapshot)
	}
}

func TestNewGameStateWithDeckCopiesRequestedDeck(t *testing.T) {
	cards := []Card{NewCard(Ace, Hearts), NewCard(Two, Clubs)}
	state := NewGameStateWithDeck(cards)
	cards[0] = NewCard(King, Spades)

	if len(state.drawPile.cards) != 2 {
		t.Fatalf("len(state.drawPile.cards) = %d; want 2", len(state.drawPile.cards))
	}
	if !cardsEqual(state.drawPile.cards[0], NewCard(Ace, Hearts)) {
		t.Fatalf("state.drawPile.cards[0] = %#v; want ace of hearts", state.drawPile.cards[0])
	}
}

func TestGameStateSnapshotForPlayer(t *testing.T) {
	state := NewGameStateWithDeck([]Card{NewCard(Ace, Hearts)})
	first := NewPlayer()
	second := NewPlayer()
	first.ID = "first"
	second.ID = "second"
	first.hand.cards = []Card{NewCard(King, Hearts), NewJoker()}
	first.totalPoints = 12
	first.hasOpened = true
	second.hand.cards = []Card{NewCard(Two, Clubs)}
	state.players = []*Player{first, nil, second}
	state.phase = PhaseInProgress
	state.round = 3
	state.dealerIndex = 1
	state.roundWinnerIndex = -1
	state.turn = Turn{number: 7, playerIndex: 2, hasDrawn: true, mustUseDiscardDraw: true}
	state.drawPile = &CardPile{cards: []Card{NewCard(Ace, Spades), NewCard(Two, Spades)}}
	state.discardPile = &CardPile{cards: []Card{NewCard(Three, Diamonds)}}
	setComp := mustSet(t, NewCard(Ten, Hearts), NewCard(Ten, Diamonds), NewJoker())
	runComp := mustRun(t, NewCard(Five, Clubs), NewCard(Six, Clubs), NewCard(Seven, Clubs), NewCard(Eight, Clubs), NewCard(Nine, Clubs), NewCard(Ten, Clubs), NewCard(Jack, Clubs), NewCard(Queen, Clubs), NewCard(King, Clubs), NewCard(Ace, Clubs), NewCard(Ace, Clubs), NewCard(Two, Clubs), NewCard(Three, Clubs), NewCard(Four, Clubs))
	state.activeCompositions = []*Composition{nil, setComp, runComp}

	snapshot, ok := state.SnapshotForPlayer("first")
	if !ok {
		t.Fatal("SnapshotForPlayer(first) ok = false; want true")
	}
	if snapshot.Phase != PhaseInProgress || snapshot.Round != 3 || snapshot.DealerIndex != 1 || snapshot.RoundWinnerIndex != -1 {
		t.Fatalf("snapshot header = %#v; want in-progress round 3", snapshot)
	}
	if snapshot.Turn.Number != 7 || snapshot.Turn.PlayerIndex != 2 || snapshot.Turn.PlayerID != "second" || !snapshot.Turn.HasDrawn || !snapshot.Turn.MustUseDiscardDraw {
		t.Fatalf("snapshot.Turn = %#v; want second player's drawn turn", snapshot.Turn)
	}
	if len(snapshot.Players) != 2 {
		t.Fatalf("len(snapshot.Players) = %d; want 2", len(snapshot.Players))
	}
	if snapshot.Players[0].PlayerID != "first" || snapshot.Players[0].HandCount != 2 || snapshot.Players[0].TotalPoints != 12 || !snapshot.Players[0].HasOpened {
		t.Fatalf("snapshot.Players[0] = %#v; want first player state", snapshot.Players[0])
	}
	if len(snapshot.Hand) != 2 || snapshot.Hand[0].Rank != King || !snapshot.Hand[1].IsJoker {
		t.Fatalf("snapshot.Hand = %#v; want first player's hand", snapshot.Hand)
	}
	if snapshot.DrawPileCount != 2 {
		t.Fatalf("snapshot.DrawPileCount = %d; want 2", snapshot.DrawPileCount)
	}
	if len(snapshot.DiscardPile) != 1 || snapshot.DiscardPile[0].Rank != Three {
		t.Fatalf("snapshot.DiscardPile = %#v; want three of diamonds", snapshot.DiscardPile)
	}
	if len(snapshot.ActiveCompositions) != 2 {
		t.Fatalf("len(snapshot.ActiveCompositions) = %d; want 2", len(snapshot.ActiveCompositions))
	}
	if snapshot.ActiveCompositions[0].Type != "set" || len(snapshot.ActiveCompositions[0].JokerRepresentations) != 1 || snapshot.ActiveCompositions[0].Points != 30 || snapshot.ActiveCompositions[0].Complete {
		t.Fatalf("set snapshot = %#v; want incomplete ten set with joker representation", snapshot.ActiveCompositions[0])
	}
	if snapshot.ActiveCompositions[1].Type != "run" || !snapshot.ActiveCompositions[1].Complete {
		t.Fatalf("run snapshot = %#v; want complete run", snapshot.ActiveCompositions[1])
	}
}

func TestGameStateSnapshotForPlayerFailuresAndInvalidTurn(t *testing.T) {
	if snapshot, ok := (*GameState)(nil).SnapshotForPlayer("missing"); ok || snapshot.Round != 0 {
		t.Fatalf("nil SnapshotForPlayer() = %#v, %v; want zero false", snapshot, ok)
	}
	if got := (*GameState)(nil).CurrentPlayerIndex(); got != 0 {
		t.Fatalf("nil CurrentPlayerIndex() = %d; want 0", got)
	}

	state := NewGameState()
	player := NewPlayer()
	player.ID = "only"
	state.players = []*Player{player}
	state.turn.playerIndex = 9

	if got := state.CurrentPlayerIndex(); got != 9 {
		t.Fatalf("CurrentPlayerIndex() = %d; want 9", got)
	}
	if snapshot, ok := state.SnapshotForPlayer("missing"); ok || snapshot.Round != 0 {
		t.Fatalf("missing SnapshotForPlayer() = %#v, %v; want zero false", snapshot, ok)
	}
	snapshot, ok := state.SnapshotForPlayer("only")
	if !ok {
		t.Fatal("SnapshotForPlayer(only) ok = false; want true")
	}
	if snapshot.Turn.PlayerID != "" {
		t.Fatalf("snapshot.Turn.PlayerID = %q; want empty for invalid turn index", snapshot.Turn.PlayerID)
	}
}
