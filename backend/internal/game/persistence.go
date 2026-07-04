package game

import (
	"errors"
	"fmt"
	"slices"
)

const PersistenceSnapshotVersion = 1

type PersistenceSnapshot struct {
	Version            int                              `json:"version"`
	Players            []PersistencePlayerSnapshot      `json:"players"`
	ActiveCompositions []PersistenceCompositionSnapshot `json:"activeCompositions"`
	DrawPile           []CardSnapshot                   `json:"drawPile"`
	DiscardPile        []CardSnapshot                   `json:"discardPile"`
	MaxPlayers         int                              `json:"maxPlayers"`
	Phase              GamePhase                        `json:"phase"`
	Round              int                              `json:"round"`
	DealerIndex        int                              `json:"dealerIndex"`
	Turn               PersistenceTurnSnapshot          `json:"turn"`
	RoundWinnerIndex   int                              `json:"roundWinnerIndex"`
}

type PersistencePlayerSnapshot struct {
	ID           string         `json:"id"`
	Hand         []CardSnapshot `json:"hand"`
	TotalPoints  int            `json:"totalPoints"`
	PointsGained int            `json:"pointsGained"`
	HasOpened    bool           `json:"hasOpened"`
}

type PersistenceCompositionSnapshot struct {
	Variant string         `json:"variant"`
	Cards   []CardSnapshot `json:"cards"`
}

type PersistenceTurnSnapshot struct {
	Number             int           `json:"number"`
	PlayerIndex        int           `json:"playerIndex"`
	HasDrawn           bool          `json:"hasDrawn"`
	MustUseDiscardDraw bool          `json:"mustUseDiscardDraw"`
	DiscardDrawCard    *CardSnapshot `json:"discardDrawCard,omitempty"`
}

func (gs *GameState) PersistenceSnapshot() PersistenceSnapshot {
	if gs == nil {
		return PersistenceSnapshot{}
	}

	players := make([]PersistencePlayerSnapshot, 0, len(gs.players))
	for _, player := range gs.players {
		if player == nil {
			continue
		}
		hand := []CardSnapshot{}
		if player.hand != nil {
			hand = cardSnapshots(player.hand.cards)
		}
		players = append(players, PersistencePlayerSnapshot{
			ID:           player.ID,
			Hand:         hand,
			TotalPoints:  player.totalPoints,
			PointsGained: player.pointsGained,
			HasOpened:    player.hasOpened,
		})
	}

	compositions := make([]PersistenceCompositionSnapshot, 0, len(gs.activeCompositions))
	for _, comp := range gs.activeCompositions {
		if comp == nil {
			continue
		}
		compositions = append(compositions, PersistenceCompositionSnapshot{
			Variant: string(comp.variant),
			Cards:   cardSnapshots(comp.cards),
		})
	}

	turn := PersistenceTurnSnapshot{
		Number:             gs.turn.number,
		PlayerIndex:        gs.turn.playerIndex,
		HasDrawn:           gs.turn.hasDrawn,
		MustUseDiscardDraw: gs.turn.mustUseDiscardDraw,
	}
	if gs.turn.mustUseDiscardDraw {
		card := gs.turn.discardDrawCard.Snapshot()
		turn.DiscardDrawCard = &card
	}

	drawPile := []CardSnapshot{}
	if gs.drawPile != nil {
		drawPile = cardSnapshots(gs.drawPile.cards)
	}
	discardPile := []CardSnapshot{}
	if gs.discardPile != nil {
		discardPile = cardSnapshots(gs.discardPile.cards)
	}

	return PersistenceSnapshot{
		Version:            PersistenceSnapshotVersion,
		Players:            players,
		ActiveCompositions: compositions,
		DrawPile:           drawPile,
		DiscardPile:        discardPile,
		MaxPlayers:         gs.maxPlayers,
		Phase:              gs.phase,
		Round:              gs.round,
		DealerIndex:        gs.dealerIndex,
		Turn:               turn,
		RoundWinnerIndex:   gs.roundWinnerIndex,
	}
}

