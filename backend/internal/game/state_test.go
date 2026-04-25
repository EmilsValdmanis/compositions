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

func TestGameStatePlayCompositionRequiresDrawFirst(t *testing.T) {
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

	err := state.PlayComposition(comp)

	if !errors.Is(err, ErrPlayerHasntDrawn) {
		t.Errorf("PlayComposition() error = %v; want %v", err, ErrPlayerHasntDrawn)
	}
}

func TestGameStatePlayCompositionRejectsNilComposition(t *testing.T) {
	state := newTurnTestState()
	state.turn.hasDrawn = true

	err := state.PlayComposition(nil)

	if !errors.Is(err, ErrInvalidComposition) {
		t.Errorf("PlayComposition(nil) error = %v; want %v", err, ErrInvalidComposition)
	}
}

func TestGameStatePlayCompositionMovesCardsToActiveCompositions(t *testing.T) {
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

	err := state.PlayComposition(comp)

	if err != nil {
		t.Fatalf("PlayComposition() error = %v", err)
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

func TestGameStatePlayCompositionRejectsCardsNotInHand(t *testing.T) {
	state := newTurnTestState()
	state.turn.hasDrawn = true
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

	err := state.PlayComposition(comp)

	if !errors.Is(err, ErrCardsNotInHand) {
		t.Errorf("PlayComposition() error = %v; want %v", err, ErrCardsNotInHand)
	}
	if len(state.activeCompositions) != 0 {
		t.Errorf("len(state.activeCompositions) = %d; want 0", len(state.activeCompositions))
	}
	if len(state.players[0].hand.cards) != 3 {
		t.Errorf("len(state.players[0].hand.cards) = %d; want 3", len(state.players[0].hand.cards))
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
