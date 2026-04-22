package game

import (
	"errors"
	"math/rand"
)

type GamePhase int

const (
	PhaseLobby = iota
	PhaseInProgress
	PhaseGameOver
)

type Turn struct {
	number      int
	playerIndex int
}

type GameState struct {
	players            []*Player
	activeCompositions []*Composition
	drawPile           *CardPile
	discardPile        *CardPile
	maxPlayers         int
	phase              GamePhase
	round              int
	turn               Turn
}

var (
	ErrGameInProgress           = errors.New("game already in progress")
	ErrGameFull                 = errors.New("game is full")
	ErrPlayerExists             = errors.New("player already in game")
	ErrNilPlayer                = errors.New("player is nil")
	ErrNotEnoughPlayers         = errors.New("need at least 2 players to start")
	ErrNotEnoughCardsInDrawPile = errors.New("not enough cards in draw pile for all players")
	ErrNoPlayers                = errors.New("no players in game")
)

func NewGameState() *GameState {
	players := make([]*Player, 0, 4)
	deck := NewGameDeck()
	deck.Shuffle()

	return &GameState{
		players:            players,
		activeCompositions: []*Composition{},
		drawPile:           deck,
		discardPile:        &CardPile{cards: make([]Card, 0, cardsInDeck*2)},
		maxPlayers:         4,
		phase:              PhaseLobby,
		round:              1,
		turn: Turn{
			number:      1,
			playerIndex: 0,
		},
	}
}

func (gs *GameState) StartGame() error {
	if gs.phase != PhaseLobby {
		return ErrGameInProgress
	}
	if len(gs.players) < 2 {
		return ErrNotEnoughPlayers
	}
	if err := gs.dealInitialHands(); err != nil {
		return err
	}
	card, ok := gs.drawPile.DrawOne()
	if !ok {
		return ErrNotEnoughCardsInDrawPile
	}
	gs.discardPile.AddToTop(card)
	if err := gs.SelectFirstPlayer(); err != nil {
		return err
	}
	gs.phase = PhaseInProgress
	return nil
}

func (gs *GameState) SelectFirstPlayer() error {
	if len(gs.players) == 0 {
		return ErrNoPlayers
	}

	gs.turn.playerIndex = rand.Intn(len(gs.players))
	return nil
}

func (gs *GameState) CurrentPlayer() (*Player, error) {
	if len(gs.players) == 0 {
		return nil, ErrNoPlayers
	}

	return gs.players[gs.turn.playerIndex], nil
}

func (gs *GameState) AddPlayer(p *Player) error {
	if gs.phase != PhaseLobby {
		return ErrGameInProgress
	}
	if p == nil {
		return ErrNilPlayer
	}
	if len(gs.players) >= gs.maxPlayers {
		return ErrGameFull
	}
	for _, existing := range gs.players {
		if existing.ID == p.ID {
			return ErrPlayerExists
		}
	}
	gs.players = append(gs.players, p)
	return nil
}

func (gs *GameState) dealInitialHands() error {
	required := InitialHandSize * len(gs.players)
	if len(gs.drawPile.cards) < required {
		return ErrNotEnoughCardsInDrawPile
	}

	for range InitialHandSize {
		for _, player := range gs.players {
			if !player.hand.Draw(gs.drawPile) {
				return ErrNotEnoughCardsInDrawPile
			}
		}
	}
	return nil
}
