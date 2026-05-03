package game

import (
	"errors"
	"math/bits"
	"math/rand"
)

type GameState struct {
	players            []*Player
	activeCompositions []*Composition
	drawPile           *CardPile
	discardPile        *CardPile
	maxPlayers         int
	phase              GamePhase
	round              int
	dealerIndex        int
	turn               Turn
	roundWinnerIndex   int
}

type GamePhase int

const (
	PhaseLobby = iota
	PhaseInProgress
	PhaseRoundOver
	PhaseGameOver
)

type Turn struct {
	number      int
	playerIndex int
	hasDrawn    bool
}

type DealTypes int

type CompositionAddition struct {
	CompositionIndex int
	Cards            []Card
}

type JokerReclaim struct {
	CompositionIndex int
	JokerIndex       int
	ReplacementCard  Card
}

type tablePlayCandidate struct {
	usedMask   uint32
	comp       *Composition
	addition   *CompositionAddition
	reclaim    *JokerReclaim
	usesDiscard bool
}

type handCardKey struct {
	rank    Rank
	suit    Suit
	isJoker bool
}

const (
	DealRoundRobin = iota
	DealInBlocks
)

var (
	ErrGameInProgress              = errors.New("game already in progress")
	ErrGameNotInProgress           = errors.New("game is not in progress")
	ErrGameFull                    = errors.New("game is full")
	ErrPlayerExists                = errors.New("player already in game")
	ErrNilPlayer                   = errors.New("player is nil")
	ErrNotEnoughPlayers            = errors.New("need at least 2 players to start")
	ErrNotEnoughCardsInDrawPile    = errors.New("not enough cards in draw pile for all players")
	ErrNoPlayers                   = errors.New("no players in game")
	ErrNotEnoughCardsInDiscardPile = errors.New("not enough cards in discard pile")
	ErrInvalidComposition          = errors.New("not a valid composition")
	ErrPlayerAlreadyDrew           = errors.New("player already drew")
	ErrPlayerHasntDrawn            = errors.New("player hasnt drawn a card yet")
	ErrCannotTakeDiscardCard       = errors.New("cannot take discard card")
	ErrRemovingCard                = errors.New("error removing card")
	ErrCardsNotInHand              = errors.New("one or more cards not in hand")
	ErrInitialPointsNotMet         = errors.New("initial compositions must total at least 40 points")
	ErrInitialPlayRequiresOwnComp  = errors.New("initial play requires at least one new composition")
	ErrMustKeepDiscardCard         = errors.New("player must keep one card for the final discard")
	ErrInvalidDealingType          = errors.New("invalid dealing type")
	ErrInvalidDealingOrder         = errors.New("invalid dealing order")
	ErrInvalidDealer               = errors.New("invalid dealer")
	ErrInvalidDealChooser          = errors.New("invalid deal chooser")
)

