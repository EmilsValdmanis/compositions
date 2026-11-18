package game

const minimumOpeningPoints = 40

type CompositionAddition struct {
	CompositionIndex int
	InsertIndex      *int
	Cards            []Card
}

type JokerReclaim struct {
	CompositionIndex int
	JokerIndex       int
	ReplacementCard  Card
}

type tablePlayState struct {
	handCards          []Card
	activeCompositions []*Composition
	hasOpened          bool
	syntheticReclaims  int
}

func tablePlayUsesCard(comps []*Composition, additions []CompositionAddition, reclaims []JokerReclaim, target Card) bool {
	for _, comp := range comps {
		if comp == nil {
			continue
		}
		for _, playedCard := range comp.cards {
			if cardsEqual(playedCard, target) {
				return true
			}
		}
	}

	for _, addition := range additions {
		for _, playedCard := range addition.Cards {
			if cardsEqual(playedCard, target) {
				return true
			}
		}
	}

	for _, reclaim := range reclaims {
		if cardsEqual(reclaim.ReplacementCard, target) {
			return true
		}
	}

	return false
}

func applyTablePlayState(state tablePlayState, comps []*Composition, additions []CompositionAddition, reclaims []JokerReclaim) (tablePlayState, error) {
	playedCards := make([]Card, 0)
	reclaimedCards := make([]Card, 0, len(reclaims))
	openingPoints := 0
	for _, comp := range comps {
		if comp == nil {
			return tablePlayState{}, ErrInvalidComposition
		}
		if !comp.isValid() {
			return tablePlayState{}, ErrInvalidComposition
		}
		if comp.variant == run {
			if !runCardsAreOrdered(comp.cards) && !runCardsAreReverseOrdered(comp.cards) {
				return tablePlayState{}, ErrInvalidComposition
			}
			_ = comp.normalizeRunCards()
		}
		playedCards = append(playedCards, comp.cards...)
		openingPoints += comp.Points()
	}

	updatedCompositions := make([]*Composition, len(state.activeCompositions))
	copy(updatedCompositions, state.activeCompositions)

	deferredReclaims := make([]deferredJokerReclaim, 0, len(reclaims))
	for _, reclaim := range reclaims {
		if reclaim.CompositionIndex < 0 || reclaim.CompositionIndex >= len(updatedCompositions) {
			return tablePlayState{}, ErrInvalidComposition
		}

		target := updatedCompositions[reclaim.CompositionIndex]
		if target == nil {
			return tablePlayState{}, ErrInvalidComposition
		}

		updated, ok := target.ReclaimJoker(reclaim.JokerIndex, reclaim.ReplacementCard)
		if !ok {
			if isAmbiguousSetReclaim(target, reclaim) {
				deferredReclaims = append(deferredReclaims, deferredJokerReclaim{reclaim: reclaim, jokerIndex: reclaim.JokerIndex})
				continue
			}
			return tablePlayState{}, ErrInvalidComposition
		}

		reclaimedCards = append(reclaimedCards, target.cards[reclaim.JokerIndex])
		playedCards = append(playedCards, reclaim.ReplacementCard)
		updatedCompositions[reclaim.CompositionIndex] = updated
	}

	for _, addition := range additions {
		if len(addition.Cards) == 0 {
			return tablePlayState{}, ErrInvalidComposition
		}
		if addition.CompositionIndex < 0 || addition.CompositionIndex >= len(updatedCompositions) {
			return tablePlayState{}, ErrInvalidComposition
		}

		target := updatedCompositions[addition.CompositionIndex]
		if target == nil {
			return tablePlayState{}, ErrInvalidComposition
		}

		insertIndex := len(target.cards)
		if addition.InsertIndex != nil {
			insertIndex = *addition.InsertIndex
		}
		for index := range deferredReclaims {
			pending := &deferredReclaims[index]
			if pending.reclaim.CompositionIndex == addition.CompositionIndex && insertIndex <= pending.jokerIndex {
				pending.jokerIndex += len(addition.Cards)
			}
		}

		extended, ok := target.WithInsertedCards(insertIndex, addition.Cards)
		if !ok {
			return tablePlayState{}, ErrInvalidComposition
		}

		updatedCompositions[addition.CompositionIndex] = extended
		playedCards = append(playedCards, addition.Cards...)
		openingPoints += extended.Points() - target.Points()
	}

	for _, pending := range deferredReclaims {
		target := updatedCompositions[pending.reclaim.CompositionIndex]
		updated, ok := target.ReclaimJoker(pending.jokerIndex, pending.reclaim.ReplacementCard)
		if !ok {
			return tablePlayState{}, ErrInvalidComposition
		}

		reclaimedCards = append(reclaimedCards, target.cards[pending.jokerIndex])
		playedCards = append(playedCards, pending.reclaim.ReplacementCard)
		updatedCompositions[pending.reclaim.CompositionIndex] = updated
	}

	if !state.hasOpened {
		if !hasOpeningCompositionFromHand(state.handCards, comps) {
			return tablePlayState{}, ErrInitialPlayRequiresOwnComp
		}
		if openingPoints < minimumOpeningPoints {
			return tablePlayState{}, ErrInitialPointsNotMet
		}
	}

	nextHand := &Hand{cards: make([]Card, 0, len(state.handCards)+len(reclaimedCards))}
	nextHand.cards = append(nextHand.cards, state.handCards...)
	nextHand.cards = append(nextHand.cards, reclaimedCards...)
	if !nextHand.RemoveCards(playedCards) {
		return tablePlayState{}, ErrCardsNotInHand
	}
	if len(nextHand.cards) == 0 {
		return tablePlayState{}, ErrMustKeepDiscardCard
	}

	updatedCompositions = append(updatedCompositions, comps...)
	return tablePlayState{
		handCards:          nextHand.cards,
		activeCompositions: updatedCompositions,
		hasOpened:          true,
	}, nil
}

type deferredJokerReclaim struct {
	reclaim    JokerReclaim
	jokerIndex int
}

func isAmbiguousSetReclaim(comp *Composition, reclaim JokerReclaim) bool {
	if comp == nil || comp.variant != set || reclaim.JokerIndex < 0 || reclaim.JokerIndex >= len(comp.cards) || !comp.cards[reclaim.JokerIndex].isJoker || reclaim.ReplacementCard.isJoker {
		return false
	}

	representations, ok := comp.JokerRepresentations(reclaim.JokerIndex)
	if !ok || len(representations) <= 1 {
		return false
	}
	for _, representation := range representations {
		if cardsEqual(representation, reclaim.ReplacementCard) {
			return true
		}
	}
	return false
}

func hasOpeningCompositionFromHand(handCards []Card, compositions []*Composition) bool {
	hand := &Hand{cards: handCards}
	for _, composition := range compositions {
		if hand.RemoveCards(composition.cards) {
			return true
		}
	}

	return false
}
