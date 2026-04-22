package game

import (
	"math/rand"
	"strings"
)

type CardPile struct {
	cards []Card
}

func (cp *CardPile) DrawOne() (Card, bool) {
	if len(cp.cards) == 0 {
		return Card{}, false
	}

	card := cp.cards[0]
	cp.cards = cp.cards[1:]
	return card, true
}

func (cp *CardPile) AddToTop(c Card) {
	cp.cards = append([]Card{c}, cp.cards...)
}

func (cp *CardPile) Shuffle() {
	rand.Shuffle(len(cp.cards), func(i, j int) {
		cp.cards[i], cp.cards[j] = cp.cards[j], cp.cards[i]
	})
}

func (cp *CardPile) String() string {
	var sb strings.Builder

	for i, card := range cp.cards {
		sb.WriteString(card.String())

		if i != len(cp.cards)-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
