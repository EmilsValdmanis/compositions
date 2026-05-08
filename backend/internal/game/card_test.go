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

func TestCardPointsUnknownRank(t *testing.T) {
	card := Card{rank: Rank(99), suit: Hearts}

	if got := card.Points(); got != 0 {
		t.Fatalf("Points() = %d; want 0", got)
	}
}

func TestRankString(t *testing.T) {
	tests := []struct {
		rank Rank
		want string
	}{
		{Ace, "Ace"},
		{Two, "Two"},
		{Three, "Three"},
		{Four, "Four"},
		{Five, "Five"},
		{Six, "Six"},
		{Seven, "Seven"},
		{Eight, "Eight"},
		{Nine, "Nine"},
		{Ten, "Ten"},
		{Jack, "Jack"},
		{Queen, "Queen"},
		{King, "King"},
		{Rank(99), "Unknown"},
	}

	for _, test := range tests {
		if got := test.rank.String(); got != test.want {
			t.Fatalf("Rank(%d).String() = %q; want %q", test.rank, got, test.want)
		}
	}
}

func TestSuitString(t *testing.T) {
	tests := []struct {
		suit Suit
		want string
	}{
		{Hearts, "Hearts"},
		{Diamonds, "Diamonds"},
		{Clubs, "Clubs"},
		{Spades, "Spades"},
		{Suit(99), "Unknown"},
	}

	for _, test := range tests {
		if got := test.suit.String(); got != test.want {
			t.Fatalf("Suit(%d).String() = %q; want %q", test.suit, got, test.want)
		}
	}
}

func TestCardString(t *testing.T) {
	if got := joker().String(); got != "Joker" {
		t.Fatalf("joker.String() = %q; want %q", got, "Joker")
	}

	if got := card(Ace, Hearts).String(); got != "{Rank: Ace, Suit: Hearts}" {
		t.Fatalf("card.String() = %q; want %q", got, "{Rank: Ace, Suit: Hearts}")
	}
}
