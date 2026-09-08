// Package rating implements the app's placement-based Elo rules.
package rating

import (
	"errors"
	"math"
)

type Player struct {
	Rating    int
	Placement int
}

// Calculate compares each pair using pre-game ratings. Averaging over opponents
// keeps the maximum change at K for both two- and four-player games. Equal
// placements score a draw. Round the final delta (half away from zero), then
// apply a zero floor. Rounding and the floor can introduce small rating drift.
func Calculate(players []Player) ([]int, error) {
	return Current().Calculate(players)
}

// Calculate uses a single rules snapshot for every participant in a game.
func (rules Rules) Calculate(players []Player) ([]int, error) {
	if rules.Version == "" || rules.Initial < 0 || rules.KFactor <= 0 || rules.Scale <= 0 {
		return nil, errors.New("invalid Elo rules")
	}
	if len(players) < 2 || len(players) > 4 {
		return nil, errors.New("Elo requires two to four players")
	}
	for _, p := range players {
		if p.Rating < 0 || p.Placement < 1 || p.Placement > len(players) {
			return nil, errors.New("invalid Elo player")
		}
	}
	result := make([]int, len(players))
	for i, p := range players {
		delta := 0.0
		for j, opponent := range players {
			if i == j {
				continue
			}
			actual := 0.5
			if p.Placement < opponent.Placement {
				actual = 1
			} else if p.Placement > opponent.Placement {
				actual = 0
			}
			expected := 1 / (1 + math.Pow(10, (float64(opponent.Rating)-float64(p.Rating))/float64(rules.Scale)))
			delta += actual - expected
		}
		result[i] = max(0, p.Rating+int(math.Round(float64(rules.KFactor)*delta/float64(len(players)-1))))
	}
	return result, nil
}

func Tier(value int) string {
	tiers := Tiers()
	for i := len(tiers) - 1; i >= 0; i-- {
		if value >= tiers[i].Minimum {
			return tiers[i].ID
		}
	}
	return tiers[0].ID
}
