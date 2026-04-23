package game

import "testing"

func TestHandDraw(t *testing.T) {
	deck := NewGameDeck()
	deck.Shuffle()
	initialDeckSize := len(deck.cards)

	hand := NewHand()

	drew := hand.Draw(deck)

	if !drew {
		t.Fatal("Draw() returned false; expected true")
	}

	if len(hand.cards) != 1 {
		t.Errorf("hand.cards length = %d; want 1", len(hand.cards))
	}

	if len(deck.cards) != initialDeckSize-1 {
		t.Errorf("deck.cards length = %d; want %d", len(deck.cards), initialDeckSize-1)
	}
}

func TestHandDrawFromEmptyDeck(t *testing.T) {
	deck := &CardPile{cards: []Card{}}
	hand := NewHand()

	drew := hand.Draw(deck)

	if drew {
		t.Error("Draw() returned true for empty deck; expected false")
	}

	if len(hand.cards) != 0 {
		t.Errorf("hand.cards length = %d; want 0", len(hand.cards))
	}
}

func TestHandDrawMultiple(t *testing.T) {
	deck := NewGameDeck()
	hand := NewHand()

	for i := 1; i <= 5; i++ {
		hand.Draw(deck)

		if len(hand.cards) != i {
			t.Errorf("after %d draws, hand.cards length = %d; want %d", i, len(hand.cards), i)
		}
	}

	if len(deck.cards) != 108-5 {
		t.Errorf("deck.cards length = %d; want %d", len(deck.cards), 108-5)
	}
}

func TestHandPoints(t *testing.T) {
	tests := []struct {
		name     string
		cards    []Card
		expected int
	}{
		{
			name:     "ace as last card counts as 1 (Special rule)",
			cards:    []Card{{rank: Ace, suit: Spades}},
			expected: 1,
		},
		{
			name:     "number cards use face value",
			cards:    []Card{{rank: Two}, {rank: Five}, {rank: Ten}},
			expected: 17,
		},
		{
			name:     "joker scores 20",
			cards:    []Card{{isJoker: true}},
			expected: 20,
		},
		{
			name:     "two jokers score 40",
			cards:    []Card{{isJoker: true}, {isJoker: true}},
			expected: 40,
		},
		{
			name:     "mixed hand",
			cards:    []Card{{rank: Ace}, {rank: Seven}, {rank: King}, {isJoker: true}},
			expected: 47, // 10 + 7 + 10 + 20
		},
		{
			name:     "multiple aces all score 10",
			cards:    []Card{{rank: Ace}, {rank: Ace}, {rank: Ace}},
			expected: 30,
		},
		{
			name:     "empty hand scores 0",
			cards:    []Card{},
			expected: 0,
		},
	}

	for _, test := range tests {
		h := &Hand{cards: test.cards}
		hp := h.Points()

		if hp != test.expected {
			t.Errorf("got %d, want %d", hp, test.expected)
		}
	}
}

func TestHandRemoveAt(t *testing.T) {
	hand := &Hand{cards: []Card{
		{rank: Ace, suit: Hearts},
		{rank: Five, suit: Clubs},
		{rank: King, suit: Spades},
	}}

	removed, ok := hand.RemoveAt(1)

	if !ok {
		t.Fatal("RemoveAt(1) returned false; want true")
	}
	if removed.rank != Five || removed.suit != Clubs {
		t.Errorf("removed card = %+v; want Five of Clubs", removed)
	}
	if len(hand.cards) != 2 {
		t.Fatalf("len(hand.cards) = %d; want 2", len(hand.cards))
	}
	if hand.cards[0].rank != Ace || hand.cards[1].rank != King {
		t.Errorf("remaining cards = %+v; want Ace then King", hand.cards)
	}
}

func TestHandRemoveAtRejectsInvalidIndex(t *testing.T) {
	hand := &Hand{cards: []Card{{rank: Ace, suit: Hearts}}}

	removed, ok := hand.RemoveAt(1)

	if ok {
		t.Error("RemoveAt(1) returned true; want false")
	}
	if removed != (Card{}) {
		t.Errorf("removed card = %+v; want zero Card", removed)
	}
	if len(hand.cards) != 1 {
		t.Errorf("len(hand.cards) = %d; want 1", len(hand.cards))
	}
}

func TestHandRemoveCards(t *testing.T) {
	hand := &Hand{cards: []Card{
		{rank: Seven, suit: Hearts},
		{rank: Seven, suit: Hearts},
		{rank: Eight, suit: Clubs},
		{rank: King, suit: Spades},
	}}

	ok := hand.RemoveCards([]Card{
		{rank: Seven, suit: Hearts},
		{rank: Eight, suit: Clubs},
	})

	if !ok {
		t.Fatal("RemoveCards() returned false; want true")
	}
	if len(hand.cards) != 2 {
		t.Fatalf("len(hand.cards) = %d; want 2", len(hand.cards))
	}
	if hand.cards[0].rank != Seven || hand.cards[0].suit != Hearts {
		t.Errorf("hand.cards[0] = %+v; want Seven of Hearts", hand.cards[0])
	}
	if hand.cards[1].rank != King || hand.cards[1].suit != Spades {
		t.Errorf("hand.cards[1] = %+v; want King of Spades", hand.cards[1])
	}
}

func TestHandRemoveCardsDoesNotMutateOnFailure(t *testing.T) {
	original := []Card{
		{rank: Four, suit: Diamonds},
		{rank: Five, suit: Diamonds},
	}
	hand := &Hand{cards: append([]Card(nil), original...)}

	ok := hand.RemoveCards([]Card{{rank: King, suit: Spades}})

	if ok {
		t.Error("RemoveCards() returned true; want false")
	}
	if len(hand.cards) != len(original) {
		t.Fatalf("len(hand.cards) = %d; want %d", len(hand.cards), len(original))
	}
	for i, card := range original {
		if hand.cards[i] != card {
			t.Errorf("hand.cards[%d] = %+v; want %+v", i, hand.cards[i], card)
		}
	}
}
