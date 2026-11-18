package game

func (gs *GameState) DrawFromDeck() error {
	if gs.phase != PhaseInProgress {
		return ErrGameNotInProgress
	}
	cp, err := gs.CurrentPlayer()
	if err != nil {
		return err
	}

	if gs.turn.hasDrawn {
		return ErrPlayerAlreadyDrew
	}

	gs.recycleDiscardIntoDrawPileIfNeeded()

	if !cp.hand.Draw(gs.drawPile) {
		return ErrNotEnoughCardsInDrawPile
	}
	cp.statistics.CardsDrawnFromDeck++
	gs.turn.hasDrawn = true
	gs.resetDiscardDrawState()
	return nil
}

func (gs *GameState) DrawFromDiscard() error {
	if gs.phase != PhaseInProgress {
		return ErrGameNotInProgress
	}
	cp, err := gs.CurrentPlayer()
	if err != nil {
		return err
	}

	if gs.turn.hasDrawn {
		return ErrPlayerAlreadyDrew
	}

	if !gs.canTakeDiscardNow() {
		return ErrCannotTakeDiscardCard
	}

	discardCard := gs.discardPile.cards[0]
	gs.discardPile.cards = gs.discardPile.cards[1:]
	cp.hand.cards = append(cp.hand.cards, discardCard)
	cp.statistics.CardsDrawnFromDiscard++

	gs.turn.hasDrawn = true
	gs.turn.mustUseDiscardDraw = true
	gs.turn.discardDrawCard = discardCard
	return nil
}

func (gs *GameState) PlayCompositions(comps []*Composition) error {
	return gs.PlayTable(comps, nil)
}

func (gs *GameState) AddToCompositions(additions []CompositionAddition) error {
	return gs.PlayTable(nil, additions)
}

func (gs *GameState) PlayTable(comps []*Composition, additions []CompositionAddition, reclaims ...JokerReclaim) error {
	if gs.phase != PhaseInProgress {
		return ErrGameNotInProgress
	}
	if !gs.turn.hasDrawn {
		return ErrPlayerHasntDrawn
	}
	if len(comps) == 0 && len(additions) == 0 && len(reclaims) == 0 {
		return ErrInvalidComposition
	}

	cp, err := gs.CurrentPlayer()
	if err != nil {
		return err
	}

	nextState, err := applyTablePlayState(tablePlayState{
		handCards:          cp.hand.cards,
		activeCompositions: gs.activeCompositions,
		hasOpened:          cp.hasOpened,
	}, comps, additions, reclaims)
	if err != nil {
		return err
	}
	// Table plays are committed per request, so the discard obligation must be
	// satisfied before any supporting cards can be committed elsewhere.
	if gs.turn.mustUseDiscardDraw && !tablePlayUsesCard(comps, additions, reclaims, gs.turn.discardDrawCard) {
		return ErrMustUseDrawnDiscardCard
	}

	wasOpened := cp.hasOpened
	cp.hand.cards = nextState.handCards
	gs.activeCompositions = nextState.activeCompositions
	cp.hasOpened = nextState.hasOpened
	cp.statistics.CompositionsCreated += len(comps)
	cp.statistics.AdditionsDone += len(additions)
	cp.statistics.JokersReclaimed += len(reclaims)
	for _, comp := range comps {
		cp.statistics.CardsPlayed += len(comp.cards)
		cp.statistics.JokersPlayed += countJokers(comp.cards)
		switch comp.variant {
		case set:
			cp.statistics.SetsCreated++
		case run:
			cp.statistics.RunsCreated++
		}
	}
	for _, addition := range additions {
		cp.statistics.CardsPlayed += len(addition.Cards)
		cp.statistics.JokersPlayed += countJokers(addition.Cards)
	}
	cp.statistics.CardsPlayed += len(reclaims)
	if !wasOpened && cp.hasOpened {
		cp.statistics.RoundsOpened++
		if cp.statistics.FastestOpeningTurn == 0 || gs.turn.number < cp.statistics.FastestOpeningTurn {
			cp.statistics.FastestOpeningTurn = gs.turn.number
		}
	}
	gs.resetDiscardDrawState()

	return nil
}

func (gs *GameState) PlayTableAndDiscard(cardIndex int, comps []*Composition, additions []CompositionAddition, reclaims ...JokerReclaim) error {
	return gs.PlayTableAndDiscardMatching(cardIndex, nil, comps, additions, reclaims...)
}

