package game

import "testing"

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
