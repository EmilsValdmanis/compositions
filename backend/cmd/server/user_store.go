package main

import (
	"context"
	"time"

	"github.com/EmilsValdmanis/compositions/internal/database"
)

const defaultUserStoreTimeout = 5 * time.Second

var openConfiguredUserStore = newConfiguredUserStore
var databaseURLFromEnv = database.URLFromEnv
var newDatabaseUserStore = database.NewUserStore
var upsertStoredUser = func(ctx context.Context, store *database.UserStore, user database.UserRecord) (database.UserRecord, error) {
	return store.UpsertUser(ctx, user)
}
var closeStoredUserStore = func(store *database.UserStore) {
	store.Close()
}
var saveStoredLobbyState = func(ctx context.Context, store *database.UserStore, data []byte) error {
	return store.SaveLobbyState(ctx, data)
}
var loadStoredLobbyState = func(ctx context.Context, store *database.UserStore) ([]byte, error) {
	return store.LoadLobbyState(ctx)
}
var createStoredGameBugReport = func(ctx context.Context, store *database.UserStore, report database.GameBugReportRecord) (database.GameBugReportRecord, error) {
	return store.CreateGameBugReport(ctx, report)
}

type userStore interface {
	UpsertUser(ctx context.Context, user authenticatedUser) (authenticatedUser, error)
	CreateSession(ctx context.Context, session authSessionRecord) error
	GetSessionUserByToken(ctx context.Context, sessionToken string, now time.Time) (database.SessionUserRecord, error)
	DeleteSession(ctx context.Context, sessionToken string) error
	SaveLobbyState(ctx context.Context, state persistedLobbyState) error
	LoadLobbyState(ctx context.Context) (persistedLobbyState, error)
	CreateGameBugReport(ctx context.Context, report database.GameBugReportRecord) (database.GameBugReportRecord, error)
	Close() error
}

type noopUserStore struct{}

func (noopUserStore) UpsertUser(_ context.Context, user authenticatedUser) (authenticatedUser, error) {
	return user, nil
}

func (noopUserStore) CreateSession(context.Context, authSessionRecord) error { return nil }

func (noopUserStore) GetSessionUserByToken(context.Context, string, time.Time) (database.SessionUserRecord, error) {
	return database.SessionUserRecord{}, database.ErrSessionNotFound
}

func (noopUserStore) DeleteSession(context.Context, string) error { return database.ErrSessionNotFound }

func (noopUserStore) SaveLobbyState(context.Context, persistedLobbyState) error { return nil }

func (noopUserStore) LoadLobbyState(context.Context) (persistedLobbyState, error) {
	return persistedLobbyState{}, nil
}

func (noopUserStore) CreateGameBugReport(_ context.Context, report database.GameBugReportRecord) (database.GameBugReportRecord, error) {
	return report, nil
}

func (noopUserStore) Close() error { return nil }
