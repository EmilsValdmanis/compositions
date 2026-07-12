package game

import "testing"

// These end-to-end game flows use deterministic decks to exercise complete
// turns and rounds without depending on random shuffle order.

func cardsOf(suit Suit, ranks ...Rank) []Card {
	cards := make([]Card, 0, len(ranks))
	for _, rank := range ranks {
		cards = append(cards, card(rank, suit))
	}
	return cards
}

func sameSuitCollectionHand(suit Suit) []Card {
	return cardsOf(suit, Ace, Two, Three, Four, Five, Six, Seven, Eight, Nine, Ten, Jack, Queen)
}

func orderedGameDeck(cards ...Card) *CardPile {
	fullDeck := NewGameDeck().cards
	remaining := append([]Card(nil), fullDeck...)
	ordered := make([]Card, 0, len(fullDeck))

	for _, want := range cards {
		index := indexOfCard(remaining, want)
		if index < 0 {
			panic("orderedGameDeck: requested card not available")
		}
		ordered = append(ordered, remaining[index])
		remaining = append(remaining[:index], remaining[index+1:]...)
	}

	ordered = append(ordered, remaining...)
	return &CardPile{cards: ordered}
}

func blockDealSetup(order []int, handsByPlayer [][]Card, discard Card, drawPile ...Card) []Card {
	stack := make([]Card, 0)
	for _, playerIndex := range order {
		stack = append(stack, handsByPlayer[playerIndex]...)
	}
	stack = append(stack, discard)
	stack = append(stack, drawPile...)
	return stack
}

func indexOfCard(cards []Card, target Card) int {
	for i, candidate := range cards {
		if candidate.isJoker != target.isJoker {
			continue
		}
		if target.isJoker || cardsEqual(candidate, target) {
			return i
		}
	}

	return -1
}

func mustSet(t *testing.T, cards ...Card) *Composition {
	t.Helper()

	comp, ok := NewSet(cards)
	if !ok {
		t.Fatalf("NewSet(%v) returned false; want true", cards)
	}

	return comp
}

func mustRun(t *testing.T, cards ...Card) *Composition {
	t.Helper()

	comp, ok := NewRun(cards)
	if !ok {
		t.Fatalf("NewRun(%v) returned false; want true", cards)
	}

	return comp
}

func mustAddPlayers(t *testing.T, state *GameState, count int) []*Player {
	t.Helper()

	players := make([]*Player, 0, count)
	for range count {
		player := NewPlayer()
		if err := state.AddPlayer(player); err != nil {
			t.Fatalf("AddPlayer() error = %v", err)
		}
		players = append(players, player)
	}

	return players
}

func withRiggedShuffledDeck(t *testing.T, cards ...Card) func() {
	t.Helper()

	previous := newShuffledGameDeck
	newShuffledGameDeck = func() *CardPile {
		return orderedGameDeck(cards...)
	}

	return func() {
		newShuffledGameDeck = previous
	}
}

func mustStartGameWithDeck(t *testing.T, dealerIndex, chooserIndex int, dealType DealTypes, order []int, cutSize int, cards ...Card) (*GameState, []*Player) {
	t.Helper()

	restore := withRiggedShuffledDeck(t, cards...)
	defer restore()

	state := NewGameState()
	players := mustAddPlayers(t, state, requiredPlayerCount(order, dealerIndex, chooserIndex))

	if err := state.StartGame(dealerIndex, chooserIndex, dealType, order, cutSize); err != nil {
		t.Fatalf("StartGame() error = %v", err)
	}

	return state, players
}

func mustStartNextRoundWithDeck(t *testing.T, state *GameState, dealType DealTypes, order []int, cutSize int, cards ...Card) {
	t.Helper()

	restore := withRiggedShuffledDeck(t, cards...)
	defer restore()

	if err := state.StartNextRound(dealType, order, cutSize); err != nil {
		t.Fatalf("StartNextRound() error = %v", err)
	}
}

func mustDrawFromDeck(t *testing.T, state *GameState) {
	t.Helper()

	if err := state.DrawFromDeck(); err != nil {
		t.Fatalf("DrawFromDeck() error = %v", err)
	}
}

func mustDrawFromDiscard(t *testing.T, state *GameState) {
	t.Helper()

	if err := state.DrawFromDiscard(); err != nil {
		t.Fatalf("DrawFromDiscard() error = %v", err)
	}
}

