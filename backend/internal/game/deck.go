package game

import (
	"math/rand"
	"strings"
)

type Deck struct {
	cards []Card
}

const cardsInDeck = 54

func newDeck() *Deck {
	cards := make([]Card, 0, cardsInDeck)

	for suit := Hearts; suit <= Spades; suit++ {
		for rank := Ace; rank <= King; rank++ {
			cards = append(cards, Card{rank: rank, suit: suit})
		}
	}
	cards = append(cards, Card{isJoker: true}, Card{isJoker: true})

	return &Deck{
		cards,
	}
}

func NewGameDeck() *Deck {
	cards := make([]Card, 0, cardsInDeck*2)

	for range 2 {
		cards = append(cards, newDeck().cards...)
	}

	return &Deck{cards}
}

func (d *Deck) Shuffle() {
	rand.Shuffle(len(d.cards), func(i, j int) {
		d.cards[i], d.cards[j] = d.cards[j], d.cards[i]
	})
}

func (d *Deck) String() string {
	var sb strings.Builder

	for i, card := range d.cards {
		sb.WriteString(card.String())

		if i != len(d.cards)-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
