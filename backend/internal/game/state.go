package game

type GameState struct {
	players     []*Player
	drawPile    *Deck
	discardPile *Deck
}

func NewGameState() *GameState {
	players := make([]*Player, 0, 4)
	deck := NewGameDeck()
	deck.Shuffle()

	return &GameState{
		players:    players,
		drawPile:   deck,
		discardPile: &Deck{cards: []Card{}},
	}
}
