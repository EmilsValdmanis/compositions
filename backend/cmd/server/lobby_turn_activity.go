package main

import (
	"fmt"
	"maps"

	"github.com/EmilsValdmanis/compositions/internal/game"
)

func (r *room) resetTurnTracking(playerID string) {
	if r == nil || r.gameState == nil {
		return
	}
	baseline, ok := r.gameState.SnapshotForPlayer(playerID)
	if !ok {
		return
	}
	r.turnBaseline = &baseline
	r.turnActivity = &game.TurnActivitySnapshot{
		PlayerID:   playerID,
		Round:      r.gameState.RoundNumber(),
		TurnNumber: r.gameState.TurnNumber(),
	}
}

func (r *room) clearTurnTracking() {
	if r == nil {
		return
	}
	r.turnBaseline = nil
	r.turnActivity = nil
}

func (r *room) turnActivityForPlayer(playerID string) *game.TurnActivitySnapshot {
	if r.turnBaseline == nil || r.turnActivity == nil || r.turnActivity.PlayerID == playerID {
		return nil
	}
	activity := *r.turnActivity
	activity.BaselineCompositions = cloneCompositionSnapshots(r.turnBaseline.ActiveCompositions)
	activity.DraftCompositions = cloneDraftCompositionSnapshots(r.turnActivity.DraftCompositions)
	activity.CompositionActivities = cloneCompositionActivitySnapshots(r.turnActivity.CompositionActivities)
	return &activity
}

func (r *room) applySubmittedTurnActivity(playerID string, comps []*game.Composition, additions []game.CompositionAddition, reclaims []game.JokerReclaim) {
	if r == nil || r.turnActivity == nil || r.turnBaseline == nil {
		return
	}
	if r.turnActivity.PlayerID != playerID {
		return
	}
	r.turnActivity.DraftCompositions = buildDraftSnapshotsFromSubmitted(comps, additions)
	mergeCompositionActivities(r.turnActivity, buildCompositionActivities(playerID, comps, additions, reclaims, r.gameState.ActiveCompositions()))
}

func buildDraftSnapshotsFromSubmitted(comps []*game.Composition, additions []game.CompositionAddition) []game.DraftCompositionSnapshot {
	drafts := make([]game.DraftCompositionSnapshot, 0, len(comps)+len(additions))
	for index, comp := range comps {
		if comp == nil {
			continue
		}
		snapshot := comp.Snapshot()
		cards := make([]game.CardSnapshot, len(snapshot.Cards))
		copy(cards, snapshot.Cards)
		drafts = append(drafts, game.DraftCompositionSnapshot{
			ID:    fmt.Sprintf("submitted-new-%d", index),
			Cards: cards,
		})
	}
	for _, addition := range additions {
		cards := make([]game.CardSnapshot, 0, len(addition.Cards))
		for _, card := range addition.Cards {
			cards = append(cards, card.Snapshot())
		}
		index := addition.CompositionIndex
		drafts = append(drafts, game.DraftCompositionSnapshot{
			ID:          fmt.Sprintf("submitted-table-%d", index),
			TableIndex:  &index,
			InsertIndex: addition.InsertIndex,
			Cards:       cards,
		})
	}
	return drafts
}

func mergeCompositionActivities(target *game.TurnActivitySnapshot, updates []game.CompositionActivitySnapshot) {
	if target == nil || len(updates) == 0 {
		return
	}
	merged := cloneCompositionActivitySnapshots(target.CompositionActivities)
	byIndex := make(map[int]int, len(merged))
	for i := range merged {
		byIndex[merged[i].TableIndex] = i
	}
	for _, update := range updates {
		if existingIndex, ok := byIndex[update.TableIndex]; ok {
			existing := &merged[existingIndex]
			if existing.Kind == "" {
				existing.Kind = update.Kind
			}
			if existing.PlayerID == "" {
				existing.PlayerID = update.PlayerID
			}
			if len(update.CardActivities) > 0 {
				if existing.CardActivities == nil {
					existing.CardActivities = make(map[int]game.CardActivitySnapshot, len(update.CardActivities))
				}
				maps.Copy(existing.CardActivities, update.CardActivities)
			}
			continue
		}
		merged = append(merged, update)
		byIndex[update.TableIndex] = len(merged) - 1
	}
	target.CompositionActivities = merged
}

