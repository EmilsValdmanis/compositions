package game

import "testing"

func TestCardPoints(t *testing.T) {
	tests := []struct {
		card     Card
		expected int
	}{
		{Card{Rank: Ace, Suit: Spades}, 10},
		{Card{Rank: Two, Suit: Hearts}, 2},
		{Card{Rank: Three, Suit: Diamonds}, 3},
		{Card{Rank: Four, Suit: Spades}, 4},
		{Card{Rank: Five, Suit: Clubs}, 5},
		{Card{Rank: Six, Suit: Spades}, 6},
		{Card{Rank: Seven, Suit: Hearts}, 7},
		{Card{Rank: Eight, Suit: Spades}, 8},
		{Card{Rank: Nine, Suit: Clubs}, 9},
		{Card{Rank: Ten, Suit: Spades}, 10},
		{Card{Rank: Jack, Suit: Diamonds}, 10},
		{Card{Rank: Queen, Suit: Spades}, 10},
		{Card{Rank: King, Suit: Clubs}, 10},
		{Card{isJoker: true}, 20},
	}

	for _, test := range tests {
		cp := test.card.Points()

		if cp != test.expected {
			t.Errorf("Points(%v) = %d; want %d", test.card, cp, test.expected)
		}
	}
}