func mustPlayTable(t *testing.T, state *GameState, comps []*Composition, additions []CompositionAddition, reclaims ...JokerReclaim) {
	t.Helper()

	if err := state.PlayTable(comps, additions, reclaims...); err != nil {
		t.Fatalf("PlayTable() error = %v", err)
	}
}

func mustDiscardCard(t *testing.T, state *GameState, want Card) {
	t.Helper()

	player, err := state.CurrentPlayer()
	if err != nil {
		t.Fatalf("CurrentPlayer() error = %v", err)
	}

	index := indexOfCard(player.hand.cards, want)
	if index < 0 {
		t.Fatalf("discard card %+v not found in hand %+v", want, player.hand.cards)
	}

	if err := state.DiscardFromHand(index); err != nil {
		t.Fatalf("DiscardFromHand() error = %v", err)
	}
}

func requiredPlayerCount(order []int, dealerIndex, chooserIndex int) int {
	count := max(2, len(order))
	count = max(count, dealerIndex+1)
	count = max(count, chooserIndex+1)
	return count
}

func TestGameStateIntegrationFirstPlayerCanTakeOpeningDiscardInsteadOfDrawingFromDeck(t *testing.T) {
	openingHand := []Card{
		card(King, Hearts),
		card(King, Diamonds),
		card(King, Clubs),
		card(Ace, Spades),
		card(Two, Spades),
		card(Three, Spades),
		card(Five, Hearts),
		card(Six, Hearts),
		card(Seven, Hearts),
		card(Five, Spades),
		card(Eight, Hearts),
		card(Two, Clubs),
	}
	otherHand := []Card{
		card(Ace, Clubs),
		card(Ace, Diamonds),
		card(Two, Clubs),
		card(Four, Clubs),
		card(Five, Diamonds),
		card(Six, Clubs),
		card(Seven, Diamonds),
		card(Eight, Clubs),
		card(Nine, Diamonds),
		card(Two, Hearts),
		card(Five, Spades),
		card(Three, Spades),
	}

	state, players := mustStartGameWithDeck(
		t,
		twoPlayerDealerIndex,
		twoPlayerChooserIndex,
		DealInBlocks,
		[]int{1, 0},
		0,
		blockDealSetup([]int{1, 0}, [][]Card{otherHand, openingHand}, card(Four, Spades), card(Three, Diamonds))...,
	)

	currentPlayer, err := state.CurrentPlayer()
	if err != nil {
		t.Fatalf("CurrentPlayer() error = %v", err)
	}
	if currentPlayer != players[1] {
		t.Fatalf("CurrentPlayer() = %p; want %p", currentPlayer, players[1])
	}
	if len(state.discardPile.cards) != 1 {
		t.Fatalf("len(state.discardPile.cards) = %d; want 1", len(state.discardPile.cards))
	}
	if got := state.discardPile.cards[0]; !cardsEqual(got, card(Four, Spades)) {
		t.Fatalf("discardPile.cards[0] = %+v; want %+v", got, card(Four, Spades))
	}

	startingDrawPileSize := len(state.drawPile.cards)
	mustDrawFromDiscard(t, state)

	if len(state.drawPile.cards) != startingDrawPileSize {
		t.Fatalf("len(state.drawPile.cards) = %d; want %d", len(state.drawPile.cards), startingDrawPileSize)
	}
	if len(state.discardPile.cards) != 0 {
		t.Fatalf("len(state.discardPile.cards) = %d; want 0", len(state.discardPile.cards))
	}
	if len(players[1].hand.cards) != len(openingHand)+1 {
		t.Fatalf("len(players[1].hand.cards) = %d; want %d", len(players[1].hand.cards), len(openingHand)+1)
	}
	if drawn := players[1].hand.cards[len(players[1].hand.cards)-1]; !cardsEqual(drawn, card(Four, Spades)) {
		t.Fatalf("drawn card = %+v; want %+v", drawn, card(Four, Spades))
	}
	if !state.turn.hasDrawn {
		t.Fatal("state.turn.hasDrawn = false; want true")
	}
}

