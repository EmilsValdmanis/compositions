//go:build integration

package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	migratepgx "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestUserStoreUpsertUser(t *testing.T) {
	ctx := context.Background()
	databaseURL := startPostgresContainer(t, ctx)

	pool, err := OpenPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("OpenPool() error = %v", err)
	}
	defer pool.Close()

	if err := RunMigrations(ctx, databaseURL, MigrationUp); err != nil {
		t.Fatalf("RunMigrations(up) error = %v", err)
	}
	if err := RunMigrations(ctx, databaseURL, MigrationUp); err != nil {
		t.Fatalf("RunMigrations(up second pass) error = %v", err)
	}

	store, err := NewUserStore(ctx, databaseURL)
	if err != nil {
		t.Fatalf("NewUserStore() error = %v", err)
	}
	defer store.Close()

	createdUser := UserRecord{
		ID:       uuid.NewString(),
		Name:     "Player One",
		Email:    "player1@example.com",
		ImageURL: "https://cdn.example.com/player-1.png",
	}
	storedCreatedUser, err := store.UpsertUser(ctx, createdUser)
	if err != nil {
		t.Fatalf("UpsertUser(create) error = %v", err)
	}
	if storedCreatedUser.ID != createdUser.ID {
		t.Fatalf("stored created id = %q; want %q", storedCreatedUser.ID, createdUser.ID)
	}

	createdRecord, err := store.GetUserByID(ctx, createdUser.ID)
	if err != nil {
		t.Fatalf("GetUserByID(create) error = %v", err)
	}
	if createdRecord.Name != createdUser.Name {
		t.Fatalf("created name = %q; want %q", createdRecord.Name, createdUser.Name)
	}
	if createdRecord.Email != createdUser.Email {
		t.Fatalf("created email = %q; want %q", createdRecord.Email, createdUser.Email)
	}
	if createdRecord.ImageURL != createdUser.ImageURL {
		t.Fatalf("created image_url = %q; want %q", createdRecord.ImageURL, createdUser.ImageURL)
	}

	updatedUser := UserRecord{
		ID:       createdUser.ID,
		Name:     "Updated Player",
		Email:    "player1@example.com",
		ImageURL: "https://cdn.example.com/player-1-updated.png",
	}
	storedUpdatedUser, err := store.UpsertUser(ctx, updatedUser)
	if err != nil {
		t.Fatalf("UpsertUser(update) error = %v", err)
	}
	if storedUpdatedUser.ID != updatedUser.ID {
		t.Fatalf("stored updated id = %q; want %q", storedUpdatedUser.ID, updatedUser.ID)
	}

	updatedRecord, err := store.GetUserByID(ctx, updatedUser.ID)
	if err != nil {
		t.Fatalf("GetUserByID(update) error = %v", err)
	}
	if updatedRecord.Name != updatedUser.Name {
		t.Fatalf("updated name = %q; want %q", updatedRecord.Name, updatedUser.Name)
	}
	if updatedRecord.Email != updatedUser.Email {
		t.Fatalf("updated email = %q; want %q", updatedRecord.Email, updatedUser.Email)
	}
	if updatedRecord.ImageURL != updatedUser.ImageURL {
		t.Fatalf("updated image_url = %q; want %q", updatedRecord.ImageURL, updatedUser.ImageURL)
	}

	conflictingUser := UserRecord{
		ID:       uuid.NewString(),
		Name:     " Conflicting Player ",
		Email:    " PLAYER1@EXAMPLE.COM ",
		ImageURL: " https://cdn.example.com/player-2.png ",
	}
	if _, err := store.UpsertUser(ctx, conflictingUser); !errors.Is(err, ErrUserConflict) {
		t.Fatalf("UpsertUser(conflicting) error = %v; want %v", err, ErrUserConflict)
	}

	if got := countUsers(t, ctx, pool); got != 1 {
		t.Fatalf("user row count = %d; want 1", got)
	}

	preservedRecord, err := store.GetUserByID(ctx, createdUser.ID)
	if err != nil {
		t.Fatalf("GetUserByID(preserved) error = %v", err)
	}
	if preservedRecord.Name != updatedUser.Name {
		t.Fatalf("preserved name = %q; want %q", preservedRecord.Name, updatedUser.Name)
	}
	if preservedRecord.Email != updatedUser.Email {
		t.Fatalf("preserved email = %q; want %q", preservedRecord.Email, updatedUser.Email)
	}
	if preservedRecord.ImageURL != updatedUser.ImageURL {
		t.Fatalf("preserved image_url = %q; want %q", preservedRecord.ImageURL, updatedUser.ImageURL)
	}
	if _, err := store.GetUserByID(ctx, strings.TrimSpace(conflictingUser.ID)); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("GetUserByID(conflicting id) error = %v; want %v", err, pgx.ErrNoRows)
	}
}

