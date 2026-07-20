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
