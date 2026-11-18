package game

func (gs *GameState) StartGame(dealerIndex, chooserIndex int, dt DealTypes, order []int, cutSize int) error {
	return gs.StartGameWithMode(dealerIndex, chooserIndex, dt, order, cutSize, GameModeFull)
}

func (gs *GameState) StartGameWithMode(dealerIndex, chooserIndex int, dt DealTypes, order []int, cutSize int, mode GameMode) error {
	if gs.phase != PhaseLobby {
		return ErrGameInProgress
	}
	if !mode.Valid() {
		return ErrInvalidGameMode
	}

	deck := gs.drawPile
	if deck == nil {
		deck = newShuffledGameDeck()
	}
	return gs.startRound(deck, gs.round, mode, dealerIndex, chooserIndex, dt, order, cutSize)
}

func (gs *GameState) StartNextRound(dt DealTypes, order []int, cutSize int) error {
	if gs.phase != PhaseRoundOver {
		return ErrCannotStartNextRound
	}

	dealerIndex, chooserIndex, err := gs.NextRoundDealerAndChooser()
	if err != nil {
		return err
	}
	return gs.startRound(newShuffledGameDeck(), gs.round+1, gs.gameMode, dealerIndex, chooserIndex, dt, order, cutSize)
}

func (gs *GameState) startRound(deck *CardPile, round int, mode GameMode, dealerIndex, chooserIndex int, dt DealTypes, order []int, cutSize int) error {
	activeIndexes := gs.ActivePlayerIndexes()
	if len(activeIndexes) < 2 {
		return ErrNotEnoughPlayers
	}
	if !gs.isActivePlayerIndex(dealerIndex) {
		return ErrInvalidDealer
	}
	if chooserIndex != gs.previousActivePlayerIndex(dealerIndex) {
		return ErrInvalidDealChooser
	}

	if dt == DealInBlocks && order == nil {
		return ErrInvalidDealingOrder
	}
	hands, remaining, err := planDeal(deck.cards, activeIndexes, dealerIndex, dt, order, cutSize)
	if err != nil {
		return err
	}

	gs.resetRoundState(&CardPile{cards: remaining})
	gs.dealerIndex = dealerIndex
	gs.round = round
	gs.gameMode = mode
	for index, cards := range hands {
		player := gs.players[index]
		player.hand.cards = cards
		player.statistics.RoundsPlayed++
	}
	card, _ := gs.drawPile.DrawOne()
	gs.discardPile.AddToTop(card)
	for i, player := range gs.players {
		if player != nil && !player.forfeited && hasSpecialWinningHand(player.hand.cards) {
			gs.finishRound(i)
			return nil
		}
	}
	gs.turn.playerIndex = gs.nextActivePlayerIndex(gs.dealerIndex)
	gs.phase = PhaseInProgress
	return nil
}

func (gs *GameState) resetRoundState(drawPile *CardPile) {
	gs.activeCompositions = []*Composition{}
	gs.drawPile = drawPile
	gs.discardPile = &CardPile{cards: make([]Card, 0, cardsInDeck*2)}
	gs.phase = PhaseLobby
	gs.roundWinnerIndex = -1
	gs.turn = Turn{
		number:      1,
		playerIndex: 0,
	}

	for _, player := range gs.players {
		if player == nil {
			continue
		}

		player.hand = NewHand()
		player.pointsGained = 0
		player.unadjustedTotalPoints = player.totalPoints
		player.hasOpened = false
	}
}

func (gs *GameState) finishRoundIfSpecialWin(playerIndex int) bool {
	if !isValidPlayerIndex(playerIndex, len(gs.players)) {
		return false
	}

	player := gs.players[playerIndex]
	if player == nil || !hasSpecialWinningHand(player.hand.cards) {
		return false
	}

	gs.finishRound(playerIndex)
	return true
}

func (gs *GameState) finishRound(winnerIndex int) {
	gs.roundWinnerIndex = winnerIndex
	gs.turn.hasDrawn = false
	gs.resetDiscardDrawState()

	pointsInflicted := 0
	for i, player := range gs.players {
		if player != nil && player.statistics.RoundsPlayed >= gs.round {
			player.statistics.recordRoundResult(gs.round, i == winnerIndex)
		}
		if i == winnerIndex || player == nil || player.forfeited {
			if player != nil {
				player.pointsGained = 0
				player.unadjustedTotalPoints = player.totalPoints
			}
			continue
		}
		roundPoints := player.hand.Points()
		player.pointsGained = roundPoints
		player.totalPoints += roundPoints
		player.unadjustedTotalPoints = player.totalPoints
		player.statistics.PenaltyPoints += roundPoints
		player.statistics.LargestRoundPenalty = max(player.statistics.LargestRoundPenalty, roundPoints)
		pointsInflicted += roundPoints
	}

	for _, player := range gs.players {
		if player == nil || player.statistics.RoundsPlayed < gs.round {
			continue
		}
		player.statistics.recordRoundHand(gs.round, len(player.hand.cards), player.hand.Points())
	}
	winner := gs.players[winnerIndex]
	winner.statistics.PointsInflicted += pointsInflicted
	winner.statistics.LargestRoundPointsInflicted = max(winner.statistics.LargestRoundPointsInflicted, pointsInflicted)
	if hasSameSuitCollection(winner.hand.cards) {
		winner.statistics.SameSuitWins++
	} else if hasSixIdenticalPairs(winner.hand.cards) {
		winner.statistics.SixPairsWins++
	}

	if gs.gameMode == GameModeQuick {
		gs.phase = PhaseGameOver
		return
	}

	if gs.allOtherPlayersOverHundred(winnerIndex) {
		gs.phase = PhaseGameOver
		return
	}

	gs.applyOverHundredAdjustment()
	gs.phase = PhaseRoundOver
}

func (gs *GameState) allOtherPlayersOverHundred(winnerIndex int) bool {
	for i, player := range gs.players {
		if i == winnerIndex || player == nil || player.forfeited {
			continue
		}
		if player.totalPoints <= 100 {
			return false
		}
	}

	return true
}

func (gs *GameState) applyOverHundredAdjustment() {
	highestRemaining := -1
	for _, player := range gs.players {
		if player == nil || player.forfeited || player.totalPoints > 100 {
			continue
		}
		if player.totalPoints > highestRemaining {
			highestRemaining = player.totalPoints
		}
	}

	if highestRemaining < 0 {
		return
	}

	for _, player := range gs.players {
		if player == nil || player.forfeited || player.totalPoints <= 100 {
			continue
		}
		player.totalPoints = highestRemaining
	}
}

func hasSpecialWinningHand(cards []Card) bool {
	return hasSameSuitCollection(cards) || hasSixIdenticalPairs(cards)
}

func hasSameSuitCollection(cards []Card) bool {
	if len(cards) != InitialHandSize {
		return false
	}

	firstSuitSet := false
	var suit Suit
	for _, card := range cards {
		if card.isJoker {
			continue
		}
		if !firstSuitSet {
			suit = card.suit
			firstSuitSet = true
			continue
		}
		if card.suit != suit {
			return false
		}
	}

	return true
}

func hasSixIdenticalPairs(cards []Card) bool {
	if len(cards) != InitialHandSize {
		return false
	}

	counts := make(map[Card]int, 6)
	jokerCount := 0
	for _, card := range cards {
		if card.isJoker {
			jokerCount++
			continue
		}
		counts[Card{rank: card.rank, suit: card.suit}]++
	}

	oddCount := 0
	for _, count := range counts {
		if count%2 != 0 {
			oddCount++
		}
	}

	return jokerCount >= oddCount
}
