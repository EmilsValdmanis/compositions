package database

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNormalizeGameBugReport(t *testing.T) {
	report, err := normalizeGameBugReport(GameBugReportRecord{
		ID:               " report-id ",
		RoomCode:         " abc123 ",
		ReporterPlayerID: " player-1 ",
		ReporterUserID:   " user-1 ",
		Description:      " Something broke ",
		GameState:        json.RawMessage(`{"phase":1}`),
		Round:            2,
		Turn:             8,
		CreatedAt:        time.Date(2026, 7, 10, 9, 0, 0, 0, time.FixedZone("test", 2*60*60)),
	})
	if err != nil {
		t.Fatalf("normalizeGameBugReport() error = %v", err)
	}
	if report.RoomCode != "ABC123" || report.ReporterPlayerID != "player-1" || report.Description != "Something broke" {
		t.Fatalf("normalized report = %#v", report)
	}
	if report.CreatedAt.Location() != time.UTC {
		t.Fatalf("created-at location = %v; want UTC", report.CreatedAt.Location())
	}
}

func TestNormalizeGameBugReportRejectsInvalidFields(t *testing.T) {
	valid := GameBugReportRecord{
		ID:               "report-id",
		RoomCode:         "ABC123",
		ReporterPlayerID: "player-1",
		Description:      "Something broke",
		GameState:        json.RawMessage(`{"phase":1}`),
		Round:            1,
		Turn:             1,
	}
	tests := []struct {
		name   string
		mutate func(*GameBugReportRecord)
	}{
		{"id", func(report *GameBugReportRecord) { report.ID = "" }},
		{"room", func(report *GameBugReportRecord) { report.RoomCode = "" }},
		{"reporter", func(report *GameBugReportRecord) { report.ReporterPlayerID = "" }},
		{"description", func(report *GameBugReportRecord) { report.Description = "" }},
		{"description length", func(report *GameBugReportRecord) { report.Description = strings.Repeat("x", 501) }},
		{"game state", func(report *GameBugReportRecord) { report.GameState = json.RawMessage(`{`) }},
		{"round", func(report *GameBugReportRecord) { report.Round = 0 }},
		{"turn", func(report *GameBugReportRecord) { report.Turn = 0 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			report := valid
			test.mutate(&report)
			if _, err := normalizeGameBugReport(report); err == nil {
				t.Fatal("normalizeGameBugReport() error = nil; want error")
			}
		})
	}
}
