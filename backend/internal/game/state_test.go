package game

import (
	"errors"
	"slices"
	"testing"
)

const (
	twoPlayerDealerIndex  = 0
	twoPlayerChooserIndex = 1
)

func TestGameStateAccessors(t *testing.T) {
	if got := (*GameState)(nil).Phase(); got != PhaseLobby {
		t.Fatalf("(*GameState)(nil).Phase() = %v; want %v", got, PhaseLobby)
	}
	if got := (*GameState)(nil).DealerIndex(); got != 0 {
		t.Fatalf("(*GameState)(nil).DealerIndex() = %d; want 0", got)
	}

	state := NewGameState()
	state.phase = PhaseInProgress
	state.dealerIndex = 2

	if got := state.Phase(); got != PhaseInProgress {
		t.Fatalf("state.Phase() = %v; want %v", got, PhaseInProgress)
	}
	if got := state.DealerIndex(); got != 2 {
		t.Fatalf("state.DealerIndex() = %d; want 2", got)
	}
}

func newTurnTestState() *GameState {
	state := NewGameState()
	first := NewPlayer()
	second := NewPlayer()
	state.players = []*Player{first, second}
	state.phase = PhaseInProgress
	state.turn = Turn{number: 1, playerIndex: 0}
	state.drawPile = &CardPile{cards: []Card{}}
	state.discardPile = &CardPile{cards: []Card{}}
	state.activeCompositions = []*Composition{}
	return state
}

func TestNewGameStateDefaults(t *testing.T) {
	state := NewGameState()

	if state.round != 1 {
		t.Errorf("state.round = %d; want 1", state.round)
	}
	if state.turn.number != 1 {
		t.Errorf("state.turn.number = %d; want 1", state.turn.number)
	}
	if state.turn.playerIndex != 0 {
		t.Errorf("state.turn.playerIndex = %d; want 0", state.turn.playerIndex)
	}
	if state.phase != PhaseLobby {
		t.Errorf("state.phase = %d; want %d", state.phase, PhaseLobby)
	}
	if state.maxPlayers != 4 {
		t.Errorf("state.maxPlayers = %d; want 4", state.maxPlayers)
	}
	if len(state.drawPile.cards) != 108 {
		t.Errorf("len(state.drawPile.cards) = %d; want 108", len(state.drawPile.cards))
	}
	if len(state.discardPile.cards) != 0 {
		t.Errorf("len(state.discardPile.cards) = %d; want 0", len(state.discardPile.cards))
	}
}

func TestGameStateAddPlayerRejectsNilPlayer(t *testing.T) {
	state := NewGameState()

	err := state.AddPlayer(nil)

	if !errors.Is(err, ErrNilPlayer) {
		t.Errorf("AddPlayer(nil) error = %v; want %v", err, ErrNilPlayer)
	}
}

func TestGameStateStartGameDealsHandsAndCreatesDiscardPile(t *testing.T) {
	state := NewGameState()
	first := NewPlayer()
	second := NewPlayer()

	if err := state.AddPlayer(first); err != nil {
		t.Fatalf("AddPlayer(first) error = %v", err)
	}
	if err := state.AddPlayer(second); err != nil {
		t.Fatalf("AddPlayer(second) error = %v", err)
	}

	err := state.StartGame(twoPlayerDealerIndex, twoPlayerChooserIndex, DealRoundRobin, nil, 0)

	if err != nil {
		t.Fatalf("StartGame() error = %v", err)
	}
	if state.phase != PhaseInProgress {
		t.Errorf("state.phase = %d; want %d", state.phase, PhaseInProgress)
	}
	if state.turn.number != 1 {
		t.Errorf("state.turn.number = %d; want 1", state.turn.number)
	}
	if state.turn.playerIndex != 1 {
		t.Fatalf("state.turn.playerIndex = %d; want 1", state.turn.playerIndex)
	}
	currentPlayer, err := state.CurrentPlayer()
	if err != nil {
		t.Fatalf("CurrentPlayer() error = %v", err)
	}
	if currentPlayer == nil {
		t.Fatal("CurrentPlayer() returned nil")
	}
	if len(first.hand.cards) != InitialHandSize {
		t.Errorf("len(first.hand.cards) = %d; want %d", len(first.hand.cards), InitialHandSize)
	}
	if len(second.hand.cards) != InitialHandSize {
		t.Errorf("len(second.hand.cards) = %d; want %d", len(second.hand.cards), InitialHandSize)
	}
	if len(state.discardPile.cards) != 1 {
		t.Errorf("len(state.discardPile.cards) = %d; want 1", len(state.discardPile.cards))
	}
	if len(state.drawPile.cards) != 108-(2*InitialHandSize)-1 {
		t.Errorf("len(state.drawPile.cards) = %d; want %d", len(state.drawPile.cards), 108-(2*InitialHandSize)-1)
	}
}

func TestGameStateStartGameRejectsInvalidDealer(t *testing.T) {
	state := NewGameState()
	first := NewPlayer()
	second := NewPlayer()

	if err := state.AddPlayer(first); err != nil {
		t.Fatalf("AddPlayer(first) error = %v", err)
	}
	if err := state.AddPlayer(second); err != nil {
		t.Fatalf("AddPlayer(second) error = %v", err)
	}

	err := state.StartGame(2, twoPlayerChooserIndex, DealRoundRobin, nil, 0)

	if !errors.Is(err, ErrInvalidDealer) {
		t.Errorf("StartGame() error = %v; want %v", err, ErrInvalidDealer)
	}
}

func TestGameStateStartGameRejectsChooserThatIsNotPreviousPlayer(t *testing.T) {
	state := NewGameState()
	first := NewPlayer()
	second := NewPlayer()
	third := NewPlayer()

	if err := state.AddPlayer(first); err != nil {
		t.Fatalf("AddPlayer(first) error = %v", err)
	}
	if err := state.AddPlayer(second); err != nil {
		t.Fatalf("AddPlayer(second) error = %v", err)
	}
	if err := state.AddPlayer(third); err != nil {
		t.Fatalf("AddPlayer(third) error = %v", err)
	}

	err := state.StartGame(1, 2, DealInBlocks, []int{2, 0, 1}, 0)

	if !errors.Is(err, ErrInvalidDealChooser) {
		t.Errorf("StartGame() error = %v; want %v", err, ErrInvalidDealChooser)
	}
}

func TestGameStateStartGameAllowsBlockDealingFromPreviousPlayerChooser(t *testing.T) {
	state := NewGameState()
	first := NewPlayer()
	second := NewPlayer()
	third := NewPlayer()

	if err := state.AddPlayer(first); err != nil {
		t.Fatalf("AddPlayer(first) error = %v", err)
	}
	if err := state.AddPlayer(second); err != nil {
		t.Fatalf("AddPlayer(second) error = %v", err)
	}
	if err := state.AddPlayer(third); err != nil {
		t.Fatalf("AddPlayer(third) error = %v", err)
	}

	err := state.StartGame(1, 0, DealInBlocks, []int{2, 0, 1}, 0)

	if err != nil {
		t.Fatalf("StartGame() error = %v", err)
	}
	if state.phase != PhaseInProgress {
		t.Errorf("state.phase = %d; want %d", state.phase, PhaseInProgress)
	}
	if len(first.hand.cards) != InitialHandSize {
		t.Errorf("len(first.hand.cards) = %d; want %d", len(first.hand.cards), InitialHandSize)
	}
	if len(second.hand.cards) != InitialHandSize {
		t.Errorf("len(second.hand.cards) = %d; want %d", len(second.hand.cards), InitialHandSize)
	}
	if len(third.hand.cards) != InitialHandSize {
		t.Errorf("len(third.hand.cards) = %d; want %d", len(third.hand.cards), InitialHandSize)
	}
	if len(state.discardPile.cards) != 1 {
		t.Errorf("len(state.discardPile.cards) = %d; want 1", len(state.discardPile.cards))
	}
}

func TestGameStateStartGameBuildsDrawPileFromUndealtCardsOnTopOfSetAsidePacket(t *testing.T) {
	state := NewGameState()
	first := NewPlayer()
	second := NewPlayer()
	state.drawPile = &CardPile{cards: []Card{
		card(Ace, Hearts),
		card(Two, Clubs),
		card(Three, Diamonds),
		card(Four, Spades),
		card(Five, Hearts),
		card(Six, Clubs),
		card(Seven, Diamonds),
		card(Eight, Spades),
		card(Nine, Hearts),
		card(Ten, Clubs),
		card(Jack, Diamonds),
		card(Queen, Spades),
		card(King, Hearts),
		card(Ace, Clubs),
		card(Two, Diamonds),
		card(Three, Spades),
		card(Four, Hearts),
		card(Five, Clubs),
		card(Six, Diamonds),
		card(Seven, Spades),
		card(Eight, Hearts),
		card(Nine, Clubs),
		card(Ten, Diamonds),
		card(Jack, Spades),
		card(Queen, Hearts),
		card(King, Clubs),
		card(Ace, Diamonds),
	}}

	if err := state.AddPlayer(first); err != nil {
		t.Fatalf("AddPlayer(first) error = %v", err)
	}
	if err := state.AddPlayer(second); err != nil {
		t.Fatalf("AddPlayer(second) error = %v", err)
	}

	err := state.StartGame(twoPlayerDealerIndex, twoPlayerChooserIndex, DealRoundRobin, nil, 2)

	if err != nil {
		t.Fatalf("StartGame() error = %v", err)
	}
	if len(first.hand.cards) != InitialHandSize {
		t.Fatalf("len(first.hand.cards) = %d; want %d", len(first.hand.cards), InitialHandSize)
	}
	if len(second.hand.cards) != InitialHandSize {
		t.Fatalf("len(second.hand.cards) = %d; want %d", len(second.hand.cards), InitialHandSize)
	}
	if len(state.discardPile.cards) != 1 {
		t.Fatalf("len(state.discardPile.cards) = %d; want 1", len(state.discardPile.cards))
	}
	if got := state.discardPile.cards[0]; !cardsEqual(got, card(Ace, Diamonds)) {
		t.Fatalf("discardPile.cards[0] = %+v; want %+v", got, card(Ace, Diamonds))
	}
	if len(state.drawPile.cards) != 2 {
		t.Fatalf("len(state.drawPile.cards) = %d; want 2", len(state.drawPile.cards))
	}
	if got := state.drawPile.cards[0]; !cardsEqual(got, card(Ace, Hearts)) {
		t.Fatalf("drawPile.cards[0] = %+v; want %+v", got, card(Ace, Hearts))
	}
	if got := state.drawPile.cards[1]; !cardsEqual(got, card(Two, Clubs)) {
		t.Fatalf("drawPile.cards[1] = %+v; want %+v", got, card(Two, Clubs))
	}
}

func TestGameStateStartGameRequiresOpeningDiscardCard(t *testing.T) {
	state := NewGameState()
	first := NewPlayer()
	second := NewPlayer()
	state.drawPile = &CardPile{cards: append([]Card(nil), NewGameDeck().cards[:2*InitialHandSize]...)}

	if err := state.AddPlayer(first); err != nil {
		t.Fatalf("AddPlayer(first) error = %v", err)
	}
	if err := state.AddPlayer(second); err != nil {
		t.Fatalf("AddPlayer(second) error = %v", err)
	}

	err := state.StartGame(twoPlayerDealerIndex, twoPlayerChooserIndex, DealRoundRobin, nil, 0)

	if !errors.Is(err, ErrNotEnoughCardsInDrawPile) {
		t.Fatalf("StartGame() error = %v; want %v", err, ErrNotEnoughCardsInDrawPile)
	}
}

func TestGameStateStartNextRoundAdvancesDealerAndResetsRoundScopedState(t *testing.T) {
	state := NewGameState()
	first := NewPlayer()
	second := NewPlayer()

	if err := state.AddPlayer(first); err != nil {
		t.Fatalf("AddPlayer(first) error = %v", err)
	}
	if err := state.AddPlayer(second); err != nil {
		t.Fatalf("AddPlayer(second) error = %v", err)
	}

	state.phase = PhaseRoundOver
	state.round = 1
	state.dealerIndex = 1
	state.roundWinnerIndex = 0
	state.turn = Turn{number: 9, playerIndex: 1, hasDrawn: true}
	state.activeCompositions = []*Composition{{cards: []Card{
		card(Four, Hearts),
		card(Five, Hearts),
		card(Six, Hearts),
	}}}
	state.drawPile = &CardPile{cards: []Card{card(King, Spades)}}
	state.discardPile = &CardPile{cards: []Card{card(Queen, Clubs), card(Jack, Diamonds)}}
	first.totalPoints = 18
	first.hasOpened = true
	first.hand.cards = []Card{card(Ace, Hearts)}
	second.totalPoints = 37
	second.hasOpened = true
	second.hand.cards = []Card{card(Two, Clubs)}

	err := state.StartNextRound(DealRoundRobin, nil, 0)

	if err != nil {
		t.Fatalf("StartNextRound() error = %v", err)
	}
	if state.phase != PhaseInProgress {
		t.Fatalf("state.phase = %d; want %d", state.phase, PhaseInProgress)
	}
	if state.round != 2 {
		t.Fatalf("state.round = %d; want 2", state.round)
	}
	if state.roundWinnerIndex != -1 {
		t.Fatalf("state.roundWinnerIndex = %d; want -1", state.roundWinnerIndex)
	}
	if state.dealerIndex != 0 {
		t.Fatalf("state.dealerIndex = %d; want 0", state.dealerIndex)
	}
	if state.turn.number != 1 {
		t.Fatalf("state.turn.number = %d; want 1", state.turn.number)
	}
	if state.turn.hasDrawn {
		t.Fatal("state.turn.hasDrawn = true; want false")
	}
	if state.turn.playerIndex != 1 {
		t.Fatalf("state.turn.playerIndex = %d; want 1", state.turn.playerIndex)
	}
	if len(state.activeCompositions) != 0 {
		t.Fatalf("len(state.activeCompositions) = %d; want 0", len(state.activeCompositions))
	}
	if len(first.hand.cards) != InitialHandSize {
		t.Fatalf("len(first.hand.cards) = %d; want %d", len(first.hand.cards), InitialHandSize)
	}
	if len(second.hand.cards) != InitialHandSize {
		t.Fatalf("len(second.hand.cards) = %d; want %d", len(second.hand.cards), InitialHandSize)
	}
	if first.hasOpened {
		t.Fatal("first.hasOpened = true; want false")
	}
	if second.hasOpened {
		t.Fatal("second.hasOpened = true; want false")
	}
	if first.totalPoints != 18 {
		t.Fatalf("first.totalPoints = %d; want 18", first.totalPoints)
	}
	if second.totalPoints != 37 {
		t.Fatalf("second.totalPoints = %d; want 37", second.totalPoints)
	}
	if len(state.discardPile.cards) != 1 {
		t.Fatalf("len(state.discardPile.cards) = %d; want 1", len(state.discardPile.cards))
	}
	if len(state.drawPile.cards) != 108-(2*InitialHandSize)-1 {
		t.Fatalf("len(state.drawPile.cards) = %d; want %d", len(state.drawPile.cards), 108-(2*InitialHandSize)-1)
	}
}

func TestGameStateStartNextRoundRequiresFinishedNonFinalRound(t *testing.T) {
	state := NewGameState()
	first := NewPlayer()
	second := NewPlayer()

	if err := state.AddPlayer(first); err != nil {
		t.Fatalf("AddPlayer(first) error = %v", err)
	}
	if err := state.AddPlayer(second); err != nil {
		t.Fatalf("AddPlayer(second) error = %v", err)
	}

	err := state.StartNextRound(DealRoundRobin, nil, 0)

	if !errors.Is(err, ErrCannotStartNextRound) {
		t.Fatalf("StartNextRound() error = %v; want %v", err, ErrCannotStartNextRound)
	}

	state.phase = PhaseGameOver
	err = state.StartNextRound(DealRoundRobin, nil, 0)

	if !errors.Is(err, ErrCannotStartNextRound) {
		t.Fatalf("StartNextRound() after game over error = %v; want %v", err, ErrCannotStartNextRound)
	}
}

func TestGameStateStartGameRejectsCutThatLeavesTooFewCardsToDeal(t *testing.T) {
	state := NewGameState()
	first := NewPlayer()
	second := NewPlayer()

	if err := state.AddPlayer(first); err != nil {
		t.Fatalf("AddPlayer(first) error = %v", err)
	}
	if err := state.AddPlayer(second); err != nil {
		t.Fatalf("AddPlayer(second) error = %v", err)
	}

	err := state.StartGame(twoPlayerDealerIndex, twoPlayerChooserIndex, DealRoundRobin, nil, len(state.drawPile.cards)-(InitialHandSize*len(state.players))+1)

	if !errors.Is(err, ErrInvalidCutSize) {
		t.Fatalf("StartGame() error = %v; want %v", err, ErrInvalidCutSize)
	}
}

func TestGameStateStartGameRejectsCutWhenDeckWasTapped(t *testing.T) {
	state := NewGameState()
	first := NewPlayer()
	second := NewPlayer()
	third := NewPlayer()

	if err := state.AddPlayer(first); err != nil {
		t.Fatalf("AddPlayer(first) error = %v", err)
	}
	if err := state.AddPlayer(second); err != nil {
		t.Fatalf("AddPlayer(second) error = %v", err)
	}
	if err := state.AddPlayer(third); err != nil {
		t.Fatalf("AddPlayer(third) error = %v", err)
	}

	err := state.StartGame(1, 0, DealInBlocks, []int{2, 0, 1}, 1)

	if !errors.Is(err, ErrInvalidCutSize) {
		t.Fatalf("StartGame() error = %v; want %v", err, ErrInvalidCutSize)
	}
}

func TestGameStateStartGameEndsRoundForDealtSpecialWinningHand(t *testing.T) {
	state := NewGameState()
	first := NewPlayer()
	second := NewPlayer()
	state.drawPile = &CardPile{cards: []Card{
		card(Ace, Hearts),
		card(Two, Hearts),
		card(Three, Hearts),
		card(Four, Hearts),
		card(Five, Hearts),
		card(Six, Hearts),
		card(Seven, Hearts),
		card(Eight, Hearts),
		card(Nine, Hearts),
		card(Ten, Hearts),
		card(Jack, Hearts),
		card(Queen, Hearts),
		card(Two, Clubs),
		card(Three, Clubs),
		card(Four, Clubs),
		card(Five, Clubs),
		card(Six, Clubs),
		card(Seven, Clubs),
		card(Eight, Clubs),
		card(Nine, Clubs),
		card(Ten, Clubs),
		card(Jack, Clubs),
		card(Queen, Clubs),
		card(King, Clubs),
		card(Ace, Spades),
	}}

	if err := state.AddPlayer(first); err != nil {
		t.Fatalf("AddPlayer(first) error = %v", err)
	}
	if err := state.AddPlayer(second); err != nil {
		t.Fatalf("AddPlayer(second) error = %v", err)
	}

	err := state.StartGame(twoPlayerDealerIndex, twoPlayerChooserIndex, DealInBlocks, []int{0, 1}, 0)

	if err != nil {
		t.Fatalf("StartGame() error = %v", err)
	}
	if state.phase != PhaseRoundOver {
		t.Fatalf("state.phase = %d; want %d", state.phase, PhaseRoundOver)
	}
	if state.roundWinnerIndex != 0 {
		t.Fatalf("state.roundWinnerIndex = %d; want 0", state.roundWinnerIndex)
	}
	if first.totalPoints != 0 {
		t.Fatalf("winner totalPoints = %d; want 0", first.totalPoints)
	}
	if second.totalPoints != 84 {
		t.Fatalf("loser totalPoints = %d; want 84", second.totalPoints)
	}
	if len(state.discardPile.cards) != 1 {
		t.Fatalf("len(state.discardPile.cards) = %d; want 1", len(state.discardPile.cards))
	}
	if top := state.discardPile.cards[0]; top.rank != Ace || top.suit != Spades {
		t.Fatalf("top discard = %+v; want Ace of Spades", top)
	}
}

