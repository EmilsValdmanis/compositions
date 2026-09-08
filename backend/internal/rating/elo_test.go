package rating

import (
	"math/rand"
	"reflect"
	"testing"
)

func TestCalculate(t *testing.T) {
	for _, tc := range []struct {
		name    string
		players []Player
		want    []int
	}{
		{"equal duel", []Player{{1000, 1}, {1000, 2}}, []int{1016, 984}},
		{"favorite wins", []Player{{1400, 1}, {1000, 2}}, []int{1403, 997}},
		{"upset", []Player{{1000, 1}, {1400, 2}}, []int{1029, 1371}},
		{"draw", []Player{{1000, 1}, {1000, 1}}, []int{1000, 1000}},
		{"unequal draw", []Player{{1400, 1}, {1000, 1}}, []int{1387, 1013}},
		{"three players", []Player{{1000, 1}, {1000, 2}, {1000, 3}}, []int{1016, 1000, 984}},
		{"four players", []Player{{1000, 1}, {1000, 2}, {1000, 3}, {1000, 4}}, []int{1016, 1005, 995, 984}},
		{"tied runners up", []Player{{1000, 1}, {1000, 2}, {1000, 2}, {1000, 4}}, []int{1016, 1000, 1000, 984}},
		{"floor", []Player{{0, 2}, {0, 1}}, []int{0, 16}},
		{"extreme upset", []Player{{0, 1}, {100000, 2}}, []int{32, 99968}},
		{"extreme favorite", []Player{{100000, 1}, {0, 2}}, []int{100000, 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original := append([]Player(nil), tc.players...)
			got, err := Calculate(tc.players)
			if err != nil || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v, %v; want %v", got, err, tc.want)
			}
			if !reflect.DeepEqual(original, tc.players) {
				t.Fatal("mutated inputs")
			}
		})
	}
}

func TestCalculateInvalid(t *testing.T) {
	for _, players := range [][]Player{nil, {{1000, 1}}, make([]Player, 5), {{-1, 1}, {1000, 2}}, {{1000, 0}, {1000, 2}}, {{1000, 1}, {1000, 3}}} {
		if _, err := Calculate(players); err == nil {
			t.Fatalf("accepted %v", players)
		}
	}
}

func TestCustomRules(t *testing.T) {
	rules := Current()
	rules.Version = "elo-test-v2"
	rules.Initial = 1500
	rules.KFactor = 64
	rules.Scale = 800
	got, err := rules.Calculate([]Player{{1500, 1}, {2300, 2}})
	if err != nil || !reflect.DeepEqual(got, []int{1558, 2242}) {
		t.Fatalf("custom rules: got %v, %v", got, err)
	}
	// Tuning a copy must not alter the active rules or the v1 regression cases.
	if Current().Version != "elo-v1" || Current().Initial != 1000 || Current().KFactor != 32 || Current().Scale != 400 {
		t.Fatal("custom rules mutated active rules")
	}
}

func TestInvalidRules(t *testing.T) {
	for _, change := range []func(*Rules){
		func(r *Rules) { r.Version = "" },
		func(r *Rules) { r.Initial = -1 },
		func(r *Rules) { r.KFactor = 0 },
		func(r *Rules) { r.Scale = 0 },
	} {
		rules := Current()
		change(&rules)
		if _, err := rules.Calculate([]Player{{1000, 1}, {1000, 2}}); err == nil {
			t.Fatalf("accepted invalid rules: %+v", rules)
		}
	}
}

func TestTierConfiguration(t *testing.T) {
	tiers := Tiers()
	seen := make(map[string]bool)
	for i, tier := range tiers {
		if tier.ID == "" || seen[tier.ID] || (i > 0 && tier.Minimum <= tiers[i-1].Minimum) {
			t.Fatalf("invalid tier configuration: %+v", tiers)
		}
		seen[tier.ID] = true
	}
	if tiers[0].Minimum != 0 {
		t.Fatal("tiers must cover the rating floor")
	}
}

func TestTierBoundaries(t *testing.T) {
	for _, tc := range []struct {
		value int
		want  string
	}{
		{0, "bronze"}, {1000, "bronze"}, {1199, "bronze"}, {1200, "silver"}, {1399, "silver"},
		{1400, "gold"}, {1599, "gold"}, {1600, "platinum"}, {1799, "platinum"},
		{1800, "diamond"}, {1999, "diamond"}, {2000, "master"}, {2199, "master"}, {2200, "grandmaster"}, {99999, "grandmaster"},
	} {
		if got := Tier(tc.value); got != tc.want {
			t.Errorf("Tier(%d) = %s, want %s", tc.value, got, tc.want)
		}
	}
}

func TestCalculateProperties(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for range 10000 {
		n := 2 + rng.Intn(3)
		players := make([]Player, n)
		for i := range players {
			players[i] = Player{rng.Intn(4001), 1 + rng.Intn(n)}
		}
		got, err := Calculate(players)
		if err != nil {
			t.Fatal(err)
		}
		// Reversing seats must not change anyone's rating.
		reversed := make([]Player, n)
		for i := range players {
			reversed[n-1-i] = players[i]
		}
		reverseGot, _ := Calculate(reversed)
		for i, p := range players {
			if got[i] < 0 || got[i]-p.Rating > Current().KFactor || p.Rating-got[i] > Current().KFactor {
				t.Fatalf("unbounded change: %v -> %v", players, got)
			}
			if got[i] != reverseGot[n-1-i] {
				t.Fatalf("seat dependent: %v -> %v / %v", players, got, reverseGot)
			}
			improved := append([]Player(nil), players...)
			improved[i].Placement = 1
			better, _ := Calculate(improved)
			if better[i] < got[i] {
				t.Fatal("better placement lowered rating")
			}
		}
	}
}

func FuzzCalculate(f *testing.F) {
	f.Add(uint16(1000), uint16(1400), uint8(4), uint8(2))
	f.Fuzz(func(t *testing.T, a, b uint16, size, place uint8) {
		n := 2 + int(size%3)
		players := make([]Player, n)
		for i := range players {
			players[i] = Player{int(b), 1 + i}
		}
		players[0] = Player{int(a), 1 + int(place)%n}
		result, err := Calculate(players)
		if err != nil {
			t.Fatal(err)
		}
		for i, p := range players {
			if result[i] < 0 || result[i] > p.Rating+Current().KFactor || result[i] < p.Rating-Current().KFactor {
				t.Fatalf("invalid %v -> %v", players, result)
			}
		}
	})
}
