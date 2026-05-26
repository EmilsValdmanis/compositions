package game

import (
	"errors"
	"math/bits"
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
	number             int
	playerIndex        int
	hasDrawn           bool
	mustUseDiscardDraw bool
	discardDrawCard    Card
}

type DealTypes int

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

type tablePlayCandidate struct {
	usedMask    uint32
	comp        *Composition
	addition    *CompositionAddition
	reclaim     *JokerReclaim
	usesDiscard bool
}

type searchScratch struct {
	maskCards   []Card
	combinedBuf []Card
}

type tablePlayState struct {
	handCards          []Card
	activeCompositions []*Composition
	hasOpened          bool
}

type selectedAddition struct {
	compositionIndex int
	insertIndex      *int
	mask             uint32
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
	ErrCannotStartNextRound        = errors.New("cannot start next round")
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
	ErrMustUseDrawnDiscardCard     = errors.New("player must use the drawn discard card before discarding")
	ErrInvalidDealingType          = errors.New("invalid dealing type")
	ErrInvalidDealingOrder         = errors.New("invalid dealing order")
	ErrInvalidDealer               = errors.New("invalid dealer")
	ErrInvalidDealChooser          = errors.New("invalid deal chooser")
	ErrInvalidCutSize              = errors.New("invalid cut size")
)

var newShuffledGameDeck = func() *CardPile {
	deck := NewGameDeck()
	deck.Shuffle()
	return deck
}

