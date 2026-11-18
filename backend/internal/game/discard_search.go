package game

import "math/bits"

type tablePlayCandidate struct {
	usedMask    uint32
	reclaim     *JokerReclaim
	usesDiscard bool
}

type searchScratch struct {
	maskCards   []Card
	combinedBuf []Card
}

type selectedAddition struct {
	compositionIndex int
	insertIndex      *int
	mask             uint32
}

func (gs *GameState) canTakeDiscardNow() bool {
	if len(gs.discardPile.cards) == 0 {
		return false
	}

	baseState, ok := gs.discardSearchBaseState()
	if !ok {
		return false
	}

	availableCards := baseState.handCards
	discardMask := uint32(1) << uint(len(availableCards)-1)
	scratch := searchScratch{
		maskCards:   make([]Card, 0, len(availableCards)),
		combinedBuf: make([]Card, 0, 14),
	}

	if hasLegalPlayWithDiscard(baseState, discardMask, scratch, nil) {
		return true
	}

	for compositionIndex, comp := range gs.activeCompositions {
		if comp == nil {
			continue
		}

		for jokerIndex, tableCard := range comp.cards {
			if !tableCard.isJoker {
				continue
			}

			for cardIndex, replacementCard := range availableCards {
				if !comp.canReclaimJoker(jokerIndex, replacementCard) {
					continue
				}

				candidateState := baseState
				candidateDiscardMask := discardMask
				usedMask := uint32(1) << uint(cardIndex)
				usesDiscard := cardIndex == len(availableCards)-1
				candidateScratch := scratch
				if !usesDiscard {
					candidateCards := make([]Card, 0, len(availableCards)+1)
					candidateCards = append(candidateCards, availableCards[:len(availableCards)-1]...)
					candidateCards = append(candidateCards, tableCard, availableCards[len(availableCards)-1])
					candidateState.handCards = candidateCards
					candidateState.syntheticReclaims = 1
					candidateDiscardMask = uint32(1) << uint(len(candidateCards)-1)
					candidateScratch.maskCards = make([]Card, 0, len(candidateCards))
				}

				reclaim := tablePlayCandidate{
					usedMask:    usedMask,
					reclaim:     &JokerReclaim{CompositionIndex: compositionIndex, JokerIndex: jokerIndex, ReplacementCard: replacementCard},
					usesDiscard: usesDiscard,
				}
				if hasLegalPlayWithDiscard(candidateState, candidateDiscardMask, candidateScratch, &reclaim) {
					return true
				}
			}
		}
	}

	return false
}

func cardsForMask(availableCards []Card, mask uint32, buf []Card) []Card {
	cards := buf[:0]
	for i, card := range availableCards {
		if mask&(uint32(1)<<uint(i)) == 0 {
			continue
		}
		cards = append(cards, card)
	}
	return cards
}

func hasLegalPlayWithDiscard(baseState tablePlayState, discardMask uint32, scratch searchScratch, reclaimCandidate *tablePlayCandidate) bool {
	selectedCompMasks := make([]uint32, 0, len(baseState.handCards))
	selectedCompVariants := make([]compositionVariant, 0, len(baseState.handCards))
	selectedAdditions := make([]selectedAddition, 0, len(baseState.handCards))
	selectedReclaims := make([]JokerReclaim, 0, 1)
	usedMask := uint32(0)
	hasDiscardPlay := false

	if reclaimCandidate != nil {
		usedMask = reclaimCandidate.usedMask
		hasDiscardPlay = reclaimCandidate.usesDiscard
		selectedReclaims = append(selectedReclaims, *reclaimCandidate.reclaim)
		if validateTablePlay(baseState, selectedCompMasks, selectedCompVariants, selectedAdditions, selectedReclaims, usedMask, hasDiscardPlay, scratch) {
			return true
		}
	}

	if hasDiscardPlay {
		return searchSupportCandidates(baseState, discardMask, 1, usedMask, selectedCompMasks, selectedCompVariants, selectedAdditions, selectedReclaims, scratch)
	}

	for subset := range discardMask {
		mask := subset | discardMask
		if usedMask&mask != 0 {
			continue
		}
		nextUsedMask := usedMask | mask
		cards := cardsForMask(baseState.handCards, mask, scratch.maskCards)

		if bits.OnesCount32(subset) >= 2 {
			for _, variant := range []compositionVariant{set, run} {
				if !isSearchCompositionValid(cards, variant) {
					continue
				}

				selectedCompMasks = append(selectedCompMasks, mask)
				selectedCompVariants = append(selectedCompVariants, variant)
				if validateTablePlay(baseState, selectedCompMasks, selectedCompVariants, selectedAdditions, selectedReclaims, nextUsedMask, true, scratch) {
					return true
				}
				if !baseState.hasOpened && searchSupportCandidates(baseState, discardMask, 1, nextUsedMask, selectedCompMasks, selectedCompVariants, selectedAdditions, selectedReclaims, scratch) {
					return true
				}
				selectedCompMasks = selectedCompMasks[:len(selectedCompMasks)-1]
				selectedCompVariants = selectedCompVariants[:len(selectedCompVariants)-1]
			}
		}

		for compositionIndex, target := range baseState.activeCompositions {
			if target == nil {
				continue
			}

			for insertIndex := 0; insertIndex <= len(target.cards); insertIndex++ {
				insertIndex := insertIndex
				if !canInsertCardsIntoComposition(target, insertIndex, cards, scratch.combinedBuf) {
					continue
				}

				selectedAdditions = append(selectedAdditions, selectedAddition{compositionIndex: compositionIndex, insertIndex: &insertIndex, mask: mask})
				if validateTablePlay(baseState, selectedCompMasks, selectedCompVariants, selectedAdditions, selectedReclaims, nextUsedMask, true, scratch) {
					return true
				}
				if !baseState.hasOpened && searchSupportCandidates(baseState, discardMask, 1, nextUsedMask, selectedCompMasks, selectedCompVariants, selectedAdditions, selectedReclaims, scratch) {
					return true
				}
				selectedAdditions = selectedAdditions[:len(selectedAdditions)-1]
			}
		}
	}

	return false
}

