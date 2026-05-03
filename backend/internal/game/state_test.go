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

	err := state.StartGame(twoPlayerDealerIndex, twoPlayerChooserIndex, DealInBlocks, []int{0, 1})

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
	if len(state.drawPile.cards) != 2 {
		t.Fatalf("len(state.drawPile.cards) = %d; want 2", len(state.drawPile.cards))
	}
	if top := state.drawPile.cards[0]; top.rank != Five || top.suit != Clubs {
		t.Fatalf("drawPile.cards[0] = %+v; want Five of Clubs", top)
	}
	if next := state.drawPile.cards[1]; next.rank != Queen || next.suit != Spades {
		t.Fatalf("drawPile.cards[1] = %+v; want Queen of Spades", next)
	}
	if len(state.discardPile.cards) != 0 {
		t.Fatalf("len(state.discardPile.cards) = %d; want 0", len(state.discardPile.cards))
	}
	if !state.turn.hasDrawn {
		t.Fatal("state.turn.hasDrawn = false; want true")
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
	if state.activeCompositions[0].cards[1].isJoker {
		t.Fatal("reclaimed joker was not replaced in base composition")
	}
	if got := state.activeCompositions[0].cards[1]; got.rank != Six || got.suit != Hearts {
		t.Fatalf("state.activeCompositions[0].cards[1] = %+v; want Six of Hearts", got)
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
	if len(state.discardPile.cards) != 0 {
		t.Fatalf("len(state.discardPile.cards) = %d; want 0", len(state.discardPile.cards))
	}
	if len(state.drawPile.cards) != 2 {
		t.Fatalf("len(state.drawPile.cards) = %d; want 2", len(state.drawPile.cards))
	}
	if top := state.drawPile.cards[0]; top.rank != Four || top.suit != Diamonds {
		t.Fatalf("drawPile.cards[0] = %+v; want Four of Diamonds", top)
	}
	if next := state.drawPile.cards[1]; next.rank != King || next.suit != Spades {
		t.Fatalf("drawPile.cards[1] = %+v; want King of Spades", next)
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

func TestGameStatePlayTableEndsRoundForSameSuitCollection(t *testing.T) {
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
	if state.phase != PhaseRoundOver {
		t.Fatalf("state.phase = %d; want %d", state.phase, PhaseRoundOver)
	}
	if state.roundWinnerIndex != 0 {
		t.Fatalf("state.roundWinnerIndex = %d; want 0", state.roundWinnerIndex)
	}
	if len(state.players[0].hand.cards) != 12 {
		t.Fatalf("len(state.players[0].hand.cards) = %d; want 12", len(state.players[0].hand.cards))
	}
	if state.players[1].totalPoints != 32 {
		t.Fatalf("player 1 totalPoints = %d; want 32", state.players[1].totalPoints)
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
