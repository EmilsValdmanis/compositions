package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/EmilsValdmanis/compositions/internal/database"
)

func TestNewConfiguredUserStore(t *testing.T) {
	originalDatabaseURLFromEnv := databaseURLFromEnv
	defer func() { databaseURLFromEnv = originalDatabaseURLFromEnv }()
	originalNewDatabaseUserStore := newDatabaseUserStore
	defer func() { newDatabaseUserStore = originalNewDatabaseUserStore }()

	t.Run("returns env error", func(t *testing.T) {
		databaseURLFromEnv = func() (string, error) {
			return "", errors.New("missing database url")
		}

		store, err := newConfiguredUserStore()
		if err == nil || err.Error() != "missing database url" {
			t.Fatalf("newConfiguredUserStore() error = %v; want missing database url", err)
		}
		if store != nil {
			t.Fatalf("store = %#v; want nil", store)
		}
	})

	t.Run("returns constructor error", func(t *testing.T) {
		databaseURLFromEnv = func() (string, error) {
			return "postgres://configured", nil
		}
		newDatabaseUserStore = func(ctx context.Context, databaseURL string) (*database.UserStore, error) {
			if databaseURL != "postgres://configured" {
				t.Fatalf("databaseURL = %q; want postgres://configured", databaseURL)
			}
			if _, ok := ctx.Deadline(); !ok {
				t.Fatal("ctx has no deadline; want timeout")
			}
			return nil, errors.New("open store boom")
		}

		store, err := newConfiguredUserStore()
		if err == nil || err.Error() != "open store boom" {
			t.Fatalf("newConfiguredUserStore() error = %v; want open store boom", err)
		}
		if store != nil {
			t.Fatalf("store = %#v; want nil", store)
		}
	})

	t.Run("returns postgres store", func(t *testing.T) {
		databaseURLFromEnv = func() (string, error) {
			return "postgres://configured", nil
		}
		newDatabaseUserStore = func(context.Context, string) (*database.UserStore, error) {
			return &database.UserStore{}, nil
		}

		store, err := newConfiguredUserStore()
		if err != nil {
			t.Fatalf("newConfiguredUserStore() error = %v", err)
		}
		postgresStore, ok := store.(*postgresUserStore)
		if !ok {
			t.Fatalf("store type = %T; want *postgresUserStore", store)
		}
		if postgresStore.store == nil {
			t.Fatal("postgresStore.store = nil; want configured database store")
		}
	})
}

func TestPostgresUserStoreUpsertUser(t *testing.T) {
	originalUpsertStoredUser := upsertStoredUser
	defer func() { upsertStoredUser = originalUpsertStoredUser }()

	t.Run("requires configured store", func(t *testing.T) {
		var store *postgresUserStore
		if _, err := store.UpsertUser(context.Background(), authenticatedUser{}); err == nil || err.Error() != "user store is not configured" {
			t.Fatalf("UpsertUser() error = %v; want user store is not configured", err)
		}
	})

	t.Run("trims fields before upsert", func(t *testing.T) {
		called := false
		upsertStoredUser = func(ctx context.Context, store *database.UserStore, user database.UserRecord) (database.UserRecord, error) {
			called = true
			if store == nil {
				t.Fatal("store = nil; want configured store")
			}
			if user.ID != "user-1" || user.Name != "Player One" || user.Email != "player@example.com" || user.ImageURL != "https://cdn.example.com/player.png" {
				t.Fatalf("user = %#v; want trimmed fields", user)
			}
			return user, nil
		}

		store := &postgresUserStore{store: &database.UserStore{}}
		storedUser, err := store.UpsertUser(context.Background(), authenticatedUser{
			ID:    " user-1 ",
			Name:  " Player One ",
			Email: " Player@Example.com ",
			Image: " https://cdn.example.com/player.png ",
		})
		if err != nil {
			t.Fatalf("UpsertUser() error = %v", err)
		}
		if storedUser.ID != "user-1" {
			t.Fatalf("stored user id = %q; want user-1", storedUser.ID)
		}
		if !called {
			t.Fatal("upsertStoredUser was not called")
		}
	})

	t.Run("returns upsert error", func(t *testing.T) {
		upsertStoredUser = func(context.Context, *database.UserStore, database.UserRecord) (database.UserRecord, error) {
			return database.UserRecord{}, errors.New("upsert boom")
		}

		store := &postgresUserStore{store: &database.UserStore{}}
		if _, err := store.UpsertUser(context.Background(), authenticatedUser{ID: "user-1"}); err == nil || err.Error() != "upsert boom" {
			t.Fatalf("UpsertUser() error = %v; want upsert boom", err)
		}
	})
}

