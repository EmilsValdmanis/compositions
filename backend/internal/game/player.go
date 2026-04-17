package game

type Player struct {
	hand  *Hand
	score int
}

func NewPlayer() *Player {
	return &Player{hand: NewHand()}
}
