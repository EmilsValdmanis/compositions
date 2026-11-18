package game

import "errors"

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
	gameMode           GameMode
}

type GameMode string

const (
	GameModeFull  GameMode = "full"
	GameModeQuick GameMode = "quick"
)

func (mode GameMode) Valid() bool {
	return mode == GameModeFull || mode == GameModeQuick
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
	ErrMustUseDrawnDiscardCard     = errors.New("player must use the drawn discard card before playing other cards or discarding")
	ErrInvalidDealingType          = errors.New("invalid dealing type")
	ErrInvalidDealingOrder         = errors.New("invalid dealing order")
	ErrInvalidDealer               = errors.New("invalid dealer")
	ErrInvalidDealChooser          = errors.New("invalid deal chooser")
	ErrInvalidCutSize              = errors.New("invalid cut size")
	ErrPlayerNotFound              = errors.New("player not found")
	ErrPlayerAlreadyForfeited      = errors.New("player already forfeited")
	ErrInvalidGameMode             = errors.New("invalid game mode")
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
		gameMode:           GameModeFull,
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

func (gs *GameState) CurrentPlayer() (*Player, error) {
	if len(gs.players) == 0 {
		return nil, ErrNoPlayers
	}

	player := gs.players[gs.turn.playerIndex]
	if player != nil && player.forfeited {
		return nil, ErrPlayerNotFound
	}
	return player, nil
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

// ForfeitPlayer permanently removes a player from active play while preserving
// their seat and score history. Their remaining hand is shuffled back into the
// draw pile so cards needed by the remaining players stay available.
func (gs *GameState) ForfeitPlayer(playerID string) (string, error) {
	if gs == nil || (gs.phase != PhaseInProgress && gs.phase != PhaseRoundOver) {
		return "", ErrGameNotInProgress
	}

	playerIndex := -1
	for index, player := range gs.players {
		if player != nil && player.ID == playerID {
			playerIndex = index
			break
		}
	}
	if playerIndex < 0 {
		return "", ErrPlayerNotFound
	}

	player := gs.players[playerIndex]
	if player.forfeited {
		return "", ErrPlayerAlreadyForfeited
	}
	nextForfeitOrder := 1
	for _, candidate := range gs.players {
		if candidate != nil && candidate != player && candidate.statistics.ForfeitOrder >= nextForfeitOrder {
			nextForfeitOrder = candidate.statistics.ForfeitOrder + 1
		}
	}
	player.statistics.ForfeitOrder = nextForfeitOrder
	if player.statistics.RoundsPlayed >= gs.round && player.hand != nil {
		player.statistics.recordRoundHand(gs.round, len(player.hand.cards), player.hand.Points())
	}

	if player.hand != nil && len(player.hand.cards) > 0 {
		gs.drawPile.cards = append(gs.drawPile.cards, player.hand.cards...)
		gs.drawPile.Shuffle()
	}
	player.hand = NewHand()
	player.pointsGained = 0
	player.unadjustedTotalPoints = player.totalPoints
	player.forfeited = true

	activeIndexes := gs.ActivePlayerIndexes()
	if len(activeIndexes) == 1 {
		winnerIndex := activeIndexes[0]
		for i, candidate := range gs.players {
			if candidate != nil && candidate.statistics.RoundsPlayed >= gs.round {
				candidate.statistics.recordRoundResult(gs.round, i == winnerIndex)
			}
		}
		gs.roundWinnerIndex = winnerIndex
		gs.phase = PhaseGameOver
		gs.turn.hasDrawn = false
		gs.resetDiscardDrawState()
		return gs.players[winnerIndex].ID, nil
	}

	if gs.phase == PhaseInProgress && gs.turn.playerIndex == playerIndex {
		gs.advanceTurn()
	}
	return "", nil
}

func (gs *GameState) EndWithoutWinner() error {
	if gs == nil || (gs.phase != PhaseInProgress && gs.phase != PhaseRoundOver) {
		return ErrGameNotInProgress
	}
	gs.phase = PhaseGameOver
	gs.roundWinnerIndex = -1
	gs.turn.hasDrawn = false
	gs.resetDiscardDrawState()
	return nil
}

func (gs *GameState) IsPlayerForfeited(playerID string) bool {
	for _, player := range gs.players {
		if player != nil && player.ID == playerID {
			return player.forfeited
		}
	}
	return false
}

func (gs *GameState) NextRoundDealerAndChooser() (int, int, error) {
	if gs == nil || len(gs.ActivePlayerIndexes()) < 2 {
		return 0, 0, ErrNotEnoughPlayers
	}
	dealerIndex := gs.nextActivePlayerIndex(gs.dealerIndex)
	return dealerIndex, gs.previousActivePlayerIndex(dealerIndex), nil
}

func (gs *GameState) ActivePlayerIndexes() []int {
	var indexes []int
	for index, player := range gs.players {
		if player != nil && !player.forfeited {
			indexes = append(indexes, index)
		}
	}
	return indexes
}

func (gs *GameState) isActivePlayerIndex(index int) bool {
	return isValidPlayerIndex(index, len(gs.players)) && gs.players[index] != nil && !gs.players[index].forfeited
}

func (gs *GameState) nextActivePlayerIndex(index int) int {
	if len(gs.players) == 0 {
		return 0
	}
	for offset := 1; offset <= len(gs.players); offset++ {
		candidate := (index + offset) % len(gs.players)
		if gs.isActivePlayerIndex(candidate) {
			return candidate
		}
	}
	return index
}

func (gs *GameState) previousActivePlayerIndex(index int) int {
	if len(gs.players) == 0 {
		return 0
	}
	for offset := 1; offset <= len(gs.players); offset++ {
		candidate := (index - offset + len(gs.players)) % len(gs.players)
		if gs.isActivePlayerIndex(candidate) {
			return candidate
		}
	}
	return index
}

func isValidPlayerIndex(playerIndex, playerCount int) bool {
	return playerIndex >= 0 && playerIndex < playerCount
}
