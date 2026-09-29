package game

import (
	"maps"
	"slices"
)

// Clone copies mutable game data for a command transaction. It does not run
// restore validation: an in-memory clone must also preserve intermediate states.
func (gs *GameState) Clone() *GameState {
	if gs == nil {
		return nil
	}
	next := *gs
	next.players = slices.Clone(gs.players)
	for i, player := range next.players {
		if player != nil {
			copy := *player
			if player.hand != nil {
				hand := *player.hand
				hand.cards = slices.Clone(player.hand.cards)
				copy.hand = &hand
			}
			next.players[i] = &copy
		}
	}
	next.activeCompositions = slices.Clone(gs.activeCompositions)
	for i, comp := range next.activeCompositions {
		if comp != nil {
			copy := *comp
			copy.cards = slices.Clone(comp.cards)
			copy.jokerRepresentations = maps.Clone(comp.jokerRepresentations)
			for index, cards := range comp.jokerRepresentations {
				copy.jokerRepresentations[index] = slices.Clone(cards)
			}
			next.activeCompositions[i] = &copy
		}
	}
	if gs.drawPile != nil {
		next.drawPile = &CardPile{cards: slices.Clone(gs.drawPile.cards)}
	}
	if gs.discardPile != nil {
		next.discardPile = &CardPile{cards: slices.Clone(gs.discardPile.cards)}
	}
	return &next
}
