package game

import "slices"

type DealTypes int

const (
	DealRoundRobin = iota
	DealInBlocks
)

func planDeal(deck []Card, seats []int, dealer int, dealType DealTypes, order []int, cutSize int) (map[int][]Card, []Card, error) {
	cardsToDeal := InitialHandSize * len(seats)
	if len(deck) <= cardsToDeal {
		return nil, nil, ErrNotEnoughCardsInDrawPile
	}

	cardsPerDeal := 1
	switch dealType {
	case DealRoundRobin:
		first := slices.Index(seats, dealer) + 1
		order = append(slices.Clone(seats[first:]), seats[:first]...)
	case DealInBlocks:
		if !validDealOrder(order, seats) {
			return nil, nil, ErrInvalidDealingOrder
		}
		cardsPerDeal = InitialHandSize
	default:
		return nil, nil, ErrInvalidDealingType
	}

	if cutSize < 0 || cutSize > len(deck)-cardsToDeal {
		return nil, nil, ErrInvalidCutSize
	}

	hands := make(map[int][]Card, len(seats))
	cards := deck[cutSize : cutSize+cardsToDeal]
	for index := 0; index < len(cards); index += cardsPerDeal {
		seat := order[(index/cardsPerDeal)%len(order)]
		hands[seat] = append(hands[seat], cards[index:index+cardsPerDeal]...)
	}
	remaining := append(slices.Clone(deck[cutSize+cardsToDeal:]), deck[:cutSize]...)
	return hands, remaining, nil
}

func validDealOrder(order, seats []int) bool {
	if len(order) != len(seats) {
		return false
	}
	remaining := slices.Clone(seats)
	for _, seat := range order {
		index := slices.Index(remaining, seat)
		if index < 0 {
			return false
		}
		remaining = slices.Delete(remaining, index, index+1)
	}
	return true
}