func TestGameStateStartNextRoundAdvancesDealerClockwiseForThreePlayers(t *testing.T) {
	state := NewGameState()
	first := NewPlayer()
	second := NewPlayer()
	third := NewPlayer()

	if err := state.AddPlayer(first); err != nil {
		t.Fatalf("AddPlayer(first) error = %v", err)
	}
	if err := state.AddPlayer(second); err != nil {
		t.Fatalf("AddPlayer(second) error = %v", err)
	}
	if err := state.AddPlayer(third); err != nil {
		t.Fatalf("AddPlayer(third) error = %v", err)
	}

	state.phase = PhaseRoundOver
	state.round = 4
	state.dealerIndex = 1

	err := state.StartNextRound(DealInBlocks, []int{0, 1, 2}, 0)

	if err != nil {
		t.Fatalf("StartNextRound() error = %v", err)
	}
	if state.round != 5 {
		t.Fatalf("state.round = %d; want 5", state.round)
	}
	if state.dealerIndex != 2 {
		t.Fatalf("state.dealerIndex = %d; want 2", state.dealerIndex)
	}
	if state.turn.playerIndex != 0 {
		t.Fatalf("state.turn.playerIndex = %d; want 0", state.turn.playerIndex)
	}
}

func TestDealRoundRobinStartsWithNextPlayerClockwiseFromDealer(t *testing.T) {
	players := []*Player{NewPlayer(), NewPlayer(), NewPlayer()}
	drawPile := &CardPile{cards: []Card{
		{rank: Ace, suit: Hearts},
		{rank: Two, suit: Hearts},
		{rank: Three, suit: Hearts},
		{rank: Four, suit: Hearts},
		{rank: Five, suit: Hearts},
		{rank: Six, suit: Hearts},
		{rank: Seven, suit: Hearts},
		{rank: Eight, suit: Hearts},
		{rank: Nine, suit: Hearts},
		{rank: Ten, suit: Hearts},
		{rank: Jack, suit: Hearts},
		{rank: Queen, suit: Hearts},
		{rank: King, suit: Hearts},
		{rank: Ace, suit: Diamonds},
		{rank: Two, suit: Diamonds},
		{rank: Three, suit: Diamonds},
		{rank: Four, suit: Diamonds},
		{rank: Five, suit: Diamonds},
		{rank: Six, suit: Diamonds},
		{rank: Seven, suit: Diamonds},
		{rank: Eight, suit: Diamonds},
		{rank: Nine, suit: Diamonds},
		{rank: Ten, suit: Diamonds},
		{rank: Jack, suit: Diamonds},
		{rank: Queen, suit: Diamonds},
		{rank: King, suit: Diamonds},
		{rank: Ace, suit: Clubs},
		{rank: Two, suit: Clubs},
		{rank: Three, suit: Clubs},
		{rank: Four, suit: Clubs},
		{rank: Five, suit: Clubs},
		{rank: Six, suit: Clubs},
		{rank: Seven, suit: Clubs},
		{rank: Eight, suit: Clubs},
		{rank: Nine, suit: Clubs},
		{rank: Ten, suit: Clubs},
	}}

	err := dealRoundRobin(players, drawPile, 1, 0)

	if err != nil {
		t.Fatalf("dealRoundRobin() error = %v", err)
	}
	if len(players[2].hand.cards) != InitialHandSize {
		t.Fatalf("len(players[2].hand.cards) = %d; want %d", len(players[2].hand.cards), InitialHandSize)
	}
	if len(players[0].hand.cards) != InitialHandSize {
		t.Fatalf("len(players[0].hand.cards) = %d; want %d", len(players[0].hand.cards), InitialHandSize)
	}
	if len(players[1].hand.cards) != InitialHandSize {
		t.Fatalf("len(players[1].hand.cards) = %d; want %d", len(players[1].hand.cards), InitialHandSize)
	}

	if first := players[2].hand.cards[0]; first.rank != Ace || first.suit != Hearts {
		t.Errorf("players[2].hand.cards[0] = %+v; want Ace of Hearts", first)
	}
	if first := players[0].hand.cards[0]; first.rank != Two || first.suit != Hearts {
		t.Errorf("players[0].hand.cards[0] = %+v; want Two of Hearts", first)
	}
	if first := players[1].hand.cards[0]; first.rank != Three || first.suit != Hearts {
		t.Errorf("players[1].hand.cards[0] = %+v; want Three of Hearts", first)
	}
}

func TestDealInBlocksUsesChosenOrder(t *testing.T) {
	players := []*Player{NewPlayer(), NewPlayer(), NewPlayer()}
	drawPile := &CardPile{cards: []Card{
		card(Ace, Hearts),
		card(Two, Hearts),
		card(Three, Hearts),
		card(Four, Hearts),
		card(Five, Hearts),
		card(Six, Hearts),
		card(Seven, Hearts),
		card(Eight, Hearts),
		card(Nine, Hearts),
		card(Ten, Hearts),
		card(Jack, Hearts),
		card(Queen, Hearts),
		card(King, Hearts),
		card(Ace, Clubs),
		card(Two, Clubs),
		card(Three, Clubs),
		card(Four, Clubs),
		card(Five, Clubs),
		card(Six, Clubs),
		card(Seven, Clubs),
		card(Eight, Clubs),
		card(Nine, Clubs),
		card(Ten, Clubs),
		card(Jack, Clubs),
		card(Queen, Clubs),
		card(King, Clubs),
		card(Ace, Spades),
		card(Two, Spades),
		card(Three, Spades),
		card(Four, Spades),
		card(Five, Spades),
		card(Six, Spades),
		card(Seven, Spades),
		card(Eight, Spades),
		card(Nine, Spades),
		card(Ten, Spades),
	}}

	err := dealInBlocks(players, drawPile, []int{2, 0, 1})

	if err != nil {
		t.Fatalf("dealInBlocks() error = %v", err)
	}
	if len(players[2].hand.cards) != InitialHandSize {
		t.Fatalf("len(players[2].hand.cards) = %d; want %d", len(players[2].hand.cards), InitialHandSize)
	}
	if len(players[0].hand.cards) != InitialHandSize {
		t.Fatalf("len(players[0].hand.cards) = %d; want %d", len(players[0].hand.cards), InitialHandSize)
	}
	if len(players[1].hand.cards) != InitialHandSize {
		t.Fatalf("len(players[1].hand.cards) = %d; want %d", len(players[1].hand.cards), InitialHandSize)
	}
	if got := players[2].hand.cards[0]; !cardsEqual(got, card(Ace, Hearts)) {
		t.Fatalf("players[2].hand.cards[0] = %+v; want %+v", got, card(Ace, Hearts))
	}
	if got := players[0].hand.cards[0]; !cardsEqual(got, card(King, Hearts)) {
		t.Fatalf("players[0].hand.cards[0] = %+v; want %+v", got, card(King, Hearts))
	}
	if got := players[1].hand.cards[0]; !cardsEqual(got, card(Queen, Clubs)) {
		t.Fatalf("players[1].hand.cards[0] = %+v; want %+v", got, card(Queen, Clubs))
	}
}

func TestGameStateCurrentPlayerRequiresPlayers(t *testing.T) {
	state := NewGameState()

	_, err := state.CurrentPlayer()

	if !errors.Is(err, ErrNoPlayers) {
		t.Errorf("CurrentPlayer() error = %v; want %v", err, ErrNoPlayers)
	}
}

func TestGameStateDrawFromDeckRequiresGameInProgress(t *testing.T) {
	state := NewGameState()

	err := state.DrawFromDeck()

	if !errors.Is(err, ErrGameNotInProgress) {
		t.Errorf("DrawFromDeck() error = %v; want %v", err, ErrGameNotInProgress)
	}
}

func TestGameStateDrawFromDeckDrawsCardAndMarksTurn(t *testing.T) {
	state := NewGameState()
	first := NewPlayer()
	second := NewPlayer()

	if err := state.AddPlayer(first); err != nil {
		t.Fatalf("AddPlayer(first) error = %v", err)
	}
	if err := state.AddPlayer(second); err != nil {
		t.Fatalf("AddPlayer(second) error = %v", err)
	}
	if err := state.StartGame(twoPlayerDealerIndex, twoPlayerChooserIndex, DealRoundRobin, nil, 0); err != nil {
		t.Fatalf("StartGame() error = %v", err)
	}

	currentPlayer, err := state.CurrentPlayer()
	if err != nil {
		t.Fatalf("CurrentPlayer() error = %v", err)
	}
	startingHandSize := len(currentPlayer.hand.cards)
	startingDrawPileSize := len(state.drawPile.cards)

	err = state.DrawFromDeck()

	if err != nil {
		t.Fatalf("DrawFromDeck() error = %v", err)
	}
	if len(currentPlayer.hand.cards) != startingHandSize+1 {
		t.Errorf("len(currentPlayer.hand.cards) = %d; want %d", len(currentPlayer.hand.cards), startingHandSize+1)
	}
	if len(state.drawPile.cards) != startingDrawPileSize-1 {
		t.Errorf("len(state.drawPile.cards) = %d; want %d", len(state.drawPile.cards), startingDrawPileSize-1)
	}
	if !state.turn.hasDrawn {
		t.Error("state.turn.hasDrawn = false; want true")
	}
	if state.turn.number != 1 {
		t.Errorf("state.turn.number = %d; want 1", state.turn.number)
	}
}

func TestGameStateDrawFromDeckDoesNotEndRoundForJokerSameSuitCollection(t *testing.T) {
	state := newTurnTestState()
	state.players[0].hand.cards = []Card{
		card(Ace, Hearts),
		card(Two, Hearts),
		card(Three, Hearts),
		card(Four, Hearts),
		card(Five, Hearts),
		card(Six, Hearts),
		card(Seven, Hearts),
		card(Eight, Hearts),
		card(Nine, Hearts),
		joker(),
		joker(),
	}
	state.drawPile = &CardPile{cards: []Card{joker()}}
	state.players[1].totalPoints = 10
	state.players[1].hand.cards = []Card{card(King, Clubs)}

	err := state.DrawFromDeck()

	if err != nil {
		t.Fatalf("DrawFromDeck() error = %v", err)
	}
	if state.phase != PhaseInProgress {
		t.Fatalf("state.phase = %d; want %d", state.phase, PhaseInProgress)
	}
	if state.roundWinnerIndex != -1 {
		t.Fatalf("state.roundWinnerIndex = %d; want -1", state.roundWinnerIndex)
	}
	if state.players[1].totalPoints != 10 {
		t.Fatalf("player 1 totalPoints = %d; want 10", state.players[1].totalPoints)
	}
	if state.turn.mustUseDiscardDraw {
		t.Fatal("state.turn.mustUseDiscardDraw = true; want false")
	}
}

func TestGameStateDrawFromDeckRejectsSecondDrawSameTurn(t *testing.T) {
	state := NewGameState()
	first := NewPlayer()
	second := NewPlayer()

	if err := state.AddPlayer(first); err != nil {
		t.Fatalf("AddPlayer(first) error = %v", err)
	}
	if err := state.AddPlayer(second); err != nil {
		t.Fatalf("AddPlayer(second) error = %v", err)
	}
	if err := state.StartGame(twoPlayerDealerIndex, twoPlayerChooserIndex, DealRoundRobin, nil, 0); err != nil {
		t.Fatalf("StartGame() error = %v", err)
	}
	if err := state.DrawFromDeck(); err != nil {
		t.Fatalf("first DrawFromDeck() error = %v", err)
	}

	err := state.DrawFromDeck()

	if !errors.Is(err, ErrPlayerAlreadyDrew) {
		t.Errorf("second DrawFromDeck() error = %v; want %v", err, ErrPlayerAlreadyDrew)
	}
}

func TestGameStateDrawFromDeckRecyclesDiscardPileWhenDrawPileIsEmpty(t *testing.T) {
	state := newTurnTestState()
	state.drawPile = &CardPile{cards: []Card{}}
	state.discardPile = &CardPile{cards: []Card{
		{rank: Queen, suit: Spades},
		{rank: Five, suit: Clubs},
		{rank: Ace, suit: Hearts},
	}}

	currentPlayer, err := state.CurrentPlayer()
	if err != nil {
		t.Fatalf("CurrentPlayer() error = %v", err)
	}
	startingHandSize := len(currentPlayer.hand.cards)

	err = state.DrawFromDeck()

	if err != nil {
		t.Fatalf("DrawFromDeck() error = %v", err)
	}
	if len(currentPlayer.hand.cards) != startingHandSize+1 {
		t.Fatalf("len(currentPlayer.hand.cards) = %d; want %d", len(currentPlayer.hand.cards), startingHandSize+1)
	}
	if drawn := currentPlayer.hand.cards[len(currentPlayer.hand.cards)-1]; drawn.rank != Ace || drawn.suit != Hearts {
		t.Fatalf("drawn card = %+v; want Ace of Hearts", drawn)
	}
	if len(state.drawPile.cards) != 1 {
		t.Fatalf("len(state.drawPile.cards) = %d; want 1", len(state.drawPile.cards))
	}
	if top := state.drawPile.cards[0]; top.rank != Five || top.suit != Clubs {
		t.Fatalf("drawPile.cards[0] = %+v; want Five of Clubs", top)
	}
	if len(state.discardPile.cards) != 1 {
		t.Fatalf("len(state.discardPile.cards) = %d; want 1", len(state.discardPile.cards))
	}
	if top := state.discardPile.cards[0]; top.rank != Queen || top.suit != Spades {
		t.Fatalf("discardPile.cards[0] = %+v; want Queen of Spades", top)
	}
	if !state.turn.hasDrawn {
		t.Fatal("state.turn.hasDrawn = false; want true")
	}
}

func TestGameStateDrawFromDeckDoesNotRecycleLastDiscardCard(t *testing.T) {
	state := newTurnTestState()
	state.drawPile = &CardPile{cards: []Card{}}
	state.discardPile = &CardPile{cards: []Card{{rank: Queen, suit: Spades}}}

	err := state.DrawFromDeck()

	if !errors.Is(err, ErrNotEnoughCardsInDrawPile) {
		t.Fatalf("DrawFromDeck() error = %v; want %v", err, ErrNotEnoughCardsInDrawPile)
	}
	if len(state.discardPile.cards) != 1 {
		t.Fatalf("len(state.discardPile.cards) = %d; want 1", len(state.discardPile.cards))
	}
}

