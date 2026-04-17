package game

type Hand struct {
	cards []Card
}

func NewHand() *Hand {
	return &Hand{cards: []Card{}}
}

func (h *Hand) Draw(d *Deck) bool {
	if len(d.cards) == 0 {
		return false
	}
	dc := d.cards[len(d.cards)-1]
	d.cards = d.cards[:len(d.cards)-1]
	h.cards = append(h.cards, dc)
	return true
}

func (h *Hand) Points() int {
	if len(h.cards) == 1 && h.cards[0].Rank == Ace {
		return 1
	}

	total := 0

	for _, c := range h.cards {
		total += c.Points()
	}

	return total
}