func RestoreGameState(snapshot PersistenceSnapshot) (*GameState, error) {
	if snapshot.Version != PersistenceSnapshotVersion {
		return nil, fmt.Errorf("unsupported game persistence snapshot version %d", snapshot.Version)
	}
	if snapshot.MaxPlayers <= 0 {
		return nil, errors.New("max players must be positive")
	}
	if snapshot.Round <= 0 {
		return nil, errors.New("round must be positive")
	}
	if snapshot.Phase < PhaseLobby || snapshot.Phase > PhaseGameOver {
		return nil, errors.New("invalid game phase")
	}

	players := make([]*Player, 0, len(snapshot.Players))
	seenPlayerIDs := make(map[string]bool, len(snapshot.Players))
	for _, playerSnapshot := range snapshot.Players {
		if playerSnapshot.ID == "" {
			return nil, errors.New("player id is required")
		}
		if seenPlayerIDs[playerSnapshot.ID] {
			return nil, fmt.Errorf("duplicate player id %q", playerSnapshot.ID)
		}
		seenPlayerIDs[playerSnapshot.ID] = true

		handCards, err := restoreCards(playerSnapshot.Hand)
		if err != nil {
			return nil, fmt.Errorf("restore hand for player %q: %w", playerSnapshot.ID, err)
		}
		players = append(players, &Player{
			ID:           playerSnapshot.ID,
			hand:         &Hand{cards: handCards},
			totalPoints:  playerSnapshot.TotalPoints,
			pointsGained: playerSnapshot.PointsGained,
			hasOpened:    playerSnapshot.HasOpened,
		})
	}
	if len(players) > snapshot.MaxPlayers {
		return nil, errors.New("player count exceeds max players")
	}
	if snapshot.Phase != PhaseLobby && len(players) == 0 {
		return nil, errors.New("non-lobby game requires players")
	}

	activeCompositions := make([]*Composition, 0, len(snapshot.ActiveCompositions))
	for _, compSnapshot := range snapshot.ActiveCompositions {
		cards, err := restoreCards(compSnapshot.Cards)
		if err != nil {
			return nil, fmt.Errorf("restore composition: %w", err)
		}
		comp, ok := NewComposition(cards, compositionVariant(compSnapshot.Variant))
		if !ok {
			return nil, fmt.Errorf("invalid persisted composition %q", compSnapshot.Variant)
		}
		activeCompositions = append(activeCompositions, comp)
	}

	drawPile, err := restoreCards(snapshot.DrawPile)
	if err != nil {
		return nil, fmt.Errorf("restore draw pile: %w", err)
	}
	discardPile, err := restoreCards(snapshot.DiscardPile)
	if err != nil {
		return nil, fmt.Errorf("restore discard pile: %w", err)
	}
	turn, err := restoreTurn(snapshot.Turn, len(players))
	if err != nil {
		return nil, err
	}
	if snapshot.DealerIndex < 0 || snapshot.DealerIndex >= max(1, len(players)) {
		return nil, errors.New("invalid dealer index")
	}
	if snapshot.RoundWinnerIndex < -1 || snapshot.RoundWinnerIndex >= max(1, len(players)) {
		return nil, errors.New("invalid round winner index")
	}

	return &GameState{
		players:            players,
		activeCompositions: activeCompositions,
		drawPile:           &CardPile{cards: drawPile},
		discardPile:        &CardPile{cards: discardPile},
		maxPlayers:         snapshot.MaxPlayers,
		phase:              snapshot.Phase,
		round:              snapshot.Round,
		dealerIndex:        snapshot.DealerIndex,
		turn:               turn,
		roundWinnerIndex:   snapshot.RoundWinnerIndex,
	}, nil
}

func restoreTurn(snapshot PersistenceTurnSnapshot, playerCount int) (Turn, error) {
	if snapshot.Number <= 0 {
		return Turn{}, errors.New("turn number must be positive")
	}
	if snapshot.PlayerIndex < 0 || snapshot.PlayerIndex >= max(1, playerCount) {
		return Turn{}, errors.New("invalid turn player index")
	}
	turn := Turn{
		number:             snapshot.Number,
		playerIndex:        snapshot.PlayerIndex,
		hasDrawn:           snapshot.HasDrawn,
		mustUseDiscardDraw: snapshot.MustUseDiscardDraw,
	}
	if snapshot.MustUseDiscardDraw {
		if snapshot.DiscardDrawCard == nil {
			return Turn{}, errors.New("discard draw card is required")
		}
		card, err := restoreCard(*snapshot.DiscardDrawCard)
		if err != nil {
			return Turn{}, fmt.Errorf("restore discard draw card: %w", err)
		}
		turn.discardDrawCard = card
	}
	return turn, nil
}

func restoreCards(snapshots []CardSnapshot) ([]Card, error) {
	cards := make([]Card, 0, len(snapshots))
	for _, snapshot := range snapshots {
		card, err := restoreCard(snapshot)
		if err != nil {
			return nil, err
		}
		cards = append(cards, card)
	}
	return cards, nil
}

func restoreCard(snapshot CardSnapshot) (Card, error) {
	if snapshot.IsJoker {
		return NewJoker(), nil
	}
	if snapshot.Rank < Ace || snapshot.Rank > King {
		return Card{}, errors.New("invalid card rank")
	}
	if snapshot.Suit < Hearts || snapshot.Suit > Spades {
		return Card{}, errors.New("invalid card suit")
	}
	return NewCard(snapshot.Rank, snapshot.Suit), nil
}

func cloneCards(cards []Card) []Card {
	return slices.Clone(cards)
}