func TestGameStateIntegrationFullTurnDrawPlayDiscardAdvancesToNextPlayer(t *testing.T) {
	openingHand := []Card{
		card(King, Hearts),
		card(King, Diamonds),
		card(King, Clubs),
		card(Ace, Spades),
		card(Two, Spades),
		card(Three, Spades),
		card(Five, Hearts),
		card(Six, Hearts),
		card(Seven, Hearts),
		card(Five, Spades),
		card(Eight, Hearts),
		card(Two, Clubs),
	}
	otherHand := []Card{
		card(Ace, Clubs),
		card(Ace, Diamonds),
		card(Two, Clubs),
		card(Four, Clubs),
		card(Five, Diamonds),
		card(Six, Clubs),
		card(Seven, Diamonds),
		card(Eight, Clubs),
		card(Nine, Diamonds),
		card(Two, Hearts),
		card(Five, Spades),
		card(Three, Spades),
	}

	state, players := mustStartGameWithDeck(
		t,
		twoPlayerDealerIndex,
		twoPlayerChooserIndex,
		DealInBlocks,
		[]int{1, 0},
		0,
		blockDealSetup([]int{1, 0}, [][]Card{otherHand, openingHand}, card(Four, Spades), card(Three, Diamonds), card(Ace, Diamonds))...,
	)

	currentPlayer, err := state.CurrentPlayer()
	if err != nil {
		t.Fatalf("CurrentPlayer() error = %v", err)
	}
	if currentPlayer != players[1] {
		t.Fatalf("CurrentPlayer() = %p; want %p", currentPlayer, players[1])
	}

	mustDrawFromDiscard(t, state)
	mustPlayTable(t, state, []*Composition{
		mustSet(t, card(King, Hearts), card(King, Diamonds), card(King, Clubs)),
		mustRun(t, card(Ace, Spades), card(Two, Spades), card(Three, Spades), card(Four, Spades)),
		mustRun(t, card(Five, Hearts), card(Six, Hearts), card(Seven, Hearts)),
	}, nil)
	mustDiscardCard(t, state, card(Two, Clubs))

	if state.phase != PhaseInProgress {
		t.Fatalf("state.phase = %d; want %d", state.phase, PhaseInProgress)
	}
	if state.turn.number != 2 {
		t.Fatalf("state.turn.number = %d; want 2", state.turn.number)
	}
	if state.turn.playerIndex != 0 {
		t.Fatalf("state.turn.playerIndex = %d; want 0", state.turn.playerIndex)
	}
	if state.turn.hasDrawn {
		t.Fatal("state.turn.hasDrawn = true; want false")
	}
	if !players[1].hasOpened {
		t.Fatal("players[1].hasOpened = false; want true")
	}
	if len(state.activeCompositions) != 3 {
		t.Fatalf("len(state.activeCompositions) = %d; want 3", len(state.activeCompositions))
	}
	if len(state.discardPile.cards) != 1 {
		t.Fatalf("len(state.discardPile.cards) = %d; want 1", len(state.discardPile.cards))
	}
	if got := state.discardPile.cards[0]; !cardsEqual(got, card(Two, Clubs)) {
		t.Fatalf("discardPile.cards[0] = %+v; want %+v", got, card(Two, Clubs))
	}
	if len(players[1].hand.cards) != 2 {
		t.Fatalf("len(players[1].hand.cards) = %d; want 2", len(players[1].hand.cards))
	}
	if got := players[1].hand.cards[0]; !cardsEqual(got, card(Five, Spades)) {
		t.Fatalf("players[1].hand.cards[0] = %+v; want %+v", got, card(Five, Spades))
	}
	if got := players[1].hand.cards[1]; !cardsEqual(got, card(Eight, Hearts)) {
		t.Fatalf("players[1].hand.cards[1] = %+v; want %+v", got, card(Eight, Hearts))
	}
}