func TestUserStoreUpsertUserByAccount(t *testing.T) {
	ctx := context.Background()
	databaseURL := startPostgresContainer(t, ctx)

	pool, err := OpenPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("OpenPool() error = %v", err)
	}
	defer pool.Close()

	if err := RunMigrations(ctx, databaseURL, MigrationUp); err != nil {
		t.Fatalf("RunMigrations(up) error = %v", err)
	}

	store, err := NewUserStore(ctx, databaseURL)
	if err != nil {
		t.Fatalf("NewUserStore() error = %v", err)
	}
	defer store.Close()

	createdUser, err := store.UpsertUser(ctx, UserRecord{
		Name:              "Player One",
		Email:             "player@example.com",
		ImageURL:          "https://cdn.example.com/player.png",
		Provider:          "google",
		ProviderAccountID: "123",
	})
	if err != nil {
		t.Fatalf("UpsertUser(create account user) error = %v", err)
	}
	if _, err := uuid.Parse(createdUser.ID); err != nil {
		t.Fatalf("created user id = %q; want uuid: %v", createdUser.ID, err)
	}

	updatedUser, err := store.UpsertUser(ctx, UserRecord{
		Name:              "Updated Player",
		Email:             "player@example.com",
		ImageURL:          "https://cdn.example.com/player-updated.png",
		Provider:          "google",
		ProviderAccountID: "123",
	})
	if err != nil {
		t.Fatalf("UpsertUser(update account user) error = %v", err)
	}
	if updatedUser.ID != createdUser.ID {
		t.Fatalf("updated user id = %q; want %q", updatedUser.ID, createdUser.ID)
	}

	record, err := store.GetUserByID(ctx, createdUser.ID)
	if err != nil {
		t.Fatalf("GetUserByID() error = %v", err)
	}
	if record.Name != "Updated Player" || record.ImageURL != "https://cdn.example.com/player-updated.png" {
		t.Fatalf("stored record = %#v; want updated account-linked user", record)
	}

	var accountCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM accounts WHERE provider = $1 AND provider_account_id = $2`, "google", "123").Scan(&accountCount); err != nil {
		t.Fatalf("count account rows error = %v", err)
	}
	if accountCount != 1 {
		t.Fatalf("google account rows = %d; want 1", accountCount)
	}
}

func TestUserStoreLobbyState(t *testing.T) {
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

	if _, err := store.LoadLobbyState(ctx); !errors.Is(err, ErrLobbyStateNotFound) {
		t.Fatalf("LoadLobbyState(empty) error = %v; want %v", err, ErrLobbyStateNotFound)
	}
	if err := store.SaveLobbyState(ctx, json.RawMessage(`{"version":1,"rooms":[{"code":"ABC123"}]}`)); err != nil {
		t.Fatalf("SaveLobbyState(first) error = %v", err)
	}
	loaded, err := store.LoadLobbyState(ctx)
	if err != nil {
		t.Fatalf("LoadLobbyState(first) error = %v", err)
	}
	var loadedFirst struct {
		Version int `json:"version"`
		Rooms   []struct {
			Code string `json:"code"`
		} `json:"rooms"`
	}
	if err := json.Unmarshal(loaded, &loadedFirst); err != nil {
		t.Fatalf("Unmarshal(first lobby state) error = %v", err)
	}
	if loadedFirst.Version != 1 || len(loadedFirst.Rooms) != 1 || loadedFirst.Rooms[0].Code != "ABC123" {
		t.Fatalf("loaded first state = %#v; want one ABC123 room", loadedFirst)
	}

	if err := store.SaveLobbyState(ctx, json.RawMessage(`{"version":1,"rooms":[]}`)); err != nil {
		t.Fatalf("SaveLobbyState(update) error = %v", err)
	}
	loaded, err = store.LoadLobbyState(ctx)
	if err != nil {
		t.Fatalf("LoadLobbyState(update) error = %v", err)
	}
	var loadedUpdated struct {
		Version int               `json:"version"`
		Rooms   []json.RawMessage `json:"rooms"`
	}
	if err := json.Unmarshal(loaded, &loadedUpdated); err != nil {
		t.Fatalf("Unmarshal(updated lobby state) error = %v", err)
	}
	if loadedUpdated.Version != 1 || len(loadedUpdated.Rooms) != 0 {
		t.Fatalf("loaded updated state = %#v; want empty room list", loadedUpdated)
	}
}

func TestUserStoreSaveCompletedGameIsIdempotentAndUpdatesLifetimeStatistics(t *testing.T) {
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
	user, err := store.UpsertUser(ctx, UserRecord{ID: uuid.NewString(), Name: "Stats Player"})
	if err != nil {
		t.Fatalf("UpsertUser() error = %v", err)
	}

	started := time.Now().UTC().Add(-time.Hour)
	first := CompletedGameRecord{
		ID: uuid.NewString(), RoomCode: "ABC123", CompletionKind: "normal", RoundsPlayed: 2,
		PlayerCount: 2, StartedAt: started, CompletedAt: started.Add(30 * time.Minute),
		Players: []CompletedGamePlayerRecord{{
			UserID: user.ID, Placement: 1, Won: true, RoundsPlayed: 2, RoundsWon: 2,
			TurnsTaken: 8, CardsPlayed: 20, PointsInflicted: 80, FastestOpeningTurn: 2,
			StartingRoundWinStreak: 2, EndingRoundWinStreak: 2, LongestRoundWinStreak: 2,
		}},
	}
	checkpoint := GameCheckpointRecord{
		ID: first.ID, RoomCode: first.RoomCode, RoundsPlayed: 1, PlayerCount: 2, StartedAt: started,
		Players: []CompletedGamePlayerRecord{{UserID: user.ID, TurnsTaken: 2, CardsPlayed: 5}},
	}
	if err := store.SaveGameCheckpoint(ctx, checkpoint); err != nil {
		t.Fatalf("SaveGameCheckpoint(first) error = %v", err)
	}
	if err := store.SaveGameCheckpoint(ctx, checkpoint); err != nil {
		t.Fatalf("SaveGameCheckpoint(retry) error = %v", err)
	}
	var checkpointStatus string
	var checkpointTurns int
	var checkpointPlacement pgtype.Int4
	var checkpointWon pgtype.Bool
	err = store.pool.QueryRow(ctx, `
		SELECT g.status, gps.turns_taken, gps.placement, gps.won
		FROM games g JOIN game_player_statistics gps ON gps.game_id = g.id
		WHERE g.id = $1 AND gps.user_id = $2
	`, first.ID, user.ID).Scan(&checkpointStatus, &checkpointTurns, &checkpointPlacement, &checkpointWon)
	if err != nil || checkpointStatus != "in_progress" || checkpointTurns != 2 || checkpointPlacement.Valid || checkpointWon.Valid {
		t.Fatalf("checkpoint = status:%q turns:%d placement:%+v won:%+v error:%v", checkpointStatus, checkpointTurns, checkpointPlacement, checkpointWon, err)
	}
	if err := store.SaveCompletedGame(ctx, first); err != nil {
		t.Fatalf("SaveCompletedGame(first) error = %v", err)
	}
	var finalizedStatus string
	if err := store.pool.QueryRow(ctx, `SELECT status FROM games WHERE id = $1`, first.ID).Scan(&finalizedStatus); err != nil || finalizedStatus != "completed" {
		t.Fatalf("finalized status = %q, error = %v", finalizedStatus, err)
	}
	if err := store.SaveCompletedGame(ctx, first); err != nil {
		t.Fatalf("SaveCompletedGame(retry) error = %v", err)
	}

	second := CompletedGameRecord{
		ID: uuid.NewString(), RoomCode: "XYZ789", CompletionKind: "normal", RoundsPlayed: 3,
		PlayerCount: 2, StartedAt: started.Add(time.Hour), CompletedAt: started.Add(2 * time.Hour),
		Players: []CompletedGamePlayerRecord{{
			UserID: user.ID, Placement: 2, RoundsPlayed: 3, RoundsWon: 1, Forfeited: true,
			TurnsTaken: 5, CardsPlayed: 10, FastestOpeningTurn: 3,
			StartingRoundWinStreak: 1, EndingRoundWinStreak: 0, LongestRoundWinStreak: 1,
		}},
	}
	if err := store.SaveCompletedGame(ctx, second); err != nil {
		t.Fatalf("SaveCompletedGame(second) error = %v", err)
	}

	var gamesPlayed, gamesWon, totalPlacement, roundsPlayed, roundsWon, forfeits int
	var turnsTaken, cardsPlayed, pointsInflicted int
	var fastestOpening, currentGameStreak, longestGameStreak, currentRoundStreak, longestRoundStreak int
	err = store.pool.QueryRow(ctx, `
		SELECT games_played, games_won, total_placement, rounds_played, rounds_won, forfeits,
		       turns_taken, cards_played, points_inflicted, fastest_opening_turn,
		       current_game_win_streak, longest_game_win_streak, current_round_win_streak, longest_round_win_streak
		FROM player_statistics WHERE user_id = $1
	`, user.ID).Scan(&gamesPlayed, &gamesWon, &totalPlacement, &roundsPlayed, &roundsWon, &forfeits,
		&turnsTaken, &cardsPlayed, &pointsInflicted, &fastestOpening,
		&currentGameStreak, &longestGameStreak, &currentRoundStreak, &longestRoundStreak)
	if err != nil {
		t.Fatalf("query player_statistics error = %v", err)
	}
	got := []int{gamesPlayed, gamesWon, totalPlacement, roundsPlayed, roundsWon, forfeits, turnsTaken,
		cardsPlayed, pointsInflicted, fastestOpening, currentGameStreak, longestGameStreak, currentRoundStreak, longestRoundStreak}
	want := []int{2, 1, 3, 5, 3, 1, 13, 30, 80, 2, 0, 1, 0, 3}
	if !slices.Equal(got, want) {
		t.Fatalf("lifetime statistics = %v; want %v", got, want)
	}

	unranked := GameCheckpointRecord{
		ID: uuid.NewString(), RoomCode: "BUG123", RoundsPlayed: 2, PlayerCount: 2, StartedAt: started,
		Players: []CompletedGamePlayerRecord{{UserID: user.ID, Forfeited: true, TurnsTaken: 7, CardsPlayed: 14}},
	}
	if err := store.SaveGameCheckpoint(ctx, unranked); err != nil {
		t.Fatalf("SaveGameCheckpoint(unranked) error = %v", err)
	}
	if err := store.SaveUnrankedGame(ctx, unranked, "technical_abort", started.Add(3*time.Hour)); err != nil {
		t.Fatalf("SaveUnrankedGame() error = %v", err)
	}
	if err := store.SaveUnrankedGame(ctx, unranked, "technical_abort", started.Add(3*time.Hour)); err != nil {
		t.Fatalf("SaveUnrankedGame(retry) error = %v", err)
	}
	stale := unranked
	stale.Players = []CompletedGamePlayerRecord{{UserID: user.ID, TurnsTaken: 1}}
	if err := store.SaveGameCheckpoint(ctx, stale); err != nil {
		t.Fatalf("SaveGameCheckpoint(after finalization) error = %v", err)
	}
	var unrankedStatus string
	var unrankedForfeited bool
	var unrankedTurns int
	var unrankedPlacement pgtype.Int4
	err = store.pool.QueryRow(ctx, `
		SELECT g.status, gps.forfeited, gps.turns_taken, gps.placement
		FROM games g JOIN game_player_statistics gps ON gps.game_id = g.id
		WHERE g.id = $1 AND gps.user_id = $2
	`, unranked.ID, user.ID).Scan(&unrankedStatus, &unrankedForfeited, &unrankedTurns, &unrankedPlacement)
	if err != nil || unrankedStatus != "technical_abort" || !unrankedForfeited || unrankedTurns != 7 || unrankedPlacement.Valid {
		t.Fatalf("unranked checkpoint = status:%q forfeited:%v turns:%d placement:%+v error:%v", unrankedStatus, unrankedForfeited, unrankedTurns, unrankedPlacement, err)
	}
	var lifetimeGames int
	if err := store.pool.QueryRow(ctx, `SELECT games_played FROM player_statistics WHERE user_id = $1`, user.ID).Scan(&lifetimeGames); err != nil || lifetimeGames != 2 {
		t.Fatalf("lifetime games after unranked finalization = %d, error = %v", lifetimeGames, err)
	}
}

func TestStatisticsCheckpointMigrationUpgradesCompletedGames(t *testing.T) {
	ctx := context.Background()
	databaseURL := startPostgresContainer(t, ctx)
	migrationDB, err := OpenMigrationDB(ctx, databaseURL)
	if err != nil {
		t.Fatalf("OpenMigrationDB() error = %v", err)
	}
	defer migrationDB.Close()
	migrator := newTestMigrator(t, migrationDB)
	if err := migrator.Steps(7); err != nil {
		t.Fatalf("migrator.Steps(7) error = %v", err)
	}
	pool, err := OpenPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("OpenPool() error = %v", err)
	}
	defer pool.Close()
	gameID := uuid.NewString()
	if _, err := pool.Exec(ctx, `
		INSERT INTO games (id, room_code, completion_kind, rounds_played, player_count, started_at, completed_at)
		VALUES ($1, 'OLD123', 'normal', 2, 2, NOW() - INTERVAL '1 hour', NOW())
	`, gameID); err != nil {
		t.Fatalf("seed migration-7 game error = %v", err)
	}
	if err := migrator.Steps(1); err != nil {
		t.Fatalf("migrator.Steps(checkpoints) error = %v", err)
	}
	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM games WHERE id = $1`, gameID).Scan(&status); err != nil || status != "completed" {
		t.Fatalf("upgraded game status = %q, error = %v", status, err)
	}
	checkpointID := uuid.NewString()
	if _, err := pool.Exec(ctx, `
		INSERT INTO games (id, room_code, status, rounds_played, player_count, started_at)
		VALUES ($1, 'NEW123', 'in_progress', 1, 2, NOW())
	`, checkpointID); err != nil {
		t.Fatalf("insert checkpoint game error = %v", err)
	}
	if err := migrator.Steps(-1); err != nil {
		t.Fatalf("migrator.Steps(-1 checkpoints) error = %v", err)
	}
	var completionKind string
	if err := pool.QueryRow(ctx, `SELECT completion_kind FROM games WHERE id = $1`, gameID).Scan(&completionKind); err != nil || completionKind != "normal" {
		t.Fatalf("downgraded completion kind = %q, error = %v", completionKind, err)
	}
	var checkpointCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM games WHERE id = $1`, checkpointID).Scan(&checkpointCount); err != nil || checkpointCount != 0 {
		t.Fatalf("checkpoint rows after downgrade = %d, error = %v", checkpointCount, err)
	}
}

func TestUserStoreGameBugReports(t *testing.T) {
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

	createdAt := time.Date(2026, 7, 10, 8, 30, 0, 0, time.UTC)
	report := GameBugReportRecord{
		ID:               uuid.NewString(),
		RoomCode:         "abc123",
		ReporterPlayerID: uuid.NewString(),
		Description:      "The discard pile stopped responding",
		GameState:        json.RawMessage(`{"version":1,"phase":1,"round":2}`),
		Round:            2,
		Turn:             9,
		RequestedAbort:   true,
		CreatedAt:        createdAt,
	}
	created, err := store.CreateGameBugReport(ctx, report)
	if err != nil {
		t.Fatalf("CreateGameBugReport() error = %v", err)
	}
	if created.ID != report.ID || created.RoomCode != "ABC123" || !json.Valid(created.GameState) {
		t.Fatalf("created bug report = %#v", created)
	}

	loaded, err := store.GetGameBugReport(ctx, report.ID)
	if err != nil {
		t.Fatalf("GetGameBugReport() error = %v", err)
	}
	if loaded.Description != report.Description || loaded.Round != 2 || loaded.Turn != 9 || !loaded.RequestedAbort {
		t.Fatalf("loaded bug report = %#v", loaded)
	}
	reports, err := store.ListGameBugReports(ctx, 20)
	if err != nil {
		t.Fatalf("ListGameBugReports() error = %v", err)
	}
	if len(reports) != 1 || reports[0].ID != report.ID {
		t.Fatalf("listed bug reports = %#v", reports)
	}
}

func TestUserEmailUniquenessMigrationDeduplicatesExistingRows(t *testing.T) {
	ctx := context.Background()
	databaseURL := startPostgresContainer(t, ctx)

	migrationDB, err := OpenMigrationDB(ctx, databaseURL)
	if err != nil {
		t.Fatalf("OpenMigrationDB() error = %v", err)
	}
	defer migrationDB.Close()

	migrator := newTestMigrator(t, migrationDB)
	if err := migrator.Steps(1); err != nil {
		t.Fatalf("migrator.Steps(1 first migration) error = %v", err)
	}

	pool, err := OpenPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("OpenPool() error = %v", err)
	}
	defer pool.Close()

	if _, err := pool.Exec(ctx, `
		INSERT INTO users (id, name, email, image_url, created_at, updated_at)
		VALUES
			($1, $2, $3, $4, $5, $6),
			($7, $8, $9, $10, $11, $12)
	`,
		"user-1",
		" First Player ",
		"Player@example.com",
		" https://cdn.example.com/player-1.png ",
		time.Date(2026, time.May, 24, 15, 11, 1, 0, time.UTC),
		time.Date(2026, time.May, 24, 15, 11, 56, 0, time.UTC),
		"user-2",
		" Second Player ",
		" PLAYER@EXAMPLE.COM ",
		" https://cdn.example.com/player-2.png ",
		time.Date(2026, time.May, 24, 15, 51, 3, 0, time.UTC),
		time.Date(2026, time.May, 24, 15, 51, 3, 0, time.UTC),
	); err != nil {
		t.Fatalf("seed duplicate users error = %v", err)
	}

	if err := migrator.Steps(1); err != nil {
		t.Fatalf("migrator.Steps(1 second migration) error = %v", err)
	}

	if got := countUsers(t, ctx, pool); got != 1 {
		t.Fatalf("user row count after migration = %d; want 1", got)
	}

	var record struct {
		ID       string
		Name     string
		Email    string
		ImageURL string
	}
	if err := pool.QueryRow(ctx, `
		SELECT id, name, email, image_url
		FROM users
		LIMIT 1
	`).Scan(&record.ID, &record.Name, &record.Email, &record.ImageURL); err != nil {
		t.Fatalf("select deduplicated user error = %v", err)
	}
	if record.ID != "user-2" {
		t.Fatalf("deduplicated id = %q; want %q", record.ID, "user-2")
	}
	if record.Name != "Second Player" {
		t.Fatalf("deduplicated name = %q; want %q", record.Name, "Second Player")
	}
	if record.Email != "player@example.com" {
		t.Fatalf("deduplicated email = %q; want %q", record.Email, "player@example.com")
	}
	if record.ImageURL != "https://cdn.example.com/player-2.png" {
		t.Fatalf("deduplicated image_url = %q; want %q", record.ImageURL, "https://cdn.example.com/player-2.png")
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO users (id, name, email, image_url)
		VALUES ($1, $2, $3, $4)
	`, "user-3", "Third Player", "PLAYER@example.com", "https://cdn.example.com/player-3.png"); err == nil {
		t.Fatal("expected duplicate email insert to fail")
	}
}

