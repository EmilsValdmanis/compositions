package game

import "sort"

func countJokers(cards []Card) int {
	count := 0
	for _, card := range cards {
		if card.isJoker {
			count++
		}
	}
	return count
}

// PlayerGameStatistics is a compact, persisted accumulator. It deliberately
// stores counters instead of an event per action so a complete game costs one
// database row per authenticated player.
type PlayerGameStatistics struct {
	RoundsPlayed                int  `json:"roundsPlayed"`
	RoundsWon                   int  `json:"roundsWon"`
	SameSuitWins                int  `json:"sameSuitWins"`
	SixPairsWins                int  `json:"sixPairsWins"`
	TurnsTaken                  int  `json:"turnsTaken"`
	CardsDrawnFromDeck          int  `json:"cardsDrawnFromDeck"`
	CardsDrawnFromDiscard       int  `json:"cardsDrawnFromDiscard"`
	CardsDiscarded              int  `json:"cardsDiscarded"`
	CardsPlayed                 int  `json:"cardsPlayed"`
	CompositionsCreated         int  `json:"compositionsCreated"`
	SetsCreated                 int  `json:"setsCreated"`
	RunsCreated                 int  `json:"runsCreated"`
	AdditionsDone               int  `json:"additionsDone"`
	CompositionsCompleted       int  `json:"compositionsCompleted"`
	SetsCompleted               int  `json:"setsCompleted"`
	RunsCompleted               int  `json:"runsCompleted"`
	JokersPlayed                int  `json:"jokersPlayed"`
	JokersReclaimed             int  `json:"jokersReclaimed"`
	CardsRemaining              int  `json:"cardsRemaining"`
	HandPoints                  int  `json:"handPoints"`
	PenaltyPoints               int  `json:"penaltyPoints"`
	PointsInflicted             int  `json:"pointsInflicted"`
	LargestRoundPenalty         int  `json:"largestRoundPenalty"`
	LargestRoundPointsInflicted int  `json:"largestRoundPointsInflicted"`
	MostCardsRemaining          int  `json:"mostCardsRemaining"`
	RoundsOpened                int  `json:"roundsOpened"`
	FastestOpeningTurn          int  `json:"fastestOpeningTurn"`
	CurrentRoundWinStreak       int  `json:"currentRoundWinStreak"`
	LongestRoundWinStreak       int  `json:"longestRoundWinStreak"`
	StartingRoundWinStreak      int  `json:"startingRoundWinStreak"`
	ForfeitOrder                int  `json:"forfeitOrder"`
	LastRecordedRound           int  `json:"lastRecordedRound,omitempty"`
	LastScoredRound             int  `json:"lastScoredRound,omitempty"`
	RoundStreakBroken           bool `json:"roundStreakBroken,omitempty"`
}

func (stats *PlayerGameStatistics) recordRoundHand(round, cards, points int) {
	if stats.LastScoredRound == round {
		return
	}
	stats.LastScoredRound = round
	stats.CardsRemaining += cards
	stats.HandPoints += points
	stats.MostCardsRemaining = max(stats.MostCardsRemaining, cards)
}

// CompletedPlayerStatistics is the final per-game record consumed by storage.
type CompletedPlayerStatistics struct {
	PlayerID    string
	Placement   int
	Winner      bool
	Forfeited   bool
	TotalPoints int
	Statistics  PlayerGameStatistics
}

// PlayerStatistics returns the current cumulative counters without requiring
// the game to have ended. It is used for compact, restart-safe checkpoints.
func (gs *GameState) PlayerStatistics() []CompletedPlayerStatistics {
	if gs == nil {
		return nil
	}
	result := make([]CompletedPlayerStatistics, 0, len(gs.players))
	for _, player := range gs.players {
		if player == nil {
			continue
		}
		result = append(result, CompletedPlayerStatistics{
			PlayerID: player.ID, Forfeited: player.forfeited,
			TotalPoints: player.totalPoints, Statistics: player.statistics,
		})
	}
	return result
}

func (gs *GameState) CompletedPlayerStatistics() []CompletedPlayerStatistics {
	if gs == nil || gs.phase != PhaseGameOver || gs.roundWinnerIndex < 0 {
		return nil
	}

	placements := make(map[int]int, len(gs.players))
	placements[gs.roundWinnerIndex] = 1
	active := make([]int, 0, len(gs.players))
	forfeited := make([]int, 0, len(gs.players))
	for i, player := range gs.players {
		if player == nil || i == gs.roundWinnerIndex {
			continue
		}
		if player.forfeited {
			forfeited = append(forfeited, i)
		} else {
			active = append(active, i)
		}
	}
	sort.SliceStable(active, func(i, j int) bool { return gs.players[active[i]].totalPoints < gs.players[active[j]].totalPoints })
	sort.SliceStable(forfeited, func(i, j int) bool {
		return gs.players[forfeited[i]].statistics.ForfeitOrder > gs.players[forfeited[j]].statistics.ForfeitOrder
	})
	previousPoints := -1
	previousPlace := 0
	for offset, i := range active {
		place := offset + 2
		if offset > 0 && gs.players[i].totalPoints == previousPoints {
			place = previousPlace
		}
		placements[i] = place
		previousPoints = gs.players[i].totalPoints
		previousPlace = place
	}
	for offset, i := range forfeited {
		placements[i] = len(active) + offset + 2
	}

	result := make([]CompletedPlayerStatistics, 0, len(gs.players))
	for i, player := range gs.players {
		if player == nil {
			continue
		}
		result = append(result, CompletedPlayerStatistics{
			PlayerID: player.ID, Placement: placements[i], Winner: i == gs.roundWinnerIndex,
			Forfeited: player.forfeited, TotalPoints: player.totalPoints, Statistics: player.statistics,
		})
	}
	return result
}

func (stats *PlayerGameStatistics) recordRoundResult(round int, won bool) {
	if stats.LastRecordedRound == round {
		return
	}
	stats.LastRecordedRound = round
	if won {
		stats.RoundsWon++
		stats.CurrentRoundWinStreak++
		if !stats.RoundStreakBroken {
			stats.StartingRoundWinStreak++
		}
		stats.LongestRoundWinStreak = max(stats.LongestRoundWinStreak, stats.CurrentRoundWinStreak)
		return
	}
	stats.CurrentRoundWinStreak = 0
	stats.RoundStreakBroken = true
}