func TestGameStateIntegrationFullRoundAcrossAlternatingTurns(t *testing.T) {
	openingHand := []Card{
		card(King, Hearts),
		card(King, Diamonds),
		card(King, Clubs),
		card(Ace, Spades),
		card(Two, Spades),
		card(Three, Spades),
		card(Five, Hearts),
		card(Six, Hearts),
		card(Seven, Hearts),
		card(Five, Spades),
		card(Eight, Hearts),
		card(Two, Clubs),
	}
	otherHand := []Card{
		card(Ace, Clubs),
		card(Ace, Diamonds),
		card(Two, Clubs),
		card(Four, Clubs),
		card(Five, Diamonds),
		card(Six, Clubs),
		card(Seven, Diamonds),
		card(Eight, Clubs),
		card(Nine, Diamonds),
		card(Two, Hearts),
		card(Five, Spades),
		card(Three, Spades),
	}

	state, players := mustStartGameWithDeck(
		t,
		twoPlayerDealerIndex,
		twoPlayerChooserIndex,
		DealInBlocks,
		[]int{1, 0},
		0,
		blockDealSetup([]int{1, 0}, [][]Card{otherHand, openingHand}, card(Four, Spades), card(Three, Diamonds), card(Ace, Diamonds))...,
	)

	mustDrawFromDiscard(t, state)
	mustPlayTable(t, state, []*Composition{
		mustSet(t, card(King, Hearts), card(King, Diamonds), card(King, Clubs)),
		mustRun(t, card(Ace, Spades), card(Two, Spades), card(Three, Spades), card(Four, Spades)),
		mustRun(t, card(Five, Hearts), card(Six, Hearts), card(Seven, Hearts)),
	}, nil)
	mustDiscardCard(t, state, card(Two, Clubs))

	mustDrawFromDeck(t, state)
	mustDiscardCard(t, state, card(Five, Spades))

	mustDrawFromDeck(t, state)
	mustPlayTable(t, state, nil, []CompositionAddition{
		{CompositionIndex: 1, Cards: []Card{card(Five, Spades)}},
		{CompositionIndex: 2, Cards: []Card{card(Eight, Hearts)}},
	})
	mustDiscardCard(t, state, card(Ace, Diamonds))

	if state.phase != PhaseRoundOver {
		t.Fatalf("state.phase = %d; want %d", state.phase, PhaseRoundOver)
	}
	if state.roundWinnerIndex != 1 {
		t.Fatalf("state.roundWinnerIndex = %d; want 1", state.roundWinnerIndex)
	}
	if state.turn.number != 3 {
		t.Fatalf("state.turn.number = %d; want 3", state.turn.number)
	}
	if state.turn.playerIndex != 1 {
		t.Fatalf("state.turn.playerIndex = %d; want 1", state.turn.playerIndex)
	}
	if state.turn.hasDrawn {
		t.Fatal("state.turn.hasDrawn = true; want false")
	}
	if players[1].totalPoints != 0 {
		t.Fatalf("winner totalPoints = %d; want 0", players[1].totalPoints)
	}
	if players[0].totalPoints != 69 {
		t.Fatalf("loser totalPoints = %d; want 69", players[0].totalPoints)
	}
	if len(players[1].hand.cards) != 0 {
		t.Fatalf("len(players[1].hand.cards) = %d; want 0", len(players[1].hand.cards))
	}
	if len(state.activeCompositions) != 3 {
		t.Fatalf("len(state.activeCompositions) = %d; want 3", len(state.activeCompositions))
	}
	if len(state.activeCompositions[1].cards) != 5 {
		t.Fatalf("len(state.activeCompositions[1].cards) = %d; want 5", len(state.activeCompositions[1].cards))
	}
	if len(state.activeCompositions[2].cards) != 4 {
		t.Fatalf("len(state.activeCompositions[2].cards) = %d; want 4", len(state.activeCompositions[2].cards))
	}
	if len(state.discardPile.cards) != 3 {
		t.Fatalf("len(state.discardPile.cards) = %d; want 3", len(state.discardPile.cards))
	}
	if got := state.discardPile.cards[0]; !cardsEqual(got, card(Ace, Diamonds)) {
		t.Fatalf("discardPile.cards[0] = %+v; want %+v", got, card(Ace, Diamonds))
	}
	if got := state.discardPile.cards[1]; !cardsEqual(got, card(Five, Spades)) {
		t.Fatalf("discardPile.cards[1] = %+v; want %+v", got, card(Five, Spades))
	}
	if got := state.discardPile.cards[2]; !cardsEqual(got, card(Two, Clubs)) {
		t.Fatalf("discardPile.cards[2] = %+v; want %+v", got, card(Two, Clubs))
	}
}