// PlayTableAndDiscardMatching commits a table play and discards expectedCard.
// cardIndex is kept as a fast path and for backwards compatibility, but the
// card value is authoritative when one is supplied. This prevents a stale
// client-side hand index from discarding a different card after table cards
// have been removed.
func (gs *GameState) PlayTableAndDiscardMatching(cardIndex int, expectedCard *Card, comps []*Composition, additions []CompositionAddition, reclaims ...JokerReclaim) error {
	next := gs.Clone()
	if err := next.PlayTable(comps, additions, reclaims...); err != nil {
		return err
	}
	if err := next.DiscardFromHandMatching(cardIndex, expectedCard); err != nil {
		return err
	}

	*gs = *next
	return nil
}

func (gs *GameState) DiscardFromHand(cardIndex int) error {
	return gs.DiscardFromHandMatching(cardIndex, nil)
}

// DiscardFromHandMatching discards the requested card even when cardIndex is
// stale. If expectedCard is nil, it preserves the legacy index-only behavior.
func (gs *GameState) DiscardFromHandMatching(cardIndex int, expectedCard *Card) error {
	if gs.phase != PhaseInProgress {
		return ErrGameNotInProgress
	}
	if !gs.turn.hasDrawn {
		return ErrPlayerHasntDrawn
	}
	if gs.turn.mustUseDiscardDraw {
		return ErrMustUseDrawnDiscardCard
	}

	cp, err := gs.CurrentPlayer()
	if err != nil {
		return err
	}
	if expectedCard != nil {
		matchesExpected := func(card Card) bool {
			return cardsEqual(card, *expectedCard)
		}

		if cardIndex < 0 || cardIndex >= len(cp.hand.cards) || !matchesExpected(cp.hand.cards[cardIndex]) {
			cardIndex = -1
			for index, card := range cp.hand.cards {
				if matchesExpected(card) {
					cardIndex = index
					break
				}
			}
		}
	}

	card, ok := cp.hand.RemoveAt(cardIndex)
	if !ok {
		return ErrRemovingCard
	}

	gs.removeCompletedCompositionsToDiscard()
	gs.discardPile.AddToTop(card)
	cp.statistics.TurnsTaken++
	cp.statistics.CardsDiscarded++
	if len(cp.hand.cards) == 0 {
		gs.finishRound(gs.turn.playerIndex)
		return nil
	}
	if gs.finishRoundIfSpecialWin(gs.turn.playerIndex) {
		return nil
	}
	gs.advanceTurn()
	return nil
}

func (gs *GameState) advanceTurn() {
	gs.turn.number++
	gs.turn.playerIndex = gs.nextActivePlayerIndex(gs.turn.playerIndex)
	gs.turn.hasDrawn = false
	gs.resetDiscardDrawState()
	gs.recycleDiscardIntoDrawPileIfNeeded()
}

func (gs *GameState) resetDiscardDrawState() {
	gs.turn.mustUseDiscardDraw = false
	gs.turn.discardDrawCard = Card{}
}

func (gs *GameState) recycleDiscardIntoDrawPileIfNeeded() {
	if len(gs.drawPile.cards) != 0 || len(gs.discardPile.cards) <= 1 {
		return
	}

	recycled := make([]Card, len(gs.discardPile.cards)-1)
	for i := range recycled {
		recycled[i] = gs.discardPile.cards[len(gs.discardPile.cards)-1-i]
	}

	gs.drawPile.cards = recycled
	gs.discardPile.cards = gs.discardPile.cards[:1]
}

func (gs *GameState) removeCompletedCompositionsToDiscard() {
	remaining := make([]*Composition, 0, len(gs.activeCompositions))
	var current *Player
	if isValidPlayerIndex(gs.turn.playerIndex, len(gs.players)) {
		current = gs.players[gs.turn.playerIndex]
	}
	for _, comp := range gs.activeCompositions {
		if comp == nil || !comp.isComplete() {
			remaining = append(remaining, comp)
			continue
		}

		for i := len(comp.cards) - 1; i >= 0; i-- {
			gs.discardPile.AddToTop(comp.cards[i])
		}
		if current != nil {
			current.statistics.CompositionsCompleted++
			switch comp.variant {
			case set:
				current.statistics.SetsCompleted++
			case run:
				current.statistics.RunsCompleted++
			}
		}
	}

	gs.activeCompositions = remaining
}
