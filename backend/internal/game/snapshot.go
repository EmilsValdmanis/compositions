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

type CardActivitySnapshot struct {
	Kind     string `json:"kind"`
	PlayerID string `json:"playerId"`
}

type CompositionActivitySnapshot struct {
	TableIndex     int                          `json:"tableIndex"`
	Kind           string                       `json:"kind,omitempty"`
	PlayerID       string                       `json:"playerId,omitempty"`
	CardActivities map[int]CardActivitySnapshot `json:"cardActivities,omitempty"`
}

type DraftCardPlacementSnapshot struct {
	InsertIndex       *int `json:"insertIndex,omitempty"`
	ReclaimJokerIndex *int `json:"reclaimJokerIndex,omitempty"`
}

type DraftCompositionSnapshot struct {
	TableIndex     *int                         `json:"tableIndex,omitempty"`
	InsertIndex    *int                         `json:"insertIndex,omitempty"`
	CardPlacements []DraftCardPlacementSnapshot `json:"cardPlacements,omitempty"`
	Cards          []CardSnapshot               `json:"cards"`
}

type TurnActivitySnapshot struct {
	PlayerID              string                        `json:"playerId"`
	Round                 int                           `json:"round"`
	TurnNumber            int                           `json:"turnNumber"`
	DrawSource            string                        `json:"drawSource,omitempty"`
	BaselineCompositions  []CompositionSnapshot         `json:"baselineCompositions,omitempty"`
	DraftCompositions     []DraftCompositionSnapshot    `json:"draftCompositions,omitempty"`
	CompositionActivities []CompositionActivitySnapshot `json:"compositionActivities,omitempty"`
}

type PlayerStateSnapshot struct {
	PlayerID     string         `json:"playerId"`
	HandCount    int            `json:"handCount"`
	Hand         []CardSnapshot `json:"hand,omitempty"`
	TotalPoints  int            `json:"totalPoints"`
	PointsGained int            `json:"pointsGained"`
	HasOpened    bool           `json:"hasOpened"`
	Forfeited    bool           `json:"forfeited,omitempty"`
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
	TurnActivity       *TurnActivitySnapshot `json:"turnActivity,omitempty"`
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
		Players:            playerStateSnapshots(gs.players, gs.phase == PhaseRoundOver || gs.phase == PhaseGameOver),
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

func (gs *GameState) RoundNumber() int {
	if gs == nil {
		return 0
	}
	return gs.round
}

func (gs *GameState) TurnNumber() int {
	if gs == nil {
		return 0
	}
	return gs.turn.number
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

func playerStateSnapshots(players []*Player, revealHands bool) []PlayerStateSnapshot {
	snapshots := make([]PlayerStateSnapshot, 0, len(players))
	for _, player := range players {
		if player == nil {
			continue
		}
		snapshot := PlayerStateSnapshot{
			PlayerID:     player.ID,
			HandCount:    len(player.hand.cards),
			TotalPoints:  player.totalPoints,
			PointsGained: player.pointsGained,
			HasOpened:    player.hasOpened,
			Forfeited:    player.forfeited,
		}
		if revealHands {
			snapshot.Hand = cardSnapshots(player.hand.cards)
		}
		snapshots = append(snapshots, snapshot)
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

func (c *Composition) Snapshot() CompositionSnapshot {
	if c == nil {
		return CompositionSnapshot{}
	}
	return c.snapshot()
}

func (gs *GameState) ActiveCompositions() []*Composition {
	if gs == nil {
		return nil
	}
	return gs.activeCompositions
}
