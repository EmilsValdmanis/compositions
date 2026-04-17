package main

import (
	"fmt"

	"github.com/EmilsValdmanis/compositions/backend/internal/game"
)

func main() {
	gs := game.NewGameState()
	fmt.Println(gs)
}
