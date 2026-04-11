package main

import (
	"fmt"
	"github.com/EmilsValdmanis/compositions/backend/internal/game"
)

func main() {
	cards := []game.Card{
		{Suit: game.Hearts, Rank: game.Ace},
		{Suit: game.Spades, Rank: game.King},
		{Suit: game.Diamonds, Rank: game.Ten},
	}

	fmt.Printf("cards: %+v\n", cards)
}
