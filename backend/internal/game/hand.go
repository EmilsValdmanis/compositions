package game

type Hand struct {
	cards []Card
}

func NewHand() *Hand {
	return &Hand{cards: []Card{}}
}

const InitialHandSize = 12

func (h *Hand) Draw(cp *CardPile) bool {
	card, ok := cp.DrawOne()
	if !ok {
		return false
	}

	h.cards = append(h.cards, card)
	return true
}

func (h *Hand) Points() int {
	if len(h.cards) == 1 && h.cards[0].rank == Ace {
		return 1
	}

	total := 0

	for _, c := range h.cards {
		total += c.Points()
	}

	return total
}
