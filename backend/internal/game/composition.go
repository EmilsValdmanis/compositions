package game

import "slices"

type compositionVariant string

const (
	set compositionVariant = "set"
	run compositionVariant = "run"
)

type Composition struct {
	variant compositionVariant
	cards   []Card
}

func NewComposition(cards []Card, variant compositionVariant) (*Composition, bool) {
	c := &Composition{variant: variant, cards: cards}
	if !c.isValid() {
		return nil, false
	}
	return c, true
}

func NewSet(cards []Card) (*Composition, bool) {
	return NewComposition(cards, set)
}

func NewRun(cards []Card) (*Composition, bool) {
	return NewComposition(cards, run)
}

func (c *Composition) isValid() bool {
	switch c.variant {
	case set:
		return c.isValidSet()
	case run:
		return c.isValidRun()
	default:
		return false
	}
}

func (c *Composition) isValidSet() bool {
	cardCount := len(c.cards)
	if cardCount < 3 || cardCount > 4 {
		return false
	}

	var realCards []Card
	var jokerCount int

	for _, card := range c.cards {
		if card.isJoker {
			jokerCount++
		} else {
			realCards = append(realCards, card)
		}
	}

	if len(realCards) > 0 {
		firstRank := realCards[0].rank
		for _, card := range realCards[1:] {
			if card.rank != firstRank {
				return false
			}
		}

		seenSuits := make(map[Suit]bool)
		for _, card := range realCards {
			if seenSuits[card.suit] {
				return false
			}
			seenSuits[card.suit] = true
		}
	}

	missingSlots := cardCount - len(realCards)
	if jokerCount < missingSlots {
		return false
	}

	return true
}

func (c *Composition) isValidRun() bool {
	if len(c.cards) < 3 || len(c.cards) > 14 {
		return false
	}

	var realCards []Card
	var jokerCount int

	for _, card := range c.cards {
		if card.isJoker {
			jokerCount++
		} else {
			realCards = append(realCards, card)
		}
	}

	if len(realCards) > 0 {
		firstSuit := realCards[0].suit
		for _, card := range realCards[1:] {
			if card.suit != firstSuit {
				return false
			}
		}
	}

	return tryFitSequence(realCards, jokerCount, false) || tryFitSequence(realCards, jokerCount, true)
}

func tryFitSequence(realCards []Card, jokerCount int, aceLow bool) bool {
	if len(realCards) == 0 {
		return jokerCount >= 3 && jokerCount <= 14
	}

	ranks := make([]int, len(realCards))
	aceCount := 0
	for _, card := range realCards {
		if card.rank == Ace {
			aceCount++
		}
	}

	aceAssignedLow := false
	for i, card := range realCards {
		if card.rank == Ace {
			if aceCount == 2 && !aceAssignedLow {
				ranks[i] = 1
				aceAssignedLow = true
			} else if aceLow {
				ranks[i] = 1
			} else {
				ranks[i] = 14
			}
		} else {
			ranks[i] = int(card.rank)
		}
	}

	slices.Sort(ranks)

	for i := 1; i < len(ranks); i++ {
		if ranks[i] == ranks[i-1] {
			return false
		}
	}

	jokersNeeded := 0
	for i := 1; i < len(ranks); i++ {
		diff := ranks[i] - ranks[i-1]
		jokersNeeded += diff - 1
	}

	if jokerCount < jokersNeeded {
		return false
	}

	extraJokers := jokerCount - jokersNeeded
	maxExtension := (ranks[0] - 1) + (14 - ranks[len(ranks)-1])

	return extraJokers <= maxExtension
}