func TestPostgresUserStoreDefaultDatabaseWrappers(t *testing.T) {
	store := &database.UserStore{}
	if _, err := upsertStoredUser(context.Background(), store, database.UserRecord{}); err == nil || err.Error() != "user store is not configured" {
		t.Fatalf("upsertStoredUser(default) error = %v; want user store is not configured", err)
	}
	if err := saveStoredLobbyState(context.Background(), store, nil); err == nil || err.Error() != "user store is not configured" {
		t.Fatalf("saveStoredLobbyState(default) error = %v; want user store is not configured", err)
	}
	if _, err := loadStoredLobbyState(context.Background(), store); err == nil || err.Error() != "user store is not configured" {
		t.Fatalf("loadStoredLobbyState(default) error = %v; want user store is not configured", err)
	}
	if _, err := createStoredGameBugReport(context.Background(), store, database.GameBugReportRecord{}); err == nil || err.Error() != "user store is not configured" {
		t.Fatalf("createStoredGameBugReport(default) error = %v; want user store is not configured", err)
	}
	closeStoredUserStore(store)
}

func TestPostgresUserStoreGameBugReports(t *testing.T) {
	originalCreateStoredGameBugReport := createStoredGameBugReport
	defer func() { createStoredGameBugReport = originalCreateStoredGameBugReport }()

	t.Run("requires configured store", func(t *testing.T) {
		var store *postgresUserStore
		if _, err := store.CreateGameBugReport(context.Background(), database.GameBugReportRecord{}); err == nil || err.Error() != "user store is not configured" {
			t.Fatalf("CreateGameBugReport(nil) error = %v", err)
		}
	})

	t.Run("delegates to database store", func(t *testing.T) {
		want := database.GameBugReportRecord{ID: "report-1", Description: "Broken turn"}
		createStoredGameBugReport = func(_ context.Context, store *database.UserStore, report database.GameBugReportRecord) (database.GameBugReportRecord, error) {
			if store == nil || report.ID != want.ID || report.Description != want.Description {
				t.Fatalf("CreateGameBugReport arguments store=%v report=%#v", store, report)
			}
			return report, nil
		}
		store := &postgresUserStore{store: &database.UserStore{}}
		got, err := store.CreateGameBugReport(context.Background(), want)
		if err != nil {
			t.Fatalf("CreateGameBugReport() error = %v", err)
		}
		if got.ID != want.ID {
			t.Fatalf("CreateGameBugReport() = %#v; want %#v", got, want)
		}
	})
}

func TestPostgresUserStoreGetPlayerProfile(t *testing.T) {
	var nilStore *postgresUserStore
	if _, err := nilStore.GetPlayerProfile(context.Background(), "player-id"); err == nil || err.Error() != "user store is not configured" {
		t.Fatalf("nil GetPlayerProfile() error = %v; want configuration error", err)
	}

	store := &postgresUserStore{store: &database.UserStore{}}
	if _, err := store.GetPlayerProfile(context.Background(), " player-id "); err == nil || err.Error() != "user store is not configured" {
		t.Fatalf("GetPlayerProfile() error = %v; want underlying store error", err)
	}
}

func TestPostgresUserStoreGetLeaderboard(t *testing.T) {
	var nilStore *postgresUserStore
	if _, err := nilStore.GetLeaderboard(context.Background(), nil, leaderboardPageSize, "player-id", database.LeaderboardMetricWins); err == nil || err.Error() != "user store is not configured" {
		t.Fatalf("nil GetLeaderboard() error = %v; want configuration error", err)
	}

	store := &postgresUserStore{store: &database.UserStore{}}
	if _, err := store.GetLeaderboard(context.Background(), nil, leaderboardPageSize, " player-id ", database.LeaderboardMetricWins); err == nil || err.Error() != "user store is not configured" {
		t.Fatalf("GetLeaderboard() error = %v; want underlying store error", err)
	}
}