func TestGameStateIntegrationMultiRoundScoringAdjustmentAndGameOver(t *testing.T) {
	roundOneHands := [][]Card{
		sameSuitCollectionHand(Hearts),
		ninetySixPointHand(),
		eightyNinePointHand(),
	}
	roundTwoHands := [][]Card{
		eightyNinePointHand(),
		thirtyTwoPointHandNoClubs(),
		sameSuitCollectionHand(Clubs),
	}
	roundThreeHands := [][]Card{
		eightyNinePointHand(),
		sameSuitCollectionHand(Spades),
		fortyTwoPointHandNoSpades(),
	}

	state, players := mustStartGameWithDeck(
		t,
		0,
		2,
		DealInBlocks,
		[]int{0, 1, 2},
		0,
		blockDealSetup([]int{0, 1, 2}, roundOneHands, card(King, Hearts))...,
	)

	if state.phase != PhaseRoundOver {
		t.Fatalf("round 1 phase = %d; want %d", state.phase, PhaseRoundOver)
	}
	if state.round != 1 {
		t.Fatalf("state.round = %d; want 1", state.round)
	}
	if state.roundWinnerIndex != 0 {
		t.Fatalf("round 1 winner = %d; want 0", state.roundWinnerIndex)
	}
	if players[0].totalPoints != 0 {
		t.Fatalf("player 0 totalPoints = %d; want 0", players[0].totalPoints)
	}
	if players[1].totalPoints != 96 {
		t.Fatalf("player 1 totalPoints = %d; want 96", players[1].totalPoints)
	}
	if players[2].totalPoints != 89 {
		t.Fatalf("player 2 totalPoints = %d; want 89", players[2].totalPoints)
	}

	mustStartNextRoundWithDeck(
		t,
		state,
		DealInBlocks,
		[]int{2, 0, 1},
		0,
		blockDealSetup([]int{2, 0, 1}, roundTwoHands, card(Ace, Spades))...,
	)

	if state.phase != PhaseRoundOver {
		t.Fatalf("round 2 phase = %d; want %d", state.phase, PhaseRoundOver)
	}
	if state.round != 2 {
		t.Fatalf("state.round = %d; want 2", state.round)
	}
	if state.roundWinnerIndex != 2 {
		t.Fatalf("round 2 winner = %d; want 2", state.roundWinnerIndex)
	}
	if players[0].totalPoints != 89 {
		t.Fatalf("player 0 totalPoints = %d; want 89", players[0].totalPoints)
	}
	if players[1].totalPoints != 89 {
		t.Fatalf("player 1 totalPoints = %d; want 89", players[1].totalPoints)
	}
	if players[2].totalPoints != 89 {
		t.Fatalf("player 2 totalPoints = %d; want 89", players[2].totalPoints)
	}

	mustStartNextRoundWithDeck(
		t,
		state,
		DealInBlocks,
		[]int{1, 0, 2},
		0,
		blockDealSetup([]int{1, 0, 2}, roundThreeHands, card(King, Diamonds))...,
	)

	if state.phase != PhaseGameOver {
		t.Fatalf("round 3 phase = %d; want %d", state.phase, PhaseGameOver)
	}
	if state.round != 3 {
		t.Fatalf("state.round = %d; want 3", state.round)
	}
	if state.roundWinnerIndex != 1 {
		t.Fatalf("round 3 winner = %d; want 1", state.roundWinnerIndex)
	}
	if players[0].totalPoints != 178 {
		t.Fatalf("player 0 totalPoints = %d; want 178", players[0].totalPoints)
	}
	if players[1].totalPoints != 89 {
		t.Fatalf("player 1 totalPoints = %d; want 89", players[1].totalPoints)
	}
	if players[2].totalPoints != 131 {
		t.Fatalf("player 2 totalPoints = %d; want 131", players[2].totalPoints)
	}
}

func ninetySixPointHand() []Card {
	return []Card{
		card(Ace, Clubs),
		card(King, Clubs),
		card(Queen, Clubs),
		card(Jack, Clubs),
		card(Ten, Clubs),
		card(Nine, Diamonds),
		card(Eight, Diamonds),
		card(Seven, Diamonds),
		card(Six, Diamonds),
		card(Five, Diamonds),
		card(Four, Spades),
		card(Seven, Spades),
	}
}

func eightyNinePointHand() []Card {
	return []Card{
		card(King, Clubs),
		card(Queen, Clubs),
		card(Jack, Clubs),
		card(Ten, Clubs),
		card(Nine, Diamonds),
		card(Eight, Diamonds),
		card(Seven, Diamonds),
		card(Six, Diamonds),
		card(Five, Diamonds),
		card(Four, Spades),
		card(Three, Spades),
		card(Seven, Spades),
	}
}

func thirtyTwoPointHandNoClubs() []Card {
	return []Card{
		card(Two, Hearts),
		card(Two, Hearts),
		card(Two, Diamonds),
		card(Two, Diamonds),
		card(Two, Spades),
		card(Two, Spades),
		card(Three, Hearts),
		card(Three, Hearts),
		card(Three, Diamonds),
		card(Three, Spades),
		card(Four, Hearts),
		card(Four, Diamonds),
	}
}

func fortyTwoPointHandNoSpades() []Card {
	return []Card{
		card(Two, Hearts),
		card(Two, Diamonds),
		card(Two, Clubs),
		card(Three, Hearts),
		card(Three, Diamonds),
		card(Three, Clubs),
		card(Four, Hearts),
		card(Four, Diamonds),
		card(Four, Clubs),
		card(Five, Hearts),
		card(Five, Diamonds),
		card(Five, Clubs),
	}
}
