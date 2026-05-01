package game

import (
	"errors"
	"testing"
)

const (
	twoPlayerDealerIndex  = 0
	twoPlayerChooserIndex = 1
)

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

	err := state.StartGame(twoPlayerDealerIndex, twoPlayerChooserIndex, DealRoundRobin, nil)

	if err != nil {
		t.Fatalf("StartGame() error = %v", err)
	}
	if state.phase != PhaseInProgress {
		t.Errorf("state.phase = %d; want %d", state.phase, PhaseInProgress)
	}
	if state.turn.number != 1 {
		t.Errorf("state.turn.number = %d; want 1", state.turn.number)
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

func TestGameStateSelectFirstPlayerRequiresPlayers(t *testing.T) {
	state := NewGameState()

	err := state.SelectFirstPlayer()

	if !errors.Is(err, ErrNoPlayers) {
		t.Errorf("SelectFirstPlayer() error = %v; want %v", err, ErrNoPlayers)
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

	err := state.StartGame(2, twoPlayerChooserIndex, DealRoundRobin, nil)

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

	err := state.StartGame(1, 2, DealInBlocks, []int{2, 0, 1})

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

	err := state.StartGame(1, 0, DealInBlocks, []int{2, 0, 1})

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

	err := dealRoundRobin(players, drawPile, 1)

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
	if err := state.StartGame(twoPlayerDealerIndex, twoPlayerChooserIndex, DealRoundRobin, nil); err != nil {
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
	if err := state.StartGame(twoPlayerDealerIndex, twoPlayerChooserIndex, DealRoundRobin, nil); err != nil {
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
		Cards: []Card{{rank: Ten, suit: Hearts}},
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
		Cards: []Card{{rank: Ten, suit: Hearts}},
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
		Cards: []Card{{rank: Ten, suit: Hearts}},
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
		Cards: []Card{{rank: Ten, suit: Hearts}},
	}, {
		CompositionIndex: 0,
		Cards: []Card{{rank: Queen, suit: Hearts}},
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