func NewGameState() *GameState {
	players := make([]*Player, 0, 4)
	deck := NewGameDeck()
	deck.Shuffle()

	return &GameState{
		players:            players,
		activeCompositions: []*Composition{},
		drawPile:           deck,
		discardPile:        &CardPile{cards: make([]Card, 0, cardsInDeck*2)},
		maxPlayers:         4,
		phase:              PhaseLobby,
		round:              1,
		dealerIndex:        0,
		roundWinnerIndex:   -1,
		turn: Turn{
			number:      1,
			playerIndex: 0,
		},
	}
}

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
	gs.turn.hasDrawn = true
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

	if !cp.hand.Draw(gs.discardPile) {
		return ErrNotEnoughCardsInDiscardPile
	}

	gs.turn.hasDrawn = true
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

	playedCards := make([]Card, 0)
	reclaimedCards := make([]Card, 0, len(reclaims))
	openingPoints := 0
	for _, comp := range comps {
		if comp == nil {
			return ErrInvalidComposition
		}
		if !comp.isValid() {
			return ErrInvalidComposition
		}
		playedCards = append(playedCards, comp.cards...)
		openingPoints += comp.Points()
	}

	updatedCompositions := make([]*Composition, len(gs.activeCompositions))
	copy(updatedCompositions, gs.activeCompositions)
	for _, reclaim := range reclaims {
		if reclaim.CompositionIndex < 0 || reclaim.CompositionIndex >= len(updatedCompositions) {
			return ErrInvalidComposition
		}

		target := updatedCompositions[reclaim.CompositionIndex]
		if target == nil {
			return ErrInvalidComposition
		}

		updated, ok := target.ReclaimJoker(reclaim.JokerIndex, reclaim.ReplacementCard)
		if !ok {
			return ErrInvalidComposition
		}

		reclaimedCards = append(reclaimedCards, target.cards[reclaim.JokerIndex])
		playedCards = append(playedCards, reclaim.ReplacementCard)
		updatedCompositions[reclaim.CompositionIndex] = updated
	}

	for _, addition := range additions {
		if len(addition.Cards) == 0 {
			return ErrInvalidComposition
		}
		if addition.CompositionIndex < 0 || addition.CompositionIndex >= len(updatedCompositions) {
			return ErrInvalidComposition
		}

		target := updatedCompositions[addition.CompositionIndex]
		if target == nil {
			return ErrInvalidComposition
		}

		addedPoints, ok := target.AddedCardsPoints(addition.Cards)
		if !ok {
			return ErrInvalidComposition
		}

		extended, ok := target.WithAddedCards(addition.Cards)
		if !ok {
			return ErrInvalidComposition
		}

		updatedCompositions[addition.CompositionIndex] = extended
		playedCards = append(playedCards, addition.Cards...)
		openingPoints += addedPoints
	}

	cp, err := gs.CurrentPlayer()
	if err != nil {
		return err
	}

	if !cp.hasOpened && len(comps) == 0 {
		return ErrInitialPlayRequiresOwnComp
	}
	if !cp.hasOpened && openingPoints < 40 {
		return ErrInitialPointsNotMet
	}

	nextHand := &Hand{cards: make([]Card, 0, len(cp.hand.cards)+len(reclaimedCards))}
	nextHand.cards = append(nextHand.cards, cp.hand.cards...)
	nextHand.cards = append(nextHand.cards, reclaimedCards...)
	if !nextHand.RemoveCards(playedCards) {
		return ErrCardsNotInHand
	}
	if len(nextHand.cards) == 0 {
		return ErrMustKeepDiscardCard
	}
	cp.hand.cards = nextHand.cards
	gs.activeCompositions = updatedCompositions
	gs.activeCompositions = append(gs.activeCompositions, comps...)
	cp.hasOpened = true
	if gs.finishRoundIfSpecialWin(gs.turn.playerIndex) {
		return nil
	}

	return nil
}

