package game

type Deck struct {
	cards []Card
}

func NewDeck() *Deck {
	cards := make([]Card, 0, 54)

	for suit := Hearts; suit <= Spades; suit++ {
		for rank := Ace; rank <= King; rank++ {
			cards = append(cards, Card{Suit: suit, Rank: rank})
		}
	}
	cards = append(cards, Card{isJoker: true}, Card{isJoker: true})

	return &Deck{
		cards,
	}
}
