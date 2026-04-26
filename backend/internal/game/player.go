package game

import "github.com/google/uuid"

type Player struct {
	ID          string
	hand        *Hand
	totalPoints int
	hasOpened   bool
}

func NewPlayer() *Player {
	return &Player{
		ID:   uuid.New().String(),
		hand: NewHand(),
	}
}
