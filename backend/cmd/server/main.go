package main

import (
	"fmt"
	"github.com/EmilsValdmanis/compositions/backend/internal/game"
)

func main() {
	gameDeck := game.NewGameDeck()
	gameDeck.Shuffle()

	fmt.Println(gameDeck)
}
