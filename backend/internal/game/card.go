package game

import "fmt"

type Suit int

const (
	Hearts Suit = iota
	Diamonds
	Clubs
	Spades
)

type Rank int

const (
	Ace Rank = iota + 1
	Two
	Three
	Four
	Five
	Six
	Seven
	Eight
	Nine
	Ten
	Jack
	Queen
	King
)

type Card struct {
	rank    Rank
	suit    Suit
	isJoker bool
}

func cardsEqual(a, b Card) bool {
	return a.rank == b.rank && a.suit == b.suit
}

func (c *Card) Points() int {
	if c.isJoker {
		return 20
	} else if c.rank >= Jack && c.rank <= King {
		return 10
	} else if c.rank == Ace {
		return 10
	} else if c.rank >= Two && c.rank <= Ten {
		return int(c.rank)
	}
	return 0
}

func (r Rank) String() string {
	switch r {
	case Ace:
		return "Ace"
	case Two:
		return "Two"
	case Three:
		return "Three"
	case Four:
		return "Four"
	case Five:
		return "Five"
	case Six:
		return "Six"
	case Seven:
		return "Seven"
	case Eight:
		return "Eight"
	case Nine:
		return "Nine"
	case Ten:
		return "Ten"
	case Jack:
		return "Jack"
	case Queen:
		return "Queen"
	case King:
		return "King"
	default:
		return "Unknown"
	}
}

func (s Suit) String() string {
	switch s {
	case Hearts:
		return "Hearts"
	case Diamonds:
		return "Diamonds"
	case Clubs:
		return "Clubs"
	case Spades:
		return "Spades"
	default:
		return "Unknown"
	}
}

func (c Card) String() string {
	if c.isJoker {
		return "Joker"
	}

	return fmt.Sprintf("{Rank: %s, Suit: %s}", c.rank, c.suit)
}
