package game

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
	Rank    Rank
	Suit    Suit
	isJoker bool
}

func (c *Card) Points() int {
	if c.isJoker {
		return 20
	} else if c.Rank >= Jack && c.Rank <= King {
		return 10
	} else if c.Rank == Ace {
		return 10
	} else if c.Rank >= Two && c.Rank <= Ten {
		return int(c.Rank)
	}
	return 0
}