func buildCompositionActivities(playerID string, comps []*game.Composition, additions []game.CompositionAddition, reclaims []game.JokerReclaim, active []*game.Composition) []game.CompositionActivitySnapshot {
	activities := make([]game.CompositionActivitySnapshot, 0, len(comps)+len(additions)+len(reclaims))
	newCount := len(active) - len(comps)
	newCount = max(newCount, 0)
	for offset, comp := range comps {
		if comp == nil {
			continue
		}
		compSnapshot := comp.Snapshot()
		tableIndex := newCount + offset
		cardActivities := make(map[int]game.CardActivitySnapshot, len(compSnapshot.Cards))
		for index := range compSnapshot.Cards {
			cardActivities[index] = game.CardActivitySnapshot{Kind: "new", PlayerID: playerID}
		}
		activities = append(activities, game.CompositionActivitySnapshot{
			TableIndex:     tableIndex,
			Kind:           "new_composition",
			PlayerID:       playerID,
			CardActivities: cardActivities,
		})
	}

	activityByIndex := make(map[int]int)
	for i := range activities {
		activityByIndex[activities[i].TableIndex] = i
	}
	for _, addition := range additions {
		activityIndex, ok := activityByIndex[addition.CompositionIndex]
		if !ok {
			activities = append(activities, game.CompositionActivitySnapshot{TableIndex: addition.CompositionIndex})
			activityIndex = len(activities) - 1
			activityByIndex[addition.CompositionIndex] = activityIndex
		}
		activity := &activities[activityIndex]
		if activity.CardActivities == nil {
			activity.CardActivities = map[int]game.CardActivitySnapshot{}
		}
		startIndex := 0
		if addition.InsertIndex != nil {
			startIndex = *addition.InsertIndex
		}
		if startIndex < 0 {
			startIndex = 0
		}
		if addition.CompositionIndex >= 0 && addition.CompositionIndex < len(active) && active[addition.CompositionIndex] != nil {
			maxStartIndex := len(active[addition.CompositionIndex].Snapshot().Cards)
			if startIndex > maxStartIndex {
				startIndex = maxStartIndex
			}
		}
		for offset := range addition.Cards {
			activity.CardActivities[startIndex+offset] = game.CardActivitySnapshot{Kind: "addition", PlayerID: playerID}
		}
	}
	for _, reclaim := range reclaims {
		activityIndex, ok := activityByIndex[reclaim.CompositionIndex]
		if !ok {
			activities = append(activities, game.CompositionActivitySnapshot{TableIndex: reclaim.CompositionIndex})
			activityIndex = len(activities) - 1
			activityByIndex[reclaim.CompositionIndex] = activityIndex
		}
		activity := &activities[activityIndex]
		if activity.CardActivities == nil {
			activity.CardActivities = map[int]game.CardActivitySnapshot{}
		}
		activity.CardActivities[reclaim.JokerIndex] = game.CardActivitySnapshot{Kind: "joker_reclaim", PlayerID: playerID}
	}
	return activities
}

func cloneCompositionSnapshots(source []game.CompositionSnapshot) []game.CompositionSnapshot {
	if len(source) == 0 {
		return nil
	}
	cloned := make([]game.CompositionSnapshot, 0, len(source))
	for _, comp := range source {
		next := comp
		next.Cards = append([]game.CardSnapshot(nil), comp.Cards...)
		if len(comp.JokerRepresentations) > 0 {
			next.JokerRepresentations = make(map[int][]game.CardSnapshot, len(comp.JokerRepresentations))
			for index, cards := range comp.JokerRepresentations {
				next.JokerRepresentations[index] = append([]game.CardSnapshot(nil), cards...)
			}
		}
		cloned = append(cloned, next)
	}
	return cloned
}

func cloneDraftCompositionSnapshots(source []game.DraftCompositionSnapshot) []game.DraftCompositionSnapshot {
	if len(source) == 0 {
		return nil
	}
	cloned := make([]game.DraftCompositionSnapshot, 0, len(source))
	for _, draft := range source {
		next := draft
		next.Cards = append([]game.CardSnapshot(nil), draft.Cards...)
		if draft.TableIndex != nil {
			index := *draft.TableIndex
			next.TableIndex = &index
		}
		if draft.InsertIndex != nil {
			index := *draft.InsertIndex
			next.InsertIndex = &index
		}
		if len(draft.CardPlacements) > 0 {
			next.CardPlacements = make([]game.DraftCardPlacementSnapshot, len(draft.CardPlacements))
			for index, placement := range draft.CardPlacements {
				next.CardPlacements[index] = game.DraftCardPlacementSnapshot{
					InsertIndex:       cloneIntPointer(placement.InsertIndex),
					ReclaimJokerIndex: cloneIntPointer(placement.ReclaimJokerIndex),
				}
			}
		}
		cloned = append(cloned, next)
	}
	return cloned
}

func cloneCompositionActivitySnapshots(source []game.CompositionActivitySnapshot) []game.CompositionActivitySnapshot {
	if len(source) == 0 {
		return nil
	}
	cloned := make([]game.CompositionActivitySnapshot, 0, len(source))
	for _, activity := range source {
		next := activity
		if len(activity.CardActivities) > 0 {
			next.CardActivities = maps.Clone(activity.CardActivities)
		}
		cloned = append(cloned, next)
	}
	return cloned
}

func cloneGameSnapshot(source *game.GameSnapshot) *game.GameSnapshot {
	if source == nil {
		return nil
	}
	cloned := *source
	cloned.Players = append([]game.PlayerStateSnapshot(nil), source.Players...)
	cloned.Hand = append([]game.CardSnapshot(nil), source.Hand...)
	cloned.DiscardPile = append([]game.CardSnapshot(nil), source.DiscardPile...)
	cloned.ActiveCompositions = cloneCompositionSnapshots(source.ActiveCompositions)
	cloned.TurnActivity = cloneTurnActivitySnapshot(source.TurnActivity)
	return &cloned
}

func cloneTurnActivitySnapshot(source *game.TurnActivitySnapshot) *game.TurnActivitySnapshot {
	if source == nil {
		return nil
	}
	cloned := *source
	cloned.BaselineCompositions = cloneCompositionSnapshots(source.BaselineCompositions)
	cloned.DraftCompositions = cloneDraftCompositionSnapshots(source.DraftCompositions)
	cloned.CompositionActivities = cloneCompositionActivitySnapshots(source.CompositionActivities)
	return &cloned
}
