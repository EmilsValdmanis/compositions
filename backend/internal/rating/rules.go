package rating

// Rules contains the tunable inputs to the placement-based Elo algorithm.
// Bump Version whenever calculation behavior or these values change. Persisted
// game changes retain their version; deploying new rules does not replay games.
type Rules struct {
	Version string
	Initial int
	KFactor int
	Scale   int
}

// Current returns a copy so callers cannot mutate the process-wide rules.
// Starting ratings, leaderboard fallbacks and game calculations all use this.
func Current() Rules {
	return Rules{Version: "elo-v1", Initial: 1000, KFactor: 32, Scale: 400}
}

type RankTier struct {
	ID      string
	Minimum int
}

// Tiers is ordered by ascending minimum. Thresholds are presentation rules,
// independent of the calculation version, and are applied to current ratings.
func Tiers() []RankTier {
	return []RankTier{
		{ID: "bronze", Minimum: 0},
		{ID: "silver", Minimum: 1200},
		{ID: "gold", Minimum: 1400},
		{ID: "platinum", Minimum: 1600},
		{ID: "diamond", Minimum: 1800},
		{ID: "master", Minimum: 2000},
		{ID: "grandmaster", Minimum: 2200},
	}
}
