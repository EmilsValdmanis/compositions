package game

import "testing"

func TestGameStatisticsTrackTablePlayAndOpening(t *testing.T) {
	state := newTurnTestState()
	player := state.players[0]
	player.statistics.RoundsPlayed = 1
	state.players[1].statistics.RoundsPlayed = 1
	state.turn.hasDrawn = true
	player.hand.cards = []Card{
		card(Ten, Hearts), card(Jack, Hearts), card(Queen, Hearts), NewJoker(), card(Two, Clubs),
	}
	run, ok := NewRun([]Card{card(Ten, Hearts), card(Jack, Hearts), card(Queen, Hearts), NewJoker()})
	if !ok {
		t.Fatal("NewRun() returned false")
	}
	if err := state.PlayTable([]*Composition{run}, nil); err != nil {
		t.Fatalf("PlayTable() error = %v", err)
	}
	if err := state.DiscardFromHand(0); err != nil {
		t.Fatalf("DiscardFromHand() error = %v", err)
	}

	stats := player.statistics
	if stats.CompositionsCreated != 1 || stats.RunsCreated != 1 || stats.SetsCreated != 0 {
		t.Fatalf("composition statistics = %+v", stats)
	}
	if stats.CardsPlayed != 4 || stats.JokersPlayed != 1 || stats.RoundsOpened != 1 || stats.FastestOpeningTurn != 1 {
		t.Fatalf("play/opening statistics = %+v", stats)
	}
	if stats.TurnsTaken != 1 || stats.CardsDiscarded != 1 || stats.RoundsWon != 1 {
		t.Fatalf("turn/result statistics = %+v", stats)
	}
}

func TestGameStatisticsTrackCompletedCompositionScoringAndResult(t *testing.T) {
	state := newTurnTestState()
	winner, loser := state.players[0], state.players[1]
	winner.statistics.RoundsPlayed = 1
	loser.statistics.RoundsPlayed = 1
	winner.hasOpened = true
	winner.hand.cards = []Card{card(Seven, Spades), card(Ace, Clubs)}
	loser.hand.cards = []Card{card(King, Hearts), card(Five, Clubs)}
	loser.totalPoints = 90
	state.turn.hasDrawn = true
	setComp, ok := NewSet([]Card{card(Seven, Hearts), card(Seven, Diamonds), card(Seven, Clubs)})
	if !ok {
		t.Fatal("NewSet() returned false")
	}
	state.activeCompositions = []*Composition{setComp}

	if err := state.AddToCompositions([]CompositionAddition{{CompositionIndex: 0, Cards: []Card{card(Seven, Spades)}}}); err != nil {
		t.Fatalf("AddToCompositions() error = %v", err)
	}
	if err := state.DiscardFromHand(0); err != nil {
		t.Fatalf("DiscardFromHand() error = %v", err)
	}
	if state.phase != PhaseGameOver {
		t.Fatalf("phase = %v; want game over", state.phase)
	}

	if winner.statistics.AdditionsDone != 1 || winner.statistics.CompositionsCompleted != 1 || winner.statistics.SetsCompleted != 1 {
		t.Fatalf("completion statistics = %+v", winner.statistics)
	}
	if winner.statistics.PointsInflicted != 15 || winner.statistics.LargestRoundPointsInflicted != 15 {
		t.Fatalf("winner scoring statistics = %+v", winner.statistics)
	}
	if loser.statistics.PenaltyPoints != 15 || loser.statistics.HandPoints != 15 || loser.statistics.CardsRemaining != 2 {
		t.Fatalf("loser scoring statistics = %+v", loser.statistics)
	}
	results := state.CompletedPlayerStatistics()
	if len(results) != 2 || !results[0].Winner || results[0].Placement != 1 || results[1].Placement != 2 {
		t.Fatalf("completed statistics = %+v", results)
	}
}

func TestGameStatisticsTrackDrawSourcesAndForfeitResult(t *testing.T) {
	state := newTurnTestState()
	state.players[0].statistics.RoundsPlayed = 1
	state.players[1].statistics.RoundsPlayed = 1
	state.drawPile.cards = []Card{card(Two, Hearts)}
	if err := state.DrawFromDeck(); err != nil {
		t.Fatalf("DrawFromDeck() error = %v", err)
	}
	if state.players[0].statistics.CardsDrawnFromDeck != 1 {
		t.Fatal("deck draw was not counted")
	}

	state.turn = Turn{number: 1, playerIndex: 0}
	state.discardPile.cards = []Card{card(Three, Clubs)}
	state.players[0].hasOpened = true
	state.activeCompositions = []*Composition{mustSet(t, card(Three, Hearts), card(Three, Diamonds), card(Three, Spades))}
	if err := state.DrawFromDiscard(); err != nil {
		t.Fatalf("DrawFromDiscard() error = %v", err)
	}
	if state.players[0].statistics.CardsDrawnFromDiscard != 1 {
		t.Fatal("discard draw was not counted")
	}

	state.turn.mustUseDiscardDraw = false
	if _, err := state.ForfeitPlayer(state.players[0].ID); err != nil {
		t.Fatalf("ForfeitPlayer() error = %v", err)
	}
	results := state.CompletedPlayerStatistics()
	if len(results) != 2 || !results[0].Forfeited || results[0].Placement != 2 || !results[1].Winner {
		t.Fatalf("forfeit results = %+v", results)
	}
}
