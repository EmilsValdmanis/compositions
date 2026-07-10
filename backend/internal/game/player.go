package game

import "github.com/google/uuid"

type Player struct {
	ID           string
	hand         *Hand
	totalPoints  int
	pointsGained int
	hasOpened    bool
	forfeited    bool
}

func NewPlayer() *Player {
	return &Player{
		ID:   uuid.New().String(),
		hand: NewHand(),
	}
}
