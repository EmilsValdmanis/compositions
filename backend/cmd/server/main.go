package main

import (
	"fmt"
	"github.com/EmilsValdmanis/compositions/backend/internal/game"
)

func main() {
	deck := game.NewDeck()

	fmt.Println(deck)
}
