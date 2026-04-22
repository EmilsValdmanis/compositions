package game

const cardsInDeck = 54

func newDeck() *CardPile {
	cards := make([]Card, 0, cardsInDeck)

	for suit := Hearts; suit <= Spades; suit++ {
		for rank := Ace; rank <= King; rank++ {
			cards = append(cards, Card{rank: rank, suit: suit})
		}
	}
	cards = append(cards, Card{isJoker: true}, Card{isJoker: true})

	return &CardPile{
		cards,
	}
}

func NewGameDeck() *CardPile {
	cards := make([]Card, 0, cardsInDeck*2)

	for range 2 {
		cards = append(cards, newDeck().cards...)
	}

	return &CardPile{cards}
}