func TestGameStateDrawFromDiscardAllowsImmediateJokerReclaim(t *testing.T) {
	state := newTurnTestState()
	state.players[0].hasOpened = true
	state.players[0].hand.cards = []Card{{rank: Two, suit: Clubs}}
	state.discardPile = &CardPile{cards: []Card{{rank: Six, suit: Hearts}}}

	base, ok := NewRun([]Card{
		{rank: Five, suit: Hearts},
		{isJoker: true},
		{rank: Seven, suit: Hearts},
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}
	state.activeCompositions = []*Composition{base}

	err := state.DrawFromDiscard()

	if err != nil {
		t.Fatalf("DrawFromDiscard() error = %v", err)
	}
	if len(state.players[0].hand.cards) != 2 {
		t.Fatalf("len(state.players[0].hand.cards) = %d; want 2", len(state.players[0].hand.cards))
	}

	err = state.PlayTable(nil, nil, JokerReclaim{
		CompositionIndex: 0,
		JokerIndex:       1,
		ReplacementCard:  card(Six, Hearts),
	})

	if err != nil {
		t.Fatalf("PlayTable() error = %v", err)
	}
	if got := state.activeCompositions[0].cards[1]; !cardsEqual(got, card(Six, Hearts)) {
		t.Fatalf("state.activeCompositions[0].cards[1] = %+v; want %+v", got, card(Six, Hearts))
	}
	if len(state.players[0].hand.cards) != 2 {
		t.Fatalf("len(state.players[0].hand.cards) = %d; want 2", len(state.players[0].hand.cards))
	}
	foundJoker := false
	foundTwoClubs := false
	for _, handCard := range state.players[0].hand.cards {
		if handCard.isJoker {
			foundJoker = true
		}
		if cardsEqual(handCard, card(Two, Clubs)) {
			foundTwoClubs = true
		}
	}
	if !foundJoker {
		t.Fatal("player hand does not contain reclaimed joker")
	}
	if !foundTwoClubs {
		t.Fatal("player hand does not contain original side card")
	}
}

func TestGameStateDrawFromDeckStillFailsWhenBothPilesAreEmpty(t *testing.T) {
	state := newTurnTestState()
	state.drawPile = &CardPile{cards: []Card{}}
	state.discardPile = &CardPile{cards: []Card{}}

	err := state.DrawFromDeck()

	if !errors.Is(err, ErrNotEnoughCardsInDrawPile) {
		t.Fatalf("DrawFromDeck() error = %v; want %v", err, ErrNotEnoughCardsInDrawPile)
	}
	if state.turn.hasDrawn {
		t.Fatal("state.turn.hasDrawn = true; want false")
	}
}

func TestGameStateDrawFromDiscardAllowsOpenedPlayerToUseDiscardInAddition(t *testing.T) {
	state := newTurnTestState()
	state.players[0].hasOpened = true
	state.players[0].hand.cards = []Card{
		{rank: Jack, suit: Hearts},
		{rank: Two, suit: Clubs},
	}
	state.discardPile = &CardPile{cards: []Card{{rank: Ten, suit: Hearts}}}

	base, ok := NewRun([]Card{
		{rank: Seven, suit: Hearts},
		{rank: Eight, suit: Hearts},
		{rank: Nine, suit: Hearts},
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}
	state.activeCompositions = []*Composition{base}

	err := state.DrawFromDiscard()

	if err != nil {
		t.Fatalf("DrawFromDiscard() error = %v", err)
	}
	if len(state.discardPile.cards) != 0 {
		t.Fatalf("len(state.discardPile.cards) = %d; want 0", len(state.discardPile.cards))
	}
	if len(state.players[0].hand.cards) != 3 {
		t.Fatalf("len(state.players[0].hand.cards) = %d; want 3", len(state.players[0].hand.cards))
	}
	if drawn := state.players[0].hand.cards[2]; drawn.rank != Ten || drawn.suit != Hearts {
		t.Fatalf("drawn card = %+v; want Ten of Hearts", drawn)
	}
	if !state.turn.hasDrawn {
		t.Fatal("state.turn.hasDrawn = false; want true")
	}
}

func TestGameStateDrawFromDiscardRejectsOpenedPlayerWhenCardIsNotUsable(t *testing.T) {
	state := newTurnTestState()
	state.players[0].hasOpened = true
	state.players[0].hand.cards = []Card{
		{rank: Two, suit: Clubs},
		{rank: Three, suit: Diamonds},
	}
	state.discardPile = &CardPile{cards: []Card{{rank: Five, suit: Spades}}}

	base, ok := NewRun([]Card{
		{rank: Seven, suit: Hearts},
		{rank: Eight, suit: Hearts},
		{rank: Nine, suit: Hearts},
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}
	state.activeCompositions = []*Composition{base}

	err := state.DrawFromDiscard()

	if !errors.Is(err, ErrCannotTakeDiscardCard) {
		t.Fatalf("DrawFromDiscard() error = %v; want %v", err, ErrCannotTakeDiscardCard)
	}
	if len(state.discardPile.cards) != 1 {
		t.Fatalf("len(state.discardPile.cards) = %d; want 1", len(state.discardPile.cards))
	}
	if state.turn.hasDrawn {
		t.Fatal("state.turn.hasDrawn = true; want false")
	}
}

func TestGameStateDrawFromDiscardAllowsOpeningWithCompositionAndAddition(t *testing.T) {
	state := newTurnTestState()
	state.players[0].hand.cards = []Card{
		{rank: King, suit: Hearts},
		{rank: King, suit: Diamonds},
		{rank: King, suit: Clubs},
		{rank: Two, suit: Spades},
	}
	state.discardPile = &CardPile{cards: []Card{{rank: Ten, suit: Hearts}}}

	base, ok := NewRun([]Card{
		{rank: Seven, suit: Hearts},
		{rank: Eight, suit: Hearts},
		{rank: Nine, suit: Hearts},
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}
	state.activeCompositions = []*Composition{base}

	err := state.DrawFromDiscard()

	if err != nil {
		t.Fatalf("DrawFromDiscard() error = %v", err)
	}
	if len(state.discardPile.cards) != 0 {
		t.Fatalf("len(state.discardPile.cards) = %d; want 0", len(state.discardPile.cards))
	}
	if len(state.players[0].hand.cards) != 5 {
		t.Fatalf("len(state.players[0].hand.cards) = %d; want 5", len(state.players[0].hand.cards))
	}
	if !state.turn.hasDrawn {
		t.Fatal("state.turn.hasDrawn = false; want true")
	}
}

func TestGameStateDrawFromDiscardRejectsUnopenedPlayerWithoutOwnComposition(t *testing.T) {
	state := newTurnTestState()
	state.players[0].hand.cards = []Card{
		{rank: Ten, suit: Hearts},
		{rank: Two, suit: Clubs},
	}
	state.discardPile = &CardPile{cards: []Card{{rank: Jack, suit: Hearts}}}

	base, ok := NewRun([]Card{
		{rank: Seven, suit: Hearts},
		{rank: Eight, suit: Hearts},
		{rank: Nine, suit: Hearts},
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}
	state.activeCompositions = []*Composition{base}

	err := state.DrawFromDiscard()

	if !errors.Is(err, ErrCannotTakeDiscardCard) {
		t.Fatalf("DrawFromDiscard() error = %v; want %v", err, ErrCannotTakeDiscardCard)
	}
	if len(state.discardPile.cards) != 1 {
		t.Fatalf("len(state.discardPile.cards) = %d; want 1", len(state.discardPile.cards))
	}
}

func TestGameStateDrawFromDiscardRejectsOpeningBelowForty(t *testing.T) {
	state := newTurnTestState()
	state.players[0].hand.cards = []Card{
		{rank: Seven, suit: Clubs},
		{rank: Seven, suit: Diamonds},
		{rank: Seven, suit: Spades},
		{rank: Two, suit: Clubs},
	}
	state.discardPile = &CardPile{cards: []Card{{rank: Ten, suit: Hearts}}}

	base, ok := NewRun([]Card{
		{rank: Seven, suit: Hearts},
		{rank: Eight, suit: Hearts},
		{rank: Nine, suit: Hearts},
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}
	state.activeCompositions = []*Composition{base}

	err := state.DrawFromDiscard()

	if !errors.Is(err, ErrCannotTakeDiscardCard) {
		t.Fatalf("DrawFromDiscard() error = %v; want %v", err, ErrCannotTakeDiscardCard)
	}
	if state.turn.hasDrawn {
		t.Fatal("state.turn.hasDrawn = true; want false")
	}
}

func TestGameStatePlayCompositionsRequiresDrawFirst(t *testing.T) {
	state := newTurnTestState()
	state.players[0].hand.cards = []Card{
		{rank: Seven, suit: Hearts},
		{rank: Seven, suit: Diamonds},
		{rank: Seven, suit: Clubs},
	}
	comp, ok := NewSet([]Card{
		{rank: Seven, suit: Hearts},
		{rank: Seven, suit: Diamonds},
		{rank: Seven, suit: Clubs},
	})
	if !ok {
		t.Fatal("NewSet() returned false; want true")
	}

	err := state.PlayCompositions([]*Composition{comp})

	if !errors.Is(err, ErrPlayerHasntDrawn) {
		t.Errorf("PlayCompositions() error = %v; want %v", err, ErrPlayerHasntDrawn)
	}
}

func TestGameStatePlayCompositionsRejectsNilComposition(t *testing.T) {
	state := newTurnTestState()
	state.turn.hasDrawn = true

	err := state.PlayCompositions([]*Composition{nil})

	if !errors.Is(err, ErrInvalidComposition) {
		t.Errorf("PlayCompositions(nil) error = %v; want %v", err, ErrInvalidComposition)
	}
}

func TestGameStatePlayCompositionsMovesCardsToActiveCompositions(t *testing.T) {
	state := newTurnTestState()
	state.turn.hasDrawn = true
	state.players[0].hasOpened = true
	state.players[0].hand.cards = []Card{
		{rank: Seven, suit: Hearts},
		{rank: Seven, suit: Diamonds},
		{rank: Seven, suit: Clubs},
		{rank: King, suit: Spades},
	}
	comp, ok := NewSet([]Card{
		{rank: Seven, suit: Hearts},
		{rank: Seven, suit: Diamonds},
		{rank: Seven, suit: Clubs},
	})
	if !ok {
		t.Fatal("NewSet() returned false; want true")
	}

	err := state.PlayCompositions([]*Composition{comp})

	if err != nil {
		t.Fatalf("PlayCompositions() error = %v", err)
	}
	if len(state.activeCompositions) != 1 {
		t.Fatalf("len(state.activeCompositions) = %d; want 1", len(state.activeCompositions))
	}
	if len(state.players[0].hand.cards) != 1 {
		t.Fatalf("len(state.players[0].hand.cards) = %d; want 1", len(state.players[0].hand.cards))
	}
	remaining := state.players[0].hand.cards[0]
	if remaining.rank != King || remaining.suit != Spades {
		t.Errorf("remaining hand card = %+v; want King of Spades", remaining)
	}
	if state.activeCompositions[0] != comp {
		t.Error("active composition was not appended correctly")
	}
}

func TestGameStatePlayCompositionsAcceptsDescendingRun(t *testing.T) {
	state := newTurnTestState()
	state.turn.hasDrawn = true
	state.players[0].hasOpened = true
	state.players[0].hand.cards = []Card{
		{rank: Seven, suit: Hearts},
		{rank: Six, suit: Hearts},
		{rank: Five, suit: Hearts},
		{rank: King, suit: Spades},
	}
	runComp := &Composition{variant: run, cards: []Card{
		{rank: Seven, suit: Hearts},
		{rank: Six, suit: Hearts},
		{rank: Five, suit: Hearts},
	}}

	err := state.PlayCompositions([]*Composition{runComp})

	if err != nil {
		t.Fatalf("PlayCompositions() error = %v", err)
	}
	if len(state.activeCompositions) != 1 {
		t.Fatalf("len(state.activeCompositions) = %d; want 1", len(state.activeCompositions))
	}
	if len(state.players[0].hand.cards) != 1 {
		t.Fatalf("len(state.players[0].hand.cards) = %d; want 1", len(state.players[0].hand.cards))
	}
	if got := state.activeCompositions[0].cards; !slices.Equal(got, []Card{
		{rank: Five, suit: Hearts},
		{rank: Six, suit: Hearts},
		{rank: Seven, suit: Hearts},
	}) {
		t.Fatalf("active run cards = %#v; want normalized ascending order", got)
	}
}

func TestGameStatePlayCompositionsRejectsMixedOrderRun(t *testing.T) {
	state := newTurnTestState()
	state.turn.hasDrawn = true
	state.players[0].hasOpened = true
	state.players[0].hand.cards = []Card{
		{rank: Seven, suit: Hearts},
		{rank: Five, suit: Hearts},
		{rank: Six, suit: Hearts},
		{rank: King, suit: Spades},
	}
	runComp := &Composition{variant: run, cards: []Card{
		{rank: Seven, suit: Hearts},
		{rank: Five, suit: Hearts},
		{rank: Six, suit: Hearts},
	}}

	err := state.PlayCompositions([]*Composition{runComp})

	if !errors.Is(err, ErrInvalidComposition) {
		t.Fatalf("PlayCompositions() error = %v; want %v", err, ErrInvalidComposition)
	}
	if len(state.activeCompositions) != 0 {
		t.Fatalf("len(state.activeCompositions) = %d; want 0", len(state.activeCompositions))
	}
	if len(state.players[0].hand.cards) != 4 {
		t.Fatalf("len(state.players[0].hand.cards) = %d; want 4", len(state.players[0].hand.cards))
	}
}

func TestGameStatePlayCompositionsPlaysMultipleAtOnce(t *testing.T) {
	state := newTurnTestState()
	state.turn.hasDrawn = true
	state.players[0].hasOpened = true
	state.players[0].hand.cards = []Card{
		{rank: Seven, suit: Hearts},
		{rank: Seven, suit: Diamonds},
		{rank: Seven, suit: Clubs},
		{rank: Three, suit: Spades},
		{rank: Four, suit: Spades},
		{rank: Five, suit: Spades},
		{rank: King, suit: Hearts},
	}
	setComp, ok := NewSet([]Card{
		{rank: Seven, suit: Hearts},
		{rank: Seven, suit: Diamonds},
		{rank: Seven, suit: Clubs},
	})
	if !ok {
		t.Fatal("NewSet() returned false; want true")
	}
	runComp, ok := NewRun([]Card{
		{rank: Three, suit: Spades},
		{rank: Four, suit: Spades},
		{rank: Five, suit: Spades},
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}

	err := state.PlayCompositions([]*Composition{setComp, runComp})

	if err != nil {
		t.Fatalf("PlayCompositions() error = %v", err)
	}
	if len(state.activeCompositions) != 2 {
		t.Fatalf("len(state.activeCompositions) = %d; want 2", len(state.activeCompositions))
	}
	if len(state.players[0].hand.cards) != 1 {
		t.Fatalf("len(state.players[0].hand.cards) = %d; want 1", len(state.players[0].hand.cards))
	}
	remaining := state.players[0].hand.cards[0]
	if remaining.rank != King || remaining.suit != Hearts {
		t.Errorf("remaining hand card = %+v; want King of Hearts", remaining)
	}
	if state.activeCompositions[0] != setComp {
		t.Error("set composition was not appended correctly")
	}
	if state.activeCompositions[1] != runComp {
		t.Error("run composition was not appended correctly")
	}
}

func TestGameStatePlayCompositionsRejectsCardsNotInHand(t *testing.T) {
	state := newTurnTestState()
	state.turn.hasDrawn = true
	state.players[0].hasOpened = true
	state.players[0].hand.cards = []Card{
		{rank: Seven, suit: Hearts},
		{rank: Seven, suit: Diamonds},
		{rank: King, suit: Spades},
	}
	comp, ok := NewSet([]Card{
		{rank: Seven, suit: Hearts},
		{rank: Seven, suit: Diamonds},
		{rank: Seven, suit: Clubs},
	})
	if !ok {
		t.Fatal("NewSet() returned false; want true")
	}

	err := state.PlayCompositions([]*Composition{comp})

	if !errors.Is(err, ErrCardsNotInHand) {
		t.Errorf("PlayCompositions() error = %v; want %v", err, ErrCardsNotInHand)
	}
	if len(state.activeCompositions) != 0 {
		t.Errorf("len(state.activeCompositions) = %d; want 0", len(state.activeCompositions))
	}
	if len(state.players[0].hand.cards) != 3 {
		t.Errorf("len(state.players[0].hand.cards) = %d; want 3", len(state.players[0].hand.cards))
	}
}

func TestGameStatePlayCompositionsRequiresCardLeftForDiscard(t *testing.T) {
	state := newTurnTestState()
	state.turn.hasDrawn = true
	state.players[0].hasOpened = true
	state.players[0].hand.cards = []Card{
		{rank: Seven, suit: Hearts},
		{rank: Seven, suit: Diamonds},
		{rank: Seven, suit: Clubs},
	}
	comp, ok := NewSet([]Card{
		{rank: Seven, suit: Hearts},
		{rank: Seven, suit: Diamonds},
		{rank: Seven, suit: Clubs},
	})
	if !ok {
		t.Fatal("NewSet() returned false; want true")
	}

	err := state.PlayCompositions([]*Composition{comp})

	if !errors.Is(err, ErrMustKeepDiscardCard) {
		t.Fatalf("PlayCompositions() error = %v; want %v", err, ErrMustKeepDiscardCard)
	}
	if len(state.activeCompositions) != 0 {
		t.Fatalf("len(state.activeCompositions) = %d; want 0", len(state.activeCompositions))
	}
	if len(state.players[0].hand.cards) != 3 {
		t.Fatalf("len(state.players[0].hand.cards) = %d; want 3", len(state.players[0].hand.cards))
	}
	if state.phase != PhaseInProgress {
		t.Fatalf("state.phase = %d; want %d", state.phase, PhaseInProgress)
	}
	if state.turn.playerIndex != 0 {
		t.Fatalf("state.turn.playerIndex = %d; want 0", state.turn.playerIndex)
	}
}

func TestGameStatePlayCompositionsDoesNotPartiallyMutateOnFailure(t *testing.T) {
	state := newTurnTestState()
	state.turn.hasDrawn = true
	state.players[0].hasOpened = true
	state.players[0].hand.cards = []Card{
		{rank: Seven, suit: Hearts},
		{rank: Seven, suit: Diamonds},
		{rank: Seven, suit: Clubs},
		{rank: Three, suit: Spades},
		{rank: Four, suit: Spades},
		{rank: King, suit: Hearts},
	}
	setComp, ok := NewSet([]Card{
		{rank: Seven, suit: Hearts},
		{rank: Seven, suit: Diamonds},
		{rank: Seven, suit: Clubs},
	})
	if !ok {
		t.Fatal("NewSet() returned false; want true")
	}
	runComp, ok := NewRun([]Card{
		{rank: Three, suit: Spades},
		{rank: Four, suit: Spades},
		{rank: Five, suit: Spades},
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}

	err := state.PlayCompositions([]*Composition{setComp, runComp})

	if !errors.Is(err, ErrCardsNotInHand) {
		t.Fatalf("PlayCompositions() error = %v; want %v", err, ErrCardsNotInHand)
	}
	if len(state.activeCompositions) != 0 {
		t.Fatalf("len(state.activeCompositions) = %d; want 0", len(state.activeCompositions))
	}
	if len(state.players[0].hand.cards) != 6 {
		t.Fatalf("len(state.players[0].hand.cards) = %d; want 6", len(state.players[0].hand.cards))
	}
}

func TestGameStatePlayCompositionsRejectsOpeningBelowFortyPoints(t *testing.T) {
	state := newTurnTestState()
	state.turn.hasDrawn = true
	state.players[0].hand.cards = []Card{
		{rank: Seven, suit: Hearts},
		{rank: Seven, suit: Diamonds},
		{rank: Seven, suit: Clubs},
		{rank: King, suit: Spades},
	}
	comp, ok := NewSet([]Card{
		{rank: Seven, suit: Hearts},
		{rank: Seven, suit: Diamonds},
		{rank: Seven, suit: Clubs},
	})
	if !ok {
		t.Fatal("NewSet() returned false; want true")
	}

	err := state.PlayCompositions([]*Composition{comp})

	if !errors.Is(err, ErrInitialPointsNotMet) {
		t.Fatalf("PlayCompositions() error = %v; want %v", err, ErrInitialPointsNotMet)
	}
	if state.players[0].hasOpened {
		t.Fatal("player.hasOpened = true; want false")
	}
	if len(state.activeCompositions) != 0 {
		t.Fatalf("len(state.activeCompositions) = %d; want 0", len(state.activeCompositions))
	}
	if len(state.players[0].hand.cards) != 4 {
		t.Fatalf("len(state.players[0].hand.cards) = %d; want 4", len(state.players[0].hand.cards))
	}
}

func TestGameStatePlayCompositionsAllowsOpeningAtFortyPoints(t *testing.T) {
	state := newTurnTestState()
	state.turn.hasDrawn = true
	state.players[0].hand.cards = []Card{
		{rank: King, suit: Hearts},
		{rank: King, suit: Diamonds},
		{rank: King, suit: Clubs},
		{rank: Ace, suit: Spades},
		{rank: Two, suit: Spades},
		{rank: Three, suit: Spades},
		{rank: Four, suit: Spades},
		{rank: Nine, suit: Hearts},
	}
	setComp, ok := NewSet([]Card{
		{rank: King, suit: Hearts},
		{rank: King, suit: Diamonds},
		{rank: King, suit: Clubs},
	})
	if !ok {
		t.Fatal("NewSet() returned false; want true")
	}
	runComp, ok := NewRun([]Card{
		{rank: Ace, suit: Spades},
		{rank: Two, suit: Spades},
		{rank: Three, suit: Spades},
		{rank: Four, suit: Spades},
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}

	err := state.PlayCompositions([]*Composition{setComp, runComp})

	if err != nil {
		t.Fatalf("PlayCompositions() error = %v", err)
	}
	if !state.players[0].hasOpened {
		t.Fatal("player.hasOpened = false; want true")
	}
	if len(state.activeCompositions) != 2 {
		t.Fatalf("len(state.activeCompositions) = %d; want 2", len(state.activeCompositions))
	}
	if len(state.players[0].hand.cards) != 1 {
		t.Fatalf("len(state.players[0].hand.cards) = %d; want 1", len(state.players[0].hand.cards))
	}
	remaining := state.players[0].hand.cards[0]
	if remaining.rank != Nine || remaining.suit != Hearts {
		t.Fatalf("remaining hand card = %+v; want Nine of Hearts", remaining)
	}
}

func TestGameStateAddToCompositionsAllowsOpenedPlayerToAddCards(t *testing.T) {
	state := newTurnTestState()
	state.turn.hasDrawn = true
	state.players[0].hasOpened = true
	base, ok := NewRun([]Card{
		{rank: Seven, suit: Hearts},
		{rank: Eight, suit: Hearts},
		{rank: Nine, suit: Hearts},
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}
	state.activeCompositions = []*Composition{base}
	state.players[0].hand.cards = []Card{
		{rank: Ten, suit: Hearts},
		{rank: Jack, suit: Hearts},
		{rank: Two, suit: Clubs},
	}

	err := state.AddToCompositions([]CompositionAddition{{
		CompositionIndex: 0,
		Cards: []Card{
			{rank: Ten, suit: Hearts},
			{rank: Jack, suit: Hearts},
		},
	}})

	if err != nil {
		t.Fatalf("AddToCompositions() error = %v", err)
	}
	if len(state.activeCompositions) != 1 {
		t.Fatalf("len(state.activeCompositions) = %d; want 1", len(state.activeCompositions))
	}
	if len(state.activeCompositions[0].cards) != 5 {
		t.Fatalf("len(state.activeCompositions[0].cards) = %d; want 5", len(state.activeCompositions[0].cards))
	}
	if got := state.activeCompositions[0].Points(); got != 44 {
		t.Fatalf("state.activeCompositions[0].Points() = %d; want 44", got)
	}
	wantRun := []Card{
		{rank: Seven, suit: Hearts},
		{rank: Eight, suit: Hearts},
		{rank: Nine, suit: Hearts},
		{rank: Ten, suit: Hearts},
		{rank: Jack, suit: Hearts},
	}
	if !slices.EqualFunc(state.activeCompositions[0].cards, wantRun, sameCard) {
		t.Fatalf("state.activeCompositions[0].cards = %#v; want %#v", state.activeCompositions[0].cards, wantRun)
	}
	if len(state.players[0].hand.cards) != 1 {
		t.Fatalf("len(state.players[0].hand.cards) = %d; want 1", len(state.players[0].hand.cards))
	}
	remaining := state.players[0].hand.cards[0]
	if remaining.rank != Two || remaining.suit != Clubs {
		t.Fatalf("remaining hand card = %+v; want Two of Clubs", remaining)
	}
}

func TestGameStatePlayTableAllowsOpeningWithCompositionAndAddition(t *testing.T) {
	state := newTurnTestState()
	state.turn.hasDrawn = true
	base, ok := NewRun([]Card{
		{rank: Seven, suit: Hearts},
		{rank: Eight, suit: Hearts},
		{rank: Nine, suit: Hearts},
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}
	state.activeCompositions = []*Composition{base}
	state.players[0].hand.cards = []Card{
		{rank: King, suit: Hearts},
		{rank: King, suit: Diamonds},
		{rank: King, suit: Clubs},
		{rank: Ten, suit: Hearts},
		{rank: Two, suit: Spades},
	}
	setComp, ok := NewSet([]Card{
		{rank: King, suit: Hearts},
		{rank: King, suit: Diamonds},
		{rank: King, suit: Clubs},
	})
	if !ok {
		t.Fatal("NewSet() returned false; want true")
	}

	err := state.PlayTable([]*Composition{setComp}, []CompositionAddition{{
		CompositionIndex: 0,
		Cards:            []Card{{rank: Ten, suit: Hearts}},
	}})

	if err != nil {
		t.Fatalf("PlayTable() error = %v", err)
	}
	if !state.players[0].hasOpened {
		t.Fatal("player.hasOpened = false; want true")
	}
	if len(state.activeCompositions) != 2 {
		t.Fatalf("len(state.activeCompositions) = %d; want 2", len(state.activeCompositions))
	}
	if got := state.activeCompositions[0].Points(); got != 34 {
		t.Fatalf("state.activeCompositions[0].Points() = %d; want 34", got)
	}
	if state.activeCompositions[1] != setComp {
		t.Fatal("new composition was not appended")
	}
	if len(state.players[0].hand.cards) != 1 {
		t.Fatalf("len(state.players[0].hand.cards) = %d; want 1", len(state.players[0].hand.cards))
	}
	remaining := state.players[0].hand.cards[0]
	if remaining.rank != Two || remaining.suit != Spades {
		t.Fatalf("remaining hand card = %+v; want Two of Spades", remaining)
	}
}

func TestGameStateAddToCompositionsRejectsUnopenedPlayerWithoutOwnComposition(t *testing.T) {
	state := newTurnTestState()
	state.turn.hasDrawn = true
	base, ok := NewRun([]Card{
		{rank: Seven, suit: Hearts},
		{rank: Eight, suit: Hearts},
		{rank: Nine, suit: Hearts},
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}
	state.activeCompositions = []*Composition{base}
	state.players[0].hand.cards = []Card{{rank: Ten, suit: Hearts}}

	err := state.AddToCompositions([]CompositionAddition{{
		CompositionIndex: 0,
		Cards:            []Card{{rank: Ten, suit: Hearts}},
	}})

	if !errors.Is(err, ErrInitialPlayRequiresOwnComp) {
		t.Fatalf("AddToCompositions() error = %v; want %v", err, ErrInitialPlayRequiresOwnComp)
	}
	if state.players[0].hasOpened {
		t.Fatal("player.hasOpened = true; want false")
	}
	if len(state.activeCompositions[0].cards) != 3 {
		t.Fatalf("len(state.activeCompositions[0].cards) = %d; want 3", len(state.activeCompositions[0].cards))
	}
	if len(state.players[0].hand.cards) != 1 {
		t.Fatalf("len(state.players[0].hand.cards) = %d; want 1", len(state.players[0].hand.cards))
	}
}

func TestGameStatePlayTableRejectsOpeningBelowFortyWithAddition(t *testing.T) {
	state := newTurnTestState()
	state.turn.hasDrawn = true
	base, ok := NewRun([]Card{
		{rank: Seven, suit: Hearts},
		{rank: Eight, suit: Hearts},
		{rank: Nine, suit: Hearts},
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}
	state.activeCompositions = []*Composition{base}
	state.players[0].hand.cards = []Card{
		{rank: Seven, suit: Clubs},
		{rank: Seven, suit: Spades},
		{rank: Seven, suit: Diamonds},
		{rank: Ten, suit: Hearts},
	}
	setComp, ok := NewSet([]Card{
		{rank: Seven, suit: Clubs},
		{rank: Seven, suit: Spades},
		{rank: Seven, suit: Diamonds},
	})
	if !ok {
		t.Fatal("NewSet() returned false; want true")
	}

	err := state.PlayTable([]*Composition{setComp}, []CompositionAddition{{
		CompositionIndex: 0,
		Cards:            []Card{{rank: Ten, suit: Hearts}},
	}})

	if !errors.Is(err, ErrInitialPointsNotMet) {
		t.Fatalf("PlayTable() error = %v; want %v", err, ErrInitialPointsNotMet)
	}
	if len(state.activeCompositions) != 1 {
		t.Fatalf("len(state.activeCompositions) = %d; want 1", len(state.activeCompositions))
	}
	if len(state.activeCompositions[0].cards) != 3 {
		t.Fatalf("len(state.activeCompositions[0].cards) = %d; want 3", len(state.activeCompositions[0].cards))
	}
	if len(state.players[0].hand.cards) != 4 {
		t.Fatalf("len(state.players[0].hand.cards) = %d; want 4", len(state.players[0].hand.cards))
	}
}

func TestGameStateAddToCompositionsDoesNotMutateOnInvalidAddition(t *testing.T) {
	state := newTurnTestState()
	state.turn.hasDrawn = true
	state.players[0].hasOpened = true
	base, ok := NewRun([]Card{
		{rank: Seven, suit: Hearts},
		{rank: Eight, suit: Hearts},
		{rank: Nine, suit: Hearts},
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}
	state.activeCompositions = []*Composition{base}
	state.players[0].hand.cards = []Card{
		{rank: Ten, suit: Hearts},
		{rank: Queen, suit: Hearts},
	}

	err := state.AddToCompositions([]CompositionAddition{{
		CompositionIndex: 0,
		Cards:            []Card{{rank: Ten, suit: Hearts}},
	}, {
		CompositionIndex: 0,
		Cards:            []Card{{rank: Queen, suit: Hearts}},
	}})

	if !errors.Is(err, ErrInvalidComposition) {
		t.Fatalf("AddToCompositions() error = %v; want %v", err, ErrInvalidComposition)
	}
	if len(state.activeCompositions) != 1 {
		t.Fatalf("len(state.activeCompositions) = %d; want 1", len(state.activeCompositions))
	}
	if len(state.activeCompositions[0].cards) != 3 {
		t.Fatalf("len(state.activeCompositions[0].cards) = %d; want 3", len(state.activeCompositions[0].cards))
	}
	if len(state.players[0].hand.cards) != 2 {
		t.Fatalf("len(state.players[0].hand.cards) = %d; want 2", len(state.players[0].hand.cards))
	}
}

func TestGameStatePlayTableWithReclaimsReturnsJokerToHand(t *testing.T) {
	state := newTurnTestState()
	state.turn.hasDrawn = true
	state.players[0].hasOpened = true

	base, ok := NewSet([]Card{
		{rank: Ten, suit: Hearts},
		{rank: Ten, suit: Diamonds},
		{rank: Ten, suit: Clubs},
		{isJoker: true},
	})
	if !ok {
		t.Fatal("NewSet() returned false; want true")
	}
	state.activeCompositions = []*Composition{base}
	state.players[0].hand.cards = []Card{
		{rank: Ten, suit: Spades},
		{rank: Two, suit: Clubs},
	}

	err := state.PlayTable(nil, nil, JokerReclaim{
		CompositionIndex: 0,
		JokerIndex:       3,
		ReplacementCard:  Card{rank: Ten, suit: Spades},
	})

	if err != nil {
		t.Fatalf("PlayTable() error = %v", err)
	}
	if len(state.activeCompositions) != 1 {
		t.Fatalf("len(state.activeCompositions) = %d; want 1", len(state.activeCompositions))
	}
	if state.activeCompositions[0].cards[3].isJoker {
		t.Fatal("state.activeCompositions[0].cards[3] is still a joker")
	}
	if got := state.activeCompositions[0].cards[3]; got.rank != Ten || got.suit != Spades {
		t.Fatalf("reclaimed replacement = %+v; want Ten of Spades", got)
	}
	if len(state.players[0].hand.cards) != 2 {
		t.Fatalf("len(state.players[0].hand.cards) = %d; want 2", len(state.players[0].hand.cards))
	}

	foundTwoClubs := false
	foundJoker := false
	for _, handCard := range state.players[0].hand.cards {
		if handCard.isJoker {
			foundJoker = true
		}
		if handCard.rank == Two && handCard.suit == Clubs {
			foundTwoClubs = true
		}
	}
	if !foundTwoClubs {
		t.Fatal("player hand does not contain Two of Clubs after reclaim")
	}
	if !foundJoker {
		t.Fatal("player hand does not contain reclaimed joker")
	}
}

func TestGameStatePlayTableWithReclaimsAllowsReusingJokerSameTurn(t *testing.T) {
	state := newTurnTestState()
	state.turn.hasDrawn = true
	state.players[0].hasOpened = true

	base, ok := NewRun([]Card{
		{rank: Five, suit: Hearts},
		{isJoker: true},
		{rank: Seven, suit: Hearts},
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}
	state.activeCompositions = []*Composition{base}
	state.players[0].hand.cards = []Card{
		{rank: Six, suit: Hearts},
		{rank: King, suit: Hearts},
		{rank: King, suit: Diamonds},
		{rank: Two, suit: Clubs},
	}

	setComp, ok := NewSet([]Card{
		{rank: King, suit: Hearts},
		{rank: King, suit: Diamonds},
		{isJoker: true},
	})
	if !ok {
		t.Fatal("NewSet() returned false; want true")
	}

	err := state.PlayTable([]*Composition{setComp}, nil, JokerReclaim{
		CompositionIndex: 0,
		JokerIndex:       1,
		ReplacementCard:  Card{rank: Six, suit: Hearts},
	})

	if err != nil {
		t.Fatalf("PlayTable() error = %v", err)
	}
	if len(state.activeCompositions) != 2 {
		t.Fatalf("len(state.activeCompositions) = %d; want 2", len(state.activeCompositions))
	}
	wantRun := []Card{
		{rank: Five, suit: Hearts},
		{rank: Six, suit: Hearts},
		{rank: Seven, suit: Hearts},
	}
	if !slices.EqualFunc(state.activeCompositions[0].cards, wantRun, sameCard) {
		t.Fatalf("state.activeCompositions[0].cards = %#v; want %#v", state.activeCompositions[0].cards, wantRun)
	}
	if len(state.players[0].hand.cards) != 1 {
		t.Fatalf("len(state.players[0].hand.cards) = %d; want 1", len(state.players[0].hand.cards))
	}
	remaining := state.players[0].hand.cards[0]
	if remaining.rank != Two || remaining.suit != Clubs {
		t.Fatalf("remaining hand card = %+v; want Two of Clubs", remaining)
	}
}

func TestGameStatePlayTableAllowsOpeningWithReclaimAndReusedJoker(t *testing.T) {
	state := newTurnTestState()
	state.turn.hasDrawn = true

	base, ok := NewRun([]Card{
		{rank: Five, suit: Hearts},
		{isJoker: true},
		{rank: Seven, suit: Hearts},
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}
	state.activeCompositions = []*Composition{base}
	state.players[0].hand.cards = []Card{
		{rank: Six, suit: Hearts},
		{rank: Eight, suit: Spades},
		{rank: Nine, suit: Spades},
		{rank: Ten, suit: Spades},
		{rank: Two, suit: Clubs},
	}

	runComp, ok := NewRun([]Card{
		{rank: Eight, suit: Spades},
		{rank: Nine, suit: Spades},
		{rank: Ten, suit: Spades},
		{isJoker: true},
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}

	err := state.PlayTable([]*Composition{runComp}, nil, JokerReclaim{
		CompositionIndex: 0,
		JokerIndex:       1,
		ReplacementCard:  Card{rank: Six, suit: Hearts},
	})

	if err != nil {
		t.Fatalf("PlayTable() error = %v", err)
	}
	if !state.players[0].hasOpened {
		t.Fatal("player.hasOpened = false; want true")
	}
	if len(state.activeCompositions) != 2 {
		t.Fatalf("len(state.activeCompositions) = %d; want 2", len(state.activeCompositions))
	}
	if got := state.activeCompositions[1].Points(); got != 37 {
		t.Fatalf("state.activeCompositions[1].Points() = %d; want 37", got)
	}
	if len(state.players[0].hand.cards) != 1 {
		t.Fatalf("len(state.players[0].hand.cards) = %d; want 1", len(state.players[0].hand.cards))
	}
	remaining := state.players[0].hand.cards[0]
	if remaining.rank != Two || remaining.suit != Clubs {
		t.Fatalf("remaining hand card = %+v; want Two of Clubs", remaining)
	}
}

func TestGameStatePlayTableAppliesInsertionBeforeReclaim(t *testing.T) {
	state := newTurnTestState()
	state.turn.hasDrawn = true
	state.players[0].hasOpened = true

	base, ok := NewRun([]Card{
		{rank: Queen, suit: Hearts},
		{rank: King, suit: Hearts},
		{isJoker: true},
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}
	state.activeCompositions = []*Composition{base}
	insertIndex := 0
	state.players[0].hand.cards = []Card{
		{rank: Jack, suit: Hearts},
		{rank: Ace, suit: Hearts},
		{rank: Two, suit: Clubs},
	}

	err := state.PlayTable(nil, []CompositionAddition{{
		CompositionIndex: 0,
		InsertIndex:      &insertIndex,
		Cards:            []Card{{rank: Jack, suit: Hearts}, {isJoker: true}},
	}}, JokerReclaim{
		CompositionIndex: 0,
		JokerIndex:       4,
		ReplacementCard:  Card{rank: Ace, suit: Hearts},
	})

	if err != nil {
		t.Fatalf("PlayTable() error = %v", err)
	}
	want := []Card{
		{isJoker: true},
		{rank: Jack, suit: Hearts},
		{rank: Queen, suit: Hearts},
		{rank: King, suit: Hearts},
		{rank: Ace, suit: Hearts},
	}
	if !slices.EqualFunc(state.activeCompositions[0].cards, want, sameCard) {
		t.Fatalf("state.activeCompositions[0].cards = %#v; want %#v", state.activeCompositions[0].cards, want)
	}
	if len(state.players[0].hand.cards) != 1 {
		t.Fatalf("len(state.players[0].hand.cards) = %d; want 1", len(state.players[0].hand.cards))
	}
	remaining := state.players[0].hand.cards[0]
	if remaining.rank != Two || remaining.suit != Clubs {
		t.Fatalf("remaining hand card = %+v; want Two of Clubs", remaining)
	}
}

func TestGameStatePlayTableWithReclaimsRejectsAmbiguousSetJoker(t *testing.T) {
	state := newTurnTestState()
	state.turn.hasDrawn = true
	state.players[0].hasOpened = true

	base, ok := NewSet([]Card{
		{rank: Ten, suit: Hearts},
		{rank: Ten, suit: Diamonds},
		{isJoker: true},
	})
	if !ok {
		t.Fatal("NewSet() returned false; want true")
	}
	state.activeCompositions = []*Composition{base}
	state.players[0].hand.cards = []Card{{rank: Ten, suit: Clubs}}

	err := state.PlayTable(nil, nil, JokerReclaim{
		CompositionIndex: 0,
		JokerIndex:       2,
		ReplacementCard:  Card{rank: Ten, suit: Clubs},
	})

	if !errors.Is(err, ErrInvalidComposition) {
		t.Fatalf("PlayTable() error = %v; want %v", err, ErrInvalidComposition)
	}
	if len(state.activeCompositions) != 1 {
		t.Fatalf("len(state.activeCompositions) = %d; want 1", len(state.activeCompositions))
	}
	if !state.activeCompositions[0].cards[2].isJoker {
		t.Fatal("active composition mutated after invalid reclaim")
	}
	if len(state.players[0].hand.cards) != 1 {
		t.Fatalf("len(state.players[0].hand.cards) = %d; want 1", len(state.players[0].hand.cards))
	}
	remaining := state.players[0].hand.cards[0]
	if remaining.rank != Ten || remaining.suit != Clubs {
		t.Fatalf("remaining hand card = %+v; want Ten of Clubs", remaining)
	}
}

func TestGameStateDiscardFromHandRequiresDrawFirst(t *testing.T) {
	state := newTurnTestState()
	state.players[0].hand.cards = []Card{{rank: Ace, suit: Hearts}}

	err := state.DiscardFromHand(0)

	if !errors.Is(err, ErrPlayerHasntDrawn) {
		t.Errorf("DiscardFromHand() error = %v; want %v", err, ErrPlayerHasntDrawn)
	}
}

func TestGameStateDiscardFromHandMovesCardAndAdvancesTurn(t *testing.T) {
	state := newTurnTestState()
	state.turn.hasDrawn = true
	state.drawPile = &CardPile{cards: []Card{{rank: Queen, suit: Clubs}}}
	state.players[0].hand.cards = []Card{
		{rank: Ace, suit: Hearts},
		{rank: King, suit: Spades},
	}
	state.players[1].hand.cards = []Card{{rank: Two, suit: Clubs}}
	startingTurnNumber := state.turn.number

	err := state.DiscardFromHand(1)

	if err != nil {
		t.Fatalf("DiscardFromHand() error = %v", err)
	}
	if len(state.players[0].hand.cards) != 1 {
		t.Fatalf("len(state.players[0].hand.cards) = %d; want 1", len(state.players[0].hand.cards))
	}
	if len(state.discardPile.cards) != 1 {
		t.Fatalf("len(state.discardPile.cards) = %d; want 1", len(state.discardPile.cards))
	}
	topDiscard := state.discardPile.cards[0]
	if topDiscard.rank != King || topDiscard.suit != Spades {
		t.Errorf("top discard = %+v; want King of Spades", topDiscard)
	}
	if state.turn.playerIndex != 1 {
		t.Errorf("state.turn.playerIndex = %d; want 1", state.turn.playerIndex)
	}
	if state.turn.number != startingTurnNumber+1 {
		t.Errorf("state.turn.number = %d; want %d", state.turn.number, startingTurnNumber+1)
	}
	if state.turn.hasDrawn {
		t.Error("state.turn.hasDrawn = true; want false")
	}
}

func TestGameStateDiscardFromHandRecyclesDiscardPileAtTurnStartWhenDrawPileIsEmpty(t *testing.T) {
	state := newTurnTestState()
	state.turn.hasDrawn = true
	state.drawPile = &CardPile{cards: []Card{}}
	state.discardPile = &CardPile{cards: []Card{{rank: Four, suit: Diamonds}}}
	state.players[0].hand.cards = []Card{
		{rank: Ace, suit: Hearts},
		{rank: King, suit: Spades},
	}
	state.players[1].hand.cards = []Card{{rank: Two, suit: Clubs}}

	err := state.DiscardFromHand(1)

	if err != nil {
		t.Fatalf("DiscardFromHand() error = %v", err)
	}
	if state.turn.playerIndex != 1 {
		t.Fatalf("state.turn.playerIndex = %d; want 1", state.turn.playerIndex)
	}
	if state.turn.number != 2 {
		t.Fatalf("state.turn.number = %d; want 2", state.turn.number)
	}
	if state.turn.hasDrawn {
		t.Fatal("state.turn.hasDrawn = true; want false")
	}
	if len(state.discardPile.cards) != 1 {
		t.Fatalf("len(state.discardPile.cards) = %d; want 1", len(state.discardPile.cards))
	}
	if top := state.discardPile.cards[0]; top.rank != King || top.suit != Spades {
		t.Fatalf("discardPile.cards[0] = %+v; want King of Spades", top)
	}
	if len(state.drawPile.cards) != 1 {
		t.Fatalf("len(state.drawPile.cards) = %d; want 1", len(state.drawPile.cards))
	}
	if top := state.drawPile.cards[0]; top.rank != Four || top.suit != Diamonds {
		t.Fatalf("drawPile.cards[0] = %+v; want Four of Diamonds", top)
	}
}

func TestGameStateDiscardFromHandEndsRoundWhenFinalDiscardEmptiesHand(t *testing.T) {
	state := newTurnTestState()
	state.turn.hasDrawn = true
	state.players[0].hasOpened = true
	state.players[0].hand.cards = []Card{
		{rank: Ten, suit: Hearts},
		{rank: Ace, suit: Clubs},
	}
	state.players[1].hand.cards = []Card{{rank: Two, suit: Clubs}}
	base, ok := NewRun([]Card{
		{rank: Seven, suit: Hearts},
		{rank: Eight, suit: Hearts},
		{rank: Nine, suit: Hearts},
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}
	state.activeCompositions = []*Composition{base}
	startingTurnNumber := state.turn.number

	err := state.AddToCompositions([]CompositionAddition{{
		CompositionIndex: 0,
		Cards:            []Card{{rank: Ten, suit: Hearts}},
	}})

	if err != nil {
		t.Fatalf("AddToCompositions() error = %v", err)
	}
	if len(state.players[0].hand.cards) != 1 {
		t.Fatalf("len(state.players[0].hand.cards) = %d; want 1", len(state.players[0].hand.cards))
	}

	err = state.DiscardFromHand(0)

	if err != nil {
		t.Fatalf("DiscardFromHand() error = %v", err)
	}
	if state.phase != PhaseRoundOver {
		t.Fatalf("state.phase = %d; want %d", state.phase, PhaseRoundOver)
	}
	if state.roundWinnerIndex != 0 {
		t.Fatalf("state.roundWinnerIndex = %d; want 0", state.roundWinnerIndex)
	}
	if len(state.players[0].hand.cards) != 0 {
		t.Fatalf("len(state.players[0].hand.cards) = %d; want 0", len(state.players[0].hand.cards))
	}
	if topDiscard := state.discardPile.cards[0]; topDiscard.rank != Ace || topDiscard.suit != Clubs {
		t.Fatalf("top discard = %+v; want Ace of Clubs", topDiscard)
	}
	if state.turn.playerIndex != 0 {
		t.Fatalf("state.turn.playerIndex = %d; want 0", state.turn.playerIndex)
	}
	if state.turn.number != startingTurnNumber {
		t.Fatalf("state.turn.number = %d; want %d", state.turn.number, startingTurnNumber)
	}
	if state.turn.hasDrawn {
		t.Fatal("state.turn.hasDrawn = true; want false")
	}
	if len(state.activeCompositions) != 1 {
		t.Fatalf("len(state.activeCompositions) = %d; want 1", len(state.activeCompositions))
	}
	if len(state.activeCompositions[0].cards) != 4 {
		t.Fatalf("len(state.activeCompositions[0].cards) = %d; want 4", len(state.activeCompositions[0].cards))
	}
}

func TestGameStateFinishRoundScoresRemainingHands(t *testing.T) {
	state := newTurnTestState()
	state.turn.hasDrawn = true
	state.players[0].hasOpened = true
	state.players[0].hand.cards = []Card{
		{rank: Ten, suit: Hearts},
		{rank: Ace, suit: Clubs},
	}
	state.players[1].totalPoints = 15
	state.players[1].hand.cards = []Card{{rank: Ace, suit: Spades}}

	base, ok := NewRun([]Card{
		{rank: Seven, suit: Hearts},
		{rank: Eight, suit: Hearts},
		{rank: Nine, suit: Hearts},
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}
	state.activeCompositions = []*Composition{base}

	err := state.AddToCompositions([]CompositionAddition{{
		CompositionIndex: 0,
		Cards:            []Card{{rank: Ten, suit: Hearts}},
	}})
	if err != nil {
		t.Fatalf("AddToCompositions() error = %v", err)
	}

	err = state.DiscardFromHand(0)

	if err != nil {
		t.Fatalf("DiscardFromHand() error = %v", err)
	}
	if state.phase != PhaseRoundOver {
		t.Fatalf("state.phase = %d; want %d", state.phase, PhaseRoundOver)
	}
	if state.players[0].totalPoints != 0 {
		t.Fatalf("winner totalPoints = %d; want 0", state.players[0].totalPoints)
	}
	if state.players[1].totalPoints != 16 {
		t.Fatalf("loser totalPoints = %d; want 16", state.players[1].totalPoints)
	}
}

func TestGameStateFinishRoundAppliesOverHundredAdjustment(t *testing.T) {
	state := newTurnTestState()
	third := NewPlayer()
	state.players = append(state.players, third)
	state.turn.hasDrawn = true
	state.players[0].hasOpened = true
	state.players[0].hand.cards = []Card{
		{rank: Ten, suit: Hearts},
		{rank: Ace, suit: Clubs},
	}
	state.players[1].totalPoints = 95
	state.players[1].hand.cards = []Card{{rank: Seven, suit: Spades}, {rank: Five, suit: Clubs}}
	state.players[2].totalPoints = 80
	state.players[2].hand.cards = []Card{{rank: Nine, suit: Hearts}}

	base, ok := NewRun([]Card{
		{rank: Seven, suit: Hearts},
		{rank: Eight, suit: Hearts},
		{rank: Nine, suit: Hearts},
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}
	state.activeCompositions = []*Composition{base}

	err := state.AddToCompositions([]CompositionAddition{{
		CompositionIndex: 0,
		Cards:            []Card{{rank: Ten, suit: Hearts}},
	}})
	if err != nil {
		t.Fatalf("AddToCompositions() error = %v", err)
	}

	err = state.DiscardFromHand(0)

	if err != nil {
		t.Fatalf("DiscardFromHand() error = %v", err)
	}
	if state.phase != PhaseRoundOver {
		t.Fatalf("state.phase = %d; want %d", state.phase, PhaseRoundOver)
	}
	if state.players[1].totalPoints != 89 {
		t.Fatalf("adjusted player totalPoints = %d; want 89", state.players[1].totalPoints)
	}
	if state.players[2].totalPoints != 89 {
		t.Fatalf("safe player totalPoints = %d; want 89", state.players[2].totalPoints)
	}
}

func TestGameStateFinishRoundEndsGameWhenAllOtherPlayersExceedHundred(t *testing.T) {
	state := newTurnTestState()
	third := NewPlayer()
	state.players = append(state.players, third)
	state.turn.hasDrawn = true
	state.players[0].hasOpened = true
	state.players[0].hand.cards = []Card{
		{rank: Ten, suit: Hearts},
		{rank: Ace, suit: Clubs},
	}
	state.players[1].totalPoints = 95
	state.players[1].hand.cards = []Card{{rank: Six, suit: Clubs}}
	state.players[2].totalPoints = 100
	state.players[2].hand.cards = []Card{{rank: Two, suit: Hearts}}

	base, ok := NewRun([]Card{
		{rank: Seven, suit: Hearts},
		{rank: Eight, suit: Hearts},
		{rank: Nine, suit: Hearts},
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}
	state.activeCompositions = []*Composition{base}

	err := state.AddToCompositions([]CompositionAddition{{
		CompositionIndex: 0,
		Cards:            []Card{{rank: Ten, suit: Hearts}},
	}})
	if err != nil {
		t.Fatalf("AddToCompositions() error = %v", err)
	}

	err = state.DiscardFromHand(0)

	if err != nil {
		t.Fatalf("DiscardFromHand() error = %v", err)
	}
	if state.phase != PhaseGameOver {
		t.Fatalf("state.phase = %d; want %d", state.phase, PhaseGameOver)
	}
	if state.players[1].totalPoints != 101 {
		t.Fatalf("player 1 totalPoints = %d; want 101", state.players[1].totalPoints)
	}
	if state.players[2].totalPoints != 102 {
		t.Fatalf("player 2 totalPoints = %d; want 102", state.players[2].totalPoints)
	}
}

func TestGameStatePlayTableDoesNotEndRoundForSameSuitCollection(t *testing.T) {
	state := newTurnTestState()
	state.turn.hasDrawn = true
	state.players[0].hasOpened = true
	state.players[1].totalPoints = 30
	state.players[1].hand.cards = []Card{{rank: Two, suit: Clubs}}

	base, ok := NewRun([]Card{
		{rank: Seven, suit: Clubs},
		{rank: Eight, suit: Clubs},
		{rank: Nine, suit: Clubs},
	})
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}
	state.activeCompositions = []*Composition{base}
	state.players[0].hand.cards = []Card{
		{rank: Ten, suit: Clubs},
		{rank: Ace, suit: Hearts},
		{rank: Two, suit: Hearts},
		{rank: Three, suit: Hearts},
		{rank: Four, suit: Hearts},
		{rank: Five, suit: Hearts},
		{rank: Six, suit: Hearts},
		{rank: Seven, suit: Hearts},
		{rank: Eight, suit: Hearts},
		{rank: Nine, suit: Hearts},
		{rank: Ten, suit: Hearts},
		{rank: Jack, suit: Hearts},
		{rank: Queen, suit: Hearts},
	}

	err := state.AddToCompositions([]CompositionAddition{{
		CompositionIndex: 0,
		Cards:            []Card{{rank: Ten, suit: Clubs}},
	}})

	if err != nil {
		t.Fatalf("AddToCompositions() error = %v", err)
	}
	if state.phase != PhaseInProgress {
		t.Fatalf("state.phase = %d; want %d", state.phase, PhaseInProgress)
	}
	if state.roundWinnerIndex != -1 {
		t.Fatalf("state.roundWinnerIndex = %d; want -1", state.roundWinnerIndex)
	}
	if len(state.players[0].hand.cards) != 12 {
		t.Fatalf("len(state.players[0].hand.cards) = %d; want 12", len(state.players[0].hand.cards))
	}
	if state.players[1].totalPoints != 30 {
		t.Fatalf("player 1 totalPoints = %d; want 30", state.players[1].totalPoints)
	}
}

func TestGameStateDiscardFromHandEndsRoundForSixIdenticalPairs(t *testing.T) {
	state := newTurnTestState()
	state.turn.hasDrawn = true
	state.players[1].totalPoints = 40
	state.players[1].hand.cards = []Card{{rank: King, suit: Hearts}}
	state.players[0].hand.cards = []Card{
		{rank: Two, suit: Hearts},
		{rank: Two, suit: Hearts},
		{rank: Three, suit: Clubs},
		{rank: Three, suit: Clubs},
		{rank: Four, suit: Diamonds},
		{rank: Four, suit: Diamonds},
		{rank: Five, suit: Spades},
		{rank: Five, suit: Spades},
		{rank: Six, suit: Hearts},
		{rank: Six, suit: Hearts},
		{rank: Seven, suit: Diamonds},
		{rank: Seven, suit: Diamonds},
		{rank: Ace, suit: Clubs},
	}

	err := state.DiscardFromHand(12)

	if err != nil {
		t.Fatalf("DiscardFromHand() error = %v", err)
	}
	if state.phase != PhaseRoundOver {
		t.Fatalf("state.phase = %d; want %d", state.phase, PhaseRoundOver)
	}
	if state.roundWinnerIndex != 0 {
		t.Fatalf("state.roundWinnerIndex = %d; want 0", state.roundWinnerIndex)
	}
	if len(state.players[0].hand.cards) != 12 {
		t.Fatalf("len(state.players[0].hand.cards) = %d; want 12", len(state.players[0].hand.cards))
	}
	if state.players[1].totalPoints != 50 {
		t.Fatalf("player 1 totalPoints = %d; want 50", state.players[1].totalPoints)
	}
}

func TestGameStateDiscardFromHandRemovesCompletedCompositionsBeforeDiscard(t *testing.T) {
	state := newTurnTestState()
	state.turn.hasDrawn = true
	state.discardPile = &CardPile{cards: []Card{{rank: Three, suit: Spades}}}
	state.players[0].hand.cards = []Card{{rank: King, suit: Spades}}

	completeSet, ok := NewSet([]Card{
		card(Nine, Hearts),
		card(Nine, Diamonds),
		card(Nine, Clubs),
		card(Nine, Spades),
	})
	if !ok {
		t.Fatal("NewSet() returned false; want true")
	}
	state.activeCompositions = []*Composition{completeSet}

	err := state.DiscardFromHand(0)

	if err != nil {
		t.Fatalf("DiscardFromHand() error = %v", err)
	}
	if len(state.activeCompositions) != 0 {
		t.Fatalf("len(state.activeCompositions) = %d; want 0", len(state.activeCompositions))
	}
	if len(state.discardPile.cards) != 6 {
		t.Fatalf("len(state.discardPile.cards) = %d; want 6", len(state.discardPile.cards))
	}
	if top := state.discardPile.cards[0]; top.rank != King || top.suit != Spades {
		t.Fatalf("top discard = %+v; want King of Spades", top)
	}
	for i, want := range []Card{
		card(Nine, Hearts),
		card(Nine, Diamonds),
		card(Nine, Clubs),
		card(Nine, Spades),
	} {
		if got := state.discardPile.cards[i+1]; !cardsEqual(got, want) {
			t.Fatalf("discardPile.cards[%d] = %+v; want %+v", i+1, got, want)
		}
	}
	if bottom := state.discardPile.cards[5]; bottom.rank != Three || bottom.suit != Spades {
		t.Fatalf("bottom discard = %+v; want Three of Spades", bottom)
	}
	if state.phase != PhaseRoundOver {
		t.Fatalf("state.phase = %d; want %d", state.phase, PhaseRoundOver)
	}
	if state.roundWinnerIndex != 0 {
		t.Fatalf("state.roundWinnerIndex = %d; want 0", state.roundWinnerIndex)
	}
	if state.turn.number != 1 {
		t.Fatalf("state.turn.number = %d; want 1", state.turn.number)
	}
}

func TestGameStateDiscardFromHandRemovesCompletedRunBeforeDiscard(t *testing.T) {
	state := newTurnTestState()
	state.turn.hasDrawn = true
	state.discardPile = &CardPile{cards: []Card{{rank: Three, suit: Spades}}}
	state.players[0].hand.cards = []Card{{rank: King, suit: Spades}}

	runCards := []Card{
		card(Ace, Hearts),
		card(Two, Hearts),
		card(Three, Hearts),
		card(Four, Hearts),
		card(Five, Hearts),
		card(Six, Hearts),
		card(Seven, Hearts),
		card(Eight, Hearts),
		card(Nine, Hearts),
		card(Ten, Hearts),
		card(Jack, Hearts),
		card(Queen, Hearts),
		card(King, Hearts),
		card(Ace, Hearts),
	}
	completeRun, ok := NewRun(runCards)
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}
	state.activeCompositions = []*Composition{completeRun}

	err := state.DiscardFromHand(0)

	if err != nil {
		t.Fatalf("DiscardFromHand() error = %v", err)
	}
	if len(state.activeCompositions) != 0 {
		t.Fatalf("len(state.activeCompositions) = %d; want 0", len(state.activeCompositions))
	}
	if len(state.discardPile.cards) != 16 {
		t.Fatalf("len(state.discardPile.cards) = %d; want 16", len(state.discardPile.cards))
	}
	if top := state.discardPile.cards[0]; top.rank != King || top.suit != Spades {
		t.Fatalf("top discard = %+v; want King of Spades", top)
	}
	for i, want := range runCards {
		if got := state.discardPile.cards[i+1]; !cardsEqual(got, want) {
			t.Fatalf("discardPile.cards[%d] = %+v; want %+v", i+1, got, want)
		}
	}
	if bottom := state.discardPile.cards[15]; bottom.rank != Three || bottom.suit != Spades {
		t.Fatalf("bottom discard = %+v; want Three of Spades", bottom)
	}
}

func TestGameStateDiscardFromHandRemovesMultipleCompletedCompositionsInOneTurn(t *testing.T) {
	state := newTurnTestState()
	state.turn.hasDrawn = true
	state.discardPile = &CardPile{cards: []Card{{rank: Four, suit: Diamonds}}}
	state.players[0].hand.cards = []Card{{rank: Jack, suit: Clubs}}

	completeSet, ok := NewSet([]Card{
		card(Nine, Hearts),
		card(Nine, Diamonds),
		card(Nine, Clubs),
		card(Nine, Spades),
	})
	if !ok {
		t.Fatal("NewSet() returned false; want true")
	}
	incompleteSet, ok := NewSet([]Card{
		card(Queen, Hearts),
		card(Queen, Diamonds),
		card(Queen, Clubs),
	})
	if !ok {
		t.Fatal("NewSet() returned false; want true")
	}
	runCards := []Card{
		card(Ace, Hearts),
		card(Two, Hearts),
		card(Three, Hearts),
		card(Four, Hearts),
		card(Five, Hearts),
		card(Six, Hearts),
		card(Seven, Hearts),
		card(Eight, Hearts),
		card(Nine, Hearts),
		card(Ten, Hearts),
		card(Jack, Hearts),
		card(Queen, Hearts),
		card(King, Hearts),
		card(Ace, Hearts),
	}
	completeRun, ok := NewRun(runCards)
	if !ok {
		t.Fatal("NewRun() returned false; want true")
	}
	state.activeCompositions = []*Composition{completeSet, incompleteSet, completeRun}

	err := state.DiscardFromHand(0)

	if err != nil {
		t.Fatalf("DiscardFromHand() error = %v", err)
	}
	if len(state.activeCompositions) != 1 {
		t.Fatalf("len(state.activeCompositions) = %d; want 1", len(state.activeCompositions))
	}
	if state.activeCompositions[0] != incompleteSet {
		t.Fatal("remaining active composition changed; want incomplete set to stay on table")
	}
	if len(state.discardPile.cards) != 20 {
		t.Fatalf("len(state.discardPile.cards) = %d; want 20", len(state.discardPile.cards))
	}
	if top := state.discardPile.cards[0]; top.rank != Jack || top.suit != Clubs {
		t.Fatalf("top discard = %+v; want Jack of Clubs", top)
	}
	for i, want := range runCards {
		if got := state.discardPile.cards[i+1]; !cardsEqual(got, want) {
			t.Fatalf("discardPile.cards[%d] = %+v; want %+v", i+1, got, want)
		}
	}
	for i, want := range []Card{
		card(Nine, Hearts),
		card(Nine, Diamonds),
		card(Nine, Clubs),
		card(Nine, Spades),
	} {
		if got := state.discardPile.cards[i+15]; !cardsEqual(got, want) {
			t.Fatalf("discardPile.cards[%d] = %+v; want %+v", i+15, got, want)
		}
	}
	if bottom := state.discardPile.cards[19]; bottom.rank != Four || bottom.suit != Diamonds {
		t.Fatalf("bottom discard = %+v; want Four of Diamonds", bottom)
	}
}

func TestGameStateDiscardFromHandRejectsInvalidIndex(t *testing.T) {
	state := newTurnTestState()
	state.turn.hasDrawn = true
	state.players[0].hand.cards = []Card{{rank: Ace, suit: Hearts}}

	err := state.DiscardFromHand(1)

	if !errors.Is(err, ErrRemovingCard) {
		t.Errorf("DiscardFromHand() error = %v; want %v", err, ErrRemovingCard)
	}
	if len(state.discardPile.cards) != 0 {
		t.Errorf("len(state.discardPile.cards) = %d; want 0", len(state.discardPile.cards))
	}
	if state.turn.playerIndex != 0 {
		t.Errorf("state.turn.playerIndex = %d; want 0", state.turn.playerIndex)
	}
}

func TestGameStateAddPlayerRejectsDuplicatePlayer(t *testing.T) {
	state := NewGameState()
	player := NewPlayer()

	if err := state.AddPlayer(player); err != nil {
		t.Fatalf("AddPlayer() error = %v", err)
	}

	err := state.AddPlayer(player)

	if !errors.Is(err, ErrPlayerExists) {
		t.Fatalf("AddPlayer() error = %v; want %v", err, ErrPlayerExists)
	}
	if len(state.players) != 1 {
		t.Fatalf("len(state.players) = %d; want 1", len(state.players))
	}
}

func TestGameStateAddPlayerRejectsWhenGameIsFull(t *testing.T) {
	state := NewGameState()
	for range state.maxPlayers {
		if err := state.AddPlayer(NewPlayer()); err != nil {
			t.Fatalf("AddPlayer() error = %v", err)
		}
	}

	err := state.AddPlayer(NewPlayer())

	if !errors.Is(err, ErrGameFull) {
		t.Fatalf("AddPlayer() error = %v; want %v", err, ErrGameFull)
	}
	if len(state.players) != state.maxPlayers {
		t.Fatalf("len(state.players) = %d; want %d", len(state.players), state.maxPlayers)
	}
}

func TestGameStateAddPlayerRejectsAfterGameStarts(t *testing.T) {
	state := NewGameState()
	players := []*Player{NewPlayer(), NewPlayer()}
	for _, player := range players {
		if err := state.AddPlayer(player); err != nil {
			t.Fatalf("AddPlayer() error = %v", err)
		}
	}
	if err := state.StartGame(twoPlayerDealerIndex, twoPlayerChooserIndex, DealRoundRobin, nil, 0); err != nil {
		t.Fatalf("StartGame() error = %v", err)
	}

	err := state.AddPlayer(NewPlayer())

	if !errors.Is(err, ErrGameInProgress) {
		t.Fatalf("AddPlayer() error = %v; want %v", err, ErrGameInProgress)
	}
}

func TestGameStateStartGameRejectsAlreadyStartedGame(t *testing.T) {
	state := NewGameState()
	players := []*Player{NewPlayer(), NewPlayer()}
	for _, player := range players {
		if err := state.AddPlayer(player); err != nil {
			t.Fatalf("AddPlayer() error = %v", err)
		}
	}
	if err := state.StartGame(twoPlayerDealerIndex, twoPlayerChooserIndex, DealRoundRobin, nil, 0); err != nil {
		t.Fatalf("StartGame() error = %v", err)
	}

	err := state.StartGame(twoPlayerDealerIndex, twoPlayerChooserIndex, DealRoundRobin, nil, 0)

	if !errors.Is(err, ErrGameInProgress) {
		t.Fatalf("StartGame() error = %v; want %v", err, ErrGameInProgress)
	}
}

func TestGameStateStartGameRequiresAtLeastTwoPlayers(t *testing.T) {
	state := NewGameState()
	if err := state.AddPlayer(NewPlayer()); err != nil {
		t.Fatalf("AddPlayer() error = %v", err)
	}

	err := state.StartGame(0, 0, DealRoundRobin, nil, 0)

	if !errors.Is(err, ErrNotEnoughPlayers) {
		t.Fatalf("StartGame() error = %v; want %v", err, ErrNotEnoughPlayers)
	}
}

func TestGameStateStartGameRejectsInvalidDealingType(t *testing.T) {
	state := NewGameState()
	players := []*Player{NewPlayer(), NewPlayer()}
	for _, player := range players {
		if err := state.AddPlayer(player); err != nil {
			t.Fatalf("AddPlayer() error = %v", err)
		}
	}

	err := state.StartGame(twoPlayerDealerIndex, twoPlayerChooserIndex, DealTypes(99), nil, 0)

	if !errors.Is(err, ErrInvalidDealingType) {
		t.Fatalf("StartGame() error = %v; want %v", err, ErrInvalidDealingType)
	}
}

func TestGameStateStartGameRejectsInvalidBlockOrder(t *testing.T) {
	state := NewGameState()
	for range 3 {
		if err := state.AddPlayer(NewPlayer()); err != nil {
			t.Fatalf("AddPlayer() error = %v", err)
		}
	}

	err := state.StartGame(1, 0, DealInBlocks, []int{2, 2, 1}, 0)

	if !errors.Is(err, ErrInvalidDealingOrder) {
		t.Fatalf("StartGame() error = %v; want %v", err, ErrInvalidDealingOrder)
	}
}

func TestGameStateDrawFromDiscardRequiresGameInProgress(t *testing.T) {
	state := NewGameState()

	err := state.DrawFromDiscard()

	if !errors.Is(err, ErrGameNotInProgress) {
		t.Fatalf("DrawFromDiscard() error = %v; want %v", err, ErrGameNotInProgress)
	}
}

func TestGameStateDrawFromDiscardRejectsSecondDrawSameTurn(t *testing.T) {
	state := newTurnTestState()
	state.players[0].hasOpened = true
	state.players[0].hand.cards = []Card{
		card(Seven, Hearts),
		card(Eight, Hearts),
		card(Nine, Hearts),
		card(Two, Clubs),
		card(Ace, Clubs),
	}
	state.discardPile = &CardPile{cards: []Card{card(Ten, Hearts)}}

	if err := state.DrawFromDiscard(); err != nil {
		t.Fatalf("first DrawFromDiscard() error = %v", err)
	}

	err := state.DrawFromDiscard()

	if !errors.Is(err, ErrPlayerAlreadyDrew) {
		t.Fatalf("second DrawFromDiscard() error = %v; want %v", err, ErrPlayerAlreadyDrew)
	}
}

func TestGameStateDrawFromDiscardDoesNotEndRoundForJokerSixIdenticalPairs(t *testing.T) {
	state := newTurnTestState()
	state.players[0].hasOpened = true
	state.players[0].hand.cards = []Card{
		card(Two, Hearts), card(Two, Hearts),
		card(Three, Clubs), card(Three, Clubs),
		card(Four, Diamonds), card(Four, Diamonds),
		card(Five, Spades), card(Five, Spades),
		card(Six, Hearts),
		card(Seven, Diamonds),
		joker(),
	}
	state.discardPile = &CardPile{cards: []Card{joker()}}
	state.players[1].totalPoints = 40
	state.players[1].hand.cards = []Card{card(King, Hearts)}

	err := state.DrawFromDiscard()

	if err != nil {
		t.Fatalf("DrawFromDiscard() error = %v", err)
	}
	if state.phase != PhaseInProgress {
		t.Fatalf("state.phase = %d; want %d", state.phase, PhaseInProgress)
	}
	if state.roundWinnerIndex != -1 {
		t.Fatalf("state.roundWinnerIndex = %d; want -1", state.roundWinnerIndex)
	}
	if state.players[1].totalPoints != 40 {
		t.Fatalf("player 1 totalPoints = %d; want 40", state.players[1].totalPoints)
	}
	if !state.turn.mustUseDiscardDraw {
		t.Fatal("state.turn.mustUseDiscardDraw = false; want true")
	}
}

func TestGameStateDiscardFromHandEndsRoundForJokerSameSuitCollection(t *testing.T) {
	state := newTurnTestState()
	state.turn.hasDrawn = true
	state.players[1].totalPoints = 10
	state.players[1].hand.cards = []Card{card(King, Clubs)}
	state.players[0].hand.cards = []Card{
		card(Ace, Hearts),
		card(Two, Hearts),
		card(Three, Hearts),
		card(Four, Hearts),
		card(Five, Hearts),
		card(Six, Hearts),
		card(Seven, Hearts),
		card(Eight, Hearts),
		card(Nine, Hearts),
		joker(),
		joker(),
		joker(),
		card(King, Clubs),
	}

	err := state.DiscardFromHand(12)

	if err != nil {
		t.Fatalf("DiscardFromHand() error = %v", err)
	}
	if state.phase != PhaseRoundOver {
		t.Fatalf("state.phase = %d; want %d", state.phase, PhaseRoundOver)
	}
	if state.roundWinnerIndex != 0 {
		t.Fatalf("state.roundWinnerIndex = %d; want 0", state.roundWinnerIndex)
	}
	if state.players[1].totalPoints != 20 {
		t.Fatalf("player 1 totalPoints = %d; want 20", state.players[1].totalPoints)
	}
}

func TestGameStateDiscardFromHandEndsRoundForJokerSixIdenticalPairs(t *testing.T) {
	state := newTurnTestState()
	state.turn.hasDrawn = true
	state.players[1].totalPoints = 40
	state.players[1].hand.cards = []Card{card(King, Hearts)}
	state.players[0].hand.cards = []Card{
		card(Two, Hearts), card(Two, Hearts),
		card(Three, Clubs), card(Three, Clubs),
		card(Four, Diamonds), card(Four, Diamonds),
		card(Five, Spades), card(Five, Spades),
		card(Six, Hearts),
		card(Seven, Diamonds),
		joker(),
		joker(),
		card(Ace, Clubs),
	}

	err := state.DiscardFromHand(12)

	if err != nil {
		t.Fatalf("DiscardFromHand() error = %v", err)
	}
	if state.phase != PhaseRoundOver {
		t.Fatalf("state.phase = %d; want %d", state.phase, PhaseRoundOver)
	}
	if state.roundWinnerIndex != 0 {
		t.Fatalf("state.roundWinnerIndex = %d; want 0", state.roundWinnerIndex)
	}
	if state.players[1].totalPoints != 50 {
		t.Fatalf("player 1 totalPoints = %d; want 50", state.players[1].totalPoints)
	}
}

func TestGameStateDrawFromDiscardRejectsEmptyDiscardPile(t *testing.T) {
	state := newTurnTestState()
	state.players[0].hasOpened = true
	state.players[0].hand.cards = []Card{
		card(Seven, Hearts),
		card(Eight, Hearts),
		card(Nine, Hearts),
	}

	err := state.DrawFromDiscard()

	if !errors.Is(err, ErrCannotTakeDiscardCard) {
		t.Fatalf("DrawFromDiscard() error = %v; want %v", err, ErrCannotTakeDiscardCard)
	}
}

func TestGameStatePlayTableRejectsEmptyActionSet(t *testing.T) {
	state := newTurnTestState()
	state.turn.hasDrawn = true

	err := state.PlayTable(nil, nil)

	if !errors.Is(err, ErrInvalidComposition) {
		t.Fatalf("PlayTable() error = %v; want %v", err, ErrInvalidComposition)
	}
}

func TestGameStateDiscardFromHandRejectsEndingTurnBeforeUsingTakenDiscardCard(t *testing.T) {
	state := newTurnTestState()
	state.players[0].hasOpened = true
	state.players[0].hand.cards = []Card{
		card(King, Hearts),
		card(King, Diamonds),
		card(King, Clubs),
		card(Two, Clubs),
	}
	state.discardPile = &CardPile{cards: []Card{card(Ten, Hearts)}}
	state.activeCompositions = []*Composition{mustRun(t,
		card(Seven, Hearts),
		card(Eight, Hearts),
		card(Nine, Hearts),
	)}

	if err := state.DrawFromDiscard(); err != nil {
		t.Fatalf("DrawFromDiscard() error = %v", err)
	}
	if err := state.PlayCompositions([]*Composition{mustSet(t,
		card(King, Hearts),
		card(King, Diamonds),
		card(King, Clubs),
	)}); err != nil {
		t.Fatalf("PlayCompositions() error = %v", err)
	}

	err := state.DiscardFromHand(indexOfCard(state.players[0].hand.cards, card(Ace, Clubs)))

	if !errors.Is(err, ErrMustUseDrawnDiscardCard) {
		t.Fatalf("DiscardFromHand() error = %v; want %v", err, ErrMustUseDrawnDiscardCard)
	}
	if len(state.players[0].hand.cards) != 2 {
		t.Fatalf("len(state.players[0].hand.cards) = %d; want 2", len(state.players[0].hand.cards))
	}
	if len(state.discardPile.cards) != 0 {
		t.Fatalf("len(state.discardPile.cards) = %d; want 0", len(state.discardPile.cards))
	}
}

func TestGameStateDiscardFromHandAllowsTurnToEndAfterUsingTakenDiscardCardInNewComposition(t *testing.T) {
	state := newTurnTestState()
	state.players[0].hasOpened = true
	state.players[0].hand.cards = []Card{
		card(Seven, Hearts),
		card(Eight, Hearts),
		card(Nine, Hearts),
		card(Two, Clubs),
		card(Ace, Clubs),
	}
	state.discardPile = &CardPile{cards: []Card{card(Ten, Hearts)}}
	state.drawPile = &CardPile{cards: []Card{card(Four, Spades)}}
	state.players[1].hand.cards = []Card{card(Ace, Spades)}

	if err := state.DrawFromDiscard(); err != nil {
		t.Fatalf("DrawFromDiscard() error = %v", err)
	}
	if err := state.PlayCompositions([]*Composition{mustRun(t,
		card(Seven, Hearts),
		card(Eight, Hearts),
		card(Nine, Hearts),
		card(Ten, Hearts),
	)}); err != nil {
		t.Fatalf("PlayCompositions() error = %v", err)
	}

	err := state.DiscardFromHand(indexOfCard(state.players[0].hand.cards, card(Ace, Clubs)))

	if err != nil {
		t.Fatalf("DiscardFromHand() error = %v", err)
	}
	if state.turn.playerIndex != 1 {
		t.Fatalf("state.turn.playerIndex = %d; want 1", state.turn.playerIndex)
	}
	if len(state.discardPile.cards) != 1 || !cardsEqual(state.discardPile.cards[0], card(Ace, Clubs)) {
		t.Fatalf("discard pile top = %+v; want %+v", state.discardPile.cards, card(Ace, Clubs))
	}
}

func TestGameStateDiscardFromHandAllowsTurnToEndAfterUsingTakenDiscardCardInAddition(t *testing.T) {
	state := newTurnTestState()
	state.players[0].hasOpened = true
	state.players[0].hand.cards = []Card{
		card(Jack, Hearts),
		card(Two, Clubs),
		card(Ace, Clubs),
	}
	state.discardPile = &CardPile{cards: []Card{card(Ten, Hearts)}}
	state.drawPile = &CardPile{cards: []Card{card(Four, Spades)}}
	state.players[1].hand.cards = []Card{card(Ace, Spades)}
	state.activeCompositions = []*Composition{mustRun(t,
		card(Seven, Hearts),
		card(Eight, Hearts),
		card(Nine, Hearts),
	)}

	if err := state.DrawFromDiscard(); err != nil {
		t.Fatalf("DrawFromDiscard() error = %v", err)
	}
	if err := state.AddToCompositions([]CompositionAddition{{
		CompositionIndex: 0,
		Cards:            []Card{card(Ten, Hearts), card(Jack, Hearts)},
	}}); err != nil {
		t.Fatalf("AddToCompositions() error = %v", err)
	}

	err := state.DiscardFromHand(indexOfCard(state.players[0].hand.cards, card(Ace, Clubs)))

	if err != nil {
		t.Fatalf("DiscardFromHand() error = %v", err)
	}
	if state.turn.playerIndex != 1 {
		t.Fatalf("state.turn.playerIndex = %d; want 1", state.turn.playerIndex)
	}
}

func TestGameStateDiscardFromHandAllowsTurnToEndAfterUsingTakenDiscardCardInReclaim(t *testing.T) {
	state := newTurnTestState()
	state.players[0].hasOpened = true
	state.players[0].hand.cards = []Card{card(Two, Clubs)}
	state.discardPile = &CardPile{cards: []Card{card(Six, Hearts)}}
	state.players[1].hand.cards = []Card{card(Ace, Spades)}
	state.activeCompositions = []*Composition{mustRun(t,
		card(Five, Hearts),
		joker(),
		card(Seven, Hearts),
	)}

	if err := state.DrawFromDiscard(); err != nil {
		t.Fatalf("DrawFromDiscard() error = %v", err)
	}
	if err := state.PlayTable(nil, nil, JokerReclaim{
		CompositionIndex: 0,
		JokerIndex:       1,
		ReplacementCard:  card(Six, Hearts),
	}); err != nil {
		t.Fatalf("PlayTable() error = %v", err)
	}

	err := state.DiscardFromHand(indexOfCard(state.players[0].hand.cards, card(Two, Clubs)))

	if err != nil {
		t.Fatalf("DiscardFromHand() error = %v", err)
	}
	if state.turn.playerIndex != 1 {
		t.Fatalf("state.turn.playerIndex = %d; want 1", state.turn.playerIndex)
	}
}

func TestGameStateDrawFromDeckPropagatesCurrentPlayerError(t *testing.T) {
	state := NewGameState()
	state.phase = PhaseInProgress

	err := state.DrawFromDeck()

	if !errors.Is(err, ErrNoPlayers) {
		t.Fatalf("DrawFromDeck() error = %v; want %v", err, ErrNoPlayers)
	}
}

func TestGameStateDrawFromDiscardPropagatesCurrentPlayerError(t *testing.T) {
	state := NewGameState()
	state.phase = PhaseInProgress

	err := state.DrawFromDiscard()

	if !errors.Is(err, ErrNoPlayers) {
		t.Fatalf("DrawFromDiscard() error = %v; want %v", err, ErrNoPlayers)
	}
}

func TestGameStatePlayTableRequiresGameInProgress(t *testing.T) {
	state := NewGameState()

	err := state.PlayTable([]*Composition{mustSet(t, card(King, Hearts), card(King, Diamonds), card(King, Clubs))}, nil)

	if !errors.Is(err, ErrGameNotInProgress) {
		t.Fatalf("PlayTable() error = %v; want %v", err, ErrGameNotInProgress)
	}
}

func TestGameStatePlayTablePropagatesCurrentPlayerError(t *testing.T) {
	state := NewGameState()
	state.phase = PhaseInProgress
	state.turn.hasDrawn = true

	err := state.PlayTable([]*Composition{mustSet(t, card(King, Hearts), card(King, Diamonds), card(King, Clubs))}, nil)

	if !errors.Is(err, ErrNoPlayers) {
		t.Fatalf("PlayTable() error = %v; want %v", err, ErrNoPlayers)
	}
}

func TestGameStateDiscardFromHandRequiresGameInProgress(t *testing.T) {
	state := NewGameState()

	err := state.DiscardFromHand(0)

	if !errors.Is(err, ErrGameNotInProgress) {
		t.Fatalf("DiscardFromHand() error = %v; want %v", err, ErrGameNotInProgress)
	}
}

func TestGameStateDiscardFromHandPropagatesCurrentPlayerError(t *testing.T) {
	state := NewGameState()
	state.phase = PhaseInProgress
	state.turn.hasDrawn = true

	err := state.DiscardFromHand(0)

	if !errors.Is(err, ErrNoPlayers) {
		t.Fatalf("DiscardFromHand() error = %v; want %v", err, ErrNoPlayers)
	}
}

func TestFinishRoundIfSpecialWinRejectsInvalidPlayerAndNonWinningHands(t *testing.T) {
	state := newTurnTestState()
	if state.finishRoundIfSpecialWin(-1) {
		t.Fatal("finishRoundIfSpecialWin(-1) = true; want false")
	}

	state.players[0] = nil
	if state.finishRoundIfSpecialWin(0) {
		t.Fatal("finishRoundIfSpecialWin(nil player) = true; want false")
	}

	state.players[0] = NewPlayer()
	state.players[0].hand.cards = []Card{card(Ace, Hearts)}
	if state.finishRoundIfSpecialWin(0) {
		t.Fatal("finishRoundIfSpecialWin(non-special hand) = true; want false")
	}
}

func TestApplyOverHundredAdjustmentWithoutSafePlayersDoesNothing(t *testing.T) {
	state := newTurnTestState()
	state.players[0].totalPoints = 101
	state.players[1].totalPoints = 150

	state.applyOverHundredAdjustment()

	if state.players[0].totalPoints != 101 || state.players[1].totalPoints != 150 {
		t.Fatalf("totals changed unexpectedly: %d %d", state.players[0].totalPoints, state.players[1].totalPoints)
	}
}

func TestHasSameSuitCollectionRejectsWrongLengthAndMixedSuits(t *testing.T) {
	if hasSameSuitCollection([]Card{card(Ace, Hearts)}) {
		t.Fatal("hasSameSuitCollection() = true; want false for wrong length")
	}
	hand := sameSuitCollectionHand(Hearts)
	hand[0] = card(Ace, Clubs)
	if hasSameSuitCollection(hand) {
		t.Fatal("hasSameSuitCollection() = true; want false for mixed suits")
	}
	withJokers := []Card{
		card(Ace, Hearts),
		card(Two, Hearts),
		card(Three, Hearts),
		card(Four, Hearts),
		card(Five, Hearts),
		card(Six, Hearts),
		card(Seven, Hearts),
		card(Eight, Hearts),
		card(Nine, Hearts),
		joker(),
		joker(),
		joker(),
	}
	if !hasSameSuitCollection(withJokers) {
		t.Fatal("hasSameSuitCollection() = false; want true with jokers completing the suit")
	}
}

func TestHasSixIdenticalPairsRejectsWrongLengthAndBadCounts(t *testing.T) {
	if hasSixIdenticalPairs([]Card{card(Ace, Hearts)}) {
		t.Fatal("hasSixIdenticalPairs() = true; want false for wrong length")
	}
	bad := []Card{
		card(Two, Hearts), card(Two, Hearts),
		card(Three, Hearts), card(Three, Hearts),
		card(Four, Hearts), card(Four, Hearts),
		card(Five, Hearts), card(Five, Hearts),
		card(Six, Hearts), card(Six, Hearts),
		card(Seven, Hearts), card(Eight, Hearts),
	}
	if hasSixIdenticalPairs(bad) {
		t.Fatal("hasSixIdenticalPairs() = true; want false for unmatched final pair")
	}
	withJokers := []Card{
		card(Two, Hearts), card(Two, Hearts),
		card(Three, Hearts), card(Three, Hearts),
		card(Four, Hearts), card(Four, Hearts),
		card(Five, Hearts), card(Five, Hearts),
		card(Six, Hearts),
		card(Seven, Hearts),
		joker(),
		joker(),
	}
	if !hasSixIdenticalPairs(withJokers) {
		t.Fatal("hasSixIdenticalPairs() = false; want true with jokers completing unmatched cards")
	}
}

func TestRemoveCompletedCompositionsToDiscardKeepsNilComposition(t *testing.T) {
	state := newTurnTestState()
	state.activeCompositions = []*Composition{nil}

	state.removeCompletedCompositionsToDiscard()

	if len(state.activeCompositions) != 1 || state.activeCompositions[0] != nil {
		t.Fatalf("active compositions = %+v; want single nil entry", state.activeCompositions)
	}
}

func TestStartNextRoundPropagatesStartRoundError(t *testing.T) {
	state := NewGameState()
	for range 2 {
		if err := state.AddPlayer(NewPlayer()); err != nil {
			t.Fatalf("AddPlayer() error = %v", err)
		}
	}
	state.phase = PhaseRoundOver

	err := state.StartNextRound(DealInBlocks, nil, 0)

	if !errors.Is(err, ErrInvalidDealingOrder) {
		t.Fatalf("StartNextRound() error = %v; want %v", err, ErrInvalidDealingOrder)
	}
}

func TestStartRoundRejectsBlockOrderWithoutOrder(t *testing.T) {
	state := NewGameState()
	for range 2 {
		if err := state.AddPlayer(NewPlayer()); err != nil {
			t.Fatalf("AddPlayer() error = %v", err)
		}
	}

	err := state.startRound(twoPlayerDealerIndex, twoPlayerChooserIndex, DealInBlocks, nil, 0)

	if !errors.Is(err, ErrInvalidDealingOrder) {
		t.Fatalf("startRound() error = %v; want %v", err, ErrInvalidDealingOrder)
	}
}

func TestResetRoundStateUsesFreshDeckAndSkipsNilPlayers(t *testing.T) {
	state := NewGameState()
	player := NewPlayer()
	player.hasOpened = true
	player.hand.cards = []Card{card(Ace, Hearts)}
	state.players = []*Player{player, nil}

	state.resetRoundState(nil)

	if len(state.drawPile.cards) != 108 {
		t.Fatalf("len(state.drawPile.cards) = %d; want 108", len(state.drawPile.cards))
	}
	if player.hasOpened {
		t.Fatal("player.hasOpened = true; want false")
	}
	if len(player.hand.cards) != 0 {
		t.Fatalf("len(player.hand.cards) = %d; want 0", len(player.hand.cards))
	}
}

func TestCurrentPlayerCanReturnNilPlayer(t *testing.T) {
	state := NewGameState()
	state.players = []*Player{nil}

	player, err := state.CurrentPlayer()

	if err != nil {
		t.Fatalf("CurrentPlayer() error = %v", err)
	}
	if player != nil {
		t.Fatalf("CurrentPlayer() = %v; want nil", player)
	}
}

func TestDealRoundRobinRejectsShortPileAndInvalidDealer(t *testing.T) {
	players := []*Player{NewPlayer(), NewPlayer()}

	if err := dealRoundRobin(players, &CardPile{cards: []Card{}}, 0, 0); !errors.Is(err, ErrNotEnoughCardsInDrawPile) {
		t.Fatalf("dealRoundRobin() error = %v; want %v", err, ErrNotEnoughCardsInDrawPile)
	}
	if err := dealRoundRobin(players, orderedGameDeck(blockDealSetup([]int{0, 1}, [][]Card{sameSuitCollectionHand(Hearts), sameSuitCollectionHand(Clubs)}, card(Ace, Spades))...), 2, 0); !errors.Is(err, ErrInvalidDealer) {
		t.Fatalf("dealRoundRobin() error = %v; want %v", err, ErrInvalidDealer)
	}
}

func TestDealInBlocksRejectsShortPile(t *testing.T) {
	players := []*Player{NewPlayer(), NewPlayer()}

	err := dealInBlocks(players, &CardPile{cards: []Card{}}, []int{0, 1})

	if !errors.Is(err, ErrNotEnoughCardsInDrawPile) {
		t.Fatalf("dealInBlocks() error = %v; want %v", err, ErrNotEnoughCardsInDrawPile)
	}
}

func TestValidateOrderRejectsShortAndOutOfRangeOrders(t *testing.T) {
	if validateOrder([]int{0}, 2) {
		t.Fatal("validateOrder() = true; want false for short order")
	}
	if validateOrder([]int{0, 2}, 2) {
		t.Fatal("validateOrder() = true; want false for out-of-range index")
	}
}

func TestHasSixIdenticalPairsRejectsSevenDistinctCards(t *testing.T) {
	bad := []Card{
		card(Two, Hearts), card(Two, Hearts),
		card(Three, Hearts), card(Three, Hearts),
		card(Four, Hearts), card(Four, Hearts),
		card(Five, Hearts), card(Five, Hearts),
		card(Six, Hearts), card(Six, Hearts),
		card(Seven, Hearts), card(Seven, Hearts),
	}
	bad[11] = card(Eight, Hearts)

	if hasSixIdenticalPairs(bad) {
		t.Fatal("hasSixIdenticalPairs() = true; want false when distinct count != 6")
	}
}

func TestHasSixIdenticalPairsRejectsTripleCount(t *testing.T) {
	bad := []Card{
		card(Two, Hearts), card(Two, Hearts), card(Two, Hearts),
		card(Three, Hearts), card(Three, Hearts),
		card(Four, Hearts), card(Four, Hearts),
		card(Five, Hearts), card(Five, Hearts),
		card(Six, Hearts), card(Six, Hearts),
		card(Seven, Hearts),
	}

	if hasSixIdenticalPairs(bad) {
		t.Fatal("hasSixIdenticalPairs() = true; want false when one count != 2")
	}
}

func TestCanTakeDiscardNowRejectsBaseStateFailure(t *testing.T) {
	state := NewGameState()
	state.phase = PhaseInProgress
	state.discardPile = &CardPile{cards: []Card{card(Ace, Hearts)}}

	if state.canTakeDiscardNow() {
		t.Fatal("canTakeDiscardNow() = true; want false")
	}
}

func TestCanTakeDiscardNowAllowsDiscardViaReclaimSearch(t *testing.T) {
	state := newTurnTestState()
	state.players[0].hasOpened = true
	state.players[0].hand.cards = []Card{card(Two, Clubs)}
	state.discardPile = &CardPile{cards: []Card{card(Six, Hearts)}}
	state.activeCompositions = []*Composition{mustRun(t, card(Five, Hearts), joker(), card(Seven, Hearts)), nil}

	if !state.canTakeDiscardNow() {
		t.Fatal("canTakeDiscardNow() = false; want true")
	}

	state.activeCompositions = []*Composition{nil, mustRun(t, card(Five, Hearts), joker(), card(Seven, Hearts)), mustRun(t, card(Seven, Clubs), card(Eight, Clubs), card(Nine, Clubs))}
	if !state.canTakeDiscardNow() {
		t.Fatal("canTakeDiscardNow() = false; want true with nil and non-joker compositions present")
	}

	state.players[0].hand.cards = []Card{card(Two, Clubs)}
	state.discardPile = &CardPile{cards: []Card{card(Six, Clubs)}}
	state.activeCompositions = []*Composition{mustRun(t, card(Five, Hearts), joker(), card(Seven, Hearts))}
	if state.canTakeDiscardNow() {
		t.Fatal("canTakeDiscardNow() = true; want false when reclaim replacement is invalid")
	}
}

func TestHasLegalPlayWithDiscardAndSearchSupportCandidates(t *testing.T) {
	base := tablePlayState{
		handCards:          []Card{card(King, Hearts), card(King, Diamonds), card(Ten, Hearts)},
		activeCompositions: []*Composition{mustRun(t, card(Seven, Hearts), card(Eight, Hearts), card(Nine, Hearts))},
		hasOpened:          true,
	}
	scratch := searchScratch{maskCards: make([]Card, 0, 8), combinedBuf: make([]Card, 0, 14)}
	discardMask := uint32(1 << 2)

	if !hasLegalPlayWithDiscard(base, discardMask, scratch, nil) {
		t.Fatal("hasLegalPlayWithDiscard() = false; want true")
	}
	if hasLegalPlayWithDiscard(base, discardMask, scratch, &tablePlayCandidate{usedMask: discardMask, reclaim: &JokerReclaim{CompositionIndex: 0, JokerIndex: 0, ReplacementCard: card(Two, Clubs)}, usesDiscard: true}) {
		t.Fatal("hasLegalPlayWithDiscard() = true; want false for invalid reclaim candidate")
	}
	if !searchSupportCandidates(tablePlayState{
		handCards:          []Card{card(King, Hearts), card(King, Diamonds), card(King, Clubs), card(Two, Clubs)},
		hasOpened:          true,
		activeCompositions: nil,
	}, 1<<3, 1, 0, nil, nil, nil, nil, scratch) {
		t.Fatal("searchSupportCandidates() = false; want true")
	}
	if !searchSupportCandidates(tablePlayState{
		handCards:          []Card{card(Ten, Hearts), card(Jack, Hearts), card(Two, Clubs), card(Ace, Spades)},
		hasOpened:          true,
		activeCompositions: []*Composition{mustRun(t, card(Seven, Hearts), card(Eight, Hearts), card(Nine, Hearts))},
	}, 1<<3, 1, 0, nil, nil, nil, nil, scratch) {
		t.Fatal("searchSupportCandidates() = false; want true for addition path")
	}
	if hasLegalPlayWithDiscard(tablePlayState{
		handCards:          []Card{card(Two, Clubs), card(Three, Diamonds), card(Five, Spades)},
		hasOpened:          true,
		activeCompositions: []*Composition{mustRun(t, card(Seven, Hearts), card(Eight, Hearts), card(Nine, Hearts))},
	}, 1<<2, scratch, nil) {
		t.Fatal("hasLegalPlayWithDiscard() = true; want false")
	}
	if hasLegalPlayWithDiscard(tablePlayState{
		handCards: []Card{
			card(Seven, Hearts),
			card(Seven, Diamonds),
			card(Seven, Clubs),
		},
		hasOpened:          false,
		activeCompositions: nil,
	}, 1<<2, scratch, nil) {
		t.Fatal("hasLegalPlayWithDiscard() = true; want false after composition search backtracking")
	}
	if searchSupportCandidates(tablePlayState{
		handCards:          []Card{card(Ten, Hearts), card(Ace, Clubs)},
		hasOpened:          false,
		activeCompositions: []*Composition{mustRun(t, card(Seven, Hearts), card(Eight, Hearts), card(Nine, Hearts))},
	}, 1<<1, 1, 0, nil, nil, nil, nil, scratch) {
		t.Fatal("searchSupportCandidates() = true; want false after addition search backtracking")
	}

	reclaim := tablePlayCandidate{usedMask: discardMask, reclaim: &JokerReclaim{CompositionIndex: 0, JokerIndex: 1, ReplacementCard: card(Six, Hearts)}, usesDiscard: true}
	reclaimBase := tablePlayState{
		handCards:          []Card{card(Six, Hearts)},
		activeCompositions: []*Composition{mustRun(t, card(Five, Hearts), joker(), card(Seven, Hearts))},
		hasOpened:          true,
	}
	if !hasLegalPlayWithDiscard(reclaimBase, 1, searchScratch{maskCards: make([]Card, 0, 4), combinedBuf: make([]Card, 0, 14)}, &reclaim) {
		t.Fatal("hasLegalPlayWithDiscard(reclaim) = false; want true")
	}
	if hasLegalPlayWithDiscard(base, discardMask, scratch, &tablePlayCandidate{usedMask: discardMask, reclaim: &JokerReclaim{CompositionIndex: 0, JokerIndex: 1, ReplacementCard: card(Two, Clubs)}, usesDiscard: true}) {
		t.Fatal("hasLegalPlayWithDiscard(reclaim) = true; want false")
	}
}

func TestCanTakeDiscardNowTraversesNilAndReclaimPaths(t *testing.T) {
	state := newTurnTestState()
	state.players[0].hasOpened = true
	state.players[0].hand.cards = []Card{card(Two, Clubs)}
	state.discardPile = &CardPile{cards: []Card{card(Ten, Spades)}}
	state.activeCompositions = []*Composition{
		nil,
		mustSet(t, card(Ten, Hearts), card(Ten, Diamonds), card(Ten, Clubs), joker()),
	}

	if !state.canTakeDiscardNow() {
		t.Fatal("canTakeDiscardNow() = false; want true after nil skip and reclaim traversal")
	}
}

func TestSearchSupportCandidatesRecursiveAdditionPath(t *testing.T) {
	scratch := searchScratch{maskCards: make([]Card, 0, 8), combinedBuf: make([]Card, 0, 14)}
	base := tablePlayState{
		handCards: []Card{
			card(Ten, Hearts),
			card(King, Hearts),
			card(King, Diamonds),
			card(King, Clubs),
			card(Ace, Clubs),
		},
		hasOpened:          false,
		activeCompositions: []*Composition{mustRun(t, card(Seven, Hearts), card(Eight, Hearts), card(Nine, Hearts))},
	}

	if !searchSupportCandidates(base, 1<<4, 1, 0, nil, nil, nil, nil, scratch) {
		t.Fatal("searchSupportCandidates() = false; want true via recursive addition path")
	}
}

func TestAdditionHelpers(t *testing.T) {
	buf := make([]Card, 0, 8)
	base := mustRun(t, card(Queen, Hearts), card(King, Hearts), card(Ace, Hearts))

	if !canAddCardsToComposition(base, []Card{card(Jack, Hearts), joker()}, buf) {
		t.Fatal("canAddCardsToComposition() = false; want true")
	}
	if canInsertCardsIntoComposition(base, -1, []Card{card(Jack, Hearts)}, buf) {
		t.Fatal("canInsertCardsIntoComposition(-1) = true; want false")
	}
	if canInsertCardsIntoComposition(base, len(base.cards)+1, []Card{card(Jack, Hearts)}, buf) {
		t.Fatal("canInsertCardsIntoComposition(out of bounds) = true; want false")
	}
	if !canInsertCardsIntoComposition(base, 0, []Card{joker(), card(Jack, Hearts)}, buf) {
		t.Fatal("canInsertCardsIntoComposition() = false; want true at insert index 0")
	}
	if searchSupportCandidates(tablePlayState{
		handCards:          []Card{card(Two, Clubs)},
		hasOpened:          true,
		activeCompositions: []*Composition{base},
	}, 1<<0, 1, 0, nil, nil, nil, nil, searchScratch{maskCards: make([]Card, 0, 4), combinedBuf: make([]Card, 0, 8)}) {
		t.Fatal("searchSupportCandidates() = true; want false for invalid inserted addition")
	}
	if searchSupportCandidates(tablePlayState{
		handCards:          []Card{card(Two, Clubs), card(Three, Diamonds)},
		hasOpened:          true,
		activeCompositions: []*Composition{nil, base},
	}, 1<<1, 1, 0, nil, nil, nil, nil, searchScratch{maskCards: make([]Card, 0, 4), combinedBuf: make([]Card, 0, 8)}) {
		t.Fatal("searchSupportCandidates() = true; want false after nil composition skip")
	}
}

func TestHasLegalPlayWithDiscardBacktracksInvalidOpeningComposition(t *testing.T) {
	scratch := searchScratch{maskCards: make([]Card, 0, 8), combinedBuf: make([]Card, 0, 14)}
	base := tablePlayState{
		handCards: []Card{
			card(Seven, Hearts),
			card(Seven, Diamonds),
			card(Two, Clubs),
			card(Seven, Clubs),
		},
		hasOpened:          false,
		activeCompositions: nil,
	}

	if hasLegalPlayWithDiscard(base, 1<<3, scratch, nil) {
		t.Fatal("hasLegalPlayWithDiscard() = true; want false after backtracking invalid opening composition")
	}
}

func TestValidateTablePlayBranches(t *testing.T) {
	scratch := searchScratch{maskCards: make([]Card, 0, 8), combinedBuf: make([]Card, 0, 14)}
	base := tablePlayState{handCards: []Card{card(King, Hearts), card(King, Diamonds), card(King, Clubs)}, hasOpened: true}

	if validateTablePlay(base, nil, nil, nil, nil, 0, false, scratch) {
		t.Fatal("validateTablePlay() = true; want false without discard play")
	}
	if validateTablePlay(base, nil, nil, nil, nil, 0, true, scratch) {
		t.Fatal("validateTablePlay() = true; want false with no actions")
	}
	if validateTablePlay(base, []uint32{0b111}, []compositionVariant{compositionVariant("bad")}, nil, nil, 0b111, true, scratch) {
		t.Fatal("validateTablePlay() = true; want false for invalid composition variant")
	}

	reclaimBase := tablePlayState{handCards: []Card{card(Six, Hearts)}, activeCompositions: []*Composition{nil, mustRun(t, card(Five, Hearts), joker(), card(Seven, Hearts))}, hasOpened: true}
	if validateTablePlay(reclaimBase, nil, nil, nil, []JokerReclaim{{CompositionIndex: 0, JokerIndex: 0, ReplacementCard: card(Six, Hearts)}}, 1, true, scratch) {
		t.Fatal("validateTablePlay() = true; want false for nil reclaim target")
	}
	if validateTablePlay(reclaimBase, nil, nil, nil, []JokerReclaim{{CompositionIndex: -1, JokerIndex: 0, ReplacementCard: card(Six, Hearts)}}, 1, true, scratch) {
		t.Fatal("validateTablePlay() = true; want false for invalid reclaim index")
	}
	if validateTablePlay(reclaimBase, nil, nil, nil, []JokerReclaim{{CompositionIndex: 1, JokerIndex: 1, ReplacementCard: card(Two, Clubs)}}, 1, true, scratch) {
		t.Fatal("validateTablePlay() = true; want false for invalid reclaim card")
	}

	additionBase := tablePlayState{handCards: []Card{card(Ten, Hearts), card(Jack, Hearts), card(Ace, Clubs)}, activeCompositions: []*Composition{nil, mustRun(t, card(Seven, Hearts), card(Eight, Hearts), card(Nine, Hearts))}, hasOpened: true}
	if validateTablePlay(additionBase, nil, nil, []selectedAddition{{compositionIndex: 0, mask: 1}}, nil, 1, true, scratch) {
		t.Fatal("validateTablePlay() = true; want false for nil addition target")
	}
	if validateTablePlay(additionBase, nil, nil, []selectedAddition{{compositionIndex: -1, mask: 1}}, nil, 1, true, scratch) {
		t.Fatal("validateTablePlay() = true; want false for invalid addition index")
	}
	if validateTablePlay(additionBase, nil, nil, []selectedAddition{{compositionIndex: 1, mask: 0}}, nil, 0, true, scratch) {
		t.Fatal("validateTablePlay() = true; want false for empty addition cards")
	}
	if validateTablePlay(additionBase, nil, nil, []selectedAddition{{compositionIndex: 1, mask: 0b101}}, nil, 0b101, true, scratch) {
		t.Fatal("validateTablePlay() = true; want false for invalid single-card addition")
	}

	closedBase := tablePlayState{handCards: []Card{card(King, Hearts), card(King, Diamonds), card(King, Clubs)}, hasOpened: false}
	if validateTablePlay(closedBase, nil, nil, []selectedAddition{{compositionIndex: 0, mask: 1}}, nil, 1, true, scratch) {
		t.Fatal("validateTablePlay() = true; want false without opening composition")
	}
	if validateTablePlay(tablePlayState{handCards: []Card{card(Seven, Hearts), card(Seven, Diamonds), card(Seven, Clubs), card(Two, Clubs)}, hasOpened: false}, []uint32{0b111}, []compositionVariant{set}, nil, nil, 0b111, true, scratch) {
		t.Fatal("validateTablePlay() = true; want false below 40 points")
	}
	if validateTablePlay(tablePlayState{handCards: []Card{card(King, Hearts), card(King, Diamonds), card(King, Clubs)}, hasOpened: true}, []uint32{0b111}, []compositionVariant{set}, nil, nil, 0b111, true, scratch) {
		t.Fatal("validateTablePlay() = true; want false when no discard remains")
	}
	if !validateTablePlay(tablePlayState{
		handCards:          []Card{card(Six, Hearts), card(Ten, Hearts), card(Jack, Hearts), card(Ace, Clubs)},
		activeCompositions: []*Composition{mustRun(t, card(Five, Hearts), joker(), card(Seven, Hearts)), mustRun(t, card(Seven, Hearts), card(Eight, Hearts), card(Nine, Hearts))},
		hasOpened:          true,
	}, nil, nil, []selectedAddition{{compositionIndex: 1, mask: 0b110}}, []JokerReclaim{{CompositionIndex: 0, JokerIndex: 1, ReplacementCard: card(Six, Hearts)}}, 0b111, true, scratch) {
		t.Fatal("validateTablePlay() = false; want true for combined reclaim and addition")
	}
	if !validateTablePlay(tablePlayState{
		handCards:          []Card{card(Six, Hearts), card(Ten, Hearts), card(Jack, Hearts), card(Queen, Hearts), card(Ace, Clubs)},
		activeCompositions: []*Composition{mustRun(t, card(Five, Hearts), joker(), card(Seven, Hearts)), mustRun(t, card(Seven, Hearts), card(Eight, Hearts), card(Nine, Hearts))},
		hasOpened:          true,
	}, nil, nil, []selectedAddition{{compositionIndex: 1, mask: 0b00110}, {compositionIndex: 1, mask: 0b01000}}, []JokerReclaim{{CompositionIndex: 0, JokerIndex: 1, ReplacementCard: card(Six, Hearts)}}, 0b01111, true, scratch) {
		t.Fatal("validateTablePlay() = false; want true for repeated composition updates")
	}
	if !validateTablePlay(tablePlayState{
		handCards:          []Card{card(Six, Hearts), card(Eight, Spades), card(Nine, Spades), card(Ten, Spades), card(Jack, Spades), card(Two, Clubs)},
		activeCompositions: []*Composition{mustRun(t, card(Five, Hearts), joker(), card(Seven, Hearts))},
		hasOpened:          false,
	}, []uint32{0b11110}, []compositionVariant{run}, nil, []JokerReclaim{{CompositionIndex: 0, JokerIndex: 1, ReplacementCard: card(Six, Hearts)}}, 0b11111, true, scratch) {
		t.Fatal("validateTablePlay() = false; want true when reclaim points complete opening requirement")
	}
	brokenReclaimBase := tablePlayState{
		handCards: []Card{card(Ten, Hearts), card(King, Clubs)},
		activeCompositions: []*Composition{{
			variant: run,
			cards:   []Card{card(Five, Hearts), joker(), card(Six, Hearts)},
			jokerRepresentations: map[int][]Card{
				1: {card(Four, Hearts)},
			},
		}},
		hasOpened: true,
	}
	if validateTablePlay(brokenReclaimBase, nil, nil, nil, []JokerReclaim{{CompositionIndex: 0, JokerIndex: 1, ReplacementCard: card(Ten, Hearts)}}, 1, true, scratch) {
		t.Fatal("validateTablePlay() = true; want false when reclaim points cannot be resolved")
	}
}

func TestDiscardSearchBaseStateAndTablePlayUsesCard(t *testing.T) {
	state := NewGameState()
	state.discardPile = &CardPile{cards: []Card{card(Ace, Hearts)}}
	if _, ok := state.discardSearchBaseState(); ok {
		t.Fatal("discardSearchBaseState() ok = true; want false")
	}

	state.players = []*Player{NewPlayer()}
	state.players[0].hand.cards = []Card{card(Two, Clubs)}
	base, ok := state.discardSearchBaseState()
	if !ok || len(base.handCards) != 2 {
		t.Fatalf("discardSearchBaseState() = (%+v, %v); want hand with discard appended", base, ok)
	}

	if tablePlayUsesCard([]*Composition{nil}, []CompositionAddition{{Cards: []Card{card(Three, Hearts)}}}, nil, card(Four, Hearts)) {
		t.Fatal("tablePlayUsesCard() = true; want false")
	}
	if !tablePlayUsesCard([]*Composition{mustSet(t, card(King, Hearts), card(King, Diamonds), card(King, Clubs))}, nil, nil, card(King, Diamonds)) {
		t.Fatal("tablePlayUsesCard() = false; want true for composition")
	}
	if !tablePlayUsesCard(nil, []CompositionAddition{{Cards: []Card{card(Ten, Hearts)}}}, nil, card(Ten, Hearts)) {
		t.Fatal("tablePlayUsesCard() = false; want true for addition")
	}
	if !tablePlayUsesCard(nil, nil, []JokerReclaim{{ReplacementCard: card(Six, Hearts)}}, card(Six, Hearts)) {
		t.Fatal("tablePlayUsesCard() = false; want true for reclaim")
	}
}

func TestApplyTablePlayStateErrorBranches(t *testing.T) {
	baseState := tablePlayState{handCards: []Card{card(King, Hearts), card(King, Diamonds), card(King, Clubs), card(Two, Clubs)}, activeCompositions: []*Composition{mustRun(t, card(Seven, Hearts), card(Eight, Hearts), card(Nine, Hearts))}}

	if _, err := applyTablePlayState(baseState, []*Composition{nil}, nil, nil); !errors.Is(err, ErrInvalidComposition) {
		t.Fatalf("applyTablePlayState() error = %v; want %v", err, ErrInvalidComposition)
	}
	if _, err := applyTablePlayState(baseState, []*Composition{{variant: compositionVariant("bad"), cards: []Card{card(Ace, Hearts), card(Ace, Diamonds), card(Ace, Clubs)}}}, nil, nil); !errors.Is(err, ErrInvalidComposition) {
		t.Fatalf("applyTablePlayState() error = %v; want %v", err, ErrInvalidComposition)
	}
	if _, err := applyTablePlayState(baseState, nil, nil, []JokerReclaim{{CompositionIndex: -1, JokerIndex: 0, ReplacementCard: card(Six, Hearts)}}); !errors.Is(err, ErrInvalidComposition) {
		t.Fatalf("applyTablePlayState() error = %v; want %v", err, ErrInvalidComposition)
	}
	if _, err := applyTablePlayState(tablePlayState{handCards: []Card{card(Six, Hearts), card(Two, Clubs)}, activeCompositions: []*Composition{nil}}, nil, nil, []JokerReclaim{{CompositionIndex: 0, JokerIndex: 0, ReplacementCard: card(Six, Hearts)}}); !errors.Is(err, ErrInvalidComposition) {
		t.Fatalf("applyTablePlayState() error = %v; want %v", err, ErrInvalidComposition)
	}
	if _, err := applyTablePlayState(tablePlayState{handCards: []Card{card(Two, Clubs)}, activeCompositions: []*Composition{mustRun(t, card(Five, Hearts), joker(), card(Seven, Hearts))}}, nil, nil, []JokerReclaim{{CompositionIndex: 0, JokerIndex: 1, ReplacementCard: card(Two, Clubs)}}); !errors.Is(err, ErrInvalidComposition) {
		t.Fatalf("applyTablePlayState() error = %v; want %v", err, ErrInvalidComposition)
	}

	if _, err := applyTablePlayState(baseState, nil, []CompositionAddition{{CompositionIndex: 0}}, nil); !errors.Is(err, ErrInvalidComposition) {
		t.Fatalf("applyTablePlayState() error = %v; want %v", err, ErrInvalidComposition)
	}
	if _, err := applyTablePlayState(baseState, nil, []CompositionAddition{{CompositionIndex: -1, Cards: []Card{card(Ten, Hearts)}}}, nil); !errors.Is(err, ErrInvalidComposition) {
		t.Fatalf("applyTablePlayState() error = %v; want %v", err, ErrInvalidComposition)
	}
	if _, err := applyTablePlayState(tablePlayState{handCards: []Card{card(Ten, Hearts), card(Two, Clubs)}, activeCompositions: []*Composition{nil}, hasOpened: true}, nil, []CompositionAddition{{CompositionIndex: 0, Cards: []Card{card(Ten, Hearts)}}}, nil); !errors.Is(err, ErrInvalidComposition) {
		t.Fatalf("applyTablePlayState() error = %v; want %v", err, ErrInvalidComposition)
	}
	if _, err := applyTablePlayState(tablePlayState{handCards: []Card{card(Ten, Hearts), card(Queen, Hearts), card(Two, Clubs)}, activeCompositions: []*Composition{mustRun(t, card(Seven, Hearts), card(Eight, Hearts), card(Nine, Hearts))}, hasOpened: true}, nil, []CompositionAddition{{CompositionIndex: 0, Cards: []Card{card(Ten, Hearts), card(Queen, Hearts)}}}, nil); !errors.Is(err, ErrInvalidComposition) {
		t.Fatalf("applyTablePlayState() error = %v; want %v", err, ErrInvalidComposition)
	}

	if _, err := applyTablePlayState(tablePlayState{handCards: []Card{card(Ten, Hearts)}}, nil, nil, nil); !errors.Is(err, ErrInitialPlayRequiresOwnComp) {
		t.Fatalf("applyTablePlayState() error = %v; want %v", err, ErrInitialPlayRequiresOwnComp)
	}
	if _, err := applyTablePlayState(tablePlayState{handCards: []Card{card(Seven, Hearts), card(Seven, Diamonds), card(Seven, Clubs), card(Two, Clubs)}}, []*Composition{mustSet(t, card(Seven, Hearts), card(Seven, Diamonds), card(Seven, Clubs))}, nil, nil); !errors.Is(err, ErrInitialPointsNotMet) {
		t.Fatalf("applyTablePlayState() error = %v; want %v", err, ErrInitialPointsNotMet)
	}
	if _, err := applyTablePlayState(tablePlayState{handCards: []Card{card(King, Hearts), card(King, Diamonds), card(King, Clubs)}, hasOpened: true}, []*Composition{mustSet(t, card(King, Hearts), card(King, Diamonds), card(King, Clubs))}, nil, nil); !errors.Is(err, ErrMustKeepDiscardCard) {
		t.Fatalf("applyTablePlayState() error = %v; want %v", err, ErrMustKeepDiscardCard)
	}
	if _, err := applyTablePlayState(tablePlayState{handCards: []Card{card(King, Hearts), card(King, Diamonds), card(Two, Clubs)}, hasOpened: true}, []*Composition{mustSet(t, card(King, Hearts), card(King, Diamonds), card(King, Clubs))}, nil, nil); !errors.Is(err, ErrCardsNotInHand) {
		t.Fatalf("applyTablePlayState() error = %v; want %v", err, ErrCardsNotInHand)
	}

	brokenReclaimComp := &Composition{
		variant: run,
		cards:   []Card{card(Five, Hearts), joker(), card(Six, Hearts)},
		jokerRepresentations: map[int][]Card{
			1: {card(Four, Hearts)},
		},
	}
	if _, err := applyTablePlayState(tablePlayState{handCards: []Card{card(Ten, Hearts), card(Two, Clubs)}, activeCompositions: []*Composition{brokenReclaimComp}, hasOpened: true}, nil, nil, []JokerReclaim{{CompositionIndex: 0, JokerIndex: 1, ReplacementCard: card(Ten, Hearts)}}); !errors.Is(err, ErrInvalidComposition) {
		t.Fatalf("applyTablePlayState() error = %v; want %v for unresolved reclaim points", err, ErrInvalidComposition)
	}
}