func TestUUIDUserMigrationPreservesLegacyGoogleAccounts(t *testing.T) {
	ctx := context.Background()
	databaseURL := startPostgresContainer(t, ctx)

	migrationDB, err := OpenMigrationDB(ctx, databaseURL)
	if err != nil {
		t.Fatalf("OpenMigrationDB() error = %v", err)
	}
	defer migrationDB.Close()

	migrator := newTestMigrator(t, migrationDB)
	if err := migrator.Steps(3); err != nil {
		t.Fatalf("migrator.Steps(3) error = %v", err)
	}

	pool, err := OpenPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("OpenPool() error = %v", err)
	}
	defer pool.Close()

	if _, err := pool.Exec(ctx, `
		INSERT INTO users (id, name, email, image_url)
		VALUES ($1, $2, $3, $4)
	`, "google_123", "Player One", "player@example.com", "https://cdn.example.com/player.png"); err != nil {
		t.Fatalf("seed legacy user error = %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO sessions (token_hash, user_id, expires_at)
		VALUES ($1, $2, $3)
	`, "session-hash", "google_123", time.Date(2026, time.May, 30, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("seed legacy session error = %v", err)
	}

	if err := migrator.Steps(1); err != nil {
		t.Fatalf("migrator.Steps(1 uuid migration) error = %v", err)
	}

	var userID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM users LIMIT 1`).Scan(&userID); err != nil {
		t.Fatalf("select migrated user id error = %v", err)
	}
	if _, err := uuid.Parse(userID); err != nil {
		t.Fatalf("migrated user id = %q; want uuid: %v", userID, err)
	}

	var accountUserID, provider, providerAccountID string
	if err := pool.QueryRow(ctx, `
		SELECT user_id::text, provider, provider_account_id
		FROM accounts
		LIMIT 1
	`).Scan(&accountUserID, &provider, &providerAccountID); err != nil {
		t.Fatalf("select migrated account error = %v", err)
	}
	if accountUserID != userID {
		t.Fatalf("account user id = %q; want %q", accountUserID, userID)
	}
	if provider != "google" || providerAccountID != "123" {
		t.Fatalf("migrated account = (%q, %q); want (google, 123)", provider, providerAccountID)
	}

	var sessionUserID string
	if err := pool.QueryRow(ctx, `SELECT user_id::text FROM sessions WHERE token_hash = $1`, "session-hash").Scan(&sessionUserID); err != nil {
		t.Fatalf("select migrated session error = %v", err)
	}
	if sessionUserID != userID {
		t.Fatalf("session user id = %q; want %q", sessionUserID, userID)
	}
}