func TestPostgresUserStoreLobbyState(t *testing.T) {
	originalSaveStoredLobbyState := saveStoredLobbyState
	defer func() { saveStoredLobbyState = originalSaveStoredLobbyState }()
	originalLoadStoredLobbyState := loadStoredLobbyState
	defer func() { loadStoredLobbyState = originalLoadStoredLobbyState }()

	t.Run("requires configured store", func(t *testing.T) {
		var store *postgresUserStore
		if err := store.SaveLobbyState(context.Background(), persistedLobbyState{}); err == nil || err.Error() != "user store is not configured" {
			t.Fatalf("SaveLobbyState(nil) error = %v; want user store is not configured", err)
		}
		if _, err := store.LoadLobbyState(context.Background()); err == nil || err.Error() != "user store is not configured" {
			t.Fatalf("LoadLobbyState(nil) error = %v; want user store is not configured", err)
		}
	})

	t.Run("saves marshaled state", func(t *testing.T) {
		saveStoredLobbyState = func(ctx context.Context, store *database.UserStore, data []byte) error {
			if store == nil {
				t.Fatal("store = nil; want configured store")
			}
			var state persistedLobbyState
			if err := json.Unmarshal(data, &state); err != nil {
				t.Fatalf("Unmarshal(saved data) error = %v", err)
			}
			if state.Version != persistedLobbyStateVersion {
				t.Fatalf("saved version = %d; want %d", state.Version, persistedLobbyStateVersion)
			}
			return nil
		}

		store := &postgresUserStore{store: &database.UserStore{}}
		if err := store.SaveLobbyState(context.Background(), persistedLobbyState{Version: persistedLobbyStateVersion}); err != nil {
			t.Fatalf("SaveLobbyState() error = %v", err)
		}
	})

	t.Run("returns save error", func(t *testing.T) {
		saveStoredLobbyState = func(context.Context, *database.UserStore, []byte) error {
			return errors.New("save boom")
		}

		store := &postgresUserStore{store: &database.UserStore{}}
		if err := store.SaveLobbyState(context.Background(), persistedLobbyState{}); err == nil || err.Error() != "save boom" {
			t.Fatalf("SaveLobbyState() error = %v; want save boom", err)
		}
	})

	t.Run("maps not found to empty state", func(t *testing.T) {
		loadStoredLobbyState = func(context.Context, *database.UserStore) ([]byte, error) {
			return nil, database.ErrLobbyStateNotFound
		}

		store := &postgresUserStore{store: &database.UserStore{}}
		state, err := store.LoadLobbyState(context.Background())
		if err != nil {
			t.Fatalf("LoadLobbyState() error = %v", err)
		}
		if state.Version != 0 || len(state.Sessions) != 0 || len(state.Rooms) != 0 {
			t.Fatalf("LoadLobbyState() = %#v; want empty state", state)
		}
	})

	t.Run("returns load error", func(t *testing.T) {
		loadStoredLobbyState = func(context.Context, *database.UserStore) ([]byte, error) {
			return nil, errors.New("load boom")
		}

		store := &postgresUserStore{store: &database.UserStore{}}
		if _, err := store.LoadLobbyState(context.Background()); err == nil || err.Error() != "load boom" {
			t.Fatalf("LoadLobbyState() error = %v; want load boom", err)
		}
	})

	t.Run("rejects invalid json", func(t *testing.T) {
		loadStoredLobbyState = func(context.Context, *database.UserStore) ([]byte, error) {
			return []byte("{"), nil
		}

		store := &postgresUserStore{store: &database.UserStore{}}
		if _, err := store.LoadLobbyState(context.Background()); err == nil {
			t.Fatal("LoadLobbyState(invalid json) error = nil; want error")
		}
	})

	t.Run("loads state", func(t *testing.T) {
		loadStoredLobbyState = func(context.Context, *database.UserStore) ([]byte, error) {
			return json.Marshal(persistedLobbyState{Version: persistedLobbyStateVersion})
		}

		store := &postgresUserStore{store: &database.UserStore{}}
		state, err := store.LoadLobbyState(context.Background())
		if err != nil {
			t.Fatalf("LoadLobbyState() error = %v", err)
		}
		if state.Version != persistedLobbyStateVersion {
			t.Fatalf("LoadLobbyState().Version = %d; want %d", state.Version, persistedLobbyStateVersion)
		}
	})
}

func TestPostgresUserStoreClose(t *testing.T) {
	originalCloseStoredUserStore := closeStoredUserStore
	defer func() { closeStoredUserStore = originalCloseStoredUserStore }()

	var nilStore *postgresUserStore
	if err := nilStore.Close(); err != nil {
		t.Fatalf("nilStore.Close() error = %v", err)
	}

	store := &postgresUserStore{}
	if err := store.Close(); err != nil {
		t.Fatalf("store.Close() error = %v", err)
	}

	closed := false
	closeStoredUserStore = func(store *database.UserStore) {
		closed = true
		if store == nil {
			t.Fatal("store = nil; want configured store")
		}
	}

	store = &postgresUserStore{store: &database.UserStore{}}
	if err := store.Close(); err != nil {
		t.Fatalf("store.Close() error = %v", err)
	}
	if !closed {
		t.Fatal("closeStoredUserStore was not called")
	}
}

func TestPostgresUserStoreStatisticsMethods(t *testing.T) {
	ctx := context.Background()
	var nilStore *postgresUserStore
	if err := nilStore.SaveGameCheckpoint(ctx, database.GameCheckpointRecord{}); err == nil {
		t.Fatal("nil SaveGameCheckpoint() error = nil")
	}
	if err := nilStore.SaveUnrankedGame(ctx, database.GameCheckpointRecord{}, "mutual_end", time.Now()); err == nil {
		t.Fatal("nil SaveUnrankedGame() error = nil")
	}

	store := &postgresUserStore{store: &database.UserStore{}}
	if err := store.SaveGameCheckpoint(ctx, database.GameCheckpointRecord{}); err == nil {
		t.Fatal("unconfigured SaveGameCheckpoint() error = nil")
	}
	if err := store.SaveUnrankedGame(ctx, database.GameCheckpointRecord{}, "mutual_end", time.Now()); err == nil {
		t.Fatal("unconfigured SaveUnrankedGame() error = nil")
	}
}
