package database

import "testing"

func TestParseLeaderboardScope(t *testing.T) {
	tests := []struct {
		value string
		want  LeaderboardScope
		ok    bool
	}{
		{"", LeaderboardScopeFriends, true},
		{" friends ", LeaderboardScopeFriends, true},
		{"global", LeaderboardScopeGlobal, true},
		{"everyone", "", false},
	}
	for _, test := range tests {
		got, ok := ParseLeaderboardScope(test.value)
		if got != test.want || ok != test.ok {
			t.Errorf("ParseLeaderboardScope(%q) = (%q, %t); want (%q, %t)", test.value, got, ok, test.want, test.ok)
		}
	}
}

func TestParseLeaderboardMetric(t *testing.T) {
	for _, value := range []string{"", " elo ", "elo", "wins", "games", "playtime", "rounds", "points"} {
		got, ok := ParseLeaderboardMetric(value)
		if !ok {
			t.Fatalf("rejected %q", value)
		}
		if value == "" || value == " elo " {
			if got != LeaderboardMetricElo {
				t.Fatalf("default = %q", got)
			}
		}
		if _, err := got.scoreExpression(); err != nil {
			t.Fatal(err)
		}
	}
	for _, value := range []string{"rating", "ELO", "elo; DROP TABLE users"} {
		if _, ok := ParseLeaderboardMetric(value); ok {
			t.Fatalf("accepted %q", value)
		}
	}
	if _, err := LeaderboardMetric("invalid").scoreExpression(); err == nil {
		t.Fatal("accepted invalid SQL metric")
	}
}
