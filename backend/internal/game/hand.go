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

func (h *Hand) RemoveAt(index int) (Card, bool) {
	if index < 0 || index >= len(h.cards) {
		return Card{}, false
	}
	card := h.cards[index]
	h.cards = append(h.cards[:index], h.cards[index+1:]...)

	return card, true
}

func (h *Hand) RemoveCards(cards []Card) bool {
	temp := make([]Card, len(h.cards))
	copy(temp, h.cards)

	for _, target := range cards {
		found := false

		for i, card := range temp {
			if cardsEqual(card, target) {
				temp = append(temp[:i], temp[i+1:]...)
				found = true
				break
			}
		}

		if !found {
			return false
		}
	}

	h.cards = temp
	return true
}