func searchSupportCandidates(baseState tablePlayState, discardMask uint32, startMask uint32, usedMask uint32, selectedCompMasks []uint32, selectedCompVariants []compositionVariant, selectedAdditions []selectedAddition, selectedReclaims []JokerReclaim, scratch searchScratch) bool {
	for mask := startMask; mask < discardMask; mask++ {
		if usedMask&mask != 0 {
			continue
		}
		cards := cardsForMask(baseState.handCards, mask, scratch.maskCards)
		nextUsedMask := usedMask | mask

		if bits.OnesCount32(mask) >= 3 {
			for _, variant := range []compositionVariant{set, run} {
				if !isSearchCompositionValid(cards, variant) {
					continue
				}

				selectedCompMasks = append(selectedCompMasks, mask)
				selectedCompVariants = append(selectedCompVariants, variant)
				if validateTablePlay(baseState, selectedCompMasks, selectedCompVariants, selectedAdditions, selectedReclaims, nextUsedMask, true, scratch) {
					return true
				}

				if searchSupportCandidates(baseState, discardMask, mask+1, nextUsedMask, selectedCompMasks, selectedCompVariants, selectedAdditions, selectedReclaims, scratch) {
					return true
				}
				selectedCompMasks = selectedCompMasks[:len(selectedCompMasks)-1]
				selectedCompVariants = selectedCompVariants[:len(selectedCompVariants)-1]
			}
		}

		for compositionIndex, target := range baseState.activeCompositions {
			if target == nil {
				continue
			}

			for insertIndex := 0; insertIndex <= len(target.cards); insertIndex++ {
				insertIndex := insertIndex
				if !canInsertCardsIntoComposition(target, insertIndex, cards, scratch.combinedBuf) {
					continue
				}

				selectedAdditions = append(selectedAdditions, selectedAddition{compositionIndex: compositionIndex, insertIndex: &insertIndex, mask: mask})
				if validateTablePlay(baseState, selectedCompMasks, selectedCompVariants, selectedAdditions, selectedReclaims, nextUsedMask, true, scratch) {
					return true
				}

				if searchSupportCandidates(baseState, discardMask, mask+1, nextUsedMask, selectedCompMasks, selectedCompVariants, selectedAdditions, selectedReclaims, scratch) {
					return true
				}
				selectedAdditions = selectedAdditions[:len(selectedAdditions)-1]
			}
		}
	}

	return false
}

func isSearchCompositionValid(cards []Card, variant compositionVariant) bool {
	comp := Composition{variant: variant, cards: cards}
	return comp.isValid()
}

func canInsertCardsIntoComposition(comp *Composition, insertIndex int, cards []Card, combinedBuf []Card) bool {
	if insertIndex < 0 || insertIndex > len(comp.cards) {
		return false
	}

	combined := combinedBuf[:0]
	combined = append(combined, comp.cards[:insertIndex]...)
	combined = append(combined, cards...)
	combined = append(combined, comp.cards[insertIndex:]...)
	extended := Composition{variant: comp.variant, cards: combined}
	return extended.isValid()
}