func countUsers(t *testing.T, ctx context.Context, pool *pgxpool.Pool) int {
	t.Helper()

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		t.Fatalf("count users error = %v", err)
	}

	return count
}

func newTestMigrator(t *testing.T, db *sql.DB) *migrate.Migrate {
	t.Helper()

	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		t.Fatalf("iofs.New() error = %v", err)
	}

	driver, err := migratepgx.WithInstance(db, &migratepgx.Config{})
	if err != nil {
		t.Fatalf("pgx.WithInstance() error = %v", err)
	}

	migrator, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", driver)
	if err != nil {
		t.Fatalf("migrate.NewWithInstance() error = %v", err)
	}

	t.Cleanup(func() {
		_, _ = migrator.Close()
	})

	return migrator
}

func startPostgresContainer(t *testing.T, ctx context.Context) string {
	t.Helper()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:16-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_DB":       "compositions",
				"POSTGRES_USER":     "postgres",
				"POSTGRES_PASSWORD": "postgres",
			},
			WaitingFor: wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(90 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Fatalf("terminate postgres container: %v", err)
		}
	})

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("container.Host() error = %v", err)
	}
	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("container.MappedPort() error = %v", err)
	}

	return fmt.Sprintf("postgres://postgres:postgres@%s:%s/compositions?sslmode=disable", host, port.Port())
}
