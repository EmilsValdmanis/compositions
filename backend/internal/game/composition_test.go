package game

import (
	"slices"
	"testing"
)

func TestRunValidationDefensiveBranches(t *testing.T) {
	if naturalRunIsValid([]Card{{rank: Rank(99), suit: Hearts}}) {
		t.Fatal("naturalRunIsValid() accepted an out-of-range rank")
	}
	invalid := &Composition{variant: run, cards: []Card{card(Two, Hearts), card(Five, Hearts), card(Nine, Hearts)}}
	if invalid.normalizeRunCards() {
		t.Fatal("normalizeRunCards() accepted an invalid run")
	}
}

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

func TestNewUnorderedRunAcceptsSortableCards(t *testing.T) {
	comp, ok := NewUnorderedRun([]Card{
		card(Seven, Clubs),
		card(Five, Clubs),
		card(Six, Clubs),
	})
	if !ok {
		t.Fatal("NewUnorderedRun() returned false; want true")
	}
	if !slices.EqualFunc(comp.cards, []Card{
		card(Five, Clubs),
		card(Six, Clubs),
		card(Seven, Clubs),
	}, cardsEqual) {
		t.Fatalf("NewUnorderedRun().cards = %#v; want sorted run", comp.cards)
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

func TestNewRun_AcceptsDescendingSequence(t *testing.T) {
	cards := []Card{
		card(Seven, Hearts),
		card(Six, Hearts),
		card(Five, Hearts),
	}

	comp, ok := NewRun(cards)
	if !ok {
		t.Fatal("NewRun() returned false; want true for descending run")
	}

	if got := comp.cards; !slices.Equal(got, []Card{card(Five, Hearts), card(Six, Hearts), card(Seven, Hearts)}) {
		t.Fatalf("NewRun() cards = %#v; want normalized ascending order", got)
	}
}

func TestNewRun_AcceptsDescendingSequenceStartingWithJoker(t *testing.T) {
	comp, ok := NewRun([]Card{
		joker(),
		card(Jack, Hearts),
		card(Ten, Hearts),
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true for Joker-Jack-Ten")
	}

	want := []Card{card(Ten, Hearts), card(Jack, Hearts), joker()}
	if !slices.EqualFunc(comp.cards, want, cardsEqual) {
		t.Fatalf("NewRun() cards = %#v; want normalized Ten-Jack-Joker", comp.cards)
	}
	expectJokerRepresentation(t, comp, 2, card(Queen, Hearts))
}

func TestNewRun_RejectsMixedOrderSequence(t *testing.T) {
	cards := []Card{
		card(Seven, Hearts),
		card(Five, Hearts),
		card(Six, Hearts),
	}

	if _, ok := NewRun(cards); ok {
		t.Fatal("NewRun() returned true; want false for mixed-order run")
	}
}

func TestNewRun_RejectsAmbiguousMiddleJokerPlacement(t *testing.T) {
	if _, ok := NewRun([]Card{card(Five, Hearts), joker(), card(Six, Hearts)}); ok {
		t.Fatal("NewRun() returned true; want false for ambiguous joker placement")
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

func TestNewComposition_NormalizesRunOrderForInternalMutations(t *testing.T) {
	comp, ok := NewComposition([]Card{
		card(Eight, Hearts),
		joker(),
		card(Nine, Hearts),
		card(Ten, Hearts),
		card(Seven, Hearts),
	}, run)
	if !ok {
		t.Fatal("NewComposition() returned false; want true")
	}

	got := comp.cards
	want := []Card{
		card(Seven, Hearts),
		card(Eight, Hearts),
		card(Nine, Hearts),
		card(Ten, Hearts),
		joker(),
	}
	if !slices.EqualFunc(got, want, sameCard) {
		t.Fatalf("normalized run = %#v; want %#v", got, want)
	}
	expectJokerRepresentation(t, comp, 4, card(Jack, Hearts))
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

func TestCompositionIsCompleteSet(t *testing.T) {
	comp, ok := NewSet([]Card{
		card(Ace, Hearts),
		card(Ace, Diamonds),
		card(Ace, Clubs),
		card(Ace, Spades),
	})
	if !ok {
		t.Fatal("NewSet() returned false; want true")
	}

	if !comp.isComplete() {
		t.Fatal("isComplete() = false; want true")
	}
	if !comp.isCompleteSet() {
		t.Fatal("isCompleteSet() = false; want true")
	}
	if comp.isCompleteRun() {
		t.Fatal("isCompleteRun() = true; want false")
	}
}

func TestCompositionIsNotCompleteSetWithJoker(t *testing.T) {
	comp, ok := NewSet([]Card{
		card(Ace, Hearts),
		card(Ace, Diamonds),
		card(Ace, Clubs),
		joker(),
	})
	if !ok {
		t.Fatal("NewSet() returned false; want true")
	}

	if comp.isComplete() {
		t.Fatal("isComplete() = true; want false")
	}
}

func TestCompositionIsCompleteRun(t *testing.T) {
	comp, ok := NewRun([]Card{
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
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}

	if !comp.isComplete() {
		t.Fatal("isComplete() = false; want true")
	}
	if !comp.isCompleteRun() {
		t.Fatal("isCompleteRun() = false; want true")
	}
	if comp.isCompleteSet() {
		t.Fatal("isCompleteSet() = true; want false")
	}
}

func TestCompositionIsNotCompleteRunWithJoker(t *testing.T) {
	comp, ok := NewRun([]Card{
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
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}

	if comp.isComplete() {
		t.Fatal("isComplete() = true; want false")
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

func TestCompositionWithAddedCards_NarrowsSetJokerRepresentation(t *testing.T) {
	base, ok := NewSet([]Card{
		card(Ten, Hearts),
		card(Ten, Diamonds),
		joker(),
	})
	if !ok {
		t.Fatal("NewSet() returned false; want true")
	}

	if _, narrowed := base.JokerRepresentation(2); narrowed {
		t.Fatal("JokerRepresentation(2) returned ok = true; want false before narrowing")
	}

	extended, ok := base.WithAddedCards([]Card{card(Ten, Clubs)})
	if !ok {
		t.Fatal("WithAddedCards() returned false; want true")
	}

	expectJokerRepresentation(t, extended, 2, card(Ten, Spades))
	if _, ok := extended.ReclaimJoker(2, card(Ten, Spades)); !ok {
		t.Fatal("ReclaimJoker() returned false after addition narrowed the joker")
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

func TestCompositionWithInsertedCards(t *testing.T) {
	base, ok := NewRun([]Card{
		card(Queen, Hearts),
		card(King, Hearts),
		card(Ace, Hearts),
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}

	inserted, ok := base.WithInsertedCards(0, []Card{joker(), card(Jack, Hearts)})
	if !ok {
		t.Fatal("WithInsertedCards() returned false; want true")
	}
	if len(inserted.cards) != 5 {
		t.Fatalf("len(inserted.cards) = %d; want 5", len(inserted.cards))
	}
	want := []Card{joker(), card(Jack, Hearts), card(Queen, Hearts), card(King, Hearts), card(Ace, Hearts)}
	if !slices.EqualFunc(inserted.cards, want, sameCard) {
		t.Fatalf("inserted.cards = %#v; want %#v", inserted.cards, want)
	}

	if _, ok := base.WithInsertedCards(-1, []Card{card(Jack, Hearts)}); ok {
		t.Fatal("WithInsertedCards(-1) returned true; want false")
	}
	if _, ok := base.WithInsertedCards(len(base.cards)+1, []Card{card(Jack, Hearts)}); ok {
		t.Fatal("WithInsertedCards(out of bounds) returned true; want false")
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

func TestCompositionAddedCardsPointsRejectsInvalidAddition(t *testing.T) {
	base := mustRun(t, card(Five, Hearts), card(Six, Hearts), card(Seven, Hearts))

	if got, ok := base.AddedCardsPoints([]Card{card(Nine, Hearts)}); ok || got != 0 {
		t.Fatalf("AddedCardsPoints() = (%d, %v); want (0, false)", got, ok)
	}
}

func TestNewCompositionRejectsUnknownVariant(t *testing.T) {
	if comp, ok := NewComposition([]Card{card(Ace, Hearts), card(Ace, Diamonds), card(Ace, Clubs)}, compositionVariant("weird")); ok || comp != nil {
		t.Fatalf("NewComposition() = (%v, %v); want (nil, false)", comp, ok)
	}
}

func TestCompositionReclaimJokerRejectsInvalidInputs(t *testing.T) {
	comp := mustRun(t, card(Five, Hearts), joker(), card(Seven, Hearts))

	if _, ok := comp.ReclaimJoker(-1, card(Six, Hearts)); ok {
		t.Fatal("ReclaimJoker() returned true for negative index")
	}
	if _, ok := comp.ReclaimJoker(3, card(Six, Hearts)); ok {
		t.Fatal("ReclaimJoker() returned true for out-of-range index")
	}
	if _, ok := comp.ReclaimJoker(0, card(Five, Hearts)); ok {
		t.Fatal("ReclaimJoker() returned true for non-joker card")
	}
	if _, ok := comp.ReclaimJoker(1, joker()); ok {
		t.Fatal("ReclaimJoker() returned true for joker replacement")
	}
	if _, ok := comp.ReclaimJoker(1, card(Eight, Hearts)); ok {
		t.Fatal("ReclaimJoker() returned true for wrong replacement")
	}
}

func TestCompositionReclaimPoints(t *testing.T) {
	setComp := mustSet(t, card(Ten, Hearts), card(Ten, Diamonds), card(Ten, Clubs), joker())
	if got, ok := setComp.ReclaimPoints(3); !ok || got != 10 {
		t.Fatalf("setComp.ReclaimPoints(3) = (%d, %v); want (10, true)", got, ok)
	}

	runComp := mustRun(t, card(Five, Hearts), joker(), card(Seven, Hearts))
	if got, ok := runComp.ReclaimPoints(1); !ok || got != 6 {
		t.Fatalf("runComp.ReclaimPoints(1) = (%d, %v); want (6, true)", got, ok)
	}

	doubleJokerAceLowComp := mustRun(t, joker(), joker(), card(Three, Clubs))
	if got, ok := doubleJokerAceLowComp.ReclaimPoints(0); !ok || got != 1 {
		t.Fatalf("doubleJokerAceLowComp.ReclaimPoints(0) = (%d, %v); want (1, true)", got, ok)
	}

	aceLowComp := mustRun(t, joker(), card(Two, Clubs), card(Three, Clubs))
	if got, ok := aceLowComp.ReclaimPoints(0); !ok || got != 1 {
		t.Fatalf("aceLowComp.ReclaimPoints(0) = (%d, %v); want (1, true)", got, ok)
	}
}

func TestCompositionReclaimPointsRejectsInvalidAndBrokenInputs(t *testing.T) {
	setComp := mustSet(t, card(Ten, Hearts), card(Ten, Diamonds), card(Ten, Clubs), joker())
	if got, ok := setComp.ReclaimPoints(-1); ok || got != 0 {
		t.Fatalf("setComp.ReclaimPoints(-1) = (%d, %v); want (0, false)", got, ok)
	}
	if got, ok := setComp.ReclaimPoints(0); ok || got != 0 {
		t.Fatalf("setComp.ReclaimPoints(0) = (%d, %v); want (0, false)", got, ok)
	}

	ambiguousSet := mustSet(t, card(Ten, Hearts), card(Ten, Diamonds), joker())
	if got, ok := ambiguousSet.ReclaimPoints(2); ok || got != 0 {
		t.Fatalf("ambiguousSet.ReclaimPoints(2) = (%d, %v); want (0, false)", got, ok)
	}

	ambiguousRun := &Composition{
		variant: run,
		cards:   []Card{card(Five, Hearts), joker(), card(Six, Hearts)},
		jokerRepresentations: map[int][]Card{
			1: {card(Four, Hearts)},
		},
	}
	if got, ok := ambiguousRun.ReclaimPoints(1); ok || got != 0 {
		t.Fatalf("ambiguousRun.ReclaimPoints(1) = (%d, %v); want (0, false)", got, ok)
	}

	unknown := &Composition{variant: compositionVariant("weird"), cards: []Card{joker()}}
	if got, ok := unknown.ReclaimPoints(0); ok || got != 0 {
		t.Fatalf("unknown.ReclaimPoints(0) = (%d, %v); want (0, false)", got, ok)
	}
}

func TestCompositionCanReclaimJoker(t *testing.T) {
	ambiguous := mustSet(t, card(Ten, Hearts), card(Ten, Diamonds), joker())
	if ambiguous.canReclaimJoker(2, card(Ten, Clubs)) {
		t.Fatal("canReclaimJoker() = true; want false for ambiguous set joker")
	}

	comp := mustRun(t, card(Five, Hearts), joker(), card(Seven, Hearts))
	if !comp.canReclaimJoker(1, card(Six, Hearts)) {
		t.Fatal("canReclaimJoker() = false; want true")
	}
	if comp.canReclaimJoker(-1, card(Six, Hearts)) {
		t.Fatal("canReclaimJoker() = true for negative index")
	}
	if comp.canReclaimJoker(0, card(Five, Hearts)) {
		t.Fatal("canReclaimJoker() = true for non-joker slot")
	}
	if comp.canReclaimJoker(1, joker()) {
		t.Fatal("canReclaimJoker() = true for joker replacement")
	}
	if comp.canReclaimJoker(1, card(Eight, Hearts)) {
		t.Fatal("canReclaimJoker() = true for wrong card")
	}
}

func TestCompositionAddedCardsPointsNoAllocRejectsInvalidAddition(t *testing.T) {
	base := mustRun(t, card(Five, Hearts), card(Six, Hearts), card(Seven, Hearts))

	if got, ok := base.addedCardsPointsNoAlloc([]Card{card(Nine, Hearts)}, make([]Card, 0, 8)); ok || got != 0 {
		t.Fatalf("addedCardsPointsNoAlloc() = (%d, %v); want (0, false)", got, ok)
	}
	if got, ok := base.addedCardsPointsNoAlloc([]Card{card(Eight, Hearts)}, make([]Card, 0, 8)); !ok || got != 8 {
		t.Fatalf("addedCardsPointsNoAlloc() = (%d, %v); want (8, true)", got, ok)
	}
}

func TestCompositionPointsUnknownVariant(t *testing.T) {
	comp := &Composition{variant: compositionVariant("weird")}

	if got := comp.Points(); got != 0 {
		t.Fatalf("Points() = %d; want 0", got)
	}
}

func TestCompositionIsCompleteUnknownVariant(t *testing.T) {
	comp := &Composition{variant: compositionVariant("weird")}

	if comp.isComplete() {
		t.Fatal("isComplete() = true; want false")
	}
}

func TestCompositionIsCompleteSetRejectsMixedRanksAndDuplicateSuits(t *testing.T) {
	if comp, ok := NewSet([]Card{card(Ace, Hearts), card(Ace, Diamonds), card(Ace, Clubs), card(Ace, Spades)}); !ok || !comp.isCompleteSet() {
		t.Fatal("expected complete natural set")
	}

	mixed := &Composition{variant: set, cards: []Card{card(Ace, Hearts), card(King, Diamonds), card(Ace, Clubs), card(Ace, Spades)}}
	if mixed.isCompleteSet() {
		t.Fatal("isCompleteSet() = true; want false for mixed ranks")
	}

	dup := &Composition{variant: set, cards: []Card{card(Ace, Hearts), card(Ace, Hearts), card(Ace, Clubs), card(Ace, Spades)}}
	if dup.isCompleteSet() {
		t.Fatal("isCompleteSet() = true; want false for duplicate suit")
	}
}

func TestCompositionIsCompleteRunRejectsWrongSuitAndWrongRankCounts(t *testing.T) {
	wrongSuit := &Composition{variant: run, cards: []Card{
		card(Ace, Hearts), card(Two, Hearts), card(Three, Hearts), card(Four, Hearts), card(Five, Hearts), card(Six, Hearts),
		card(Seven, Hearts), card(Eight, Hearts), card(Nine, Hearts), card(Ten, Hearts), card(Jack, Hearts), card(Queen, Hearts),
		card(King, Hearts), card(Ace, Clubs),
	}}
	if wrongSuit.isCompleteRun() {
		t.Fatal("isCompleteRun() = true; want false for mixed suit")
	}

	wrongAces := &Composition{variant: run, cards: []Card{
		card(Ace, Hearts), card(Two, Hearts), card(Three, Hearts), card(Four, Hearts), card(Five, Hearts), card(Six, Hearts),
		card(Seven, Hearts), card(Eight, Hearts), card(Nine, Hearts), card(Ten, Hearts), card(Jack, Hearts), card(Queen, Hearts),
		card(King, Hearts), card(King, Hearts),
	}}
	if wrongAces.isCompleteRun() {
		t.Fatal("isCompleteRun() = true; want false for wrong ace count")
	}

	missingRank := &Composition{variant: run, cards: []Card{
		card(Ace, Hearts), card(Two, Hearts), card(Three, Hearts), card(Four, Hearts), card(Five, Hearts), card(Six, Hearts),
		card(Seven, Hearts), card(Eight, Hearts), card(Nine, Hearts), card(Ten, Hearts), card(Jack, Hearts), card(Queen, Hearts),
		card(Ace, Hearts), card(Ace, Hearts),
	}}
	if missingRank.isCompleteRun() {
		t.Fatal("isCompleteRun() = true; want false for missing rank")
	}

	wrongMiddleCount := &Composition{variant: run, cards: []Card{
		card(Ace, Hearts), card(Two, Hearts), card(Three, Hearts), card(Four, Hearts), card(Five, Hearts), card(Six, Hearts),
		card(Seven, Hearts), card(Eight, Hearts), card(Nine, Hearts), card(Ten, Hearts), card(Jack, Hearts), card(Queen, Hearts),
		card(Queen, Hearts), card(Ace, Hearts),
	}}
	if wrongMiddleCount.isCompleteRun() {
		t.Fatal("isCompleteRun() = true; want false for duplicate middle rank")
	}
}

func TestCompositionSetPointsFallsBackWhenSetRankUnknownOrJokerRepresentationMissing(t *testing.T) {
	empty := &Composition{variant: set}
	if got := empty.setPoints(); got != 0 {
		t.Fatalf("setPoints() = %d; want 0", got)
	}

	allJokers := mustSet(t, joker(), joker(), joker())
	if got := allJokers.setRank(); got != Ace {
		t.Fatalf("setRank() = %v; want Ace", got)
	}

	broken := &Composition{variant: set, cards: []Card{card(Ten, Hearts), card(Ten, Diamonds), joker()}, jokerRepresentations: map[int][]Card{}}
	if got := broken.setPoints(); got != 30 {
		t.Fatalf("setPoints() = %d; want 30", got)
	}

	exact := mustSet(t, card(Ten, Hearts), card(Ten, Diamonds), joker())
	if got := exact.setPoints(); got != 30 {
		t.Fatalf("setPoints() = %d; want 30", got)
	}

	narrow := mustSet(t, card(Ten, Hearts), card(Ten, Diamonds), card(Ten, Clubs), joker())
	if got := narrow.setPoints(); got != 40 {
		t.Fatalf("setPoints() = %d; want 40", got)
	}
}

func TestRunCardsPointsTwoAcesForcesFirstAceLow(t *testing.T) {
	got := runCardsPoints([]Card{card(Ace, Hearts), card(Ace, Hearts), card(King, Hearts)}, false)

	if got != 21 {
		t.Fatalf("runCardsPoints() = %d; want 21", got)
	}
}

func TestRankPointsUnknownRank(t *testing.T) {
	if got := rankPoints(Rank(99), false); got != 0 {
		t.Fatalf("rankPoints() = %d; want 0", got)
	}
}

func TestCompositionIsValidUnknownVariant(t *testing.T) {
	comp := &Composition{variant: compositionVariant("weird")}

	if comp.isValid() {
		t.Fatal("isValid() = true; want false")
	}
}

func TestCompositionIsValidSetRejectsTooFewRealCardsForMissingSlots(t *testing.T) {
	comp := &Composition{variant: set, cards: []Card{card(Five, Hearts), card(Five, Diamonds), card(Five, Clubs), joker(), joker()}}

	if comp.isValidSet() {
		t.Fatal("isValidSet() = true; want false")
	}
}

func TestCompositionAssignJokersEmptyAndUnknownVariant(t *testing.T) {
	empty := &Composition{}
	if !empty.assignJokers() {
		t.Fatal("assignJokers() = false; want true for empty composition")
	}
	if len(empty.jokerRepresentations) != 0 {
		t.Fatalf("len(jokerRepresentations) = %d; want 0", len(empty.jokerRepresentations))
	}

	unknown := &Composition{variant: compositionVariant("weird"), cards: []Card{joker()}}
	if unknown.assignJokers() {
		t.Fatal("assignJokers() = true; want false for unknown variant")
	}
}

func TestCompositionAssignSetJokersRejectsNoMissingSuit(t *testing.T) {
	comp := &Composition{variant: set, cards: []Card{card(Ten, Hearts), card(Ten, Diamonds), card(Ten, Clubs), card(Ten, Spades), joker()}}

	if comp.assignSetJokers() {
		t.Fatal("assignSetJokers() = true; want false")
	}
}

func TestCompositionAssignRunJokersRejectsUnfillableRun(t *testing.T) {
	comp := &Composition{variant: run, cards: []Card{card(Two, Hearts), joker(), card(Six, Hearts)}}

	if comp.assignRunJokers() {
		t.Fatal("assignRunJokers() = true; want false")
	}
}

func TestMissingSetCardsRejectsFullyUsedSuits(t *testing.T) {
	if options, ok := missingSetCards(Ten, map[Suit]bool{Hearts: true, Diamonds: true, Clubs: true, Spades: true}); ok || options != nil {
		t.Fatalf("missingSetCards() = (%v, %v); want (nil, false)", options, ok)
	}
}

func TestTryFitSequenceRejectsInvalidAllJokerLengthsAndOverflow(t *testing.T) {
	if replacements, ok := tryFitSequence(nil, 2, false); ok || replacements != nil {
		t.Fatalf("tryFitSequence() = (%v, %v); want (nil, false)", replacements, ok)
	}
	if replacements, ok := tryFitSequence(nil, 15, false); ok || replacements != nil {
		t.Fatalf("tryFitSequence() = (%v, %v); want (nil, false)", replacements, ok)
	}
	if replacements, ok := tryFitSequence([]Card{card(Ace, Hearts), card(Ace, Hearts)}, 13, false); ok || replacements != nil {
		t.Fatalf("tryFitSequence() = (%v, %v); want (nil, false)", replacements, ok)
	}
}

func TestSequenceRanksForCardsRejectsJokerInput(t *testing.T) {
	if ranks, ok := sequenceRanksForCards([]Card{card(Five, Hearts), joker()}, false); ok || ranks != nil {
		t.Fatalf("sequenceRanksForCards() = (%v, %v); want (nil, false)", ranks, ok)
	}
}

func TestTryFitSequenceRanksRejectsJokerInput(t *testing.T) {
	if ranks, ok := tryFitSequenceRanks([]Card{card(Five, Hearts), joker()}, 0, false); ok || ranks != nil {
		t.Fatalf("tryFitSequenceRanks() = (%v, %v); want (nil, false)", ranks, ok)
	}
}

func TestBestRunOrder_AllJokersAndInvalidRun(t *testing.T) {
	ordered, jokerRepresentations, matchesInput, ok := bestRunOrder([]Card{joker(), joker(), joker()})
	if !ok {
		t.Fatal("bestRunOrder(all jokers) ok = false; want true")
	}
	if !matchesInput {
		t.Fatal("bestRunOrder(all jokers) matchesInput = false; want true")
	}
	if len(ordered) != 3 || len(jokerRepresentations) != 3 {
		t.Fatalf("bestRunOrder(all jokers) = (%#v, %#v); want 3 ordered cards and representations", ordered, jokerRepresentations)
	}
	if jokerRepresentations[0].rank != Ace || jokerRepresentations[1].rank != Two || jokerRepresentations[2].rank != Three {
		t.Fatalf("jokerRepresentations = %#v; want Ace, Two, Three", jokerRepresentations)
	}

	if ordered, jokerRepresentations, matchesInput, ok := bestRunOrder([]Card{card(King, Hearts), card(Ace, Hearts), card(Two, Hearts)}); ok || ordered != nil || jokerRepresentations != nil || matchesInput {
		t.Fatalf("bestRunOrder(invalid) = (%#v, %#v, %v, %v); want (nil, nil, false, false)", ordered, jokerRepresentations, matchesInput, ok)
	}
}

func TestBestRunOrder_PrefersClosestCandidateWhenInputIsUnordered(t *testing.T) {
	ordered, jokerRepresentations, matchesInput, ok := bestRunOrder([]Card{
		card(Eight, Hearts),
		card(Seven, Hearts),
		joker(),
		card(Ten, Hearts),
	})
	if !ok {
		t.Fatal("bestRunOrder() ok = false; want true")
	}
	if matchesInput {
		t.Fatal("bestRunOrder() matchesInput = true; want false")
	}
	want := []Card{card(Seven, Hearts), card(Eight, Hearts), joker(), card(Ten, Hearts)}
	if !slices.EqualFunc(ordered, want, sameCard) {
		t.Fatalf("bestRunOrder() ordered = %#v; want %#v", ordered, want)
	}
	if replacement := jokerRepresentations[2]; replacement.rank != Nine || replacement.suit != Hearts {
		t.Fatalf("jokerRepresentations[2] = %#v; want Nine of Hearts", replacement)
	}
}

func TestBestRunOrder_RejectsAmbiguousJokerPlacement(t *testing.T) {
	ordered, jokerRepresentations, matchesInput, ok := bestRunOrder([]Card{
		card(Seven, Hearts),
		joker(),
		card(Six, Hearts),
	})
	if ok || ordered != nil || jokerRepresentations != nil || matchesInput {
		t.Fatalf("bestRunOrder() = (%#v, %#v, %v, %v); want (nil, nil, false, false)", ordered, jokerRepresentations, matchesInput, ok)
	}
}

func TestRunOrderCandidatesEqual(t *testing.T) {
	if !runOrderCandidatesEqual(nil, nil) {
		t.Fatal("runOrderCandidatesEqual(nil, nil) = false; want true")
	}
	if runOrderCandidatesEqual(nil, &runOrderCandidate{}) {
		t.Fatal("runOrderCandidatesEqual(nil, candidate) = true; want false")
	}

	base := &runOrderCandidate{
		ordered:              []Card{card(Five, Hearts), joker(), card(Seven, Hearts)},
		jokerRepresentations: map[int]Card{1: card(Six, Hearts)},
	}
	if !runOrderCandidatesEqual(base, &runOrderCandidate{
		ordered:              []Card{card(Five, Hearts), joker(), card(Seven, Hearts)},
		jokerRepresentations: map[int]Card{1: card(Six, Hearts)},
	}) {
		t.Fatal("runOrderCandidatesEqual(equal candidates) = false; want true")
	}
	if runOrderCandidatesEqual(base, &runOrderCandidate{
		ordered:              []Card{joker(), card(Six, Hearts), card(Seven, Hearts)},
		jokerRepresentations: map[int]Card{0: card(Five, Hearts)},
	}) {
		t.Fatal("runOrderCandidatesEqual(different ordered cards) = true; want false")
	}
	if runOrderCandidatesEqual(base, &runOrderCandidate{
		ordered:              []Card{card(Five, Hearts), joker(), card(Seven, Hearts)},
		jokerRepresentations: map[int]Card{1: card(Six, Hearts), 2: card(Eight, Hearts)},
	}) {
		t.Fatal("runOrderCandidatesEqual(different representation count) = true; want false")
	}
	if runOrderCandidatesEqual(base, &runOrderCandidate{
		ordered:              []Card{card(Five, Hearts), joker(), card(Seven, Hearts)},
		jokerRepresentations: map[int]Card{0: card(Six, Hearts)},
	}) {
		t.Fatal("runOrderCandidatesEqual(different representation keys) = true; want false")
	}
	if runOrderCandidatesEqual(base, &runOrderCandidate{
		ordered:              []Card{card(Five, Hearts), joker(), card(Seven, Hearts)},
		jokerRepresentations: map[int]Card{1: card(Eight, Hearts)},
	}) {
		t.Fatal("runOrderCandidatesEqual(different representation card) = true; want false")
	}
}

func TestBestRunOrder_RejectsImpossibleJokerCountForWindow(t *testing.T) {
	ordered, jokerRepresentations, matchesInput, ok := bestRunOrder([]Card{
		card(Two, Hearts),
		card(Two, Hearts),
		joker(),
	})
	if ok || ordered != nil || jokerRepresentations != nil || matchesInput {
		t.Fatalf("bestRunOrder() = (%#v, %#v, %v, %v); want (nil, nil, false, false)", ordered, jokerRepresentations, matchesInput, ok)
	}
}

func TestRunCardsAreOrdered(t *testing.T) {
	if !runCardsAreOrdered([]Card{card(Queen, Diamonds), card(King, Diamonds), card(Ace, Diamonds)}) {
		t.Fatal("runCardsAreOrdered() = false; want true for ordered ace-high run")
	}
	if runCardsAreOrdered([]Card{card(King, Diamonds), card(Queen, Diamonds), card(Ace, Diamonds)}) {
		t.Fatal("runCardsAreOrdered() = true; want false for unordered ace-high run")
	}
	if !runCardsAreOrdered([]Card{card(Eight, Clubs), card(Nine, Clubs), card(Ten, Clubs), joker()}) {
		t.Fatal("runCardsAreOrdered() = false; want true for ordered run with joker")
	}
	if runCardsAreOrdered([]Card{card(Eight, Clubs), joker(), card(Nine, Clubs), card(Ten, Clubs)}) {
		t.Fatal("runCardsAreOrdered() = true; want false for unordered joker placement")
	}
}

func TestRunCardsAreReverseOrdered(t *testing.T) {
	if !runCardsAreReverseOrdered([]Card{card(Ace, Diamonds), card(King, Diamonds), card(Queen, Diamonds)}) {
		t.Fatal("runCardsAreReverseOrdered() = false; want true for reverse-ordered ace-high run")
	}
	if !runCardsAreReverseOrdered([]Card{joker(), card(Ten, Clubs), card(Nine, Clubs), card(Eight, Clubs)}) {
		t.Fatal("runCardsAreReverseOrdered() = false; want true for reverse-ordered run with joker")
	}
	if runCardsAreReverseOrdered([]Card{card(King, Diamonds), card(Queen, Diamonds), card(Ace, Diamonds)}) {
		t.Fatal("runCardsAreReverseOrdered() = true; want false for mixed ace-high run")
	}
}

func TestJokerRepresentationsRejectsInvalidRequests(t *testing.T) {
	comp := mustRun(t, card(Five, Hearts), joker(), card(Seven, Hearts))

	if got, ok := comp.JokerRepresentations(-1); ok || got != nil {
		t.Fatalf("JokerRepresentations(-1) = (%v, %v); want (nil, false)", got, ok)
	}
	if got, ok := comp.JokerRepresentations(0); ok || got != nil {
		t.Fatalf("JokerRepresentations(0) = (%v, %v); want (nil, false)", got, ok)
	}

	broken := &Composition{variant: run, cards: []Card{card(Five, Hearts), joker(), card(Seven, Hearts)}, jokerRepresentations: map[int][]Card{}}
	if got, ok := broken.JokerRepresentations(1); ok || got != nil {
		t.Fatalf("JokerRepresentations(1) = (%v, %v); want (nil, false)", got, ok)
	}
}

func TestSequenceRankToCardRank(t *testing.T) {
	if got := sequenceRankToCardRank(14); got != Ace {
		t.Fatalf("sequenceRankToCardRank(14) = %v; want Ace", got)
	}
	if got := sequenceRankToCardRank(5); got != Five {
		t.Fatalf("sequenceRankToCardRank(5) = %v; want Five", got)
	}
}
