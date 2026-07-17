package database

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func validCheckpointRecord() GameCheckpointRecord {
	return GameCheckpointRecord{
		ID: uuid.NewString(), RoomCode: "ABC123", RoundsPlayed: 1, PlayerCount: 2,
		StartedAt: time.Now().UTC(),
		Players:   []CompletedGamePlayerRecord{{UserID: uuid.NewString(), RoundsPlayed: 1}},
	}
}

func TestValidateCheckpoint(t *testing.T) {
	if _, err := validateCheckpoint(validCheckpointRecord()); err != nil {
		t.Fatalf("validateCheckpoint(valid) error = %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*GameCheckpointRecord)
	}{
		{"missing identity", func(record *GameCheckpointRecord) { record.ID = "" }},
		{"incomplete counts", func(record *GameCheckpointRecord) { record.RoundsPlayed = 0 }},
		{"too many players", func(record *GameCheckpointRecord) { record.PlayerCount = 5 }},
		{"zero start time", func(record *GameCheckpointRecord) { record.StartedAt = time.Time{} }},
		{"negative playtime", func(record *GameCheckpointRecord) { record.PlaytimeSeconds = -1 }},
		{"invalid game uuid", func(record *GameCheckpointRecord) { record.ID = "bad" }},
		{"invalid game mode", func(record *GameCheckpointRecord) { record.GameMode = "arcade" }},
		{"invalid user uuid", func(record *GameCheckpointRecord) { record.Players[0].UserID = "bad" }},
		{"duplicate user", func(record *GameCheckpointRecord) { record.Players = append(record.Players, record.Players[0]) }},
		{"negative statistic", func(record *GameCheckpointRecord) { record.Players[0].TurnsTaken = -1 }},
		{"oversized statistic", func(record *GameCheckpointRecord) { record.Players[0].TurnsTaken = int(maxStoredStatistic + 1) }},
		{"round wins", func(record *GameCheckpointRecord) { record.Players[0].RoundsWon = 2 }},
		{"player rounds exceed game", func(record *GameCheckpointRecord) { record.Players[0].RoundsPlayed = 2 }},
		{"round openings", func(record *GameCheckpointRecord) { record.Players[0].RoundsOpened = 2 }},
		{"same suit wins", func(record *GameCheckpointRecord) { record.Players[0].SameSuitWins = 1 }},
		{"six pairs wins", func(record *GameCheckpointRecord) { record.Players[0].SixPairsWins = 1 }},
		{"starting streak", func(record *GameCheckpointRecord) { record.Players[0].StartingRoundWinStreak = 1 }},
		{"ending streak", func(record *GameCheckpointRecord) { record.Players[0].EndingRoundWinStreak = 1 }},
		{"longest streak", func(record *GameCheckpointRecord) { record.Players[0].LongestRoundWinStreak = 1 }},
		{"opening speed", func(record *GameCheckpointRecord) { record.Players[0].FastestOpeningTurn = 1 }},
		{"missing opening speed", func(record *GameCheckpointRecord) { record.Players[0].RoundsOpened = 1 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			record := validCheckpointRecord()
			test.mutate(&record)
			if _, err := validateCheckpoint(record); err == nil {
				t.Fatal("validateCheckpoint() error = nil")
			}
		})
	}
}

func TestPlaceholdersAndBoolInt(t *testing.T) {
	if got := placeholders(3); got != "$1, $2, $3" {
		t.Fatalf("placeholders(3) = %q", got)
	}
	if boolInt(false) != 0 || boolInt(true) != 1 {
		t.Fatal("boolInt() returned unexpected values")
	}
	if strings.TrimSpace(placeholders(0)) != "" {
		t.Fatal("placeholders(0) should be empty")
	}
}