func (gs *GameState) DiscardFromHand(cardIndex int) error {
	if gs.phase != PhaseInProgress {
		return ErrGameNotInProgress
	}
	if !gs.turn.hasDrawn {
		return ErrPlayerHasntDrawn
	}

	cp, err := gs.CurrentPlayer()
	if err != nil {
		return err
	}

	card, ok := cp.hand.RemoveAt(cardIndex)
	if !ok {
		return ErrRemovingCard
	}

	gs.removeCompletedCompositionsToDiscard()
	gs.discardPile.AddToTop(card)
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

	for i, player := range gs.players {
		if i == winnerIndex || player == nil {
			continue
		}
		player.totalPoints += player.hand.Points()
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
		if i == winnerIndex || player == nil {
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
		if player == nil || player.totalPoints > 100 {
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
		if player == nil || player.totalPoints <= 100 {
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
			return false
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

	counts := make(map[handCardKey]int, 6)
	for _, card := range cards {
		key := handCardKey{rank: card.rank, suit: card.suit, isJoker: card.isJoker}
		counts[key]++
	}

	if len(counts) != 6 {
		return false
	}

	for _, count := range counts {
		if count != 2 {
			return false
		}
	}

	return true
}

func (gs *GameState) removeCompletedCompositionsToDiscard() {
	remaining := make([]*Composition, 0, len(gs.activeCompositions))
	for _, comp := range gs.activeCompositions {
		if comp == nil || !comp.isComplete() {
			remaining = append(remaining, comp)
			continue
		}

		for i := len(comp.cards) - 1; i >= 0; i-- {
			gs.discardPile.AddToTop(comp.cards[i])
		}
	}

	gs.activeCompositions = remaining
}

func (gs *GameState) StartGame(dealerIndex, chooserIndex int, dt DealTypes, order []int) error {
	if gs.phase != PhaseLobby {
		return ErrGameInProgress
	}
	if len(gs.players) < 2 {
		return ErrNotEnoughPlayers
	}
	if !isValidPlayerIndex(dealerIndex, len(gs.players)) {
		return ErrInvalidDealer
	}
	if chooserIndex != dealChooserIndex(dealerIndex, len(gs.players)) {
		return ErrInvalidDealChooser
	}

	if dt == DealInBlocks && order == nil {
		return ErrInvalidDealingOrder
	}
	gs.dealerIndex = dealerIndex
	gs.roundWinnerIndex = -1
	if err := gs.dealInitialHands(dt, order); err != nil {
		return err
	}

	card, ok := gs.drawPile.DrawOne()
	if !ok {
		return ErrNotEnoughCardsInDrawPile
	}
	gs.discardPile.AddToTop(card)
	for i, player := range gs.players {
		if player == nil {
			continue
		}
		if hasSpecialWinningHand(player.hand.cards) {
			gs.finishRound(i)
			return nil
		}
	}
	if err := gs.SelectFirstPlayer(); err != nil {
		return err
	}
	gs.phase = PhaseInProgress
	return nil
}

func (gs *GameState) SelectFirstPlayer() error {
	if len(gs.players) == 0 {
		return ErrNoPlayers
	}

	gs.turn.playerIndex = rand.Intn(len(gs.players))
	return nil
}

func (gs *GameState) CurrentPlayer() (*Player, error) {
	if len(gs.players) == 0 {
		return nil, ErrNoPlayers
	}

	return gs.players[gs.turn.playerIndex], nil
}

func (gs *GameState) AddPlayer(p *Player) error {
	if gs.phase != PhaseLobby {
		return ErrGameInProgress
	}
	if p == nil {
		return ErrNilPlayer
	}
	if len(gs.players) >= gs.maxPlayers {
		return ErrGameFull
	}
	for _, existing := range gs.players {
		if existing.ID == p.ID {
			return ErrPlayerExists
		}
	}
	gs.players = append(gs.players, p)
	return nil
}

func (gs *GameState) advanceTurn() {
	gs.turn.number++
	gs.turn.playerIndex = (gs.turn.playerIndex + 1) % len(gs.players)
	gs.turn.hasDrawn = false
	gs.recycleDiscardIntoDrawPileIfNeeded()
}

func (gs *GameState) recycleDiscardIntoDrawPileIfNeeded() {
	if len(gs.drawPile.cards) != 0 || len(gs.discardPile.cards) == 0 {
		return
	}

	recycled := make([]Card, len(gs.discardPile.cards))
	for i := range gs.discardPile.cards {
		recycled[i] = gs.discardPile.cards[len(gs.discardPile.cards)-1-i]
	}

	gs.drawPile.cards = recycled
	gs.discardPile.cards = gs.discardPile.cards[:0]
}

func (gs *GameState) canTakeDiscardNow() bool {
	if len(gs.discardPile.cards) == 0 {
		return false
	}

	cp, err := gs.CurrentPlayer()
	if err != nil {
		return false
	}

	availableCards := make([]Card, 0, len(cp.hand.cards)+1)
	availableCards = append(availableCards, cp.hand.cards...)
	availableCards = append(availableCards, gs.discardPile.cards[0])
	discardMask := uint32(1) << uint(len(availableCards)-1)

	candidates := make([]tablePlayCandidate, 0)
	candidates = append(candidates, buildCompositionCandidates(availableCards, discardMask)...)
	candidates = append(candidates, buildAdditionCandidates(gs.activeCompositions, availableCards, discardMask)...)
	reclaimCandidates := buildReclaimCandidates(gs.activeCompositions, availableCards, discardMask)

	if gs.hasLegalPlayWithDiscard(candidates, nil, discardMask) {
		return true
	}

	for _, reclaim := range reclaimCandidates {
		if gs.hasLegalPlayWithDiscard(candidates, &reclaim, discardMask) {
			return true
		}
	}

	return false
}

func buildCompositionCandidates(availableCards []Card, discardMask uint32) []tablePlayCandidate {
	limit := uint32(1) << uint(len(availableCards))
	candidates := make([]tablePlayCandidate, 0)

	for mask := uint32(1); mask < limit; mask++ {
		cardCount := bits.OnesCount32(mask)
		if cardCount < 3 {
			continue
		}

		cards := cardsForMask(availableCards, mask)
		if comp, ok := NewSet(cards); ok {
			candidates = append(candidates, tablePlayCandidate{
				usedMask:    mask,
				comp:        comp,
				usesDiscard: mask&discardMask != 0,
			})
		}
		if comp, ok := NewRun(cards); ok {
			candidates = append(candidates, tablePlayCandidate{
				usedMask:    mask,
				comp:        comp,
				usesDiscard: mask&discardMask != 0,
			})
		}
	}

	return candidates
}

func buildAdditionCandidates(activeCompositions []*Composition, availableCards []Card, discardMask uint32) []tablePlayCandidate {
	limit := uint32(1) << uint(len(availableCards))
	candidates := make([]tablePlayCandidate, 0)

	for compositionIndex, comp := range activeCompositions {
		if comp == nil {
			continue
		}

		for mask := uint32(1); mask < limit; mask++ {
			cards := cardsForMask(availableCards, mask)
			if _, ok := comp.WithAddedCards(cards); !ok {
				continue
			}

			addition := CompositionAddition{
				CompositionIndex: compositionIndex,
				Cards:            cards,
			}
			candidates = append(candidates, tablePlayCandidate{
				usedMask:    mask,
				addition:    &addition,
				usesDiscard: mask&discardMask != 0,
			})
		}
	}

	return candidates
}

func buildReclaimCandidates(activeCompositions []*Composition, availableCards []Card, discardMask uint32) []tablePlayCandidate {
	candidates := make([]tablePlayCandidate, 0)

	for compositionIndex, comp := range activeCompositions {
		if comp == nil {
			continue
		}

		for jokerIndex, card := range comp.cards {
			if !card.isJoker {
				continue
			}

			for cardIndex, replacementCard := range availableCards {
				if replacementCard.isJoker {
					continue
				}
				if _, ok := comp.ReclaimJoker(jokerIndex, replacementCard); !ok {
					continue
				}

				usedMask := uint32(1) << uint(cardIndex)
				reclaim := JokerReclaim{
					CompositionIndex: compositionIndex,
					JokerIndex:       jokerIndex,
					ReplacementCard:  replacementCard,
				}
				candidates = append(candidates, tablePlayCandidate{
					usedMask:    usedMask,
					reclaim:     &reclaim,
					usesDiscard: usedMask&discardMask != 0,
				})
			}
		}
	}

	return candidates
}

func cardsForMask(availableCards []Card, mask uint32) []Card {
	cards := make([]Card, 0, bits.OnesCount32(mask))
	for i, card := range availableCards {
		if mask&(uint32(1)<<uint(i)) == 0 {
			continue
		}
		cards = append(cards, card)
	}
	return cards
}

func (gs *GameState) hasLegalPlayWithDiscard(candidates []tablePlayCandidate, reclaimCandidate *tablePlayCandidate, discardMask uint32) bool {
	selectedComps := make([]*Composition, 0)
	selectedAdditions := make([]CompositionAddition, 0)
	selectedReclaims := make([]JokerReclaim, 0, 1)
	usedMask := uint32(0)
	hasDiscardPlay := false

	if reclaimCandidate != nil {
		usedMask = reclaimCandidate.usedMask
		hasDiscardPlay = reclaimCandidate.usesDiscard
		selectedReclaims = append(selectedReclaims, *reclaimCandidate.reclaim)
		if gs.simulatePlayTable(selectedComps, selectedAdditions, selectedReclaims, hasDiscardPlay) {
			return true
		}
	}

	return gs.searchLegalDiscardPlay(candidates, 0, usedMask, hasDiscardPlay, selectedComps, selectedAdditions, selectedReclaims, discardMask)
}

func (gs *GameState) searchLegalDiscardPlay(candidates []tablePlayCandidate, start int, usedMask uint32, hasDiscardPlay bool, selectedComps []*Composition, selectedAdditions []CompositionAddition, selectedReclaims []JokerReclaim, discardMask uint32) bool {
	for i := start; i < len(candidates); i++ {
		candidate := candidates[i]
		if usedMask&candidate.usedMask != 0 {
			continue
		}

		nextUsedMask := usedMask | candidate.usedMask
		nextHasDiscardPlay := hasDiscardPlay || candidate.usesDiscard
		nextComps := append([]*Composition{}, selectedComps...)
		nextAdditions := append([]CompositionAddition{}, selectedAdditions...)

		if candidate.comp != nil {
			nextComps = append(nextComps, candidate.comp)
		}
		if candidate.addition != nil {
			nextAdditions = append(nextAdditions, *candidate.addition)
		}

		if gs.simulatePlayTable(nextComps, nextAdditions, selectedReclaims, nextHasDiscardPlay) {
			return true
		}

		if gs.searchLegalDiscardPlay(candidates, i+1, nextUsedMask, nextHasDiscardPlay, nextComps, nextAdditions, selectedReclaims, discardMask) {
			return true
		}
	}

	return false
}

func (gs *GameState) simulatePlayTable(comps []*Composition, additions []CompositionAddition, reclaims []JokerReclaim, hasDiscardPlay bool) bool {
	if !hasDiscardPlay {
		return false
	}
	if len(comps) == 0 && len(additions) == 0 && len(reclaims) == 0 {
		return false
	}

	clone := gs.cloneForDiscardTableSearch()
	if err := clone.PlayTable(comps, additions, reclaims...); err != nil {
		return false
	}

	return true
}

func (gs *GameState) cloneForDiscardTableSearch() *GameState {
	players := make([]*Player, len(gs.players))
	for i, player := range gs.players {
		if player == nil {
			continue
		}

		clonedHand := &Hand{cards: append([]Card{}, player.hand.cards...)}
		players[i] = &Player{
			ID:          player.ID,
			hand:        clonedHand,
			totalPoints: player.totalPoints,
			hasOpened:   player.hasOpened,
		}
	}

	if len(gs.discardPile.cards) != 0 && isValidPlayerIndex(gs.turn.playerIndex, len(players)) && players[gs.turn.playerIndex] != nil {
		players[gs.turn.playerIndex].hand.cards = append(players[gs.turn.playerIndex].hand.cards, gs.discardPile.cards[0])
	}

	activeCompositions := append([]*Composition{}, gs.activeCompositions...)
	return &GameState{
		players:            players,
		activeCompositions: activeCompositions,
		drawPile:           &CardPile{cards: append([]Card{}, gs.drawPile.cards...)},
		discardPile:        &CardPile{cards: append([]Card{}, gs.discardPile.cards...)},
		maxPlayers:         gs.maxPlayers,
		phase:              gs.phase,
		round:              gs.round,
		dealerIndex:        gs.dealerIndex,
		turn: Turn{
			number:      gs.turn.number,
			playerIndex: gs.turn.playerIndex,
			hasDrawn:    true,
		},
		roundWinnerIndex:   gs.roundWinnerIndex,
	}
}

func (gs *GameState) dealInitialHands(dt DealTypes, order []int) error {
	switch dt {
	case DealRoundRobin:
		return dealRoundRobin(gs.players, gs.drawPile, gs.dealerIndex)
	case DealInBlocks:
		return dealInBlocks(gs.players, gs.drawPile, order)
	default:
		return ErrInvalidDealingType
	}
}

func dealRoundRobin(players []*Player, drawPile *CardPile, dealerIndex int) error {
	required := InitialHandSize * len(players)
	if len(drawPile.cards) < required {
		return ErrNotEnoughCardsInDrawPile
	}
	if !isValidPlayerIndex(dealerIndex, len(players)) {
		return ErrInvalidDealer
	}

	for range InitialHandSize {
		for offset := 1; offset <= len(players); offset++ {
			player := players[(dealerIndex+offset)%len(players)]
			if !player.hand.Draw(drawPile) {
				return ErrNotEnoughCardsInDrawPile
			}
		}
	}
	return nil
}

func dealInBlocks(players []*Player, drawPile *CardPile, order []int) error {
	required := InitialHandSize * len(players)
	if len(drawPile.cards) < required {
		return ErrNotEnoughCardsInDrawPile
	}
	if !validateOrder(order, len(players)) {
		return ErrInvalidDealingOrder
	}

	for _, i := range order {
		player := players[i]

		for range InitialHandSize {
			if !player.hand.Draw(drawPile) {
				return ErrNotEnoughCardsInDrawPile
			}
		}
	}

	return nil
}

func validateOrder(order []int, playerCount int) bool {
	if len(order) != playerCount {
		return false
	}

	seen := make(map[int]bool)
	for _, i := range order {
		if i < 0 || i >= playerCount || seen[i] {
			return false
		}
		seen[i] = true
	}
	return true
}

func isValidPlayerIndex(playerIndex, playerCount int) bool {
	return playerIndex >= 0 && playerIndex < playerCount
}

func dealChooserIndex(dealerIndex, playerCount int) int {
	return (dealerIndex - 1 + playerCount) % playerCount
}
