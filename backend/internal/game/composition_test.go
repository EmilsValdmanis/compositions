package game

import "testing"

func card(rank Rank, suit Suit) Card {
	return Card{rank: rank, suit: suit}
}

func joker() Card {
	return Card{isJoker: true}
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
