package game

import "testing"

func card(rank Rank, suit Suit) Card {
	return Card{rank: rank, suit: suit}
}

func joker() Card {
	return Card{isJoker: true}
}

func expectJokerRepresentation(t *testing.T, comp *Composition, cardIndex int, want Card) {
	t.Helper()

	got, ok := comp.JokerRepresentation(cardIndex)
	if !ok {
		t.Fatalf("JokerRepresentation(%d) returned ok = false; want true", cardIndex)
	}
	if !cardsEqual(got, want) {
		t.Fatalf("JokerRepresentation(%d) = %+v; want %+v", cardIndex, got, want)
	}
}

func expectJokerRepresentations(t *testing.T, comp *Composition, cardIndex int, want []Card) {
	t.Helper()

	got, ok := comp.JokerRepresentations(cardIndex)
	if !ok {
		t.Fatalf("JokerRepresentations(%d) returned ok = false; want true", cardIndex)
	}
	if len(got) != len(want) {
		t.Fatalf("len(JokerRepresentations(%d)) = %d; want %d", cardIndex, len(got), len(want))
	}

	for i := range want {
		if !cardsEqual(got[i], want[i]) {
			t.Fatalf("JokerRepresentations(%d)[%d] = %+v; want %+v", cardIndex, i, got[i], want[i])
		}
	}
}

func TestNewSet_ValidThreeOfAKind(t *testing.T) {
	cards := []Card{
		card(Seven, Hearts),
		card(Seven, Diamonds),
		card(Seven, Clubs),
	}
	_, ok := NewSet(cards)
	if !ok {
		t.Error("expected valid set of three 7s with different suits")
	}
}

func TestNewSet_ValidFourOfAKind(t *testing.T) {
	cards := []Card{
		card(King, Hearts),
		card(King, Diamonds),
		card(King, Clubs),
		card(King, Spades),
	}
	_, ok := NewSet(cards)
	if !ok {
		t.Error("expected valid set of four Kings")
	}
}

func TestNewSet_ValidWithOneJoker(t *testing.T) {
	cards := []Card{
		card(Ten, Hearts),
		card(Ten, Diamonds),
		joker(),
	}
	_, ok := NewSet(cards)
	if !ok {
		t.Error("expected valid set with one joker filling missing suit")
	}
}

func TestNewSet_ValidWithTwoJokers(t *testing.T) {
	cards := []Card{
		card(Five, Spades),
		joker(),
		joker(),
	}
	_, ok := NewSet(cards)
	if !ok {
		t.Error("expected valid set with two jokers")
	}
}

func TestNewSet_ValidAllJokers(t *testing.T) {
	cards := []Card{joker(), joker(), joker()}
	_, ok := NewSet(cards)
	if !ok {
		t.Error("expected valid set of three jokers")
	}
}

func TestNewSet_ValidAllJokersFour(t *testing.T) {
	cards := []Card{joker(), joker(), joker(), joker()}
	_, ok := NewSet(cards)
	if !ok {
		t.Error("expected valid set of four jokers")
	}
}

func TestNewSet_AssignsExactJokerRepresentationWhenOnlyOneSuitIsMissing(t *testing.T) {
	comp, ok := NewSet([]Card{
		card(Ten, Hearts),
		card(Ten, Diamonds),
		card(Ten, Clubs),
		joker(),
	})
	if !ok {
		t.Fatal("NewSet() returned false; want true")
	}

	expectJokerRepresentation(t, comp, 3, card(Ten, Spades))
}

func TestNewSet_TracksAmbiguousJokerRepresentationWhenMultipleSuitsArePossible(t *testing.T) {
	comp, ok := NewSet([]Card{
		card(Ten, Hearts),
		card(Ten, Diamonds),
		joker(),
	})
	if !ok {
		t.Fatal("NewSet() returned false; want true")
	}

	if _, narrowed := comp.JokerRepresentation(2); narrowed {
		t.Fatal("JokerRepresentation(2) returned ok = true; want false for ambiguous set joker")
	}

	expectJokerRepresentations(t, comp, 2, []Card{
		card(Ten, Clubs),
		card(Ten, Spades),
	})
}

