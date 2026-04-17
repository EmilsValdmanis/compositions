package main

import (
	"fmt"
	"github.com/EmilsValdmanis/compositions/backend/internal/game"
)

func main() {
	gameDeck := game.NewGameDeck()
	gameDeck.Shuffle()

	hand := game.NewHand()
	hand.Draw(gameDeck)
	fmt.Println(hand)
}