func NewGameState() *GameState {
	players := make([]*Player, 0, 4)
	deck := newShuffledGameDeck()

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

func NewGameStateWithDeck(cards []Card) *GameState {
	state := NewGameState()
	state.drawPile = &CardPile{cards: append([]Card(nil), cards...)}
	return state
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

	cp.hand.cards = nextState.handCards
	gs.activeCompositions = nextState.activeCompositions
	cp.hasOpened = nextState.hasOpened
	if gs.turn.mustUseDiscardDraw && tablePlayUsesCard(comps, additions, reclaims, gs.turn.discardDrawCard) {
		gs.turn.mustUseDiscardDraw = false
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
	if gs.turn.mustUseDiscardDraw {
		return ErrMustUseDrawnDiscardCard
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
	gs.resetDiscardDrawState()

	for i, player := range gs.players {
		if i == winnerIndex || player == nil {
			if player != nil {
				player.pointsGained = 0
			}
			continue
		}
		roundPoints := player.hand.Points()
		player.pointsGained = roundPoints
		player.totalPoints += roundPoints
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

func (gs *GameState) StartGame(dealerIndex, chooserIndex int, dt DealTypes, order []int, cutSize int) error {
	if gs.phase != PhaseLobby {
		return ErrGameInProgress
	}

	gs.resetRoundState(gs.drawPile)

	return gs.startRound(dealerIndex, chooserIndex, dt, order, cutSize)
}

func (gs *GameState) StartNextRound(dt DealTypes, order []int, cutSize int) error {
	if gs.phase != PhaseRoundOver {
		return ErrCannotStartNextRound
	}

	deck := newShuffledGameDeck()
	gs.resetRoundState(deck)
	dealerIndex := nextPlayerIndex(gs.dealerIndex, len(gs.players))
	chooserIndex := dealChooserIndex(dealerIndex, len(gs.players))

	if err := gs.startRound(dealerIndex, chooserIndex, dt, order, cutSize); err != nil {
		return err
	}

	gs.round++
	return nil
}

func (gs *GameState) startRound(dealerIndex, chooserIndex int, dt DealTypes, order []int, cutSize int) error {
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
	if dt == DealInBlocks && cutSize != 0 {
		return ErrInvalidCutSize
	}
	if len(gs.drawPile.cards) < InitialHandSize*len(gs.players)+1 {
		return ErrNotEnoughCardsInDrawPile
	}
	gs.dealerIndex = dealerIndex
	if err := gs.dealInitialHands(dt, order, cutSize); err != nil {
		return err
	}
	card, _ := gs.drawPile.DrawOne()
	gs.discardPile.AddToTop(card)
	for i, player := range gs.players {
		if hasSpecialWinningHand(player.hand.cards) {
			gs.finishRound(i)
			return nil
		}
	}
	gs.turn.playerIndex = nextPlayerIndex(gs.dealerIndex, len(gs.players))
	gs.phase = PhaseInProgress
	return nil
}

func (gs *GameState) resetRoundState(drawPile *CardPile) {
	if drawPile == nil {
		drawPile = newShuffledGameDeck()
	}

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
		player.hasOpened = false
	}
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

func (gs *GameState) Phase() GamePhase {
	if gs == nil {
		return PhaseLobby
	}
	return gs.phase
}

func (gs *GameState) DealerIndex() int {
	if gs == nil {
		return 0
	}
	return gs.dealerIndex
}

func (gs *GameState) advanceTurn() {
	gs.turn.number++
	gs.turn.playerIndex = (gs.turn.playerIndex + 1) % len(gs.players)
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
				if cardIndex != len(availableCards)-1 {
					continue
				}
				if !comp.canReclaimJoker(jokerIndex, replacementCard) {
					continue
				}

				reclaim := tablePlayCandidate{
					usedMask:    discardMask,
					reclaim:     &JokerReclaim{CompositionIndex: compositionIndex, JokerIndex: jokerIndex, ReplacementCard: replacementCard},
					usesDiscard: true,
				}
				if hasLegalPlayWithDiscard(baseState, discardMask, scratch, &reclaim) {
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
		cards := cardsForMask(baseState.handCards, mask, scratch.maskCards)

		if bits.OnesCount32(subset) >= 2 {
			for _, variant := range []compositionVariant{set, run} {
				if !isSearchCompositionValid(cards, variant) {
					continue
				}

				selectedCompMasks = append(selectedCompMasks, mask)
				selectedCompVariants = append(selectedCompVariants, variant)
				if validateTablePlay(baseState, selectedCompMasks, selectedCompVariants, selectedAdditions, selectedReclaims, mask, true, scratch) {
					return true
				}
				if searchSupportCandidates(baseState, discardMask, 1, mask, selectedCompMasks, selectedCompVariants, selectedAdditions, selectedReclaims, scratch) {
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
				if validateTablePlay(baseState, selectedCompMasks, selectedCompVariants, selectedAdditions, selectedReclaims, mask, true, scratch) {
					return true
				}
				if searchSupportCandidates(baseState, discardMask, 1, mask, selectedCompMasks, selectedCompVariants, selectedAdditions, selectedReclaims, scratch) {
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

func canAddCardsToComposition(comp *Composition, cards []Card, combinedBuf []Card) bool {
	combined := combinedBuf[:0]
	combined = append(combined, comp.cards...)
	combined = append(combined, cards...)
	extended := Composition{variant: comp.variant, cards: combined}
	return extended.isValid()
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

		extended, ok := target.WithInsertedCards(insertIndex, cards)
		if !ok {
			return false
		}
		openingPoints += extended.Points() - target.Points()
		storeComposition(addition.compositionIndex, extended)
	}

	for _, reclaim := range reclaims {
		if reclaim.CompositionIndex < 0 || reclaim.CompositionIndex >= len(baseState.activeCompositions) {
			return false
		}

		target := currentComposition(reclaim.CompositionIndex)
		if target == nil {
			return false
		}

		if !target.canReclaimJoker(reclaim.JokerIndex, reclaim.ReplacementCard) {
			return false
		}
		reclaimPoints, _ := target.ReclaimPoints(reclaim.JokerIndex)
		openingPoints += reclaimPoints
		updated, _ := target.ReclaimJoker(reclaim.JokerIndex, reclaim.ReplacementCard)
		storeComposition(reclaim.CompositionIndex, updated)
	}

	if !baseState.hasOpened && len(compMasks) == 0 {
		return false
	}
	if !baseState.hasOpened && openingPoints < 40 {
		return false
	}
	if len(baseState.handCards)-bits.OnesCount32(usedMask)+len(reclaims) == 0 {
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

func tablePlayUsesCard(comps []*Composition, additions []CompositionAddition, reclaims []JokerReclaim, target Card) bool {
	for _, comp := range comps {
		if comp == nil {
			continue
		}
		for _, playedCard := range comp.cards {
			if cardsEqual(playedCard, target) && playedCard.isJoker == target.isJoker {
				return true
			}
		}
	}

	for _, addition := range additions {
		for _, playedCard := range addition.Cards {
			if cardsEqual(playedCard, target) && playedCard.isJoker == target.isJoker {
				return true
			}
		}
	}

	for _, reclaim := range reclaims {
		if cardsEqual(reclaim.ReplacementCard, target) && reclaim.ReplacementCard.isJoker == target.isJoker {
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
			comp.normalizeRunCards()
		}
		playedCards = append(playedCards, comp.cards...)
		openingPoints += comp.Points()
	}

	updatedCompositions := make([]*Composition, len(state.activeCompositions))
	copy(updatedCompositions, state.activeCompositions)

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

		extended, ok := target.WithInsertedCards(insertIndex, addition.Cards)
		if !ok {
			return tablePlayState{}, ErrInvalidComposition
		}

		updatedCompositions[addition.CompositionIndex] = extended
		playedCards = append(playedCards, addition.Cards...)
		openingPoints += extended.Points() - target.Points()
	}

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
			return tablePlayState{}, ErrInvalidComposition
		}
		reclaimPoints, _ := target.ReclaimPoints(reclaim.JokerIndex)

		reclaimedCards = append(reclaimedCards, target.cards[reclaim.JokerIndex])
		playedCards = append(playedCards, reclaim.ReplacementCard)
		openingPoints += reclaimPoints
		updatedCompositions[reclaim.CompositionIndex] = updated
	}

	if !state.hasOpened && len(comps) == 0 {
		return tablePlayState{}, ErrInitialPlayRequiresOwnComp
	}
	if !state.hasOpened && openingPoints < 40 {
		return tablePlayState{}, ErrInitialPointsNotMet
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

func (gs *GameState) dealInitialHands(dt DealTypes, order []int, cutSize int) error {
	switch dt {
	case DealRoundRobin:
		return dealRoundRobin(gs.players, gs.drawPile, gs.dealerIndex, cutSize)
	case DealInBlocks:
		return dealInBlocks(gs.players, gs.drawPile, order)
	default:
		return ErrInvalidDealingType
	}
}

func dealRoundRobin(players []*Player, drawPile *CardPile, dealerIndex, cutSize int) error {
	required := InitialHandSize * len(players)
	if len(drawPile.cards) < required {
		return ErrNotEnoughCardsInDrawPile
	}
	if !isValidPlayerIndex(dealerIndex, len(players)) {
		return ErrInvalidDealer
	}
	if cutSize < 0 || cutSize > len(drawPile.cards)-required {
		return ErrInvalidCutSize
	}

	setAside := append([]Card{}, drawPile.cards[:cutSize]...)
	mainPile := &CardPile{cards: append([]Card{}, drawPile.cards[cutSize:]...)}

	for range InitialHandSize {
		for offset := 1; offset <= len(players); offset++ {
			player := players[(dealerIndex+offset)%len(players)]
			card, _ := mainPile.DrawOne()
			player.hand.cards = append(player.hand.cards, card)
		}
	}

	drawPile.cards = append(mainPile.cards, setAside...)
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
			card, _ := drawPile.DrawOne()
			player.hand.cards = append(player.hand.cards, card)
		}
	}

	return nil
}

func validateOrder(order []int, playerCount int) bool {
	if len(order) != playerCount {
		return false
	}

	seen := make([]bool, playerCount)
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

func nextPlayerIndex(playerIndex, playerCount int) int {
	return (playerIndex + 1) % playerCount
}
