package game

import (
	"errors"
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
}

type GamePhase int

const (
	PhaseLobby = iota
	PhaseInProgress
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
	cp.hand.cards = nextHand.cards
	gs.activeCompositions = updatedCompositions
	gs.activeCompositions = append(gs.activeCompositions, comps...)
	cp.hasOpened = true

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
	gs.advanceTurn()
	return nil
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
	if err := gs.dealInitialHands(dt, order); err != nil {
		return err
	}

	card, ok := gs.drawPile.DrawOne()
	if !ok {
		return ErrNotEnoughCardsInDrawPile
	}
	gs.discardPile.AddToTop(card)
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
}

func (gs *GameState) canTakeDiscardNow() bool {
	// TODO:
	// it must either:
	// - create at least 1 valid run in hand
	// - create at least 1 valid set in hand
	// and the total for all runs and sets
	// in player's hand must be >= 40 points

	// if they have already opened, they can always take the discard card as long as they can use it:
	// - to add to an existing run or set on the table
	// - to create a new run or set in hand
	return true
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
