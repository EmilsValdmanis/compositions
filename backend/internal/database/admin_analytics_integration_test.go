//go:build integration

package database

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestUserStoreAdminAnalytics(t *testing.T) {
	ctx := context.Background()
	databaseURL := startPostgresContainer(t, ctx)

	if err := RunMigrations(ctx, databaseURL, MigrationUp); err != nil {
		t.Fatalf("RunMigrations(up) error = %v", err)
	}

	store, err := NewUserStore(ctx, databaseURL)
	if err != nil {
		t.Fatalf("NewUserStore() error = %v", err)
	}
	defer store.Close()

	users := []UserRecord{
		{ID: uuid.NewString(), Name: "Returning Player", Email: "returning@example.com"},
		{ID: uuid.NewString(), Name: "New Player", Email: "new@example.com"},
		{ID: uuid.NewString(), Name: "Previous Opponent", Email: "previous@example.com"},
	}
	for _, user := range users {
		if _, err := store.UpsertUser(ctx, user); err != nil {
			t.Fatalf("UpsertUser(%q) error = %v", user.Name, err)
		}
	}

	riga, err := time.LoadLocation(adminAnalyticsTimezone)
	if err != nil {
		t.Fatalf("LoadLocation(%q) error = %v", adminAnalyticsTimezone, err)
	}
	localTime := func(day, hour int) time.Time {
		return time.Date(2026, time.July, day, hour, 0, 0, 0, riga)
	}

	previousGame := CompletedGameRecord{
		ID:              uuid.NewString(),
		RoomCode:        "PREVIOUS",
		CompletionKind:  "normal",
		RoundsPlayed:    1,
		PlayerCount:     2,
		StartedAt:       localTime(0, 10),
		CompletedAt:     localTime(0, 10).Add(20 * time.Minute),
		PlaytimeSeconds: 1200,
		Players: []CompletedGamePlayerRecord{
			analyticsTestPlayer(users[0].ID, 1, true),
			analyticsTestPlayer(users[2].ID, 2, false),
		},
	}
	if err := store.SaveCompletedGame(ctx, previousGame); err != nil {
		t.Fatalf("SaveCompletedGame(previous) error = %v", err)
	}

	currentGame := CompletedGameRecord{
		ID:              uuid.NewString(),
		RoomCode:        "CURRENT",
		CompletionKind:  "forfeit",
		RoundsPlayed:    1,
		PlayerCount:     2,
		StartedAt:       localTime(1, 10),
		CompletedAt:     localTime(1, 10).Add(30 * time.Minute),
		PlaytimeSeconds: 1800,
		Players: []CompletedGamePlayerRecord{
			analyticsTestPlayer(users[0].ID, 1, true),
			analyticsTestPlayer(users[1].ID, 2, false),
		},
	}
	currentGame.Players[1].Forfeited = true
	if err := store.SaveCompletedGame(ctx, currentGame); err != nil {
		t.Fatalf("SaveCompletedGame(current) error = %v", err)
	}

	abortedGame := GameCheckpointRecord{
		ID:              uuid.NewString(),
		RoomCode:        "ABORTED",
		RoundsPlayed:    1,
		PlayerCount:     2,
		StartedAt:       localTime(2, 10),
		PlaytimeSeconds: 600,
		Players: []CompletedGamePlayerRecord{
			analyticsTestPlayer(users[0].ID, 0, false),
		},
	}
	if err := store.SaveUnrankedGame(ctx, abortedGame, "technical_abort", localTime(2, 10).Add(10*time.Minute)); err != nil {
		t.Fatalf("SaveUnrankedGame() error = %v", err)
	}

	insertAnalyticsBugReport(t, ctx, store, users[0].ID, localTime(1, 11), localTime(1, 13))
	insertAnalyticsBugReport(t, ctx, store, users[1].ID, localTime(2, 11), time.Time{})
	insertAnalyticsBugReport(t, ctx, store, users[2].ID, localTime(0, 11), localTime(0, 12))

	got, err := store.GetAdminAnalytics(ctx, AdminAnalyticsRange{
		From:         localTime(1, 0),
		To:           localTime(4, 0),
		PreviousFrom: time.Date(2026, time.June, 28, 0, 0, 0, 0, riga),
		PreviousTo:   localTime(1, 0),
	})
	if err != nil {
		t.Fatalf("GetAdminAnalytics() error = %v", err)
	}

	if got.Current.Games != 2 || got.Current.ActivePlayers != 2 || got.Current.ActivePlaytimeSeconds != 2400 {
		t.Fatalf("current activity totals = %#v", got.Current)
	}
	if math.Abs(got.Current.HealthyFinishRate) > 0.0001 {
		t.Fatalf("current healthy finish rate = %v; want 0", got.Current.HealthyFinishRate)
	}
	if got.Current.BugReports != 2 || got.Current.BugsResolved != 1 || got.Current.MedianBugResolutionSeconds != 7200 {
		t.Fatalf("current bug totals = %#v", got.Current)
	}
	if got.Previous.Games != 1 || got.Previous.ActivePlayers != 2 || got.Previous.ActivePlaytimeSeconds != 1200 {
		t.Fatalf("previous activity totals = %#v", got.Previous)
	}
	if got.Previous.BugReports != 1 || got.Previous.BugsResolved != 1 || got.Previous.MedianBugResolutionSeconds != 3600 {
		t.Fatalf("previous bug totals = %#v", got.Previous)
	}

	if len(got.Points) != 3 {
		t.Fatalf("daily point count = %d; want 3", len(got.Points))
	}
	first := got.Points[0]
	if first.Date != "2026-07-01" || first.Games != 1 || first.ActivePlayers != 2 ||
		first.NewPlayers != 1 || first.ReturningPlayers != 1 || first.Forfeit != 1 ||
		first.BugReports != 1 || first.BugsResolved != 1 {
		t.Fatalf("first daily point = %#v", first)
	}
	second := got.Points[1]
	if second.Date != "2026-07-02" || second.Games != 1 || second.ActivePlayers != 1 ||
		second.NewPlayers != 0 || second.ReturningPlayers != 1 || second.TechnicalAbort != 1 ||
		second.BugReports != 1 || second.BugsResolved != 0 {
		t.Fatalf("second daily point = %#v", second)
	}
	if third := got.Points[2]; third.Date != "2026-07-03" || third.Games != 0 || third.ActivePlayers != 0 {
		t.Fatalf("third daily point = %#v", third)
	}
}

func analyticsTestPlayer(userID string, placement int, won bool) CompletedGamePlayerRecord {
	return CompletedGamePlayerRecord{
		UserID:       userID,
		Placement:    placement,
		Won:          won,
		RoundsPlayed: 1,
	}
}

func insertAnalyticsBugReport(t *testing.T, ctx context.Context, store *UserStore, userID string, createdAt, completedAt time.Time) {
	t.Helper()
	var completed any
	if !completedAt.IsZero() {
		completed = completedAt
	}
	if _, err := store.pool.Exec(ctx, `
		INSERT INTO game_bug_reports (
			room_code, reporter_player_id, reporter_user_id, description,
			game_state, round, turn, created_at, completed_at
		) VALUES ('ANALYTICS', 'player', $1, 'Analytics test report', '{}'::jsonb, 1, 1, $2, $3)
	`, userID, createdAt, completed); err != nil {
		t.Fatalf("insert analytics bug report error = %v", err)
	}
}
