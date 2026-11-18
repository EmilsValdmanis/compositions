package main

import (
	"context"
	"encoding/json"
	"time"

	"github.com/EmilsValdmanis/compositions/internal/database"
)

const defaultUserStoreTimeout = 5 * time.Second

var openConfiguredUserStore = newConfiguredUserStore
var newDatabaseUserStore = database.NewUserStore

type userStore interface {
	authStore
	Close() error
	SaveLobbyState(ctx context.Context, state json.RawMessage) error
	LoadLobbyState(ctx context.Context) (json.RawMessage, error)
	CreateGameBugReport(ctx context.Context, report database.GameBugReportRecord) (database.GameBugReportRecord, error)
}

var _ leaderboardStore = (*database.UserStore)(nil)
var _ adminBugReportStore = (*database.UserStore)(nil)
var _ adminAnalyticsStore = (*database.UserStore)(nil)

func newConfiguredUserStore() (userStore, error) {
	databaseURL, err := database.URLFromEnv()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultUserStoreTimeout)
	defer cancel()
	store, err := newDatabaseUserStore(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	return store, nil
}