func validateTablePlay(baseState tablePlayState, compMasks []uint32, compVariants []compositionVariant, additions []selectedAddition, reclaims []JokerReclaim, usedMask uint32, hasDiscardPlay bool, scratch searchScratch) bool {
	if !hasDiscardPlay {
		return false
	}
	if len(compMasks) == 0 && len(additions) == 0 && len(reclaims) == 0 {
		return false
	}

	openingPoints := 0
	for i, mask := range compMasks {
		cards := cardsForMask(baseState.handCards, mask, scratch.maskCards)
		comp, ok := NewComposition(cards, compVariants[i])
		if !ok {
			return false
		}
		openingPoints += comp.Points()
	}

	var updatedIndices [32]int
	var updatedComps [32]*Composition
	updatedCount := 0

	currentComposition := func(index int) *Composition {
		for i := 0; i < updatedCount; i++ {
			if updatedIndices[i] == index {
				return updatedComps[i]
			}
		}
		return baseState.activeCompositions[index]
	}

	storeComposition := func(index int, comp *Composition) {
		for i := 0; i < updatedCount; i++ {
			if updatedIndices[i] == index {
				updatedComps[i] = comp
				return
			}
		}
		updatedIndices[updatedCount] = index
		updatedComps[updatedCount] = comp
		updatedCount++
	}

	deferredReclaims := make([]deferredJokerReclaim, 0, len(reclaims))
	for _, reclaim := range reclaims {
		if reclaim.CompositionIndex < 0 || reclaim.CompositionIndex >= len(baseState.activeCompositions) {
			return false
		}

		target := currentComposition(reclaim.CompositionIndex)
		if target == nil {
			return false
		}

		if !target.canReclaimJoker(reclaim.JokerIndex, reclaim.ReplacementCard) {
			if isAmbiguousSetReclaim(target, reclaim) {
				deferredReclaims = append(deferredReclaims, deferredJokerReclaim{reclaim: reclaim, jokerIndex: reclaim.JokerIndex})
				continue
			}
			return false
		}
		updated, _ := target.ReclaimJoker(reclaim.JokerIndex, reclaim.ReplacementCard)
		storeComposition(reclaim.CompositionIndex, updated)
	}

	for _, addition := range additions {
		if addition.compositionIndex < 0 || addition.compositionIndex >= len(baseState.activeCompositions) {
			return false
		}

		cards := cardsForMask(baseState.handCards, addition.mask, scratch.maskCards)
		if len(cards) == 0 {
			return false
		}

		target := currentComposition(addition.compositionIndex)
		if target == nil {
			return false
		}

		insertIndex := len(target.cards)
		if addition.insertIndex != nil {
			insertIndex = *addition.insertIndex
		}
		for index := range deferredReclaims {
			pending := &deferredReclaims[index]
			if pending.reclaim.CompositionIndex == addition.compositionIndex && insertIndex <= pending.jokerIndex {
				pending.jokerIndex += len(cards)
			}
		}

		extended, ok := target.WithInsertedCards(insertIndex, cards)
		if !ok {
			return false
		}
		openingPoints += extended.Points() - target.Points()
		storeComposition(addition.compositionIndex, extended)
	}

	for _, pending := range deferredReclaims {
		target := currentComposition(pending.reclaim.CompositionIndex)
		if target == nil || !target.canReclaimJoker(pending.jokerIndex, pending.reclaim.ReplacementCard) {
			return false
		}
		updated, _ := target.ReclaimJoker(pending.jokerIndex, pending.reclaim.ReplacementCard)
		storeComposition(pending.reclaim.CompositionIndex, updated)
	}

	if !baseState.hasOpened && len(compMasks) == 0 {
		return false
	}
	if !baseState.hasOpened && openingPoints < minimumOpeningPoints {
		return false
	}
	if len(baseState.handCards)-bits.OnesCount32(usedMask)+len(reclaims)-baseState.syntheticReclaims == 0 {
		return false
	}

	return true
}

func (gs *GameState) discardSearchBaseState() (tablePlayState, bool) {
	cp, err := gs.CurrentPlayer()
	if err != nil {
		return tablePlayState{}, false
	}

	handCards := make([]Card, 0, len(cp.hand.cards)+1)
	handCards = append(handCards, cp.hand.cards...)
	handCards = append(handCards, gs.discardPile.cards[0])

	return tablePlayState{
		handCards:          handCards,
		activeCompositions: gs.activeCompositions,
		hasOpened:          cp.hasOpened,
	}, true
}
