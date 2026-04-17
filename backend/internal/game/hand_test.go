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
	deck := &Deck{cards: []Card{}}
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
