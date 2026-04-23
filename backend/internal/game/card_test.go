package game

import "testing"

func TestCardPoints(t *testing.T) {
	tests := []struct {
		card     Card
		expected int
	}{
		{Card{rank: Ace, suit: Spades}, 10},
		{Card{rank: Two, suit: Hearts}, 2},
		{Card{rank: Three, suit: Diamonds}, 3},
		{Card{rank: Four, suit: Spades}, 4},
		{Card{rank: Five, suit: Clubs}, 5},
		{Card{rank: Six, suit: Spades}, 6},
		{Card{rank: Seven, suit: Hearts}, 7},
		{Card{rank: Eight, suit: Spades}, 8},
		{Card{rank: Nine, suit: Clubs}, 9},
		{Card{rank: Ten, suit: Spades}, 10},
		{Card{rank: Jack, suit: Diamonds}, 10},
		{Card{rank: Queen, suit: Spades}, 10},
		{Card{rank: King, suit: Clubs}, 10},
		{Card{isJoker: true}, 20},
	}

	for _, test := range tests {
		cp := test.card.Points()

		if cp != test.expected {
			t.Errorf("Points(%v) = %d; want %d", test.card, cp, test.expected)
		}
	}
}

func TestCardsEqual(t *testing.T) {
	tests := []struct {
	a, b     Card
	expected bool
}{
	{Card{rank: Ace, suit: Hearts}, Card{rank: Ace, suit: Hearts}, true},
	{Card{rank: Ace, suit: Hearts}, Card{rank: Ace, suit: Diamonds}, false},
	{Card{rank: Two, suit: Spades}, Card{rank: Three, suit: Spades}, false},
	{Card{isJoker: true}, Card{isJoker: true}, true},
	{Card{isJoker: true}, Card{rank: Ace, suit: Hearts}, false},
}

	for _, test := range tests {
		eq := cardsEqual(test.a, test.b)

		if eq != test.expected {
			t.Errorf("cardsEqual(%v, %v) = %t; want %t", test.a, test.b, eq, test.expected)
		}
	}
}
