package game

type CardSnapshot struct {
	Rank    Rank `json:"rank,omitempty"`
	Suit    Suit `json:"suit"`
	IsJoker bool `json:"isJoker,omitempty"`
}

type CompositionSnapshot struct {
	Type                 string                 `json:"type"`
	Cards                []CardSnapshot         `json:"cards"`
	JokerRepresentations map[int][]CardSnapshot `json:"jokerRepresentations,omitempty"`
	Points               int                    `json:"points"`
	Complete             bool                   `json:"complete"`
}

type PlayerStateSnapshot struct {
	PlayerID    string `json:"playerId"`
	HandCount   int    `json:"handCount"`
	TotalPoints int    `json:"totalPoints"`
	HasOpened   bool   `json:"hasOpened"`
}

type TurnSnapshot struct {
	Number             int    `json:"number"`
	PlayerIndex        int    `json:"playerIndex"`
	PlayerID           string `json:"playerId,omitempty"`
	HasDrawn           bool   `json:"hasDrawn"`
	MustUseDiscardDraw bool   `json:"mustUseDiscardDraw"`
}

type GameSnapshot struct {
	Phase              GamePhase             `json:"phase"`
	Round              int                   `json:"round"`
	DealerIndex        int                   `json:"dealerIndex"`
	RoundWinnerIndex   int                   `json:"roundWinnerIndex"`
	Turn               TurnSnapshot          `json:"turn"`
	Players            []PlayerStateSnapshot `json:"players"`
	Hand               []CardSnapshot        `json:"hand"`
	DrawPileCount      int                   `json:"drawPileCount"`
	DiscardPile        []CardSnapshot        `json:"discardPile"`
	ActiveCompositions []CompositionSnapshot `json:"activeCompositions"`
}

func (gs *GameState) SnapshotForPlayer(playerID string) (GameSnapshot, bool) {
	if gs == nil {
		return GameSnapshot{}, false
	}

	snapshot := GameSnapshot{
		Phase:              gs.phase,
		Round:              gs.round,
		DealerIndex:        gs.dealerIndex,
		RoundWinnerIndex:   gs.roundWinnerIndex,
		Turn:               gs.turn.snapshot(gs.players),
		Players:            playerStateSnapshots(gs.players),
		DrawPileCount:      len(gs.drawPile.cards),
		DiscardPile:        cardSnapshots(gs.discardPile.cards),
		ActiveCompositions: compositionSnapshots(gs.activeCompositions),
	}

	for _, player := range gs.players {
		if player == nil || player.ID != playerID {
			continue
		}
		snapshot.Hand = cardSnapshots(player.hand.cards)
		return snapshot, true
	}

	return GameSnapshot{}, false
}

func (gs *GameState) CurrentPlayerIndex() int {
	if gs == nil {
		return 0
	}
	return gs.turn.playerIndex
}

func cardSnapshots(cards []Card) []CardSnapshot {
	snapshots := make([]CardSnapshot, 0, len(cards))
	for _, card := range cards {
		snapshots = append(snapshots, card.Snapshot())
	}
	return snapshots
}

func (c Card) Snapshot() CardSnapshot {
	return CardSnapshot{Rank: c.rank, Suit: c.suit, IsJoker: c.isJoker}
}

func (t Turn) snapshot(players []*Player) TurnSnapshot {
	snapshot := TurnSnapshot{
		Number:             t.number,
		PlayerIndex:        t.playerIndex,
		HasDrawn:           t.hasDrawn,
		MustUseDiscardDraw: t.mustUseDiscardDraw,
	}
	if isValidPlayerIndex(t.playerIndex, len(players)) && players[t.playerIndex] != nil {
		snapshot.PlayerID = players[t.playerIndex].ID
	}
	return snapshot
}

func playerStateSnapshots(players []*Player) []PlayerStateSnapshot {
	snapshots := make([]PlayerStateSnapshot, 0, len(players))
	for _, player := range players {
		if player == nil {
			continue
		}
		snapshots = append(snapshots, PlayerStateSnapshot{
			PlayerID:    player.ID,
			HandCount:   len(player.hand.cards),
			TotalPoints: player.totalPoints,
			HasOpened:   player.hasOpened,
		})
	}
	return snapshots
}

func compositionSnapshots(comps []*Composition) []CompositionSnapshot {
	snapshots := make([]CompositionSnapshot, 0, len(comps))
	for _, comp := range comps {
		if comp == nil {
			continue
		}
		snapshots = append(snapshots, comp.snapshot())
	}
	return snapshots
}

func (c *Composition) snapshot() CompositionSnapshot {
	representations := make(map[int][]CardSnapshot, len(c.jokerRepresentations))
	for index, cards := range c.jokerRepresentations {
		representations[index] = cardSnapshots(cards)
	}
	if len(representations) == 0 {
		representations = nil
	}

	return CompositionSnapshot{
		Type:                 string(c.variant),
		Cards:                cardSnapshots(c.cards),
		JokerRepresentations: representations,
		Points:               c.Points(),
		Complete:             c.isComplete(),
	}
}