func TestNewSet_TracksAmbiguousRepresentationsForMultipleJokers(t *testing.T) {
	comp, ok := NewSet([]Card{
		card(Ace, Hearts),
		card(Ace, Diamonds),
		joker(),
		joker(),
	})
	if !ok {
		t.Fatal("NewSet() returned false; want true")
	}

	if _, narrowed := comp.JokerRepresentation(2); narrowed {
		t.Fatal("JokerRepresentation(2) returned ok = true; want false for ambiguous set joker")
	}
	if _, narrowed := comp.JokerRepresentation(3); narrowed {
		t.Fatal("JokerRepresentation(3) returned ok = true; want false for ambiguous set joker")
	}

	want := []Card{
		card(Ace, Clubs),
		card(Ace, Spades),
	}
	expectJokerRepresentations(t, comp, 2, want)
	expectJokerRepresentations(t, comp, 3, want)
}

func TestCompositionReclaimJokerReplacesExactRepresentedRunCard(t *testing.T) {
	comp, ok := NewRun([]Card{
		card(Five, Hearts),
		joker(),
		card(Seven, Hearts),
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}

	updated, ok := comp.ReclaimJoker(1, card(Six, Hearts))
	if !ok {
		t.Fatal("ReclaimJoker() returned false; want true")
	}
	if !comp.cards[1].isJoker {
		t.Fatal("original composition mutated; joker was removed")
	}
	if updated.cards[1].isJoker {
		t.Fatal("updated composition still has joker at reclaimed index")
	}
	if !cardsEqual(updated.cards[1], card(Six, Hearts)) {
		t.Fatalf("updated.cards[1] = %+v; want Six of Hearts", updated.cards[1])
	}
}

func TestCompositionReclaimJokerRejectsAmbiguousSetJoker(t *testing.T) {
	comp, ok := NewSet([]Card{
		card(Ten, Hearts),
		card(Ten, Diamonds),
		joker(),
	})
	if !ok {
		t.Fatal("NewSet() returned false; want true")
	}

	if _, ok := comp.ReclaimJoker(2, card(Ten, Clubs)); ok {
		t.Fatal("ReclaimJoker() returned true; want false for ambiguous set joker")
	}
}

func TestNewSet_InvalidTwoCards(t *testing.T) {
	cards := []Card{
		card(Nine, Hearts),
		card(Nine, Diamonds),
	}
	_, ok := NewSet(cards)
	if ok {
		t.Error("expected invalid: set needs at least 3 cards")
	}
}

func TestNewSet_InvalidFiveCards(t *testing.T) {
	cards := []Card{
		card(Three, Hearts),
		card(Three, Diamonds),
		card(Three, Clubs),
		card(Three, Spades),
		joker(),
	}
	_, ok := NewSet(cards)
	if ok {
		t.Error("expected invalid: set cannot have more than 4 cards")
	}
}

func TestNewSet_InvalidMixedRanks(t *testing.T) {
	cards := []Card{
		card(Seven, Hearts),
		card(Eight, Diamonds),
		card(Seven, Clubs),
	}
	_, ok := NewSet(cards)
	if ok {
		t.Error("expected invalid: all real cards must share the same rank")
	}
}

func TestNewSet_InvalidDuplicateSuit(t *testing.T) {
	cards := []Card{
		card(Jack, Hearts),
		card(Jack, Hearts),
		card(Jack, Clubs),
	}
	_, ok := NewSet(cards)
	if ok {
		t.Error("expected invalid: duplicate suits not allowed in a set")
	}
}

func TestNewSet_InvalidDuplicateSuitWithJoker(t *testing.T) {
	cards := []Card{
		card(Queen, Spades),
		card(Queen, Spades),
		joker(),
	}
	_, ok := NewSet(cards)
	if ok {
		t.Error("expected invalid: joker cannot fix duplicate suits from two decks")
	}
}

func TestNewRun_ValidSimpleSequence(t *testing.T) {
	cards := []Card{
		card(Five, Hearts),
		card(Six, Hearts),
		card(Seven, Hearts),
	}
	_, ok := NewRun(cards)
	if !ok {
		t.Error("expected valid run 5-6-7 of Hearts")
	}
}

func TestNewRun_ValidLongerSequence(t *testing.T) {
	cards := []Card{
		card(Three, Spades),
		card(Four, Spades),
		card(Five, Spades),
		card(Six, Spades),
		card(Seven, Spades),
	}
	_, ok := NewRun(cards)
	if !ok {
		t.Error("expected valid run 3-4-5-6-7 of Spades")
	}
}

func TestNewRun_ValidAceLow(t *testing.T) {
	cards := []Card{
		card(Ace, Clubs),
		card(Two, Clubs),
		card(Three, Clubs),
	}
	_, ok := NewRun(cards)
	if !ok {
		t.Error("expected valid run Ace-2-3 of Clubs (ace low)")
	}
}

func TestNewRun_ValidAceHigh(t *testing.T) {
	cards := []Card{
		card(Queen, Diamonds),
		card(King, Diamonds),
		card(Ace, Diamonds),
	}
	_, ok := NewRun(cards)
	if !ok {
		t.Error("expected valid run Q-K-Ace of Diamonds (ace high)")
	}
}

func TestNewRun_ValidWithOneJoker(t *testing.T) {
	cards := []Card{
		card(Five, Hearts),
		joker(),
		card(Seven, Hearts),
	}
	_, ok := NewRun(cards)
	if !ok {
		t.Error("expected valid run 5-J-7 of Hearts with joker filling gap")
	}
}

func TestNewRun_ValidWithJokerExtendingEnd(t *testing.T) {
	cards := []Card{
		card(Five, Clubs),
		card(Six, Clubs),
		joker(),
	}
	_, ok := NewRun(cards)
	if !ok {
		t.Error("expected valid run with joker extending at end")
	}
}

func TestNewRun_AssignsJokerRepresentationForGap(t *testing.T) {
	comp, ok := NewRun([]Card{
		card(Five, Hearts),
		joker(),
		card(Seven, Hearts),
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}

	expectJokerRepresentation(t, comp, 1, card(Six, Hearts))
}

func TestNewRun_AssignsJokerRepresentationForAceLowRun(t *testing.T) {
	comp, ok := NewRun([]Card{
		card(Ace, Clubs),
		joker(),
		card(Three, Clubs),
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}

	expectJokerRepresentation(t, comp, 1, card(Two, Clubs))
}

func TestNewRun_AssignsJokerRepresentationForAceHighRun(t *testing.T) {
	comp, ok := NewRun([]Card{
		card(Queen, Diamonds),
		joker(),
		card(Ace, Diamonds),
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}

	expectJokerRepresentation(t, comp, 1, card(King, Diamonds))
}

func TestNewRun_AssignsJokerRepresentationForExtension(t *testing.T) {
	comp, ok := NewRun([]Card{
		card(Five, Clubs),
		card(Six, Clubs),
		joker(),
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}

	expectJokerRepresentation(t, comp, 2, card(Seven, Clubs))
}

func TestNewRun_AssignsRepresentationsForMultipleJokersInLongRun(t *testing.T) {
	comp, ok := NewRun([]Card{
		card(Four, Hearts),
		joker(),
		card(Six, Hearts),
		joker(),
		card(Eight, Hearts),
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}

	expectJokerRepresentation(t, comp, 1, card(Five, Hearts))
	expectJokerRepresentation(t, comp, 3, card(Seven, Hearts))
}

func TestNewRun_AssignsRepresentationsForLongFaceRunWithMultipleJokers(t *testing.T) {
	comp, ok := NewRun([]Card{
		card(Nine, Diamonds),
		card(Ten, Diamonds),
		joker(),
		card(Queen, Diamonds),
		joker(),
		card(Ace, Diamonds),
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}

	expectJokerRepresentation(t, comp, 2, card(Jack, Diamonds))
	expectJokerRepresentation(t, comp, 4, card(King, Diamonds))
}

func TestNewRun_ValidWithMultipleJokers(t *testing.T) {
	cards := []Card{
		card(Two, Spades),
		joker(),
		joker(),
		card(Five, Spades),
	}
	_, ok := NewRun(cards)
	if !ok {
		t.Error("expected valid run 2-J-J-5 of Spades with two jokers filling gap")
	}
}

func TestNewRun_ValidAllJokers(t *testing.T) {
	cards := []Card{joker(), joker(), joker()}
	_, ok := NewRun(cards)
	if !ok {
		t.Error("expected valid run of three jokers")
	}
}

func TestNewRun_InvalidTwoCards(t *testing.T) {
	cards := []Card{
		card(Four, Hearts),
		card(Five, Hearts),
	}
	_, ok := NewRun(cards)
	if ok {
		t.Error("expected invalid: run needs at least 3 cards")
	}
}

func TestNewRun_InvalidMixedSuits(t *testing.T) {
	cards := []Card{
		card(Four, Hearts),
		card(Five, Diamonds),
		card(Six, Hearts),
	}
	_, ok := NewRun(cards)
	if ok {
		t.Error("expected invalid: all real cards in a run must share the same suit")
	}
}

func TestNewRun_InvalidNonSequential(t *testing.T) {
	cards := []Card{
		card(Two, Clubs),
		card(Four, Clubs),
		card(Seven, Clubs),
	}
	_, ok := NewRun(cards)
	if ok {
		t.Error("expected invalid: gap too large, not enough jokers to fill")
	}
}

func TestNewRun_InvalidDuplicateRank(t *testing.T) {
	cards := []Card{
		card(Six, Hearts),
		card(Six, Hearts),
		card(Seven, Hearts),
	}
	_, ok := NewRun(cards)
	if ok {
		t.Error("expected invalid: duplicate rank in same suit run")
	}
}

func TestNewRun_InvalidNotEnoughJokersForGap(t *testing.T) {
	cards := []Card{
		card(Two, Diamonds),
		joker(),
		card(Six, Diamonds),
	}
	_, ok := NewRun(cards)
	if ok {
		t.Error("expected invalid: one joker cannot fill a gap of 3 (2 to 6)")
	}
}

func TestNewRun_InvalidAceMiddle(t *testing.T) {
	cards := []Card{
		card(King, Spades),
		card(Ace, Spades),
		card(Two, Spades),
	}
	_, ok := NewRun(cards)
	if ok {
		t.Error("expected invalid: Ace cannot wrap around (K-A-2 is not valid)")
	}
}

func TestNewRun_ValidFullSuitRun(t *testing.T) {
	cards := []Card{
		card(Ace, Hearts),
		card(Two, Hearts),
		card(Three, Hearts),
		card(Four, Hearts),
		card(Five, Hearts),
		card(Six, Hearts),
		card(Seven, Hearts),
		card(Eight, Hearts),
		card(Nine, Hearts),
		card(Ten, Hearts),
		card(Jack, Hearts),
		card(Queen, Hearts),
		card(King, Hearts),
		card(Ace, Hearts), // second Ace from second deck, acting as high
	}
	_, ok := NewRun(cards)
	if !ok {
		t.Error("expected valid full suit run with Ace on both ends")
	}
}

func TestNewRun_InvalidNaturalSuitRunWithTooManyJokers(t *testing.T) {
	cards := []Card{
		card(Ace, Hearts),
		card(Two, Hearts),
		card(Three, Hearts),
		card(Four, Hearts),
		card(Five, Hearts),
		card(Six, Hearts),
		card(Seven, Hearts),
		card(Eight, Hearts),
		card(Nine, Hearts),
		card(Ten, Hearts),
		card(Jack, Hearts),
		card(Queen, Hearts),
		card(King, Hearts),
		joker(),
		joker(),
	}

	_, ok := NewRun(cards)
	if ok {
		t.Error("expected invalid: only one joker can extend A-2-3-4-5-6-7-8-9-10-J-Q-K into a complete suit run")
	}
}

func TestNewRun_InvalidFullSuitRunPlusExtraCard(t *testing.T) {
	cards := []Card{
		card(Ace, Hearts),
		card(Two, Hearts),
		card(Three, Hearts),
		card(Four, Hearts),
		card(Five, Hearts),
		card(Six, Hearts),
		card(Seven, Hearts),
		card(Eight, Hearts),
		card(Nine, Hearts),
		card(Ten, Hearts),
		card(Jack, Hearts),
		card(Queen, Hearts),
		card(King, Hearts),
		card(Ace, Hearts),
		joker(),
	}

	_, ok := NewRun(cards)
	if ok {
		t.Error("expected invalid: nothing can be added to a complete Ace-low to Ace-high suit run")
	}
}

func TestCompositionPoints_SetUsesCompositionValueForJokers(t *testing.T) {
	comp, ok := NewSet([]Card{
		card(Ten, Hearts),
		card(Ten, Diamonds),
		joker(),
	})
	if !ok {
		t.Fatal("NewSet() returned false; want true")
	}

	if got := comp.Points(); got != 30 {
		t.Fatalf("Points() = %d; want 30", got)
	}
}

func TestCompositionPoints_SetWithMultipleJokersUsesSetValue(t *testing.T) {
	comp, ok := NewSet([]Card{
		card(Ace, Hearts),
		joker(),
		joker(),
	})
	if !ok {
		t.Fatal("NewSet() returned false; want true")
	}

	if got := comp.Points(); got != 30 {
		t.Fatalf("Points() = %d; want 30", got)
	}
}

func TestCompositionPoints_RunTreatsAceAsLow(t *testing.T) {
	comp, ok := NewRun([]Card{
		card(Ace, Clubs),
		card(Two, Clubs),
		card(Three, Clubs),
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}

	if got := comp.Points(); got != 6 {
		t.Fatalf("Points() = %d; want 6", got)
	}
}

func TestCompositionPoints_RunUsesRepresentedJokerValue(t *testing.T) {
	comp, ok := NewRun([]Card{
		card(Queen, Diamonds),
		joker(),
		card(Ace, Diamonds),
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}

	if got := comp.Points(); got != 30 {
		t.Fatalf("Points() = %d; want 30", got)
	}
}

func TestCompositionPoints_LongRunWithMultipleJokers(t *testing.T) {
	comp, ok := NewRun([]Card{
		card(Nine, Diamonds),
		card(Ten, Diamonds),
		joker(),
		card(Queen, Diamonds),
		joker(),
		card(Ace, Diamonds),
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}

	if got := comp.Points(); got != 59 {
		t.Fatalf("Points() = %d; want 59", got)
	}
}

func TestCompositionPoints_AceLowRunWithMultipleJokers(t *testing.T) {
	comp, ok := NewRun([]Card{
		card(Ace, Clubs),
		joker(),
		joker(),
		card(Four, Clubs),
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}

	if got := comp.Points(); got != 10 {
		t.Fatalf("Points() = %d; want 10", got)
	}
}

func TestCompositionWithAddedCards_ExtendsSet(t *testing.T) {
	base, ok := NewSet([]Card{
		card(Seven, Hearts),
		card(Seven, Diamonds),
		card(Seven, Clubs),
	})
	if !ok {
		t.Fatal("NewSet() returned false; want true")
	}

	extended, ok := base.WithAddedCards([]Card{card(Seven, Spades)})
	if !ok {
		t.Fatal("WithAddedCards() returned false; want true")
	}

	if len(extended.cards) != 4 {
		t.Fatalf("len(extended.cards) = %d; want 4", len(extended.cards))
	}
	if got := extended.Points(); got != 28 {
		t.Fatalf("extended.Points() = %d; want 28", got)
	}
}

func TestCompositionWithAddedCards_ExtendsRunWithMultipleCards(t *testing.T) {
	base, ok := NewRun([]Card{
		card(Five, Hearts),
		card(Six, Hearts),
		card(Seven, Hearts),
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}

	extended, ok := base.WithAddedCards([]Card{card(Eight, Hearts), card(Nine, Hearts)})
	if !ok {
		t.Fatal("WithAddedCards() returned false; want true")
	}

	if len(extended.cards) != 5 {
		t.Fatalf("len(extended.cards) = %d; want 5", len(extended.cards))
	}
	if got := extended.Points(); got != 35 {
		t.Fatalf("extended.Points() = %d; want 35", got)
	}
}

func TestCompositionWithAddedCards_RejectsInvalidAddition(t *testing.T) {
	base, ok := NewRun([]Card{
		card(Five, Hearts),
		card(Six, Hearts),
		card(Seven, Hearts),
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}

	if _, ok := base.WithAddedCards([]Card{card(Nine, Hearts)}); ok {
		t.Fatal("WithAddedCards() returned true; want false")
	}
}

func TestCompositionAddedCardsPoints_UsesContextualAceValue(t *testing.T) {
	base, ok := NewRun([]Card{
		card(Two, Clubs),
		card(Three, Clubs),
		card(Four, Clubs),
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}

	got, ok := base.AddedCardsPoints([]Card{card(Ace, Clubs)})
	if !ok {
		t.Fatal("AddedCardsPoints() returned false; want true")
	}
	if got != 1 {
		t.Fatalf("AddedCardsPoints() = %d; want 1", got)
	}
}

func TestCompositionAddedCardsPoints_UsesRepresentedJokerValue(t *testing.T) {
	base, ok := NewRun([]Card{
		card(Queen, Diamonds),
		card(King, Diamonds),
		card(Ace, Diamonds),
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}

	got, ok := base.AddedCardsPoints([]Card{joker()})
	if !ok {
		t.Fatal("AddedCardsPoints() returned false; want true")
	}
	if got != 10 {
		t.Fatalf("AddedCardsPoints() = %d; want 10", got)
	}
}
